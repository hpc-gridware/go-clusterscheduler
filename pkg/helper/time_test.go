/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
)

var _ = Describe("Time Helper", func() {

	Describe("ParseTimeResourceValueToSeconds", func() {

		// Test valid time strings.
		DescribeTable("should correctly parse valid time strings",
			func(input string, expected int64) {
				actual, err := helper.ParseTimeResourceValueToSeconds(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			},
			Entry("parses zero time", "00:00:00", int64(0)),
			Entry("parses early time", "01:02:03", int64(3723)),     // 1h + 2m + 3s = 3600 + 120 + 3
			Entry("parses mid time", "10:20:30", int64(37230)),      // 10h + 20m + 30s = 36000 + 1200 + 30
			Entry("parses boundary time", "23:59:59", int64(86399)), // latest valid time before midnight ends
		)

		// Test invalid time strings.
		DescribeTable("should return errors for invalid time strings",
			func(input string) {
				_, err := helper.ParseTimeResourceValueToSeconds(input)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid time format"))
			},
			Entry("with too few components", "12:34"),
			Entry("with too many components", "01:02:03:04"),
			Entry("with non-numeric hour", "ab:12:34"),
			Entry("with non-numeric minute", "12:cd:34"),
			Entry("with non-numeric second", "12:34:ef"),
		)
	})

	Describe("FormatSecondsToTimeResourceValue", func() {

		DescribeTable("should correctly format seconds into time string",
			func(input int64, expected string) {
				actual := helper.FormatSecondsToTimeResourceValue(input)
				Expect(actual).To(Equal(expected))
			},
			Entry("formats zero seconds", int64(0), "00:00:00"),
			Entry("formats standard seconds", int64(3723), "01:02:03"),
			Entry("formats another standard value", int64(37230), "10:20:30"),
			Entry("formats boundary seconds", int64(86399), "23:59:59"),
			Entry("clamps negative seconds to zero", int64(-10), "00:00:00"),
			Entry("formats time beyond one day", int64(86461), "24:01:01"),
		)
	})
})
