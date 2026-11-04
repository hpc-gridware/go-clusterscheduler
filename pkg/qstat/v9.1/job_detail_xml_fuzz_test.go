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

package qstat_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.1"
)

// FuzzDecodeJobDetailXML feeds arbitrary bytes to the decoder: it must
// return, never panic, and hand back a job whenever it reports success.
func FuzzDecodeJobDetailXML(f *testing.F) {
	for _, name := range append(jobDetailFixtures, "unknown") {
		b, err := os.ReadFile(filepath.Join("testdata", jobDetailFixture(name)))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add([]byte(""))
	f.Add([]byte("<detailed_job_info><djob_info><element><JB_ja_u_h_ids><x><RN_min>1</RN_min>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		job, err := qstat.DecodeJobDetailXML(bytes.NewReader(data))
		if err == nil && job == nil {
			t.Fatal("success without a job")
		}
	})
}
