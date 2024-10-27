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

package qsub_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
)

var _ = Describe("QsubImpl", func() {

	Context("NativeSpecification", func() {

		It("should return the native qsub command line specification for the given options", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())

			output, err := qs.SubmitWithNativeSpecification(ctx,
				[]string{"-help"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("usage: qsub"))
		})

		It("should return the job ID", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())

			output, err := qs.SubmitWithNativeSpecification(ctx,
				[]string{"-b", "y", "sleep", "10"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("Your job"))

			jobId, output, err := qs.SubmitSimpleBinary(ctx, "sleep", "0")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobId).To(BeNumerically(">", int64(0)))
			Expect(output).NotTo(Equal(""))

			//submit again
			jobId2, output, err := qs.SubmitSimpleBinary(ctx, "sleep", "0")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobId2).To(BeNumerically(">", int64(0)))
			Expect(output).NotTo(Equal(""))
			// jobId2 should be higher than jobId
			Expect(jobId2).To(BeNumerically(">", jobId))
		})

	})

	Context("Qsub command line options", func() {

		It("should be able to submit with scoped resources", func() {

			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(
				qsub.CommandLineQSubConfig{
					DryRun: true,
				})
			Expect(err).NotTo(HaveOccurred())

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				ScopedResources: map[string]map[string]qsub.ResourceRequest{
					qsub.ResourceRequestScopeGlobal: {
						qsub.ResourceRequestTypeHard: {
							Resources: map[string]string{
								"mem_free":    "4G",
								"np_load_avg": "1"},
						},
						qsub.ResourceRequestTypeSoft: {
							Resources: map[string]string{"m_core": "32"},
						},
					},
				},
			})

			Expect(output).To(Or(
				ContainSubstring("-scope global -hard -l mem_free=4G,np_load_avg=1 -soft -l m_core=32"),
				ContainSubstring("-scope global -hard -l np_load_avg=1,mem_free=4G -soft -l m_core=32")))
		})

		It("should be able to submit a job with all options", func() {

			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(
				qsub.CommandLineQSubConfig{
					DryRun: true,
				})
			Expect(err).NotTo(HaveOccurred())

			startTime := time.Now().Add(time.Hour)
			deadline := startTime.Add(time.Hour * 24)

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				StartTime:            qsub.ToPtr(startTime),
				Deadline:             qsub.ToPtr(deadline),
				AdvanceReservationID: qsub.ToPtr("12"),
				Account:              qsub.ToPtr("123"),
				Project:              qsub.ToPtr("456"),
				Priority:             qsub.ToPtr(100),
				Queue:                []string{"test.q", "highmem.q"},
				ParallelEnvironment:  qsub.ToPtr("smp 1-10"),
				StdOut:               []string{"/path/to/output"},
				StdIn:                []string{"/path/to/input"},
				StdErr:               []string{"/path/to/error"},
				WorkingDir:           qsub.ToPtr("/path/to/working"),
				Command:              "echo",
				CommandArgs:          []string{"hello", "world"},
				Binary:               qsub.ToPtr(true),
				ScopedResources: map[string]map[string]qsub.ResourceRequest{
					qsub.ResourceRequestScopeGlobal: {
						qsub.ResourceRequestTypeHard: {
							Resources: map[string]string{"mem_free": "4G", "np_load_avg": "1"},
						},
					},
				},
				ExportAllEnv:                    qsub.ToPtr(true),
				EnvVariables:                    map[string]string{"FOO": "BAR", "FOO2": "BAR2"},
				CommandPrefix:                   qsub.ToPtr("#DIRECTIVE"),
				Shell:                           qsub.ToPtr(true),
				CommandInterpreter:              qsub.ToPtr("/bin/bash"),
				JobName:                         qsub.ToPtr("test-job"),
				JobArray:                        qsub.ToPtr("1-10:2"),
				MaxRunningTasks:                 qsub.ToPtr(2),
				NotifyBeforeSuspend:             qsub.ToPtr(true),
				MasterQueue:                     []string{"test.q"},
				CommandFile:                     qsub.ToPtr("/path/to/command"),
				JobSubmissionVerificationScript: qsub.ToPtr("/path/to/script"),
				PTTY:                            qsub.ToPtr(true),
				Restartable:                     qsub.ToPtr(true),
				StartImmediately:                qsub.ToPtr(true),
				ReservationDesired:              qsub.ToPtr(true),
				MergeStdOutErr:                  qsub.ToPtr(true),
				JobShare:                        qsub.ToPtr(10),
				MailList:                        []string{"some@email.com"},
				Notify:                          qsub.ToPtr(true),
				Hold:                            qsub.ToPtr(true),
			})

			Expect(output).To(ContainSubstring(
				"-a " + qsub.ConvertTimeToQsubDateTime(startTime)))
			Expect(output).To(ContainSubstring(
				"-dl " + qsub.ConvertTimeToQsubDateTime(deadline)))
			Expect(output).To(ContainSubstring("-ar 12"))
			Expect(output).To(ContainSubstring("-A 123"))
			Expect(output).To(ContainSubstring("-P 456"))
			Expect(output).To(ContainSubstring("-p 100"))
			Expect(output).To(ContainSubstring("-q test.q,highmem.q"))
			Expect(output).To(ContainSubstring("-pe smp 1-10"))
			Expect(output).To(ContainSubstring("-o /path/to/output"))
			Expect(output).To(ContainSubstring("-i /path/to/input"))
			Expect(output).To(ContainSubstring("-e /path/to/error"))
			Expect(output).To(ContainSubstring("-wd /path/to/working"))
			Expect(output).To(ContainSubstring("echo hello world"))
			Expect(output).To(ContainSubstring("-b y"))
			Expect(output).To((Or(
				ContainSubstring("-l mem_free=4G,np_load_avg=1"),
				ContainSubstring("-l np_load_avg=1,mem_free=4G"))))
			Expect(output).To(ContainSubstring("-V"))
			Expect(output).To(Or(ContainSubstring("-v FOO=BAR,FOO2=BAR2"),
				ContainSubstring("-v FOO2=BAR2,FOO=BAR")))
			Expect(output).To(ContainSubstring("#DIRECTIVE"))
			Expect(output).To(ContainSubstring("-S /bin/bash"))
			Expect(output).To(ContainSubstring("-N test-job"))
			Expect(output).To(ContainSubstring("-t 1-10:2"))
			Expect(output).To(ContainSubstring("-notify"))
			Expect(output).To(ContainSubstring("-masterq test.q"))
			Expect(output).To(ContainSubstring("-@ /path/to/command"))
			Expect(output).To(ContainSubstring("-jsv /path/to/script"))
			Expect(output).To(ContainSubstring("-pty y"))
			Expect(output).To(ContainSubstring("-r y"))
			Expect(output).To(ContainSubstring("-now y"))
			Expect(output).To(ContainSubstring("-R y"))
			Expect(output).To(ContainSubstring("-j y"))
			Expect(output).To(ContainSubstring("-js 10"))
			Expect(output).To(ContainSubstring("-M some@email.com"))
			Expect(output).To(ContainSubstring("-notify"))
			Expect(output).To(ContainSubstring("-h"))
		})

		It("should be able to submit a job with simple resource requests", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())

			jobId, output, err := qs.Submit(ctx, qsub.JobOptions{
				ScopedResources: qsub.SimpleLRequest(map[string]string{"mem_free": "1M"}),
				Binary:          qsub.ToPtr(true),
				Command:         "echo",
				CommandArgs:     []string{"hello", "world"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(jobId).To(BeNumerically(">", int64(0)))
			Expect(output).To(ContainSubstring(fmt.Sprintf("%d", jobId)))
		})

	})

})
