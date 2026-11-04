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

// Streaming, context-aware access to "qquota -u * -xml".
//
// qquota asks qmaster for the resource quota sets with their usage, the
// complexes, user sets, host groups and the execution hosts' load and
// consumables -- no jobs -- so a whole-cluster quota read costs in hosts and
// rules, not in jobs. The shape follows pkg/qhost/core/stream.go on purpose.
//
// What qquota prints, verified against a 9.1.7 cluster and its source
// (source/clients/qquota/ocs_qquota_print.cc). The -xml document was byte
// identical on OCS 9.0.12, 9.1.0 and 9.1.7 and GCS 9.1.4 and 9.1.7:
//
//   - Without -u it shows only the rows that match the CALLING user; a
//     cluster-wide read needs "-u *" (passed without a shell, so literally).
//   - Disabled rule sets are skipped silently.
//   - A consumable limit prints one row per usage entry: a rule with nothing
//     in use has no row, and "users {*}" yields one row per user with usage.
//     Within one rule set a job is counted by the FIRST rule that matches it,
//     so a later rule of the same set can stay without rows.
//   - A non-consumable limit prints the limit and no value: it limits each
//     job, it does not count.
//   - A dynamic limit ($num_proc*2) is evaluated per host, printed as an
//     integer.
//   - Values: integers, memory as "%.3f" with K/M/G (1024-based) or bare,
//     time as hh:mm:ss, "infinity" for no limit.
//   - The XML gives a "{*}" scope as "*".
//   - qmaster unreachable: exit 1, the reason on stderr.

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// QuotaStreamer is the bounded-memory, context-aware qquota client.
// *QQuotaImpl implements it; it is a small interface of its own so callers
// and fakes need nothing else.
type QuotaStreamer interface {
	// RunContext runs qquota with args under ctx, with exactly env as the
	// child environment (nil inherits the parent's), and returns stdout and
	// stderr. A failure carries the stderr text.
	RunContext(ctx context.Context, env []string, args ...string) (string, string, error)
	// StreamQuotasXML runs "qquota -u * -xml" and calls fn once per
	// <qquota_rule> element as it is read. An error from fn stops the stream
	// and is returned as it is.
	StreamQuotasXML(ctx context.Context, env []string, fn func(QuotaRuleXML) error) (QuotaStats, error)
}

// QuotaRuleXML is one <qquota_rule> element: one rule of a resource quota
// set, for one scope (one user, one host, ...), with its limits.
type QuotaRuleXML struct {
	// Name is "<rule set>/<rule>", the rule being its name or its 1-based
	// position in the set.
	Name string
	// The scope, as qquota prints it: the expanded value for a "{...}"
	// filter ("carol"), the filter itself otherwise ("*", "@gpu"); the X
	// lists are the excluded values ("!bob").
	Users, XUsers       []string
	Projects, XProjects []string
	PEs, XPEs           []string
	Queues, XQueues     []string
	Hosts, XHosts       []string
	Limits              []LimitXML
}

// RuleSet is the resource quota set the rule belongs to.
func (r QuotaRuleXML) RuleSet() string {
	set, _, _ := strings.Cut(r.Name, "/")
	return set
}

// Rule is the rule's name, or its 1-based position in the set.
func (r QuotaRuleXML) Rule() string {
	_, rule, _ := strings.Cut(r.Name, "/")
	return rule
}

// LimitXML is one <limit> of a rule.
type LimitXML struct {
	// Resource is the complex name, Limit the limit as qquota printed it
	// (evaluated for a dynamic limit).
	Resource string
	Limit    string
	// Value is what is in use; HasValue is false for a non-consumable limit,
	// which limits each job and counts nothing ("absent" is not "0").
	Value    string
	HasValue bool
}

// QuotaStats describes one streamed document.
type QuotaStats struct {
	// Decoded counts rule elements handed to the callback.
	Decoded int
	// Skipped counts rule elements without a name.
	Skipped int
	// Complete is true when the root close tag was seen. False means the
	// document was cut short and the rules decoded so far are a prefix.
	Complete bool
}

// ErrIncompleteQuotaDocument is returned when qquota's output did not end
// with the root close tag.
var ErrIncompleteQuotaDocument = errors.New("qquota XML document ended before its root close tag")

// maxStderrBytes bounds the stderr a streamed command may leave behind.
const maxStderrBytes = 64 * 1024

// streamWaitDelay bounds how long Wait blocks after the context is done when
// a grandchild still holds the pipes.
const streamWaitDelay = time.Second

// streamArgs is the cluster-wide read: every user's rows, as XML.
var streamArgs = []string{"-u", "*", "-xml"}

var _ QuotaStreamer = (*QQuotaImpl)(nil)

func commandError(stderr fmt.Stringer, cause error) error {
	return fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), cause)
}

// RunContext implements QuotaStreamer.
func (q *QQuotaImpl) RunContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qquota %v", args), "", nil
	}
	return helper.RunContext(ctx, q.config.Executable, env, nil, args...)
}

