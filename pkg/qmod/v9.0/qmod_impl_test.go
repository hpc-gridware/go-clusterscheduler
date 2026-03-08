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

package qmod_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qmod "github.com/hpc-gridware/go-clusterscheduler/pkg/qmod/v9.0"
)

var _ = Describe("QmodImpl", func() {

	Context("Basic functionality", func() {

		It("should be able to create a new qmod instance", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{})
			Expect(q).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should default executable to qmod", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{})
			Expect(err).To(BeNil())
			Expect(q).NotTo(BeNil())
		})

		It("should accept a custom executable path", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				Executable: "/usr/bin/qmod",
			})
			Expect(err).To(BeNil())
			Expect(q).NotTo(BeNil())
		})

		It("should accept force flag", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				Force: true,
			})
			Expect(err).To(BeNil())
			Expect(q).NotTo(BeNil())
		})
	})

	Context("Dry-run mode", func() {

		It("should run ClearErrorState in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.ClearErrorState([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run ClearJobErrorState in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.ClearJobErrorState([]string{"123"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run ClearQueueErrorState in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.ClearQueueErrorState([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run Disable in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.Disable([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run Enable in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.Enable([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run Suspend in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.Suspend([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run SuspendJobs in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.SuspendJobs([]string{"123", "456"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run SuspendQueues in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.SuspendQueues([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run Unsuspend in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.Unsuspend([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run UnsuspendJobs in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.UnsuspendJobs([]string{"123"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run UnsuspendQueues in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.UnsuspendQueues([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run RescheduleJobs in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.RescheduleJobs([]string{"123"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run RescheduleQueues in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.RescheduleQueues([]string{"all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run NativeSpecification in dry-run mode", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.NativeSpecification([]string{"-d", "all.q"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Error handling", func() {

		It("should return error when no targets specified for Disable", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			_, err = q.Disable([]string{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("no targets specified"))
		})

		It("should return error when no targets specified for Enable", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			_, err = q.Enable([]string{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("no targets specified"))
		})

		It("should return error when no targets specified for SuspendJobs", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			_, err = q.SuspendJobs([]string{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("no targets specified"))
		})
	})

	Context("Interface compliance", func() {

		It("should implement the QMod interface", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{})
			Expect(err).To(BeNil())
			var _ qmod.QMod = q
		})
	})
})
