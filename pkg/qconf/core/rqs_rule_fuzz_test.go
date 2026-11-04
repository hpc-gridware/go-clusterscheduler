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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// FuzzParseRQSRule feeds arbitrary limit lines to the rule parser, seeded
// with every limit line of the captured "qconf -srqs". It must not panic,
// and an accepted rule must carry at least one complete resource=value
// limit.
func FuzzParseRQSRule(f *testing.F) {
	data, err := os.ReadFile(filepath.Join("testdata", "qconf-srqs-all-9.1.7.txt"))
	if err != nil {
		f.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "limit") {
			f.Add(line)
		}
	}
	f.Add("limit name r1 users !alice,{@staff} hosts {*} to slots=$num_proc*2,h_vmem=4G")
	f.Add("")
	f.Fuzz(func(t *testing.T, limit string) {
		rule, err := core.ParseRQSRule(limit)
		if err != nil {
			return
		}
		if len(rule.Limits) == 0 {
			t.Fatalf("accepted %q without a limit", limit)
		}
		for _, l := range rule.Limits {
			if l.Resource == "" || l.Value == "" {
				t.Fatalf("accepted %q with an incomplete limit %+v", limit, l)
			}
		}
	})
}
