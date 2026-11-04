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
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qalter "github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
)

// FuzzParseVerifyOutput feeds arbitrary stdout and stderr to the parser,
// seeded with every captured answer. It must not panic, must always return
// a result with a known outcome, must keep the capped reasons within the cap
// and must account for every reason line either as kept or as dropped: the
// count of an uncapped run of the same input. The keep argument switches on
// a Keep function, whose accepted lines are not counted against the cap.
func FuzzParseVerifyOutput(f *testing.F) {
	files, err := filepath.Glob(filepath.Join("testdata", "qalter-w-*.txt"))
	if err != nil {
		f.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(string(data), "", false)
		f.Add(string(data), "", true)
	}
	f.Add("", `denied: job "1" does not exist`, false)
	f.Add("Job 1 (-l gpu=1) cannot run globally because it offers only gc:gpu=0.000000\nverification: found possible assignment with 4 slots\n", "", true)
	const maxReasons = 3
	known := map[qalter.VerifyOutcome]bool{
		qalter.VerifyNoSuitableQueues:    true,
		qalter.VerifyFoundSuitableQueues: true,
		qalter.VerifyFoundAssignment:     true,
		qalter.VerifyAlreadyRunning:      true,
		qalter.VerifyOutcomeUnknown:      true,
	}
	// onWorker1 keeps the lines about one host's queue instances, as a
	// caller asking about that host would.
	onWorker1 := func(r qalter.VerifyReason) bool {
		return strings.HasSuffix(r.Target, "@ocs-worker1")
	}
	f.Fuzz(func(t *testing.T, stdout, stderr string, keep bool) {
		opts := qalter.VerifyOptions{MaxReasons: maxReasons}
		if keep {
			opts.Keep = onWorker1
		}
		result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, stdout, stderr, opts)
		if result == nil {
			t.Fatal("no result")
		}
		if !known[result.Outcome] {
			t.Fatalf("unknown outcome %q", result.Outcome)
		}
		if (err != nil) != (result.Denied != "") {
			t.Fatalf("error %v does not match denial %q", err, result.Denied)
		}
		all, _ := qalter.ParseVerifyOutput(qalter.VerifyNow, stdout, stderr, qalter.VerifyOptions{MaxReasons: math.MaxInt})
		if all.Dropped != 0 {
			t.Fatalf("uncapped run dropped %d reasons", all.Dropped)
		}
		if got := len(result.Reasons) + result.Dropped; got != len(all.Reasons) {
			t.Fatalf("%d kept + %d dropped, want %d reason lines", len(result.Reasons), result.Dropped, len(all.Reasons))
		}
		capped := 0
		for _, r := range result.Reasons {
			if r.Text == "" || strings.ContainsAny(r.Text, "\n") {
				t.Fatalf("reason without a single-line text: %+v", r)
			}
			if opts.Keep == nil || !opts.Keep(r) {
				capped++
			}
		}
		if capped > maxReasons {
			t.Fatalf("%d capped reasons kept, cap is %d", capped, maxReasons)
		}
		if result.Outcome != all.Outcome || result.Slots != all.Slots || result.Denied != all.Denied {
			t.Fatalf("cap changed the result line: %+v vs %+v", result, all)
		}
	})
}
