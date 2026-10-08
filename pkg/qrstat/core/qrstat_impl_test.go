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
	"errors"
	"math"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/core"
)

// fakeQrstat writes a shell script that prints the given fixture to stdout
// and exits with the given code. It stands in for qrstat to test the exit
// code handling without a cluster.
func fakeQrstat(stdout string, exitCode string) string {
	dir := GinkgoT().TempDir()
	out := filepath.Join(dir, "out")
	Expect(os.WriteFile(out, []byte(stdout), 0o600)).To(Succeed())
	script := filepath.Join(dir, "qrstat")
	Expect(os.WriteFile(script, []byte("#!/bin/sh\ncat '"+out+"'\nexit "+exitCode+"\n"), 0o700)).To(Succeed())
	return script
}

var _ = Describe("CommandLineQRStat", func() {
	ctx := context.Background()

	It("fails if the executable does not exist", func() {
		_, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{Executable: "no-such-qrstat"})
		Expect(err).To(HaveOccurred())
	})

	Context("argument building", func() {
		It("builds the list arguments", func() {
			args, err := core.BuildListArgs(core.ListOptions{Users: []string{"alice", "*"}, Explain: true})
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"-u", "alice,*", "-explain"}))
			args, err = core.BuildListArgs(core.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(BeEmpty())
		})

		It("builds the detail arguments", func() {
			args, err := core.BuildGetArgs(1, 42)
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"-ar", "1,42"}))
		})

		DescribeTable("rejects invalid ids",
			func(ids ...int64) {
				_, err := core.BuildGetArgs(ids...)
				Expect(err).To(HaveOccurred())
			},
			Entry("none"),
			Entry("zero", int64(0)),
			Entry("negative", int64(1), int64(-1)),
			Entry("above 32 bit", int64(math.MaxUint32)+1),
		)

		It("separates exact users from patterns, independent of GCS_VALIDATION", func() {
			GinkgoT().Setenv(validate.EnvVar, "off")
			Expect(core.ValidateUsers([]string{"*"})).To(Succeed())
			Expect(core.ValidateExactUsers([]string{"*"})).To(HaveOccurred())
			Expect(core.ValidateExactUsers([]string{"a?"})).To(HaveOccurred())
			Expect(core.ValidateExactUsers([]string{"alice"})).To(Succeed())
		})

		DescribeTable("rejects injected user entries",
			func(user string) {
				_, err := core.BuildListArgs(core.ListOptions{Users: []string{user}})
				Expect(err).To(HaveOccurred())
				Expect(core.ValidateUsers([]string{user})).To(HaveOccurred())
			},
			Entry("flag", "-explain"),
			Entry("comma", "alice,bob"),
			Entry("space", "alice bob"),
			Entry("newline", "alice\n"),
			Entry("empty", ""),
		)

		It("keeps ValidateUsers enforcing when GCS_VALIDATION=off", func() {
			GinkgoT().Setenv(validate.EnvVar, "off")
			Expect(core.ValidateUsers([]string{"-u"})).To(HaveOccurred())
			_, err := core.BuildListArgs(core.ListOptions{Users: []string{"-u"}})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("dry run", func() {
		It("returns the command line and runs nothing", func() {
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{DryRun: true})
			Expect(err).NotTo(HaveOccurred())
			out, err := q.NativeSpecification(ctx, []string{"-u", "*"})
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qrstat -u *"))
			ars, err := q.ListReservations(ctx, core.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ars).To(BeEmpty())
			details, err := q.GetReservations(ctx, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(details).To(BeEmpty())
		})

		It("rejects control characters before anything runs", func() {
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{DryRun: true})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.NativeSpecification(ctx, []string{"-u", "a\nb"})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("with a fake qrstat", func() {
		It("parses the summary", func() {
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{
				Executable: fakeQrstat(fixture("9.1", "qrstat_summary.stdout"), "0"),
			})
			Expect(err).NotTo(HaveOccurred())
			ars, err := q.ListReservations(ctx, core.ListOptions{Users: []string{"*"}})
			Expect(err).NotTo(HaveOccurred())
			Expect(ars).To(HaveLen(7))
		})

		DescribeTable("maps the not found output to ErrNotFound",
			func(version string) {
				q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{
					Executable: fakeQrstat(fixture(version, "qrstat_missing.stdout"), "1"),
				})
				Expect(err).NotTo(HaveOccurred())
				_, err = q.GetReservations(ctx, 9999)
				Expect(errors.Is(err, core.ErrNotFound)).To(BeTrue())
				var cmdErr *core.CommandError
				Expect(errors.As(err, &cmdErr)).To(BeTrue())
			},
			Entry("9.0", "9.0"),
			Entry("9.1", "9.1"),
		)

		It("returns the details of the found ARs", func() {
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{
				Executable: fakeQrstat(fixture("9.1", "qrstat_partial.stdout"), "0"),
			})
			Expect(err).NotTo(HaveOccurred())
			ars, err := q.GetReservations(ctx, 8, 9999)
			Expect(err).NotTo(HaveOccurred())
			Expect(ars).To(HaveLen(1))
			Expect(ars[0].ID).To(Equal(int64(8)))
		})

		It("reports an executable which cannot be started", func() {
			exe := filepath.Join(GinkgoT().TempDir(), "qrstat")
			Expect(os.WriteFile(exe, []byte("#!/no/such/interpreter\n"), 0o700)).To(Succeed())
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.ListReservations(ctx, core.ListOptions{})
			Expect(err).To(MatchError(ContainSubstring("failed to run")))
		})

		It("returns a CommandError for other failures", func() {
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{
				Executable: fakeQrstat("error: commlib error\n", "1"),
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.ListReservations(ctx, core.ListOptions{})
			var cmdErr *core.CommandError
			Expect(errors.As(err, &cmdErr)).To(BeTrue())
			Expect(cmdErr.ExitCode).To(Equal(1))
			Expect(cmdErr.Output).To(ContainSubstring("commlib error"))
			Expect(errors.Is(err, core.ErrNotFound)).To(BeFalse())
		})

		It("never spawns qrstat for invalid input", func() {
			GinkgoT().Setenv(validate.EnvVar, "")
			q, err := core.NewCommandLineQRStat(core.CommandLineQRStatConfig{
				Executable: fakeQrstat("must not run", "0"),
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.ListReservations(ctx, core.ListOptions{Users: []string{"-x"}})
			Expect(err).To(MatchError(ContainSubstring("must not start with '-'")))
		})
	})
})
