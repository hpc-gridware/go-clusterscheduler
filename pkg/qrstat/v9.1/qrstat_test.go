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

package qrstat_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qrstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/v9.1"
)

var _ = Describe("QRStat v9.1", func() {
	It("lists and gets reservations through the core client", func() {
		c, err := qrstat.NewCommandLineQRStat(qrstat.CommandLineQRStatConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		var _ qrstat.QRStat = c
		out, err := c.NativeSpecification(context.Background(), []string{"-ar", "1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qrstat -ar 1"))
		Expect(qrstat.ValidateUsers([]string{"-u"})).To(HaveOccurred())
		Expect(qrstat.StateWaiting.Description()).To(Equal("waiting"))
	})
})
