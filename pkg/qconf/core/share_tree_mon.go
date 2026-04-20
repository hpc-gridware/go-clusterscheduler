/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package core

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrShareTreeMonNotAvail indicates that the sge_share_mon binary is not
// available on PATH or returned a non-zero exit code. Callers can use
// errors.Is to distinguish "feature unavailable" from unexpected I/O errors.
var ErrShareTreeMonNotAvail = errors.New("sge_share_mon is not available")

// ShareTreeNodeStats mirrors the per-node fields emitted by sge_share_mon in
// its name=value (-n) output mode. See sge_share_mon(1) OUTPUT FORMAT.
type ShareTreeNodeStats struct {
	NodeName         string  `json:"node_name"`
	UserName         string  `json:"user_name,omitempty"`
	ProjectName      string  `json:"project_name,omitempty"`
	Shares           int     `json:"shares"`
	JobCount         int     `json:"job_count"`
	LevelPercent     float64 `json:"level_percent"`
	TotalPercent     float64 `json:"total_percent"`
	LongTargetShare  float64 `json:"long_target_share"`
	ShortTargetShare float64 `json:"short_target_share"`
	ActualShare      float64 `json:"actual_share"`
	Usage            float64 `json:"usage"`
	CPU              float64 `json:"cpu"`
	Mem              float64 `json:"mem"`
	IO               float64 `json:"io"`
	LtCPU            float64 `json:"lt_cpu"`
	LtMem            float64 `json:"lt_mem"`
	LtIO             float64 `json:"lt_io"`
}

// ShareTreeMonitoring is a flat snapshot of runtime share-tree statistics.
// The map is keyed by node_name as reported by sge_share_mon.
type ShareTreeMonitoring struct {
	CollectedAt time.Time                     `json:"collected_at"`
	Nodes       map[string]ShareTreeNodeStats `json:"nodes"`
}

// ParseShareMonOutput parses the name=value output produced by
// `sge_share_mon -c 1 -n`.
//
// Observed ground-truth format (OCS 9.0.x):
//
//	curr_time=<n>\tusage_time=<n>\tnode_name=/\tuser_name=\tproject_name=\t...\n
//	curr_time=<n>\tusage_time=<n>\tnode_name=/default\t...\n
//	curr_time=<n>\tusage_time=<n>\tnode_name=/P1\tuser_name=\tproject_name=P1\t...\n
//
// Each non-empty line is one complete record; fields within a record are
// separated by TABs; the record ends at the newline. The tool's own
// -l/-r flags take delimiters as *literal strings* (no backslash escape
// interpretation), so forcing multi-character separators is a trap.
//
// A single leading "No share tree" line is a valid signal that qmaster
// has no share tree configured; it is surfaced as a successful parse of
// zero nodes and the envelope-level handler (ShowShareTreeMonitoring
// via runShareMon) maps it to ErrNoShareTree.
//
// Unknown fields are ignored so the parser tolerates scheduler versions
// that add new columns. Records with an empty node_name are dropped.
// The `-h` header row (field names without "=" signs) is skipped
// because each header token fails the `=` check below.
func ParseShareMonOutput(r io.Reader) (*ShareTreeMonitoring, error) {
	mon := &ShareTreeMonitoring{
		CollectedAt: time.Now().UTC(),
		Nodes:       make(map[string]ShareTreeNodeStats),
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t\r")
		if line == "" {
			continue
		}
		// "No share tree" is the literal body sge_share_mon prints when
		// qmaster has no tree configured. Parser returns zero nodes;
		// runShareMon maps exit-code 2 + this body to ErrNoShareTree.
		if strings.EqualFold(strings.TrimSpace(line), "No share tree") {
			continue
		}
		record := parseShareMonRecord(line)
		if record.NodeName == "" {
			continue
		}
		mon.Nodes[record.NodeName] = record
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mon, nil
}

// parseShareMonRecord turns one TAB-delimited line of key=value tokens
// into a ShareTreeNodeStats. Tokens without "=" (e.g. the -h header
// row) are silently ignored, preserving parser tolerance.
func parseShareMonRecord(line string) ShareTreeNodeStats {
	var s ShareTreeNodeStats
	for _, field := range strings.Split(line, "\t") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		eq := strings.IndexByte(field, '=')
		if eq < 0 {
			continue
		}
		assignShareMonField(&s, strings.TrimSpace(field[:eq]), strings.TrimSpace(field[eq+1:]))
	}
	return s
}

