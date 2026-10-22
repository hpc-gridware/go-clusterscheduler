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
	"testing"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

// BenchmarkDecodeJobsXML200k decodes a generated 220,000-element document
// (200k running tasks, 20k pending ranges), the shape of a cluster with
// 200k jobs. Run with: go test -run xxx -bench DecodeJobsXML -benchmem ./pkg/qstat/v9.0/
func BenchmarkDecodeJobsXML200k(b *testing.B) {
	var doc bytes.Buffer
	generatedJobs(&doc, 200000)
	b.SetBytes(int64(doc.Len()))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := 0
		stats, err := qstat.DecodeJobsXML(bytes.NewReader(doc.Bytes()), func(qstat.JobXML) error {
			n++
			return nil
		})
		if err != nil || !stats.Complete || n != 220000 {
			b.Fatalf("decode: %v, complete=%v, n=%d", err, stats.Complete, n)
		}
	}
}
