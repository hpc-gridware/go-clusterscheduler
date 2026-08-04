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

// Package validate guards caller-supplied strings before they are assembled
// into the argv of the cluster-scheduler client binaries (qconf, qsub, qstat,
// qhost, qacct, qalter, qdel, qmod).
//
// The library builds argv arrays and never uses a shell, so the threat is
// argument injection (CWE-88), not shell injection: a value containing a comma,
// an equals sign, a control character, or a leading hyphen can be reinterpreted
// by the target binary as an extra list entry, an extra name=value pair, or an
// extra flag/subcommand. For example, qconf's object-list parser stops at the
// first token starting with "-" and hands it back to the top-level option loop,
// so "goodq -ke host" smuggled behind a valid object name becomes a new
// subcommand.
//
// The validators reject only characters that are both dangerous and
// structurally impossible in a legitimate scheduler identifier in that context,
// verified against the OCS source (verify_str_key/KEY_TABLE, QSUB_TABLE,
// verify_host_name). Every rejected input is one the scheduler itself would
// already refuse, so no working call site regresses. This generalises the
// existing ValidateSharePath pattern in pkg/qconf/core.
//
// Two layers are used by callers:
//
//   - Layer 1: Args (NoControl on every assembled argv token) at each exec
//     choke point. Safe on already-joined tokens because it does not reject
//     ",", "=", or space, only control characters and invalid UTF-8.
//   - Layer 2: the context validators (Operand, ListElement, NameValueKey,
//     NameValueValue, LocalFileName) applied to individual caller strings
//     before they are concatenated.
//
// Enforcement is governed by the GCS_VALIDATION environment variable via the
// Enforce gate: unset (default) blocks, "warn" logs and allows, "off" disables.
// This toggle is process-global and relaxes only the in-library wrapper checks;
// it is a developer/operator escape hatch for a false positive. Network trust
// boundaries (the MCP and REST handlers) call the validators directly, without
// Enforce, so they keep rejecting dangerous input even under "off"/"warn" -- a
// trust boundary must not be disableable by an env var.
package validate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// EnvVar is the environment variable that selects the enforcement mode.
const EnvVar = "GCS_VALIDATION"

// Mode is the enforcement level applied by Enforce.
type Mode int

const (
	// ModeEnforce blocks a violating input with an error. It is the default
	// and is safe because the structural rules reject only input that is
	// impossible in a legitimate identifier.
	ModeEnforce Mode = iota
	// ModeStrict adds the per-identifier character allowlist (see strict.go)
	// on top of the structural checks, at the call sites that opt into it via
	// StrictObjectName. Where a site does not opt in, ModeStrict behaves like
	// ModeEnforce.
	ModeStrict
	// ModeWarn logs each violation and allows the call to proceed. It is the
	// operator escape hatch for an unexpected false positive.
	ModeWarn
	// ModeOff disables validation entirely.
	ModeOff
)

// WarnLogger receives one message per violation when the mode is ModeWarn. It
// is replaceable so tests and hosts can capture or redirect the output, but it
// is a plain package var with no synchronisation: set it once at process init,
// before any concurrent validation runs. Do not reassign it while requests are
// in flight.
var WarnLogger = func(msg string) {
	log.Printf("gcs validation (warn): %s", msg)
}

// CurrentMode reads GCS_VALIDATION. An empty or unrecognised value yields
// ModeEnforce. The lookup is cheap; validation runs only on process-spawning
// paths, never in a hot loop.
func CurrentMode() Mode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvVar))) {
	case "off":
		return ModeOff
	case "warn":
		return ModeWarn
	case "strict":
		return ModeStrict
	default:
		return ModeEnforce
	}
}

// Enforce applies the current mode to one or more validation results and
// returns the first non-nil error after the mode gate. A call with no non-nil
// error returns nil. ModeOff returns nil; ModeWarn logs and returns nil;
// otherwise the first error is returned unchanged. The variadic form lets a
// call site combine several checks in one guard, e.g.
// Enforce(Operand(name), StrictObjectName(name)).
func Enforce(errs ...error) error {
	var first error
	for _, e := range errs {
		if e != nil {
			first = e
			break
		}
	}
	if first == nil {
		return nil
	}
	switch CurrentMode() {
	case ModeOff:
		return nil
	case ModeWarn:
		WarnLogger(first.Error())
		return nil
	default:
		return first
	}
}

// NoControl rejects NUL, all C0 controls (including newline, carriage return
// and tab), DEL, the C1 range, and invalid UTF-8. Printable multi-byte UTF-8 is
// accepted, matching SanitizeExtraValue in pkg/qconf/core. It deliberately does
// not reject ",", "=", or space because assembled argv tokens legitimately
// contain them (for example "all.q,gpu.q" or "slots=8").
func NoControl(s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("contains invalid UTF-8")
	}
	for i, r := range s {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return fmt.Errorf("contains control character %q at position %d", r, i)
		}
	}
	return nil
}

