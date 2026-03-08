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

package qmod_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qmod "github.com/hpc-gridware/go-clusterscheduler/pkg/qmod/v9.1"
)

var _ = Describe("QmodImpl", func() {

	Context("Basic functionality", func() {

		It("should be able to create a new qmod instance", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{})
			Expect(q).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should implement the QMod interface", func() {
			q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{})
			Expect(err).To(BeNil())
			var _ qmod.QMod = q
		})
	})

	Context("Dry-run mode", func() {

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
	})
})
