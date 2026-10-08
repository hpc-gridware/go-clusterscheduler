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

package qrdel_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qrdel "github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/v9.0"
)

var _ = Describe("QRDel v9.0", func() {
	It("deletes through the core client", func() {
		c, err := qrdel.NewCommandLineQRDel(qrdel.CommandLineQRDelConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		var _ qrdel.QRDel = c
		out, err := c.DeleteReservations(context.Background(), 7)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qrdel 7"))
		Expect(qrdel.ValidateReservationName("-f")).To(HaveOccurred())
		Expect(qrdel.ValidateUsers([]string{"a,b"})).To(HaveOccurred())
	})
})