// Operand validates a single caller-supplied operand (an object name, host,
// user, and so on) that becomes exactly one argv token. It rejects the empty
// string and a leading "-" (which the target binary would parse as a flag or
// subcommand) in addition to NoControl. It intentionally allows ".", "_", "=",
// "@", and the other characters that appear in legitimate identifiers.
func Operand(s string) error {
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	if s[0] == '-' {
		return fmt.Errorf("must not start with '-' (would be parsed as a flag)")
	}
	return NoControl(s)
}

// ListElement validates one element of a comma- or space-separated argv list,
// such as an entry in "qconf -ah host1,host2" or one object instance of
// "qconf -mattr ... a.q,b.q". In addition to Operand it rejects "," and space,
// since qconf treats both as element separators and a value containing either
// would forge extra entries. "@" is allowed so queue instances (cqueue@host)
// validate.
func ListElement(s string) error {
	if err := Operand(s); err != nil {
		return err
	}
	if i := strings.IndexAny(s, ", "); i >= 0 {
		return fmt.Errorf("must not contain a list separator %q at position %d", s[i], i)
	}
	return nil
}

// NameValueKey validates the key of a name=value argv pair, such as a resource
// name in "qsub -l name=value" or a variable name in "-v key=value". It rejects
// "=" and "," (the pair and list separators) and a leading "-", in addition to
// NoControl.
func NameValueKey(s string) error {
	if err := Operand(s); err != nil {
		return err
	}
	if i := strings.IndexAny(s, "=,"); i >= 0 {
		return fmt.Errorf("key must not contain %q at position %d", s[i], i)
	}
	return nil
}

// NameValueValue validates the value of a name=value argv pair. It rejects ","
// (the list separator) in addition to NoControl. "=" is allowed because values
// may legitimately contain it (a complex value like "slots=8" or an environment
// value "A=B"); a leading "-" is allowed because the value is not a separate
// argv token.
func NameValueValue(s string) error {
	if err := NoControl(s); err != nil {
		return err
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		return fmt.Errorf("value must not contain ',' at position %d", i)
	}
	return nil
}

// LocalFileName validates a caller-supplied object name that is used to build a
// temp-file path (the qconf -A*/-M* file-input pattern). It requires a single
// local path segment: filepath.IsLocal must hold and the name must not contain
// a path separator. This prevents a name such as ".." or "a/b" from escaping
// the temp directory.
func LocalFileName(s string) error {
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	if err := NoControl(s); err != nil {
		return err
	}
	if strings.ContainsAny(s, "/\\") {
		return fmt.Errorf("must not contain a path separator")
	}
	if !filepath.IsLocal(s) {
		return fmt.Errorf("%q is not a local file name", s)
	}
	return nil
}

// Args applies the Layer 1 NoControl guard to every token of an assembled argv.
// It is called at the exec choke point and covers flags, operands, and
// pre-joined list tokens alike.
func Args(args ...string) error {
	for i, a := range args {
		if err := NoControl(a); err != nil {
			return fmt.Errorf("argument %d (%q): %w", i, a, err)
		}
	}
	return nil
}

// SplitAndValidateList splits a pre-joined comma-separated list token (such as
// the objIDList argument of qconf -mattr) and validates each element with
// ListElement. The comma separator itself is preserved; only the elements are
// checked, so a legitimate multi-object token like "all.q,gpu.q" passes while a
// forged "-ke host" element is rejected.
func SplitAndValidateList(s string) error {
	if err := NoControl(s); err != nil {
		return err
	}
	for _, elem := range strings.Split(s, ",") {
		if err := ListElement(elem); err != nil {
			return fmt.Errorf("list element %q: %w", elem, err)
		}
	}
	return nil
}

// List validates each element of a caller-supplied slice that will be
// comma-joined into a single argv token, using ListElement. what names the
// element class for error messages (e.g. "queue", "job id").
func List(what string, xs []string) error {
	for _, x := range xs {
		if err := ListElement(x); err != nil {
			return fmt.Errorf("%s %q: %w", what, x, err)
		}
	}
	return nil
}

// JobTaskList validates a pre-joined job/task reference list token as passed to
// qdel, qmod, and qalter. Elements may be separated by a comma or a space (the
// scheduler parser accepts both), so both are treated as separators here and
// each element is checked as an Operand: no leading '-' (which qdel/qmod would
// parse as a flag such as -t), no control characters, but the job-id, name, and
// fnmatch/boolean wildcard grammar (`* ? [ ] ! & | ( ) . : -` mid-token) all
// pass. An empty token is allowed; the caller may supply the range elsewhere.
func JobTaskList(s string) error {
	if err := NoControl(s); err != nil {
		return err
	}
	for _, elem := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	}) {
		if err := Operand(elem); err != nil {
			return fmt.Errorf("job/task element %q: %w", elem, err)
		}
	}
	return nil
}
