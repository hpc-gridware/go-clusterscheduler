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
	"strings"
	"testing"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

// FuzzBuildQrsubArgsTwoResources asserts that a pair of resources and a
// pair of queues stay intact: a bracket or separator in one entry must not
// swallow or forge the other (the -l parser carries an open '[' across
// the following pairs).
func FuzzBuildQrsubArgsTwoResources(f *testing.F) {
	f.Add("a", "1", "b", "2", "q1", "q2")
	f.Add("a[", "1", "b", "2]", "q[1", "q2]")
	f.Add("h", "n[1-3]", "arch", "x", "q", "q@h")
	f.Fuzz(func(t *testing.T, k1, v1, k2, v2, q1, q2 string) {
		if k1 == k2 {
			return
		}
		args, err := core.BuildQrsubArgs(core.ReservationOptions{
			Duration:  core.ToPtr(time.Hour),
			Resources: map[string]string{k1: v1, k2: v2},
			Queues:    []string{q1, q2},
		})
		if err != nil {
			return
		}
		for _, s := range []string{k1, v1, k2, v2} {
			if strings.ContainsAny(s, ", ") {
				t.Fatalf("accepted resource part %q with a separator", s)
			}
		}
		for _, k := range []string{k1, k2} {
			if strings.ContainsAny(k, "[]=") {
				t.Fatalf("accepted resource name %q", k)
			}
		}
		for _, v := range []string{v1, v2} {
			if strings.Count(v, "[") != strings.Count(v, "]") {
				t.Fatalf("accepted unbalanced resource value %q", v)
			}
		}
		if !strings.Contains(strings.Join(args, "\x00"), "-q\x00"+q1+","+q2) {
			t.Fatalf("queues not intact in %q", args)
		}
	})
}

// FuzzBuildQrsubArgs asserts that caller supplied values can only end up
// in value positions: for any accepted input, the argv is a sequence of
// known switches each followed by exactly the expected value, the list
// elements survive intact, and no element can split into two.
func FuzzBuildQrsubArgs(f *testing.F) {
	f.Add("all.q", "alice", "h_rt", "600", "maint", "mpi", "2")
	f.Add("a,b", "-f", "a=b", "1,x=y", "-N", "-u", "-8")
	f.Add("q@h", "!@acl", "gpu", "", "x y", "pe", "1-")
	f.Fuzz(func(t *testing.T, queue, user, resName, resValue, name, peName, peSlots string) {
		opts := core.ReservationOptions{
			Duration:  core.ToPtr(time.Hour),
			Queues:    []string{queue},
			Users:     []string{user},
			Resources: map[string]string{resName: resValue},
			Name:      &name,
			PEName:    &peName,
			PESlots:   &peSlots,
		}
		args, err := core.BuildQrsubArgs(opts)
		if err != nil {
			return
		}
		want := [][]string{
			{"-d", "01:00:00"},
			{"-N", name},
			{"-l", resName + "=" + resValue},
			{"-q", queue},
			{"-pe", peName, peSlots},
			{"-u", user},
		}
		var flat []string
		for _, w := range want {
			flat = append(flat, w...)
		}
		if strings.Join(args, "\x00") != strings.Join(flat, "\x00") {
			t.Fatalf("unexpected argv %q, want %q", args, flat)
		}
		for _, element := range []string{queue, user, resName} {
			if strings.ContainsAny(element, ", ") || strings.HasPrefix(element, "-") {
				t.Fatalf("accepted list element %q that could split or become a flag", element)
			}
		}
		if strings.ContainsAny(resValue, ", ") || strings.ContainsAny(resName, "=, ") {
			t.Fatalf("accepted resource %q=%q that could forge a second pair", resName, resValue)
		}
	})
}
