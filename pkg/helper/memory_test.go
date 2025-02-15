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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
)

var _ = Describe("Memory", func() {

	Describe("ParseMemoryFromString", func() {

		DescribeTable("should correctly parse valid memory strings",
			func(input string, expected int64) {
				result, err := helper.ParseMemoryFromString(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(expected))
			},
			// No unit provided, parsed as an integer.
			Entry("parses integer string without unit", "15", int64(15)),

			// Special zero values.
			Entry("parses \"0\" string", "0", int64(0)),
			Entry("parses \"0.0\" string", "0.0", int64(0)),
			Entry("parses \"0.00\" string", "0.00", int64(0)),
			Entry("parses \"0.000\" string", "0.000", int64(0)),

			// With multiplier units.
			Entry("parses with uppercase K", "1K", int64(1024)),
			Entry("parses with lowercase k", "1k", int64(1000)),
			Entry("parses with uppercase M", "1M", int64(1024*1024)),
			Entry("parses with lowercase m", "1m", int64(1000000)),
			Entry("parses with uppercase G", "1G", int64(1024*1024*1024)),
			Entry("parses with lowercase g", "1g", int64(1000000000)),

			// Fractional values.
			Entry("parses fractional with uppercase G", "15.6G", int64(16750372454)), // 15.6G -> 15.6 * 1073741824, truncating the fractional part
			Entry("parses fractional with uppercase M", "2.5M", int64(2.5*1048576)),  // 1048576 = 1024^2
			Entry("parses fractional with lowercase m", "2.5m", int64(2.5*1000000)),
		)

		DescribeTable("should return error for invalid memory strings",
			func(input string) {
				_, err := helper.ParseMemoryFromString(input)
				Expect(err).To(HaveOccurred())
			},
			Entry("empty string", ""),
			Entry("non-numeric value with unit", "abcG"),
			Entry("floating-point number without unit", "15.6"),
			Entry("invalid unit because of unrecognized multiplier", "10X"),
			Entry("invalid fractional part", "15.twoG"),
		)
	})

	Describe("MemoryToString", func() {

		DescribeTable("should correctly convert memory bytes to string",
			func(input int64, expected string) {
				result := helper.MemoryToString(input)
				Expect(result).To(Equal(expected))
			},
			// Below 1024 bytes are printed as is.
			Entry("returns bytes as string if less than 1024", int64(512), "512"),
			Entry("returns bytes as string if less than 1024", int64(1023), "1023"),

			// For values between 1024 and 1024^2, floor-divide by 1024 and append "K".
			Entry("converts 1024 bytes to 1K", int64(1024), "1K"),
			Entry("converts 2048 bytes to 2K", int64(2048), "2K"),
			Entry("edge case: value just below 1M", int64(1024*1024-1),
				strconv.FormatInt((1024*1024-1)/1024, 10)+"K"),

			// For values between 1024^2 and 1024^3.
			Entry("converts 1M bytes to 1M", int64(1024*1024), "1M"),
			Entry("converts value just above 1M to 1M", int64(1024*1024+1), "1M"),
			Entry("edge case: value just below 1G", int64(1024*1024*1024-1),
				strconv.FormatInt((1024*1024*1024-1)/(1024*1024), 10)+"M"),

			// For values >= 1024^3.
			Entry("converts 1G bytes to 1G", int64(1024*1024*1024), "1G"),
			Entry("large value conversion to G", int64(2*1024*1024*1024), "2G"),

			// Negative values are returned as a simple numeric string.
			Entry("handles negative values", int64(-100), "-100"),
		)

	})
})
