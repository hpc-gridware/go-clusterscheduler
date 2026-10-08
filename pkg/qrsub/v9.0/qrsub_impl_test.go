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

package qrsub_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qrsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/v9.0"
)

var _ = Describe("QRSub v9.0", func() {
	It("submits through the core client", func() {
		c, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		var _ qrsub.QRSub = c
		_, out, err := c.Submit(context.Background(), qrsub.ReservationOptions{
			Name:     qrsub.ToPtr("maint"),
			Duration: qrsub.ToPtr(time.Hour),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qrsub -d 01:00:00 -N maint"))
	})

	It("offers the builder and the validators", func() {
		c, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		args := qrsub.NewReservationBuilder(c).Duration(time.Minute).Args()
		Expect(args).To(Equal([]string{"-d", "00:01:00"}))
		Expect(qrsub.ValidateReservationOptions(qrsub.ReservationOptions{})).To(HaveOccurred())
		Expect(qrsub.IsLimitReached(nil)).To(BeFalse())
	})
})
