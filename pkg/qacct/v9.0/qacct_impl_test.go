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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
)

var _ = Describe("QacctImpl", func() {

	// We need to test the native specification.

	Context("Native specification", func() {

		It("should return the help output using the native specification", func() {
			q, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())

			result, err := q.NativeSpecification([]string{"-help"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).
				To(ContainSubstring("usage: qacct [options]"))
		})

		It("should return an error if the command does not exist", func() {
			q, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "nonexistent-command",
				})
			Expect(err).To(HaveOccurred())
			Expect(q).To(BeNil())
		})

		It("should return an error if the qacct argument is invalid", func() {
			q, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(q).NotTo(BeNil())

			result, err := q.NativeSpecification([]string{"-invalid-argument"})
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

	})

	Context("ShowJobDetails", func() {
		var qa qacct.QAcct
		var qs qsub.Qsub

		BeforeEach(func() {
			var err error
			qa, err = qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(qa).NotTo(BeNil())

			qs, err = qsub.NewCommandLineQSub(
				qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())
			Expect(qs).NotTo(BeNil())

		})

		It("should return the job details", func() {
			// submit a job
			var err error

			jobID, _, err := qs.Submit(
				context.Background(),
				qsub.JobOptions{
					Command:     "sleep",
					CommandArgs: []string{"0"},
					Binary:      qsub.ToPtr(true),
					JobName:     qsub.ToPtr("test-job"),
					Account:     qsub.ToPtr("test-account"),
					Synchronize: qsub.ToPtr(true),
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(jobID).NotTo(BeZero())

			// ensure that accounting has been updated
			<-time.After(1 * time.Second)

			result, err := qa.ShowJobDetails([]int64{jobID})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())
			Expect(result[0].JobNumber).To(Equal(jobID))
			Expect(result[0].JobName).To(Equal("test-job"))
			Expect(result[0].Account).To(Equal("test-account"))
		})

		It("should return the job details for multiple jobs", func() {

			jobID, _, err := qs.Submit(
				context.Background(),
				qsub.JobOptions{
					Command:     "sleep",
					CommandArgs: []string{"0"},
					Binary:      qsub.ToPtr(true),
					JobName:     qsub.ToPtr("test-job2"),
					Account:     qsub.ToPtr("test-account"),
					Synchronize: qsub.ToPtr(true),
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(jobID).NotTo(BeZero())

			// ensure that accounting has been updated
			<-time.After(1 * time.Second)

			result, err := qa.ShowJobDetails([]int64{jobID})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())
			Expect(result[0].JobNumber).To(Equal(jobID))
			Expect(result[0].JobName).To(Equal("test-job2"))
			Expect(result[0].Account).To(Equal("test-account"))

			// list all jobs
			result, err = qa.ShowJobDetails(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())
			// more than 1 job
			Expect(len(result)).To(BeNumerically(">", 1))
			// contains our job
			Expect(result).To(ContainElement(
				SatisfyAll(
					HaveField("JobNumber", Equal(jobID)),
					HaveField("JobName", Equal("test-job2")),
					HaveField("Account", Equal("test-account")),
				)))

		})
	})

})
