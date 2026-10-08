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
	"testing"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/core"
)

// addFixtureSeeds seeds a fuzz target with all captured outputs.
func addFixtureSeeds(f *testing.F, pattern string) {
	files, _ := filepath.Glob(filepath.Join("testdata", "*", pattern))
	for _, file := range files {
		if data, err := os.ReadFile(file); err == nil {
			f.Add(string(data))
		}
	}
}

// FuzzParseSummary asserts the summary parser never panics and that every
// returned row has a positive id.
func FuzzParseSummary(f *testing.F) {
	addFixtureSeeds(f, "qrstat_*.stdout")
	f.Fuzz(func(t *testing.T, s string) {
		ars, err := core.ParseSummary(s)
		if err != nil {
			return
		}
		for _, ar := range ars {
			if ar.ID <= 0 {
				t.Fatalf("summary row with id %d from %q", ar.ID, s)
			}
		}
	})
}

// FuzzParseDetail asserts the detail parser never panics and that every
// returned AR has an id.
func FuzzParseDetail(f *testing.F) {
	addFixtureSeeds(f, "qrstat_*.stdout")
	f.Fuzz(func(t *testing.T, s string) {
		ars, err := core.ParseDetail(s)
		if err != nil {
			return
		}
		for _, ar := range ars {
			if ar.ID <= 0 {
				t.Fatalf("detail block without id from %q", s)
			}
		}
	})
}
