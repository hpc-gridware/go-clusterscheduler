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

package validate_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// hasControl reports whether s contains a rune the validators must reject.
func hasControl(s string) bool {
	if !utf8.ValidString(s) {
		return true
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return true
		}
	}
	return false
}

// FuzzOperand asserts the acceptance invariant: if Operand accepts a value it
// is non-empty, does not start with '-', and contains no control characters or
// invalid UTF-8. A hole in the validator surfaces as an accepted-but-dangerous
// input.
func FuzzOperand(f *testing.F) {
	for _, s := range []string{"", "-x", "node", "all.q@host", "node\x00", "n\u00f6de"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if validate.Operand(s) == nil {
			if s == "" {
				t.Fatalf("Operand accepted the empty string")
			}
			if s[0] == '-' {
				t.Fatalf("Operand accepted a leading-dash value: %q", s)
			}
			if hasControl(s) {
				t.Fatalf("Operand accepted a value with a control char / bad UTF-8: %q", s)
			}
		}
	})
}

// FuzzListElement asserts that an accepted list element additionally contains
// no comma and no space (the qconf element separators).
func FuzzListElement(f *testing.F) {
	for _, s := range []string{"node", "a,b", "a b", "-x", "all.q@host"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if validate.ListElement(s) == nil {
			if strings.ContainsAny(s, ", ") {
				t.Fatalf("ListElement accepted a value with a separator: %q", s)
			}
			if s == "" || s[0] == '-' || hasControl(s) {
				t.Fatalf("ListElement accepted an invalid operand: %q", s)
			}
		}
	})
}
