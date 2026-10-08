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

package core_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/core"
)

// FuzzBuildDeleteByNameArgs asserts that an accepted owner and name can
// only select ARs by name of exactly that owner: the argv is always
// "-u owner name", the owner is one user without pattern characters, and
// the name can neither become a flag, a second selector nor an AR id.
func FuzzBuildDeleteByNameArgs(f *testing.F) {
	f.Add("alice", "maint")
	f.Add("*", "x")
	f.Add("alice", "0x2a")
	f.Add("alice", "a,b")
	f.Add("-f", "-u")
	f.Fuzz(func(t *testing.T, owner, name string) {
		args, err := core.BuildDeleteByNameArgs(false, owner, name)
		if err != nil {
			return
		}
		if len(args) != 3 || args[0] != "-u" || args[1] != owner || args[2] != name {
			t.Fatalf("unexpected argv %q", args)
		}
		if owner == "" || strings.ContainsAny(owner, "*?[], ") || strings.HasPrefix(owner, "-") {
			t.Fatalf("accepted owner %q which is not exactly one user", owner)
		}
		if strings.ContainsAny(name, ", ") || strings.HasPrefix(name, "-") {
			t.Fatalf("accepted name %q which could split or become a flag", name)
		}
		if _, err := strconv.ParseInt(strings.TrimPrefix(name, "+"), 0, 64); err == nil {
			t.Fatalf("accepted name %q which qmaster reads as an AR id", name)
		}
	})
}
