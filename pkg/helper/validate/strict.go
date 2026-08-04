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

// Strict-mode charset validators. These mirror the character sets the OCS
// clients actually enforce (verify_str_key / KEY_TABLE, verify_host_name, and
// QSUB_TABLE in the OCS source), so anything they reject the scheduler would
// also reject. They are stricter than the default structural checks and are
// opt-in: a caller applies them only when CurrentMode() == ModeStrict, via the
// Strict* gates below, because a site may legitimately use unusual names that
// the documented charset does not cover.

package validate

import (
	"fmt"
	"strings"
)

// objectNameForbidden are the characters KEY_TABLE rejects within a single
// object-name segment (queue, PE, checkpoint, calendar, complex, project,
// userset, department, user, host-group name after '@'). '@' itself is a
// structural delimiter handled by ObjectName splitting on it, so it is not
// listed here. NoControl already rejects the C0/C1/DEL control characters, but
// KEY_TABLE additionally forbids the plain ASCII space (0x20), which NoControl
// accepts, so it is listed explicitly.
const objectNameForbidden = "/:'\"\\[]{}|()%, "

// maxObjectNameLen is MAX_VERIFY_STRING in the OCS source.
const maxObjectNameLen = 512

// reservedNames are rejected case-insensitively by the scheduler for object
// names.
var reservedNames = map[string]bool{"none": true, "all": true, "template": true}

// ObjectName validates a name against the OCS KEY_TABLE rules: non-empty, at
// most 512 characters, and composed of KEY_TABLE segments. A name may be a
// plain object name, an '@'-prefixed host group (@group), or a queue instance
// (cqueue@host or cqueue@@hostgroup); '@' is a structural delimiter in those
// forms, so each '@'-separated segment is validated as a KEY_TABLE name (not
// beginning with '.' or '#', none of objectNameForbidden, not reserved). It
// intentionally allows '-' '_' '.' '+' '=' '&' '!' '*' '?' which KEY_TABLE
// accepts.
func ObjectName(s string) error {
	if err := NoControl(s); err != nil {
		return err
	}
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	if len(s) > maxObjectNameLen {
		return fmt.Errorf("length %d exceeds max %d", len(s), maxObjectNameLen)
	}
	nonEmpty := 0
	for _, seg := range strings.Split(s, "@") {
		if seg == "" {
			continue // leading '@' (host group) or '@@' (hostgroup instance)
		}
		nonEmpty++
		if err := keyTableSegment(seg); err != nil {
			return err
		}
	}
	if nonEmpty == 0 {
		return fmt.Errorf("must contain a name")
	}
	return nil
}

// keyTableSegment validates one '@'-delimited segment of an object name.
func keyTableSegment(s string) error {
	if s[0] == '.' || s[0] == '#' {
		return fmt.Errorf("segment %q must not begin with %q", s, s[0])
	}
	if i := strings.IndexAny(s, objectNameForbidden); i >= 0 {
		return fmt.Errorf("contains forbidden character %q at position %d", s[i], i)
	}
	if reservedNames[strings.ToLower(s)] {
		return fmt.Errorf("%q is a reserved name", s)
	}
	return nil
}

// maxHostNameLen is CL_MAXHOSTNAMELEN in the OCS source.
const maxHostNameLen = 256

// HostName validates a host name the way verify_host_name does: non-empty and
// at most 256 characters, with no server-side character-set restriction beyond
// NoControl. It must not contain '@' (which delimits queue@host) or whitespace.
func HostName(s string) error {
	if err := NoControl(s); err != nil {
		return err
	}
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	if len(s) > maxHostNameLen {
		return fmt.Errorf("length %d exceeds max %d", len(s), maxHostNameLen)
	}
	if i := strings.IndexAny(s, "@ "); i >= 0 {
		return fmt.Errorf("contains invalid character %q at position %d", s[i], i)
	}
	return nil
}

// jobNameForbidden are the characters QSUB_TABLE rejects in a job name.
const jobNameForbidden = `/:@\*?`

// JobName validates a qsub -N job name per QSUB_TABLE plus the qmaster rule
// that a name may not start with a digit: non-empty, at most 512 characters,
// no leading digit, and none of jobNameForbidden. Spaces, '.', ',', '|', '%',
// brackets and parentheses are accepted (QSUB_TABLE permits them).
func JobName(s string) error {
	if err := NoControl(s); err != nil {
		return err
	}
	if s == "" {
		return fmt.Errorf("must not be empty")
	}
	// The qmaster job-name check uses ">=" against MAX_VERIFY_STRING, so the
	// effective maximum is one shorter than the object-name limit.
	if len(s) >= maxObjectNameLen {
		return fmt.Errorf("length %d exceeds max %d", len(s), maxObjectNameLen-1)
	}
	if s[0] >= '0' && s[0] <= '9' {
		return fmt.Errorf("must not begin with a digit")
	}
	if i := strings.IndexAny(s, jobNameForbidden); i >= 0 {
		return fmt.Errorf("contains forbidden character %q at position %d", s[i], i)
	}
	if reservedNames[strings.ToLower(s)] {
		return fmt.Errorf("%q is a reserved name", s)
	}
	return nil
}

// StrictObjectName returns ObjectName(s) only when the current mode is
// ModeStrict; otherwise it returns nil. Object-name call sites wrap this in
// Enforce alongside the structural Operand check, so the extra charset is
// applied only when the operator opts in with GCS_VALIDATION=strict.
func StrictObjectName(s string) error {
	if CurrentMode() != ModeStrict {
		return nil
	}
	return ObjectName(s)
}

// StrictHostName returns HostName(s) only in ModeStrict; otherwise nil. Host
// call sites use this instead of StrictObjectName because host names follow the
// verify_host_name rules (no charset restriction, 256 max, no '@'), not the
// object-name KEY_TABLE.
func StrictHostName(s string) error {
	if CurrentMode() != ModeStrict {
		return nil
	}
	return HostName(s)
}

// StrictJobName returns JobName(s) only in ModeStrict; otherwise nil. The qsub
// -N site uses this because job names follow QSUB_TABLE (spaces and brackets
// allowed, leading digit and '@'/'*' forbidden), not the object-name rules.
func StrictJobName(s string) error {
	if CurrentMode() != ModeStrict {
		return nil
	}
	return JobName(s)
}
