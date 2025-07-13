/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
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

package qacct_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
)

var _ = Describe("Builders", func() {

	Context("ParseSummaryOutput", func() {
		It("should parse summary output with Total System Usage", func() {
			input := `Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
           72        25.861        12.399        38.259              0.284              0.000              0.000`

			usage, err := qacct.ParseSummaryOutput(input)
			Expect(err).To(BeNil())
			Expect(usage.WallClock).To(Equal(72.0))
			Expect(usage.UserTime).To(Equal(25.861))
			Expect(usage.SystemTime).To(Equal(12.399))
			Expect(usage.CPU).To(Equal(38.259))
			Expect(usage.Memory).To(Equal(0.284))
			Expect(usage.IO).To(Equal(0.000))
			Expect(usage.IOWait).To(Equal(0.000))
		})

		It("should parse dynamic header format with owner", func() {
			input := `OWNER     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
======================================================================================================================
root            133         1.422         1.081         2.503              0.236              0.000              0.000`

			usage, err := qacct.ParseSummaryOutput(input)
			Expect(err).To(BeNil())
			Expect(usage.WallClock).To(Equal(133.0))
			Expect(usage.UserTime).To(Equal(1.422))
			Expect(usage.SystemTime).To(Equal(1.081))
			Expect(usage.CPU).To(Equal(2.503))
			Expect(usage.Memory).To(Equal(0.236))
			Expect(usage.IO).To(Equal(0.000))
			Expect(usage.IOWait).To(Equal(0.000))
		})

		It("should parse dynamic header format with host/queue", func() {
			input := `HOST   CLUSTER QUEUE OWNER     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
===========================================================================================================================================
master all.q         root            133         1.422         1.081         2.503              0.236              0.000              0.000`

			usage, err := qacct.ParseSummaryOutput(input)
			Expect(err).To(BeNil())
			Expect(usage.WallClock).To(Equal(133.0))
			Expect(usage.UserTime).To(Equal(1.422))
			Expect(usage.SystemTime).To(Equal(1.081))
			Expect(usage.CPU).To(Equal(2.503))
			Expect(usage.Memory).To(Equal(0.236))
			Expect(usage.IO).To(Equal(0.000))
			Expect(usage.IOWait).To(Equal(0.000))
		})

		It("should handle empty input", func() {
			usage, err := qacct.ParseSummaryOutput("")
			Expect(err).To(BeNil())
			Expect(usage).To(Equal(qacct.Usage{}))
		})

		It("should handle input without summary data", func() {
			input := `Some other output
without summary section`

			usage, err := qacct.ParseSummaryOutput(input)
			Expect(err).To(BeNil())
			Expect(usage).To(Equal(qacct.Usage{}))
		})

		It("should handle header with no data rows", func() {
			input := `OWNER     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
======================================================================================================================`

			usage, err := qacct.ParseSummaryOutput(input)
			Expect(err).To(BeNil())
			Expect(usage).To(Equal(qacct.Usage{}))
		})
	})

	Context("SummaryBuilder", func() {
		var qa qacct.QAcct

		BeforeEach(func() {
			var err error
			qa, err = qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should build summary query with multiple filters", func() {
			builder := qa.Summary().
				LastDays(7).
				Group("developers").
				Owner("alice")

			// Test that builder can be created (actual execution would need cluster)
			Expect(builder).NotTo(BeNil())
		})

		It("should build summary query with time range", func() {
			builder := qa.Summary().
				BeginTime("2024-01-01").
				EndTime("2024-01-31").
				Department("engineering")

			Expect(builder).NotTo(BeNil())
		})
	})

	Context("JobsBuilder", func() {
		var qa qacct.QAcct

		BeforeEach(func() {
			var err error
			qa, err = qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should build job detail query with filters", func() {
			builder := qa.Jobs().
				Owner("bob").
				Queue("all.q").
				LastDays(1)

			Expect(builder).NotTo(BeNil())
		})

		It("should build job query with job pattern", func() {
			builder := qa.Jobs().
				JobPattern("test-*").
				Project("research")

			Expect(builder).NotTo(BeNil())
		})

		It("should build task query", func() {
			builder := qa.Jobs().
				Tasks("123", "1-10").
				Host("master")

			Expect(builder).NotTo(BeNil())
		})
	})

	Context("Interface backward compatibility", func() {
		var qa qacct.QAcct

		BeforeEach(func() {
			var err error
			qa, err = qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should keep ShowJobDetails method unchanged", func() {
			// This method should still work as before
			// Note: actual execution would require finished jobs in cluster
			_, err := qa.ShowJobDetails(nil)
			// We expect this to work but may return empty results or error if no jobs exist
			// The important thing is the method signature is preserved
			Expect(err).To(BeNil())
		})

		It("should provide ShowHelp method", func() {
			help, err := qa.ShowHelp()
			Expect(err).To(BeNil())
			Expect(help).To(ContainSubstring("usage: qacct"))
		})

		It("should provide NativeSpecification method", func() {
			output, err := qa.NativeSpecification([]string{"-help"})
			Expect(err).To(BeNil())
			Expect(output).To(ContainSubstring("usage: qacct"))
		})
	})
})