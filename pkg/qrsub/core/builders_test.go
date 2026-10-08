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

package core_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

var _ = Describe("ReservationBuilder", func() {
	ctx := context.Background()
	var client *core.CommandLineQRSub

	BeforeEach(func() {
		var err error
		client, err = core.NewCommandLineQRSub(core.CommandLineQRSubConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
	})

	It("builds the same argv as the options path", func() {
		start := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
		b := core.NewReservationBuilder(client).
			Start(start).
			Duration(4*time.Hour).
			Name("maint-gpu").
			Account("ops").
			Resource("gpu", "2").
			Queue("gpu.q").
			MasterQueue("gpu.q@node1").
			PE("mpi", "16").
			Checkpoint("ck").
			Users("alice", "@hpcteam").
			MailOptions("be").
			MailTo("ops@example.com").
			HardErrorHandling().
			Immediate()
		fromOptions, err := core.BuildQrsubArgs(core.ReservationOptions{
			StartTime:         &start,
			Duration:          core.ToPtr(4 * time.Hour),
			Name:              core.ToPtr("maint-gpu"),
			Account:           core.ToPtr("ops"),
			Resources:         map[string]string{"gpu": "2"},
			Queues:            []string{"gpu.q"},
			MasterQueues:      []string{"gpu.q@node1"},
			PEName:            core.ToPtr("mpi"),
			PESlots:           core.ToPtr("16"),
			Checkpoint:        core.ToPtr("ck"),
			Users:             []string{"alice", "@hpcteam"},
			MailOptions:       core.ToPtr("be"),
			MailList:          []string{"ops@example.com"},
			HardErrorHandling: core.ToPtr(true),
			Immediate:         core.ToPtr(true),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Args()).To(Equal(fromOptions))

		id, out, err := b.Submit(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(BeZero())
		Expect(out).To(HavePrefix("Dry run: qrsub -a "))
	})

	It("verifies with -w v", func() {
		found, out, err := core.NewReservationBuilder(client).Duration(time.Hour).Resource("gpu", "8").Verify(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeFalse())
		Expect(out).To(Equal("Dry run: qrsub -d 01:00:00 -l gpu=8 -w v"))
	})

	It("passes 9.1 binding switches through Flag", func() {
		b := core.NewReservationBuilder(client).Duration(time.Hour).Flag("-bamount", "2").Flag("-bunit", "C")
		_, _, err := b.Verify(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Args()).To(Equal([]string{"-d", "01:00:00", "-bamount", "2", "-bunit", "C"}))
	})

	It("checks binding values with the v9.1 grammar", func() {
		_, _, err := core.NewReservationBuilder(client).Duration(time.Hour).Flag("-bunit", "X1").Submit(ctx)
		Expect(err).To(MatchError(ContainSubstring("binding -bunit")))
		_, _, err = core.NewReservationBuilder(client).Duration(time.Hour).Flag("-bstrategy", "packed").Submit(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns a copy of the arguments", func() {
		b := core.NewReservationBuilder(client).Duration(time.Hour)
		args := b.Args()
		args[0] = "-@"
		Expect(b.Args()[0]).To(Equal("-d"))
	})

	It("reports a recorded error from Verify before running qrsub", func() {
		_, _, err := core.NewReservationBuilder(client).Duration(time.Hour).Queue("-u").Verify(ctx)
		Expect(err).To(MatchError(ContainSubstring("must not start with '-'")))
	})

	It("requires at least one element in list setters", func() {
		_, _, err := core.NewReservationBuilder(client).Duration(time.Hour).MailTo().Submit(ctx)
		Expect(err).To(MatchError(ContainSubstring("list must not be empty")))
	})

	DescribeTable("rejects the same bad input as the options path",
		func(build func(*core.ReservationBuilder), opts core.ReservationOptions) {
			b := core.NewReservationBuilder(client).Duration(time.Hour)
			build(b)
			_, _, builderErr := b.Submit(ctx)
			Expect(builderErr).To(HaveOccurred())
			opts.Duration = core.ToPtr(time.Hour)
			_, optionsErr := core.BuildQrsubArgs(opts)
			Expect(optionsErr).To(HaveOccurred())
			Expect(builderErr.Error()).To(Equal(optionsErr.Error()))
		},
		Entry("queue", func(b *core.ReservationBuilder) { b.Queue("a,b") }, core.ReservationOptions{Queues: []string{"a,b"}}),
		Entry("user", func(b *core.ReservationBuilder) { b.Users("-f") }, core.ReservationOptions{Users: []string{"-f"}}),
		Entry("resource", func(b *core.ReservationBuilder) { b.Resource("h_rt", "1,a=b") }, core.ReservationOptions{Resources: map[string]string{"h_rt": "1,a=b"}}),
		Entry("pe name", func(b *core.ReservationBuilder) { b.PE("-u", "2") }, core.ReservationOptions{PEName: core.ToPtr("-u"), PESlots: core.ToPtr("2")}),
		Entry("pe slots", func(b *core.ReservationBuilder) { b.PE("mpi", "x") }, core.ReservationOptions{PEName: core.ToPtr("mpi"), PESlots: core.ToPtr("x")}),
		Entry("checkpoint", func(b *core.ReservationBuilder) { b.Checkpoint("") }, core.ReservationOptions{Checkpoint: core.ToPtr("")}),
		Entry("mail options", func(b *core.ReservationBuilder) { b.MailOptions("s") }, core.ReservationOptions{MailOptions: core.ToPtr("s")}),
		Entry("name", func(b *core.ReservationBuilder) { b.Name("") }, core.ReservationOptions{Name: core.ToPtr("")}),
		Entry("account", func(b *core.ReservationBuilder) { b.Account("a\nb") }, core.ReservationOptions{Account: core.ToPtr("a\nb")}),
	)

	DescribeTable("refuses dangerous or conflicting Flag names",
		func(name string) {
			_, _, err := core.NewReservationBuilder(client).Duration(time.Hour).Flag(name, "x").Submit(ctx)
			Expect(err).To(MatchError(ContainSubstring("is not allowed")))
			GinkgoT().Setenv(validate.EnvVar, "off")
			_, _, err = core.NewReservationBuilder(client).Duration(time.Hour).Flag(name, "x").Submit(ctx)
			Expect(err).To(MatchError(ContainSubstring("is not allowed")))
		},
		Entry("options file", "-@"),
		Entry("verify mode", "-w"),
		Entry("resource list, which has its own checks", "-l"),
		Entry("user list, which has its own checks", "-u"),
		Entry("start time, which the builder must track", "-a"),
		Entry("pe, which takes two values", "-pe"),
		Entry("no dash", "N"),
		Entry("value as name", "-N x"),
	)

	DescribeTable("refuses Flag values which would shift or inject switches",
		func(value string) {
			_, _, err := core.NewReservationBuilder(client).Duration(time.Hour).Flag("-bamount", value).Name("-now").Submit(ctx)
			Expect(err).To(MatchError(ContainSubstring("binding -bamount")))
		},
		Entry("missing value", ""),
		Entry("switch as value", "-now"),
	)

	It("requires an end time or duration", func() {
		_, _, err := core.NewReservationBuilder(client).Name("x").Submit(ctx)
		Expect(err).To(MatchError(ContainSubstring("end time or a duration")))
	})

	It("requires the end after the start", func() {
		start := time.Now().Add(time.Hour)
		_, _, err := core.NewReservationBuilder(client).Start(start).End(start.Add(-time.Minute)).Submit(ctx)
		Expect(err).To(MatchError(ContainSubstring("not after start")))
	})

	It("rejects invalid durations even with GCS_VALIDATION=off", func() {
		GinkgoT().Setenv(validate.EnvVar, "off")
		_, _, err := core.NewReservationBuilder(client).Duration(-time.Second).Submit(ctx)
		Expect(err).To(HaveOccurred())
		_, _, err = core.NewReservationBuilder(client).Duration(time.Hour).Queue("-u").Submit(ctx)
		Expect(err).NotTo(HaveOccurred())
	})
})
