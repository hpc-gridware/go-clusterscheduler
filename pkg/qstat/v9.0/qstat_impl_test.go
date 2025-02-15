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

package qstat_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

var _ = Describe("QstatImpl", func() {

	Context("NativeSpecification", func() {

		It("should return the command line", func() {
			var err error
			var q qstat.QStat

			q, err = qstat.NewCommandLineQstat(
				qstat.CommandLineQStatConfig{
					DryRun: true,
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(q.NativeSpecification([]string{"-j", "123"})).
				To(Equal("Dry run: qstat [-j 123]"))
		})

		It("should return the help", func() {
			q, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{
				DryRun: false,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(q.NativeSpecification([]string{"-help"})).
				To(ContainSubstring("usage: qstat [options]"))
		})

		It("should work without arguments", func() {
			q, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.NativeSpecification(nil)
			Expect(err).NotTo(HaveOccurred())
		})

	})

})
