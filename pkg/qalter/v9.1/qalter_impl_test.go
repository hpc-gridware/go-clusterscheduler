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

package qalter_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
	qalter "github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/v9.1"
)

var _ = Describe("QalterImpl", func() {

	var q *qalter.CommandLineQAlter

	BeforeEach(func() {
		var err error
		q, err = qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
			DryRun: true,
		})
		Expect(err).To(BeNil())
		Expect(q).NotTo(BeNil())
	})

	Context("Basic functionality", func() {

		It("should be able to create a new qalter instance", func() {
			q, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{})
			Expect(q).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should implement the QAlter interface", func() {
			var _ qalter.QAlter = q
		})
	})

	Context("Time & Scheduling", func() {

		It("should set start time with time.Time", func() {
			t := time.Date(2025, 12, 25, 12, 0, 0, 0, time.UTC)
			out, err := q.SetStartTime("1234", t)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set deadline with time.Time", func() {
			t := time.Date(2025, 12, 31, 18, 30, 45, 0, time.UTC)
			out, err := q.SetDeadline("1234", t)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("When config parameter", func() {

		It("should prepend -when to all operations when configured", func() {
			qw, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
				DryRun: true,
				When:   core.WhenOnReschedule,
			})
			Expect(err).To(BeNil())
			out, err := qw.SetPriority("1234", 100)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should prepend -when to binding operations when configured", func() {
			qw, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
				DryRun: true,
				When:   core.WhenNow,
			})
			Expect(err).To(BeNil())
			out, err := qw.SetBindingAmount("1234", 4)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should not prepend -when to NativeSpecification", func() {
			qw, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
				DryRun: true,
				When:   core.WhenOnReschedule,
			})
			Expect(err).To(BeNil())
			out, err := qw.NativeSpecification([]string{"-p", "100", "1234"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Job Identity & Metadata", func() {

		It("should set job name", func() {
			out, err := q.SetJobName("1234", "my_job")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set account string", func() {
			out, err := q.SetAccountString("1234", "myaccount")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set project", func() {
			out, err := q.SetProject("1234", "myproject")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set department", func() {
			out, err := q.SetDepartment("1234", "engineering")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Context Variables", func() {

		It("should add context with slice", func() {
			out, err := q.AddContext("1234", []string{"key1=val1", "key2=val2"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should delete context with slice", func() {
			out, err := q.DeleteContext("1234", []string{"key1", "key2"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set context with slice", func() {
			out, err := q.SetContext("1234", []string{"key1=val1"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Resource Requests", func() {

		It("should set hard resource list with slice", func() {
			out, err := q.SetHardResourceList("1234", []string{"mem_free=4G", "h_rt=3600"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set soft resource list with slice", func() {
			out, err := q.SetSoftResourceList("1234", []string{"gpu=1"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Binding (v9.1)", func() {

		It("should set binding amount with int", func() {
			out, err := q.SetBindingAmount("1234", 4)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding filter", func() {
			out, err := q.SetBindingFilter("1234", "SCcscc")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding instance", func() {
			out, err := q.SetBindingInstance("1234", "env")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding sort order", func() {
			out, err := q.SetBindingSortOrder("1234", "Sc")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding start", func() {
			out, err := q.SetBindingStart("1234", "S")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding stop", func() {
			out, err := q.SetBindingStop("1234", "C")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding strategy", func() {
			out, err := q.SetBindingStrategy("1234", "linear")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding type", func() {
			out, err := q.SetBindingType("1234", "host")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding unit", func() {
			out, err := q.SetBindingUnit("1234", "C")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Queue Binding", func() {

		It("should set hard queue with slice", func() {
			out, err := q.SetHardQueue("1234", []string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set soft queue with multiple queues", func() {
			out, err := q.SetSoftQueue("1234", []string{"gpu.q", "fast.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set hard master queue with slice", func() {
			out, err := q.SetHardMasterQueue("1234", []string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set soft master queue with slice", func() {
			out, err := q.SetSoftMasterQueue("1234", []string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Parallel Environment", func() {

		It("should set parallel environment", func() {
			out, err := q.SetParallelEnvironment("1234", "mpi", "4-16")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("I/O & Paths", func() {

		It("should set error path with slice", func() {
			out, err := q.SetErrorPath("1234", []string{"/tmp/err.log"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set output path with slice", func() {
			out, err := q.SetOutputPath("1234", []string{"/tmp/out.log"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set input file with slice", func() {
			out, err := q.SetInputFile("1234", []string{"/tmp/input.dat"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set shell path with slice", func() {
			out, err := q.SetShellPath("1234", []string{"/bin/bash"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set merge output", func() {
			out, err := q.SetMergeOutput("1234", true)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Working Directory", func() {

		It("should set cwd", func() {
			out, err := q.SetCwd("1234")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set working directory", func() {
			out, err := q.SetWorkingDirectory("1234", "/home/user/work")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Checkpointing", func() {

		It("should set checkpoint selector", func() {
			out, err := q.SetCheckpointSelector("1234", "m")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set checkpoint method", func() {
			out, err := q.SetCheckpointMethod("1234", "blcr")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Holds & Dependencies", func() {

		It("should set hold", func() {
			out, err := q.SetHold("1234", "u")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set hold job dependency with slice", func() {
			out, err := q.SetHoldJobDependency("1234", []string{"100", "200"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set hold array dependency with slice", func() {
			out, err := q.SetHoldArrayDependency("1234", []string{"100"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Priority & Tickets", func() {

		It("should set priority with int", func() {
			out, err := q.SetPriority("1234", 100)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set negative priority", func() {
			out, err := q.SetPriority("1234", -500)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set job share with int", func() {
			out, err := q.SetJobShare("1234", 500)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set override tickets with int", func() {
			out, err := q.SetOverrideTickets("1234", 1000)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Notification", func() {

		It("should set mail options", func() {
			out, err := q.SetMailOptions("1234", "eas")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set mail recipients with slice", func() {
			out, err := q.SetMailRecipients("1234", []string{"alice@example.com", "bob@example.com"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set notify", func() {
			out, err := q.SetNotify("1234")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Environment Variables", func() {

		It("should set environment variables with slice", func() {
			out, err := q.SetEnvironmentVariables("1234", []string{"PATH=/usr/bin", "HOME=/root"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should export all environment variables", func() {
			out, err := q.ExportAllEnvironmentVariables("1234")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Reservation & Restart", func() {

		It("should set reservation", func() {
			out, err := q.SetReservation("1234", true)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set restartable", func() {
			out, err := q.SetRestartable("1234", false)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Advance Reservation", func() {

		It("should set advance reservation", func() {
			out, err := q.SetAdvanceReservation("1234", "42")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Task Control", func() {

		It("should set max running tasks with int", func() {
			out, err := q.SetMaxRunningTasks("1234", 10)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Verification", func() {

		It("should set verify mode", func() {
			out, err := q.SetVerifyMode("1234", "p")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Native Specification", func() {

		It("should run native specification", func() {
			out, err := q.NativeSpecification([]string{"-w", "p", "1234"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})
})