// StreamQuotasXML implements QuotaStreamer.
func (q *QQuotaImpl) StreamQuotasXML(ctx context.Context, env []string, fn func(QuotaRuleXML) error) (QuotaStats, error) {
	if q.config.DryRun {
		return QuotaStats{Complete: true}, nil
	}
	cmd := exec.CommandContext(ctx, q.config.Executable, streamArgs...)
	if env != nil {
		cmd.Env = env
	}
	cmd.WaitDelay = streamWaitDelay
	stderr := helper.NewCappedBuffer(maxStderrBytes)
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return QuotaStats{}, fmt.Errorf("failed to start qquota: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return QuotaStats{}, fmt.Errorf("failed to start qquota: %w", err)
	}
	stats, decodeErr := DecodeQuotasXML(stdout, fn)
	if decodeErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()

	// Same precedence as the qhost and qstat streams: a complete document is
	// the answer, then the deadline, then the process, then the decoder.
	switch {
	case decodeErr == nil && (waitErr == nil || errors.Is(waitErr, ctx.Err())):
		return stats, nil
	case ctx.Err() != nil:
		return stats, commandError(stderr, ctx.Err())
	case waitErr != nil && (decodeErr == nil || errors.Is(decodeErr, ErrIncompleteQuotaDocument)):
		return stats, commandError(stderr, waitErr)
	}
	return stats, decodeErr
}

// DecodeQuotasXML reads a "qquota -xml" document from r and calls fn once
// per <qquota_rule> element. It is the pure half of StreamQuotasXML and works
// on captured output as well.
func DecodeQuotasXML(r io.Reader, fn func(QuotaRuleXML) error) (QuotaStats, error) {
	var stats QuotaStats
	dec := xml.NewDecoder(r)
	// Scope values are what the admin configured; let invalid UTF-8 through
	// rather than fail the whole list.
	dec.Strict = false
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if !stats.Complete {
					return stats, incompleteQuotaDocument(stats)
				}
				return stats, nil
			}
			return stats, quotaDecodeFailure(stats, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "qquota_rule" {
				var raw quotaRuleXML
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return stats, quotaDecodeFailure(stats, err)
				}
				rule, ok := raw.typed()
				if !ok {
					stats.Skipped++
					continue
				}
				if err := fn(rule); err != nil {
					return stats, err
				}
				stats.Decoded++
				continue
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				stats.Complete = true
			}
		}
	}
}

func quotaDecodeFailure(stats QuotaStats, err error) error {
	var syntax *xml.SyntaxError
	if errors.As(err, &syntax) {
		if syntax.Msg == "unexpected EOF" {
			return incompleteQuotaDocument(stats)
		}
		return fmt.Errorf("malformed qquota XML after %d rules: %w", stats.Decoded+stats.Skipped, err)
	}
	return err
}

func incompleteQuotaDocument(stats QuotaStats) error {
	return fmt.Errorf("after %d rules: %w", stats.Decoded+stats.Skipped, ErrIncompleteQuotaDocument)
}

// quotaRuleXML is the raw shape of a <qquota_rule> element (qquota.xsd).
type quotaRuleXML struct {
	Name      string   `xml:"name,attr"`
	Users     []string `xml:"users"`
	XUsers    []string `xml:"xusers"`
	Projects  []string `xml:"projects"`
	XProjects []string `xml:"xprojects"`
	PEs       []string `xml:"pes"`
	XPEs      []string `xml:"xpes"`
	Queues    []string `xml:"queues"`
	XQueues   []string `xml:"xqueues"`
	Hosts     []string `xml:"hosts"`
	XHosts    []string `xml:"xhosts"`
	Limits    []struct {
		Resource string  `xml:"resource,attr"`
		Limit    string  `xml:"limit,attr"`
		Value    *string `xml:"value,attr"`
	} `xml:"limit"`
}

func (r quotaRuleXML) typed() (QuotaRuleXML, bool) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return QuotaRuleXML{}, false
	}
	rule := QuotaRuleXML{
		Name:  name,
		Users: trimmed(r.Users), XUsers: trimmed(r.XUsers),
		Projects: trimmed(r.Projects), XProjects: trimmed(r.XProjects),
		PEs: trimmed(r.PEs), XPEs: trimmed(r.XPEs),
		Queues: trimmed(r.Queues), XQueues: trimmed(r.XQueues),
		Hosts: trimmed(r.Hosts), XHosts: trimmed(r.XHosts),
	}
	for _, l := range r.Limits {
		limit := LimitXML{Resource: strings.TrimSpace(l.Resource), Limit: strings.TrimSpace(l.Limit)}
		if l.Value != nil {
			limit.Value, limit.HasValue = strings.TrimSpace(*l.Value), true
		}
		rule.Limits = append(rule.Limits, limit)
	}
	return rule, true
}

func trimmed(values []string) []string {
	var out []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
