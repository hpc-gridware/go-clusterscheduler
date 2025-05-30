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

package qalter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qalter "github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/v9.0"
)

var _ = Describe("QalterImpl", func() {

	Context("Basic functionality", func() {

		It("should be able to create a new qalter instance", func() {
			qalter, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{})
			Expect(qalter).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should be able to run in dry-run mode", func() {
			qalter, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
				DryRun: true,
			})
			Expect(qalter).NotTo(BeNil())
			Expect(err).To(BeNil())
			output, err := qalter.RunCommand("qalter", "-w", "p", "whymyjobnotrunning")
			Expect(err).To(BeNil())
			Expect(output).To(BeEmpty())
		})

	})

})
