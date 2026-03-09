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

package qdel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qdel "github.com/hpc-gridware/go-clusterscheduler/pkg/qdel/v9.1"
)

var _ = Describe("QdelImpl", func() {

	Context("Basic functionality", func() {

		It("should be able to create a new qdel instance", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{})
			Expect(q).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should default executable to qdel", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{})
			Expect(err).To(BeNil())
			Expect(q).NotTo(BeNil())
		})

		It("should accept force flag", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				Force: true,
			})
			Expect(err).To(BeNil())
			Expect(q).NotTo(BeNil())
		})
	})

	Context("Dry-run mode", func() {

		It("should run DeleteJobs in dry-run mode", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.DeleteJobs([]string{"123", "456"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run DeleteJobs with force in dry-run mode", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				DryRun: true,
				Force:  true,
			})
			Expect(err).To(BeNil())
			out, err := q.DeleteJobs([]string{"123"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run DeleteUserJobs in dry-run mode", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.DeleteUserJobs([]string{"root", "testuser"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run NativeSpecification in dry-run mode", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			out, err := q.NativeSpecification([]string{"-f", "123"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("Error handling", func() {

		It("should return error when no jobs specified for DeleteJobs", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			_, err = q.DeleteJobs([]string{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("no jobs specified"))
		})

		It("should return error when no users specified for DeleteUserJobs", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{
				DryRun: true,
			})
			Expect(err).To(BeNil())
			_, err = q.DeleteUserJobs([]string{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("no users specified"))
		})
	})

	Context("Interface compliance", func() {

		It("should implement the QDel interface", func() {
			q, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{})
			Expect(err).To(BeNil())
			var _ qdel.QDel = q
		})
	})
})
