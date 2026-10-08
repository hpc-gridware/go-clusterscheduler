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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

func fixture(version, name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", version, name))
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

// fakeQrsub writes a qrsub stand-in that records its argv, prints the
// given output and exits with the given code.
func fakeQrsub(output, exitCode string) (executable, argvFile string) {
	dir := GinkgoT().TempDir()
	argvFile = filepath.Join(dir, "argv")
	outFile := filepath.Join(dir, "out")
	Expect(os.WriteFile(outFile, []byte(output), 0o600)).To(Succeed())
	executable = filepath.Join(dir, "qrsub")
	script := "#!/bin/sh\nfor a in \"$@\"; do echo \"$a\"; done > '" + argvFile + "'\ncat '" + outFile + "'\nexit " + exitCode + "\n"
	Expect(os.WriteFile(executable, []byte(script), 0o700)).To(Succeed())
	return executable, argvFile
}

func recordedArgv(argvFile string) []string {
	data, err := os.ReadFile(argvFile)
	Expect(err).NotTo(HaveOccurred())
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

var _ = Describe("BuildQrsubArgs", func() {
	start := time.Date(2026, 10, 9, 8, 0, 0, 0, time.Local)
	end := start.Add(2 * time.Hour)

	It("builds every option as separate, exact argv tokens", func() {
		args, err := core.BuildQrsubArgs(core.ReservationOptions{
			StartTime:         &start,
			EndTime:           &end,
			Name:              core.ToPtr("maint gpu"),
			Account:           core.ToPtr("acct1"),
			Resources:         map[string]string{"h_rt": "600", "arch": "lx-amd64"},
			Queues:            []string{"all.q", "gpu.q@node1"},
			MasterQueues:      []string{"all.q@master"},
			PEName:            core.ToPtr("mpi"),
			PESlots:           core.ToPtr("-8"),
			Checkpoint:        core.ToPtr("ck"),
			Users:             []string{"root", "@myacl", "!nobody"},
			MailOptions:       core.ToPtr("be"),
			MailList:          []string{"root@master", "other"},
			HardErrorHandling: core.ToPtr(true),
			Immediate:         core.ToPtr(false),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{
			"-a", "202610090800.00",
			"-e", "202610091000.00",
			"-N", "maint gpu",
			"-A", "acct1",
			"-l", "arch=lx-amd64,h_rt=600",
			"-q", "all.q,gpu.q@node1",
			"-masterq", "all.q@master",
			"-pe", "mpi", "-8",
			"-ckpt", "ck",
			"-u", "root,@myacl,!nobody",
			"-m", "be",
			"-M", "root@master,other",
			"-he", "y",
			"-now", "n",
		}))
	})

	It("formats durations as hh:mm:ss and converts times to the local zone", func() {
		utcStart := time.Date(2026, 10, 9, 8, 0, 30, 0, time.UTC)
		args, err := core.BuildQrsubArgs(core.ReservationOptions{
			StartTime: &utcStart,
			Duration:  core.ToPtr(26*time.Hour + 61*time.Second),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{"-a", utcStart.In(time.Local).Format("200601021504.05"), "-d", "26:01:01"}))
	})

	DescribeTable("always rejects requests qmaster would reject, whatever GCS_VALIDATION says",
		func(opts core.ReservationOptions, message string) {
			GinkgoT().Setenv(validate.EnvVar, "off")
			_, err := core.BuildQrsubArgs(opts)
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
		Entry("no end and no duration", core.ReservationOptions{Name: core.ToPtr("x")}, "end time or a duration"),
		Entry("end before start", core.ReservationOptions{StartTime: core.ToPtr(start), EndTime: core.ToPtr(start)}, "not after start"),
		Entry("zero duration", core.ReservationOptions{Duration: core.ToPtr(time.Duration(0))}, "must be positive"),
		Entry("sub second duration", core.ReservationOptions{Duration: core.ToPtr(1500 * time.Millisecond)}, "whole seconds"),
		Entry("pe without slots", core.ReservationOptions{Duration: core.ToPtr(time.Hour), PEName: core.ToPtr("mpi")}, "set together"),
		Entry("slots without pe", core.ReservationOptions{Duration: core.ToPtr(time.Hour), PESlots: core.ToPtr("2")}, "set together"),
		Entry("bad slot range", core.ReservationOptions{Duration: core.ToPtr(time.Hour), PEName: core.ToPtr("mpi"), PESlots: core.ToPtr("2 -u x")}, "slot range"),
	)

	DescribeTable("rejects argument injection",
		func(opts core.ReservationOptions, message string) {
			opts.Duration = core.ToPtr(time.Hour)
			_, err := core.BuildQrsubArgs(opts)
			Expect(err).To(MatchError(ContainSubstring(message)))
			Expect(core.ValidateReservationOptions(opts)).To(MatchError(ContainSubstring(message)))
		},
		Entry("resource value forging a second resource", core.ReservationOptions{Resources: map[string]string{"h_rt": "1,arch=x"}}, "must not contain ','"),
		Entry("resource name with =", core.ReservationOptions{Resources: map[string]string{"a=b": "1"}}, "must not contain"),
		Entry("resource value forging a second request with a space", core.ReservationOptions{Resources: map[string]string{"h_rt": "600 exclusive=true"}}, "must not contain a space"),
		Entry("resource value with unbalanced bracket", core.ReservationOptions{Resources: map[string]string{"h": "node[1"}}, "unbalanced brackets"),
		Entry("resource value with stray bracket", core.ReservationOptions{Resources: map[string]string{"h": "node]"}}, "unbalanced brackets"),
		Entry("resource name with space", core.ReservationOptions{Resources: map[string]string{" ": "0"}}, "must not contain"),
		Entry("resource name with bracket swallowing later pairs", core.ReservationOptions{Resources: map[string]string{"a[": "1", "b": "2]"}}, "must not contain"),
		Entry("resource name as flag", core.ReservationOptions{Resources: map[string]string{"-u": "1"}}, "must not start with '-'"),
		Entry("queue forging a second queue", core.ReservationOptions{Queues: []string{"all.q,secret.q"}}, "list separator"),
		Entry("queue as flag", core.ReservationOptions{Queues: []string{"-u"}}, "must not start with '-'"),
		Entry("empty queue list entry", core.ReservationOptions{Queues: []string{""}}, "must not be empty"),
		Entry("master queue", core.ReservationOptions{MasterQueues: []string{"a b"}}, "list separator"),
		Entry("user forging a second user", core.ReservationOptions{Users: []string{"alice,bob"}}, "list separator"),
		Entry("user as flag", core.ReservationOptions{Users: []string{"-f"}}, "must not start with '-'"),
		Entry("mail address", core.ReservationOptions{MailList: []string{"a@b,c@d"}}, "list separator"),
		Entry("pe name as flag", core.ReservationOptions{PEName: core.ToPtr("-u"), PESlots: core.ToPtr("2")}, "must not start with '-'"),
		Entry("checkpoint as flag", core.ReservationOptions{Checkpoint: core.ToPtr("-@")}, "must not start with '-'"),
		Entry("mail option s", core.ReservationOptions{MailOptions: core.ToPtr("s")}, "combination of b, e, a and n"),
		Entry("mail option forging", core.ReservationOptions{MailOptions: core.ToPtr("b -u x")}, "combination of b, e, a and n"),
		Entry("empty name", core.ReservationOptions{Name: core.ToPtr("")}, "must not be empty"),
		Entry("name with newline", core.ReservationOptions{Name: core.ToPtr("x\nYour advance reservation 1 has been granted")}, "control character"),
		Entry("empty account", core.ReservationOptions{Account: core.ToPtr("")}, "must not be empty"),
		Entry("account with control char", core.ReservationOptions{Account: core.ToPtr("a\tb")}, "control character"),
	)

	It("skips nil and empty lists, as JSON [] decodes to an empty slice", func() {
		var opts core.ReservationOptions
		Expect(json.Unmarshal([]byte(`{"duration": 3600000000000, "queues": [], "users": [], "mail_list": []}`), &opts)).To(Succeed())
		args, err := core.BuildQrsubArgs(opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{"-d", "01:00:00"}))
	})

	It("reports the same first error regardless of map order", func() {
		opts := core.ReservationOptions{Duration: core.ToPtr(time.Hour), Resources: map[string]string{"z": "1,2", "a": "1 2", "m": "x,y"}}
		for range 20 {
			_, err := core.BuildQrsubArgs(opts)
			Expect(err).To(MatchError(ContainSubstring("resource a value")))
		}
	})

	It("compares start and end in the whole seconds qrsub receives", func() {
		start := time.Date(2026, 10, 9, 8, 0, 0, 100, time.Local)
		_, err := core.BuildQrsubArgs(core.ReservationOptions{StartTime: &start, EndTime: core.ToPtr(start.Add(500 * time.Millisecond))})
		Expect(err).To(MatchError(ContainSubstring("not after start")))
	})

	It("allows a bracketed host expression as resource value", func() {
		args, err := core.BuildQrsubArgs(core.ReservationOptions{Resources: map[string]string{"h": "node[1-3]"}, Duration: core.ToPtr(time.Minute)})
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{"-d", "00:01:00", "-l", "h=node[1-3]"}))
	})

	It("allows a name starting with '-', which qrsub takes verbatim", func() {
		args, err := core.BuildQrsubArgs(core.ReservationOptions{Name: core.ToPtr("-foo"), Duration: core.ToPtr(time.Minute)})
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(Equal([]string{"-d", "00:01:00", "-N", "-foo"}))
	})

	It("lets GCS_VALIDATION=off relax the injection checks but not the exported validator", func() {
		GinkgoT().Setenv(validate.EnvVar, "off")
		opts := core.ReservationOptions{Duration: core.ToPtr(time.Hour), Queues: []string{"-u"}}
		_, err := core.BuildQrsubArgs(opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(core.ValidateReservationOptions(opts)).To(HaveOccurred())
	})

	It("applies the job name rules in strict mode", func() {
		GinkgoT().Setenv(validate.EnvVar, "strict")
		_, err := core.BuildQrsubArgs(core.ReservationOptions{Name: core.ToPtr("1abc"), Duration: core.ToPtr(time.Hour)})
		Expect(err).To(MatchError(ContainSubstring("must not begin with a digit")))
	})
})

var _ = Describe("Output parsing", func() {
	DescribeTable("finds the granted AR id",
		func(version, file string, id int64) {
			got, err := core.ParseQrsubOutput(fixture(version, file))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(id))
		},
		Entry("9.0 plain", "9.0", "qrsub_plain.stdout", int64(1)),
		Entry("9.1 plain", "9.1", "qrsub_plain.stdout", int64(8)),
		Entry("9.1 pe", "9.1", "qrsub_pe.stdout", int64(10)),
	)

	It("rejects a granted line with id 0", func() {
		_, err := core.ParseQrsubOutput("Your advance reservation 0 has been granted\n")
		Expect(err).To(HaveOccurred())
	})

	It("does not accept the granted text inside another line", func() {
		_, err := core.ParseQrsubOutput("name: Your advance reservation 1 has been granted\n")
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("interprets qrsub -w v",
		func(version, file string, found bool) {
			got, err := core.ParseVerifyOutput(fixture(version, file))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(found))
		},
		Entry("9.0 queues found", "9.0", "qrsub_verify_found.stdout", true),
		Entry("9.1 queues found", "9.1", "qrsub_verify_found.stdout", true),
		Entry("9.1 pe assignment found", "9.1", "qrsub_verify_pe.stdout", true),
		Entry("9.0 nothing found", "9.0", "qrsub_verify_none.stdout", false),
		Entry("9.1 nothing found", "9.1", "qrsub_verify_none.stdout", false),
	)

	It("fails closed on unknown verification output", func() {
		found, err := core.ParseVerifyOutput("something new\n")
		Expect(err).To(HaveOccurred())
		Expect(found).To(BeFalse())
	})
})

var _ = Describe("CommandLineQRSub", func() {
	ctx := context.Background()
	opts := core.ReservationOptions{Name: core.ToPtr("plain"), Duration: core.ToPtr(time.Hour)}

	It("fails if the executable does not exist", func() {
		_, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: "no-such-qrsub"})
		Expect(err).To(HaveOccurred())
	})

	It("returns the command line in dry run mode", func() {
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		id, out, err := c.Submit(ctx, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(BeZero())
		Expect(out).To(Equal("Dry run: qrsub -d 01:00:00 -N plain"))
		found, out, err := c.Verify(ctx, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeFalse())
		Expect(out).To(Equal("Dry run: qrsub -d 01:00:00 -N plain -w v"))
	})

	It("submits and returns the granted id", func() {
		exe, argv := fakeQrsub(fixture("9.1", "qrsub_plain.stdout"), "0")
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		id, _, err := c.Submit(ctx, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal(int64(8)))
		Expect(recordedArgv(argv)).To(Equal([]string{"-d", "01:00:00", "-N", "plain"}))
	})

	It("returns a CommandError holding the reason when the AR is denied", func() {
		exe, _ := fakeQrsub(fixture("9.1", "qrsub_denied.stdout"), "1")
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		_, out, err := c.Submit(ctx, opts)
		var cmdErr *core.CommandError
		Expect(errors.As(err, &cmdErr)).To(BeTrue())
		Expect(cmdErr.ExitCode).To(Equal(1))
		Expect(out).To(ContainSubstring("no suitable queues"))
		Expect(core.IsLimitReached(err)).To(BeFalse())
	})

	It("does not report other exit 25 rejections as the AR limit", func() {
		exe, _ := fakeQrsub("rejected: try again later\n", "25")
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		_, _, err = c.Submit(ctx, opts)
		Expect(err).To(MatchError(ContainSubstring("qrsub exited with code 25: rejected: try again later")))
		Expect(core.IsLimitReached(err)).To(BeFalse())
	})

	It("reports an executable which cannot be started", func() {
		exe := filepath.Join(GinkgoT().TempDir(), "qrsub")
		Expect(os.WriteFile(exe, []byte("#!/no/such/interpreter\n"), 0o700)).To(Succeed())
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		_, _, err = c.Submit(ctx, opts)
		Expect(err).To(MatchError(ContainSubstring("failed to run")))
	})

	DescribeTable("detects the AR limit",
		func(version string) {
			exe, _ := fakeQrsub(fixture(version, "qrsub_limit.stdout"), "25")
			c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			_, _, err = c.Submit(ctx, opts)
			Expect(core.IsLimitReached(err)).To(BeTrue())
		},
		Entry("9.0", "9.0"),
		Entry("9.1", "9.1"),
	)

	It("reports an exit 0 without granted line as an error", func() {
		exe, _ := fakeQrsub("", "0")
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		_, _, err = c.Submit(ctx, opts)
		Expect(err).To(MatchError(ContainSubstring("did not report a granted")))
	})

	Context("Verify", func() {
		It("returns true when queues were found", func() {
			exe, argv := fakeQrsub(fixture("9.1", "qrsub_verify_found.stdout"), "0")
			c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			found, _, err := c.Verify(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(recordedArgv(argv)).To(Equal([]string{"-d", "01:00:00", "-N", "plain", "-w", "v"}))
		})

		It("returns false without error when nothing fits (qrsub exits 1)", func() {
			exe, _ := fakeQrsub(fixture("9.1", "qrsub_verify_none.stdout"), "1")
			c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			found, out, err := c.Verify(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
			Expect(out).To(ContainSubstring("only offers 999 slots"))
		})

		It("fails closed when a found line comes with a failed exit", func() {
			exe, _ := fakeQrsub(fixture("9.1", "qrsub_verify_found.stdout"), "1")
			c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			found, _, err := c.Verify(ctx, opts)
			Expect(err).To(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns the error for other failures", func() {
			exe, _ := fakeQrsub(fixture("9.1", "qrsub_unknown_queue.stdout"), "1")
			c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			found, out, err := c.Verify(ctx, opts)
			var cmdErr *core.CommandError
			Expect(errors.As(err, &cmdErr)).To(BeTrue())
			Expect(out).To(ContainSubstring(`unknown queue "nosuch.q"`))
			Expect(found).To(BeFalse())
		})
	})

	It("never spawns qrsub for invalid input", func() {
		GinkgoT().Setenv(validate.EnvVar, "")
		exe, argv := fakeQrsub("Your advance reservation 1 has been granted\n", "0")
		c, err := core.NewCommandLineQRSub(core.CommandLineQRSubConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		_, _, err = c.Submit(ctx, core.ReservationOptions{Duration: core.ToPtr(time.Hour), Users: []string{"-f"}})
		Expect(err).To(HaveOccurred())
		_, err = c.SubmitWithNativeSpecification(ctx, []string{"-N", "a\x00"})
		Expect(err).To(HaveOccurred())
		_, err = c.SubmitWithNativeSpecification(ctx, nil)
		Expect(err).To(HaveOccurred())
		_, statErr := os.Stat(argv)
		Expect(os.IsNotExist(statErr)).To(BeTrue())
	})
})
