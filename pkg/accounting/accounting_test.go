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

package accounting_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/accounting"
)

var _ = Describe("Accounting", func() {

	Context("GetUsageFilePath", func() {

		It("should return the usage file path", func() {
			os.Unsetenv("SGE_JOB_SPOOL_DIR")
			usageFilePath, err := accounting.GetUsageFilePath()
			Expect(err).To(HaveOccurred())
			Expect(usageFilePath).To(Equal(""))
		})

		It("should return the usage file path", func() {
			os.Setenv("SGE_JOB_SPOOL_DIR", "/var/spool/gridengine/job_spool")
			usageFilePath, err := accounting.GetUsageFilePath()
			Expect(err).NotTo(HaveOccurred())
			Expect(usageFilePath).To(Equal("/var/spool/gridengine/job_spool/usage"))
		})

	})
})
