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

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
	qrsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/v9.1"
)

var _ = Describe("QRSub v9.1", func() {
	ctx := context.Background()
	common := core.ReservationOptions{Name: qrsub.ToPtr("bind"), Duration: qrsub.ToPtr(10 * time.Minute)}

	It("appends the binding switches after the common options", func() {
		args, err := qrsub.BuildQrsubArgs(qrsub.ReservationOptions{
			ReservationOptions: common,
			BindingAmount:      qrsub.ToPtr(2),
			BindingUnit:        qrsub.ToPtr("C"),
			BindingFilter:      qrsub.ToPtr("ScCC"),
			BindingSort:        qrsub.ToPtr("Sc"),
			BindingStart:       qrsub.ToPtr("s"),
			BindingStop:        qrsub.ToPtr("S"),
			BindingStrategy:    qrsub.ToPtr("packed"),
			BindingInstance:    qrsub.ToPtr("set"),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{
			"-d", "00:10:00", "-N", "bind",
			"-bamount", "2", "-bunit", "C", "-bfilter", "ScCC", "-bsort", "Sc",
			"-bstart", "s", "-bstop", "S", "-bstrategy", "packed", "-binstance", "set",
		}))
	})

	It("omits unset binding switches", func() {
		args, err := qrsub.BuildQrsubArgs(qrsub.ReservationOptions{ReservationOptions: common})
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{"-d", "00:10:00", "-N", "bind"}))
	})

	DescribeTable("rejects invalid binding values",
		func(opts qrsub.ReservationOptions) {
			opts.ReservationOptions = common
			_, err := qrsub.BuildQrsubArgs(opts)
			Expect(err).To(HaveOccurred())
			Expect(qrsub.ValidateReservationOptions(opts)).To(HaveOccurred())
		},
		Entry("amount", qrsub.ReservationOptions{BindingAmount: qrsub.ToPtr(0)}),
		Entry("unit", qrsub.ReservationOptions{BindingUnit: qrsub.ToPtr("-u")}),
		Entry("instance", qrsub.ReservationOptions{BindingInstance: qrsub.ToPtr("host")}),
		Entry("strategy", qrsub.ReservationOptions{BindingStrategy: qrsub.ToPtr("linear")}),
		Entry("start", qrsub.ReservationOptions{BindingStart: qrsub.ToPtr("Ss")}),
		Entry("stop", qrsub.ReservationOptions{BindingStop: qrsub.ToPtr("-")}),
		Entry("sort", qrsub.ReservationOptions{BindingSort: qrsub.ToPtr("S,c")}),
		Entry("filter", qrsub.ReservationOptions{BindingFilter: qrsub.ToPtr("-ScC")}),
	)

	It("validates the common options too", func() {
		opts := qrsub.ReservationOptions{ReservationOptions: core.ReservationOptions{
			Duration: qrsub.ToPtr(time.Hour), Queues: []string{"a,b"},
		}}
		_, err := qrsub.BuildQrsubArgs(opts)
		Expect(err).To(HaveOccurred())
		GinkgoT().Setenv(validate.EnvVar, "off")
		Expect(qrsub.ValidateReservationOptions(opts)).To(HaveOccurred())
	})

	It("submits and verifies in dry run mode", func() {
		c, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		var _ qrsub.QRSub = c
		opts := qrsub.ReservationOptions{ReservationOptions: common, BindingAmount: qrsub.ToPtr(1)}
		id, out, err := c.Submit(ctx, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(BeZero())
		Expect(out).To(Equal("Dry run: qrsub -d 00:10:00 -N bind -bamount 1"))
		_, out, err = c.Verify(ctx, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(HaveSuffix("-bamount 1 -w v"))
		out, err = c.SubmitWithNativeSpecification(ctx, []string{"-d", "60"})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qrsub -d 60"))
	})

	It("works with the builder", func() {
		c, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		_, out, err := qrsub.NewReservationBuilder(c).Duration(time.Minute).Flag("-bamount", "1").Submit(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qrsub -d 00:01:00 -bamount 1"))
	})

	It("fails if the executable does not exist", func() {
		_, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{Executable: "no-such-qrsub"})
		Expect(err).To(HaveOccurred())
	})
})
