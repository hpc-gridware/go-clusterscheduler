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

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct"
)

var _ = Describe("QacctImpl", func() {

	// We need to test the native specification.

	Context("Native specification", func() {

		It("should return the native specification", func() {

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
})