// assignShareMonField sets a single field on a ShareTreeNodeStats. Unknown
// keys are silently ignored.
func assignShareMonField(s *ShareTreeNodeStats, key, val string) {
	switch key {
	// curr_time / usage_time deliberately fall through to the default:
	// CollectedAt on the envelope already carries the parse-time clock,
	// and any unknown or future key is silently ignored to keep the
	// parser tolerant across scheduler versions. strconv errors are also
	// ignored on purpose — sge_share_mon never emits malformed numerics
	// on the happy path, and field values already default to zero.
	case "node_name":
		s.NodeName = val
	case "user_name":
		s.UserName = val
	case "project_name":
		s.ProjectName = val
	case "shares":
		s.Shares, _ = strconv.Atoi(val)
	case "job_count":
		s.JobCount, _ = strconv.Atoi(val)
	case "level%":
		s.LevelPercent, _ = strconv.ParseFloat(val, 64)
	case "total%":
		s.TotalPercent, _ = strconv.ParseFloat(val, 64)
	case "long_target_share":
		s.LongTargetShare, _ = strconv.ParseFloat(val, 64)
	case "short_target_share":
		s.ShortTargetShare, _ = strconv.ParseFloat(val, 64)
	case "actual_share":
		s.ActualShare, _ = strconv.ParseFloat(val, 64)
	case "usage":
		s.Usage, _ = strconv.ParseFloat(val, 64)
	case "cpu":
		s.CPU, _ = strconv.ParseFloat(val, 64)
	case "mem":
		s.Mem, _ = strconv.ParseFloat(val, 64)
	case "io":
		s.IO, _ = strconv.ParseFloat(val, 64)
	case "ltcpu":
		s.LtCPU, _ = strconv.ParseFloat(val, 64)
	case "ltmem":
		s.LtMem, _ = strconv.ParseFloat(val, 64)
	case "ltio":
		s.LtIO, _ = strconv.ParseFloat(val, 64)
	}
}

// locateShareMonBinary resolves the path of sge_share_mon. The scheduler
// ships the binary under $SGE_ROOT/utilbin/$SGE_ARCH and does not export
// either variable unless the operator has sourced settings.sh, so PATH
// lookup alone fails against a cluster that is otherwise fully usable.
// The search order is:
//
//  1. $PATH (honors a developer-supplied binary).
//  2. $SGE_ROOT/utilbin/$SGE_ARCH when both are set.
//  3. $SGE_ROOT/utilbin/*/sge_share_mon (first match) when only SGE_ROOT
//     is exported.
//  4. Well-known install prefixes (/opt/ocs, /opt/gridengine) scanned
//     the same way, for the common dev-container layout.
//
// Returns "sge_share_mon" (bare name) when nothing matches so that
// exec.Command surfaces the usual "not found" error for the caller.
func locateShareMonBinary() string {
	if p, err := exec.LookPath("sge_share_mon"); err == nil {
		return p
	}

	candidates := []string{}
	if root := os.Getenv("SGE_ROOT"); root != "" {
		candidates = append(candidates, root)
	}
	// Default-install prefixes used by this project's container and its
	// documented CLAUDE.md workflow. Operators with non-standard installs
	// should export SGE_ROOT instead of us growing this list indefinitely.
	candidates = append(candidates, "/opt/ocs", "/opt/cs-install")

	arch := os.Getenv("SGE_ARCH")
	for _, root := range candidates {
		if arch != "" {
			exact := filepath.Join(root, "utilbin", arch, "sge_share_mon")
			if _, err := os.Stat(exact); err == nil {
				return exact
			}
		}
		if p := scanUtilbin(root); p != "" {
			return p
		}
	}
	return "sge_share_mon"
}

// scanUtilbin walks $root/utilbin/*/sge_share_mon and returns the first
// hit. Used when SGE_ARCH is not set but SGE_ROOT (or a well-known
// install prefix) is reachable.
func scanUtilbin(root string) string {
	entries, err := os.ReadDir(filepath.Join(root, "utilbin"))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(root, "utilbin", e.Name(), "sge_share_mon")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// defaultShareMonRunner invokes sge_share_mon for a single snapshot
// in name=value (-n) format.
//
// The tool emits one record per line with TAB-separated key=value
// fields and a newline between records. -l/-r take literal delimiter
// strings (no backslash interpretation), so overriding them is the
// wrong lever; the defaults produce the format ParseShareMonOutput
// expects.
//
// Exit codes:
//
//   - rc=0: share tree present, stdout is records.
//   - rc=2: no share tree configured. stdout is "No share tree".
//     Mapped to ErrNoShareTree so callers can branch on empty-tree.
//   - binary missing: mapped to ErrShareTreeMonNotAvail.
//   - other non-zero: wrapped in ErrShareTreeMonNotAvail for backward
//     compatibility with earlier callers that treated any failure that
//     way.
//
// This is a method on *CommandLineQConf so tests can swap the
// shareMonRunner field on their own instance without touching a
// package-level variable (previously this was a var, and concurrent
// test-level swaps would race).
func (c *CommandLineQConf) defaultShareMonRunner() (io.Reader, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, locateShareMonBinary(), "-c", "1", "-n")
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()

	// The "no tree" signal is rc=2 with "No share tree" on stdout. The
	// body check is resilient to rc differences across scheduler builds.
	trimmed := strings.TrimSpace(out.String())
	if strings.EqualFold(trimmed, "No share tree") {
		return nil, ErrNoShareTree
	}

	if err != nil {
		execErr := &exec.Error{}
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return nil, ErrShareTreeMonNotAvail
		}
		return nil, fmt.Errorf("%w: %v: %s", ErrShareTreeMonNotAvail, err, errBuf.String())
	}
	return bytes.NewReader(out.Bytes()), nil
}
