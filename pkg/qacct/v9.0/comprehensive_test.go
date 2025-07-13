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

var _ = Describe("Comprehensive qacct parsing", func() {

	Context("All qacct filter combinations", func() {
		var qa qacct.QAcct

		BeforeEach(func() {
			var err error
			qa, err = qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle single filter options", func() {
			testCases := []struct {
				name    string
				builder func() *qacct.SummaryBuilder
			}{
				{"owner filter", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root") }},
				{"group filter", func() *qacct.SummaryBuilder { return qa.Summary().Group("root") }},
				{"department filter", func() *qacct.SummaryBuilder { return qa.Summary().Department("defaultdepartment") }},
				{"project filter", func() *qacct.SummaryBuilder { return qa.Summary().Project("default") }},
				{"queue filter", func() *qacct.SummaryBuilder { return qa.Summary().Queue("all.q") }},
				{"host filter", func() *qacct.SummaryBuilder { return qa.Summary().Host("master") }},
				{"days filter", func() *qacct.SummaryBuilder { return qa.Summary().LastDays(1) }},
				{"days 0 filter", func() *qacct.SummaryBuilder { return qa.Summary().LastDays(0) }},
				{"slots filter", func() *qacct.SummaryBuilder { return qa.Summary().Slots(1) }},
			}

			for _, tc := range testCases {
				By("Testing " + tc.name)
				usage, err := tc.builder().Execute()
				Expect(err).NotTo(HaveOccurred())
				// Should get valid usage struct (may be empty)
				Expect(usage.WallClock).To(BeNumerically(">=", 0))
				Expect(usage.CPU).To(BeNumerically(">=", 0))
				Expect(usage.Memory).To(BeNumerically(">=", 0))
			}
		})

		It("should handle two-filter combinations", func() {
			testCases := []struct {
				name    string
				builder func() *qacct.SummaryBuilder
			}{
				{"owner + group", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root").Group("root") }},
				{"owner + department", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root").Department("defaultdepartment") }},
				{"owner + project", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root").Project("default") }},
				{"owner + queue", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root").Queue("all.q") }},
				{"owner + host", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root").Host("master") }},
				{"owner + days", func() *qacct.SummaryBuilder { return qa.Summary().Owner("root").LastDays(1) }},
				{"queue + host", func() *qacct.SummaryBuilder { return qa.Summary().Queue("all.q").Host("master") }},
				{"queue + project", func() *qacct.SummaryBuilder { return qa.Summary().Queue("all.q").Project("default") }},
				{"group + department", func() *qacct.SummaryBuilder { return qa.Summary().Group("root").Department("defaultdepartment") }},
			}

			for _, tc := range testCases {
				By("Testing " + tc.name)
				usage, err := tc.builder().Execute()
				Expect(err).NotTo(HaveOccurred())
				// Should get valid usage struct (may be empty)
				Expect(usage.WallClock).To(BeNumerically(">=", 0))
				Expect(usage.CPU).To(BeNumerically(">=", 0))
				Expect(usage.Memory).To(BeNumerically(">=", 0))
			}
		})

		It("should handle three-filter combinations", func() {
			testCases := []struct {
				name    string
				builder func() *qacct.SummaryBuilder
			}{
				{"owner + queue + host", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").Queue("all.q").Host("master") 
				}},
				{"owner + queue + project", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").Queue("all.q").Project("default") 
				}},
				{"owner + group + department", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").Group("root").Department("defaultdepartment") 
				}},
				{"queue + host + project", func() *qacct.SummaryBuilder { 
					return qa.Summary().Queue("all.q").Host("master").Project("default") 
				}},
				{"owner + days + slots", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").LastDays(1).Slots(1) 
				}},
			}

			for _, tc := range testCases {
				By("Testing " + tc.name)
				usage, err := tc.builder().Execute()
				Expect(err).NotTo(HaveOccurred())
				// Should get valid usage struct (may be empty)
				Expect(usage.WallClock).To(BeNumerically(">=", 0))
				Expect(usage.CPU).To(BeNumerically(">=", 0))
				Expect(usage.Memory).To(BeNumerically(">=", 0))
			}
		})

		It("should handle complex multi-filter combinations", func() {
			testCases := []struct {
				name    string
				builder func() *qacct.SummaryBuilder
			}{
				{"owner + queue + host + project", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").Queue("all.q").Host("master").Project("default") 
				}},
				{"owner + group + department + days", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").Group("root").Department("defaultdepartment").LastDays(1) 
				}},
				{"all common filters", func() *qacct.SummaryBuilder { 
					return qa.Summary().Owner("root").Queue("all.q").Host("master").LastDays(1).Slots(1) 
				}},
			}

			for _, tc := range testCases {
				By("Testing " + tc.name)
				usage, err := tc.builder().Execute()
				Expect(err).NotTo(HaveOccurred())
				// Should get valid usage struct (may be empty)
				Expect(usage.WallClock).To(BeNumerically(">=", 0))
				Expect(usage.CPU).To(BeNumerically(">=", 0))
				Expect(usage.Memory).To(BeNumerically(">=", 0))
			}
		})

		It("should handle filters that return no data", func() {
			testCases := []struct {
				name    string
				builder func() *qacct.SummaryBuilder
			}{
				{"nonexistent owner", func() *qacct.SummaryBuilder { return qa.Summary().Owner("nonexistent") }},
				{"nonexistent queue", func() *qacct.SummaryBuilder { return qa.Summary().Queue("nonexistent.q") }},
				{"nonexistent host", func() *qacct.SummaryBuilder { return qa.Summary().Host("nonexistent") }},
				{"future days", func() *qacct.SummaryBuilder { return qa.Summary().LastDays(0) }},
				{"high slot count", func() *qacct.SummaryBuilder { return qa.Summary().Slots(999) }},
			}

			for _, tc := range testCases {
				By("Testing " + tc.name)
				usage, err := tc.builder().Execute()
				Expect(err).NotTo(HaveOccurred())
				// Should get empty but valid usage struct
				Expect(usage.WallClock).To(Equal(0.0))
				Expect(usage.CPU).To(Equal(0.0))
				Expect(usage.Memory).To(Equal(0.0))
				Expect(usage.IO).To(Equal(0.0))
				Expect(usage.IOWait).To(Equal(0.0))
			}
		})
	})

	Context("Job detail filter combinations", func() {
		var qa qacct.QAcct

		BeforeEach(func() {
			var err error
			qa, err = qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle job detail filter combinations", func() {
			testCases := []struct {
				name    string
				builder func() *qacct.JobsBuilder
			}{
				{"owner filter", func() *qacct.JobsBuilder { return qa.Jobs().Owner("root") }},
				{"owner + queue", func() *qacct.JobsBuilder { return qa.Jobs().Owner("root").Queue("all.q") }},
				{"owner + days", func() *qacct.JobsBuilder { return qa.Jobs().Owner("root").LastDays(1) }},
				{"queue + host", func() *qacct.JobsBuilder { return qa.Jobs().Queue("all.q").Host("master") }},
				{"complex combination", func() *qacct.JobsBuilder { 
					return qa.Jobs().Owner("root").Queue("all.q").Host("master").LastDays(1) 
				}},
			}

			for _, tc := range testCases {
				By("Testing job details: " + tc.name)
				jobs, err := tc.builder().Execute()
				Expect(err).NotTo(HaveOccurred())
				// Should get valid job list (may be empty)
				for _, job := range jobs {
					Expect(job.JobNumber).To(BeNumerically(">", 0))
					Expect(job.Owner).NotTo(BeEmpty())
				}
			}
		})
	})
})