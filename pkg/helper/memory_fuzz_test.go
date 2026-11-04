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

package helper_test

import (
	"testing"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
)

// FuzzParseMemoryFromString checks that the parser never panics and never
// returns a negative byte count, whatever ParseFloat accepts (exponents,
// hex floats, signs, special values).
func FuzzParseMemoryFromString(f *testing.F) {
	for _, seed := range []string{
		"0", "1", "15.6G", "2.5m", "1P", "8192P", "1e30", "-1G", "NaN", "Inf",
		"-InfK", "0x1p4G", "1e3K", "9223372036854775807", "9223372036854775808",
		"", "G", ".5", "1_000", "+1K",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		value, err := helper.ParseMemoryFromString(in)
		if err != nil {
			if value != 0 {
				t.Fatalf("%q: error %v with non-zero value %d", in, err, value)
			}
			return
		}
		if value < 0 {
			t.Fatalf("%q: negative result %d", in, value)
		}
	})
}
