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

package qsub_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.1"
)

var _ = Describe("Qsub v9.1", func() {

	Context("Basic functionality", func() {

		It("should be able to create a new qsub instance in dry-run mode", func() {
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(qs).NotTo(BeNil())
		})

		It("should return an error if the executable does not exist", func() {
			_, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				QsubPath: "nonexistent-qsub-binary",
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Common options (inherited from core)", func() {

		It("should build common options correctly in dry-run", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			startTime := time.Now().Add(time.Hour)

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				JobOptions: core.JobOptions{
					StartTime:          qsub.ToPtr(startTime),
					Account:            qsub.ToPtr("test-account"),
					Project:            qsub.ToPtr("test-project"),
					Priority:           qsub.ToPtr(100),
					Queue:              []string{"all.q"},
					Binary:             qsub.ToPtr(true),
					JobName:            qsub.ToPtr("test-job"),
					Command:            "echo",
					CommandArgs:        []string{"hello"},
					ExportAllEnv:       qsub.ToPtr(true),
					MergeStdOutErr:     qsub.ToPtr(true),
					Restartable:        qsub.ToPtr(true),
					ReservationDesired: qsub.ToPtr(true),
					StartImmediately:   qsub.ToPtr(true),
					PTTY:               qsub.ToPtr(true),
					MasterQueue:        []string{"all.q"},
				},
			})

			Expect(output).To(ContainSubstring(
				"-a " + qsub.ConvertTimeToQsubDateTime(startTime)))
			Expect(output).To(ContainSubstring("-A test-account"))
			Expect(output).To(ContainSubstring("-P test-project"))
			Expect(output).To(ContainSubstring("-p 100"))
			Expect(output).To(ContainSubstring("-q all.q"))
			Expect(output).To(ContainSubstring("-b y"))
			Expect(output).To(ContainSubstring("-N test-job"))
			Expect(output).To(ContainSubstring("echo hello"))
			Expect(output).To(ContainSubstring("-V"))
			Expect(output).To(ContainSubstring("-j y"))
			Expect(output).To(ContainSubstring("-r y"))
			Expect(output).To(ContainSubstring("-R y"))
			Expect(output).To(ContainSubstring("-now y"))
			Expect(output).To(ContainSubstring("-pty y"))
			Expect(output).To(ContainSubstring("-masterq all.q"))
		})

		It("should support scoped resources", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				JobOptions: core.JobOptions{
					ScopedResources: qsub.SimpleLRequest(
						map[string]string{"mem_free": "4G"}),
					Binary:      qsub.ToPtr(true),
					Command:     "echo",
					CommandArgs: []string{"test"},
				},
			})

			Expect(output).To(ContainSubstring("-l mem_free=4G"))
		})
	})

	Context("v9.1 binding options", func() {

		It("should support all new binding options", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				JobOptions: core.JobOptions{
					Binary:      qsub.ToPtr(true),
					Command:     "sleep",
					CommandArgs: []string{"0"},
				},
				BindingAmount:   qsub.ToPtr(4),
				BindingStop:     qsub.ToPtr("S"),
				BindingFilter:   qsub.ToPtr("SCc"),
				BindingInstance: qsub.ToPtr("set"),
				BindingSort:     qsub.ToPtr("Sc"),
				BindingStart:    qsub.ToPtr("s"),
				BindingStrategy: qsub.ToPtr("linear"),
				BindingType:     qsub.ToPtr("host"),
				BindingUnit:     qsub.ToPtr("C"),
			})

			Expect(output).To(ContainSubstring("-bamount 4"))
			Expect(output).To(ContainSubstring("-bstop S"))
			Expect(output).To(ContainSubstring("-bfilter SCc"))
			Expect(output).To(ContainSubstring("-binstance set"))
			Expect(output).To(ContainSubstring("-bsort Sc"))
			Expect(output).To(ContainSubstring("-bstart s"))
			Expect(output).To(ContainSubstring("-bstrategy linear"))
			Expect(output).To(ContainSubstring("-btype host"))
			Expect(output).To(ContainSubstring("-bunit C"))
		})

		It("should support partial binding options", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				JobOptions: core.JobOptions{
					Binary:      qsub.ToPtr(true),
					Command:     "sleep",
					CommandArgs: []string{"0"},
				},
				BindingAmount:   qsub.ToPtr(8),
				BindingStrategy: qsub.ToPtr("linear"),
				BindingType:     qsub.ToPtr("slot"),
			})

			Expect(output).To(ContainSubstring("-bamount 8"))
			Expect(output).To(ContainSubstring("-bstrategy linear"))
			Expect(output).To(ContainSubstring("-btype slot"))
			Expect(output).NotTo(ContainSubstring("-bstop"))
			Expect(output).NotTo(ContainSubstring("-bfilter"))
			Expect(output).NotTo(ContainSubstring("-binding"))
		})

		It("should not emit legacy -binding flag", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, output, _ := qs.Submit(ctx, qsub.JobOptions{
				JobOptions: core.JobOptions{
					Binary:           qsub.ToPtr(true),
					Command:          "sleep",
					CommandArgs:      []string{"0"},
					ProcessorBinding: qsub.ToPtr("linear:4"),
				},
				BindingAmount: qsub.ToPtr(4),
			})

			Expect(output).NotTo(ContainSubstring("-binding"))
			Expect(output).To(ContainSubstring("-bamount 4"))
		})
	})

	Context("Simplified submission", func() {

		It("should support SubmitSimpleBinary in dry-run", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, output, _ := qs.SubmitSimpleBinary(ctx, "sleep", "0")
			Expect(output).To(ContainSubstring("-b y"))
			Expect(output).To(ContainSubstring("sleep 0"))
		})

		It("should support SubmitSimple in dry-run", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			opts := &qsub.JobOptions{}
			opts.Binary = qsub.ToPtr(true)
			_, output, _ := qs.SubmitSimple(ctx, opts, "echo", "hello")
			Expect(output).To(ContainSubstring("-b y"))
			Expect(output).To(ContainSubstring("echo hello"))
		})

		It("should support SubmitWithNativeSpecification in dry-run", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())

			output, err := qs.SubmitWithNativeSpecification(ctx,
				[]string{"-b", "y", "-bamount", "4", "sleep", "0"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("-b y"))
			Expect(output).To(ContainSubstring("-bamount 4"))
		})
	})

	Context("Interface compliance", func() {

		It("should implement the Qsub interface", func() {
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())
			var _ qsub.Qsub = qs
		})
	})

	Context("JobBuilder", func() {

		var qs qsub.Qsub

		BeforeEach(func() {
			var err error
			qs, err = qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{
				DryRun: true,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should build a simple binary job", func() {
			ctx := context.Background()
			_, output, err := qsub.NewJobBuilder(qs, "sleep", "10").
				Binary().
				Name("builder-job").
				Submit(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("-terse"))
			Expect(output).To(ContainSubstring("-b y"))
			Expect(output).To(ContainSubstring("-N builder-job"))
			Expect(output).To(ContainSubstring("sleep 10"))
		})

		It("should support common options with v9.1 client", func() {
			ctx := context.Background()
			startTime := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)

			_, output, err := qsub.NewJobBuilder(qs, "echo", "hello").
				Binary().
				Name("full-job").
				Account("acct1").
				Project("proj1").
				Priority(100).
				Queue("all.q").
				Resource("mem_free", "4G").
				StdOut("/tmp/out").
				MergeOutput().
				StartTime(startTime).
				ExportAllEnv().
				Env("FOO", "bar").
				Reservation().
				Restartable(true).
				Submit(ctx)

			Expect(err).NotTo(HaveOccurred())
			for _, s := range []string{
				"-b y", "-N full-job", "-A acct1", "-P proj1",
				"-p 100", "-q all.q", "-l mem_free=4G",
				"-o /tmp/out", "-j y",
				"-a " + qsub.ConvertTimeToQsubDateTime(startTime),
				"-V", "-v FOO=bar", "-R y", "-r y",
				"echo hello",
			} {
				Expect(output).To(ContainSubstring(s),
					fmt.Sprintf("expected output to contain %q", s))
			}
		})

		It("should support v9.1 binding options via Flag()", func() {
			ctx := context.Background()
			_, output, err := qsub.NewJobBuilder(qs, "sleep", "0").
				Binary().
				Flag("-bamount", "4").
				Flag("-bstrategy", "linear").
				Flag("-btype", "host").
				Flag("-bunit", "C").
				Submit(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("-bamount 4"))
			Expect(output).To(ContainSubstring("-bstrategy linear"))
			Expect(output).To(ContainSubstring("-btype host"))
			Expect(output).To(ContainSubstring("-bunit C"))
			Expect(output).NotTo(ContainSubstring("-binding"))
		})

		It("should support Flag() without value for boolean flags", func() {
			ctx := context.Background()
			_, output, err := qsub.NewJobBuilder(qs, "sleep", "0").
				Binary().
				Flag("-notify").
				Submit(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("-notify"))
		})
	})
})
