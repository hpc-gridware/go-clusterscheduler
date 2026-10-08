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
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/core"
)

func fixture(version, name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", version, name))
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

// fakeQrdel writes a qrdel stand-in that records its argv in a file, prints
// the given fixture and exits with the given code.
func fakeQrdel(fixtureFile, exitCode string) (executable, argvFile string) {
	dir := GinkgoT().TempDir()
	argvFile = filepath.Join(dir, "argv")
	executable = filepath.Join(dir, "qrdel")
	src, err := filepath.Abs(fixtureFile)
	Expect(err).NotTo(HaveOccurred())
	script := "#!/bin/sh\nfor a in \"$@\"; do echo \"$a\"; done > '" + argvFile + "'\ncat '" + src + "'\nexit " + exitCode + "\n"
	Expect(os.WriteFile(executable, []byte(script), 0o700)).To(Succeed())
	return executable, argvFile
}

var _ = Describe("CommandLineQRDel", func() {
	ctx := context.Background()

	BeforeEach(func() {
		// The injection checks follow GCS_VALIDATION; pin the default so the
		// specs do not depend on the caller's environment.
		GinkgoT().Setenv(validate.EnvVar, "")
	})

	It("fails if the executable does not exist", func() {
		_, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: "no-such-qrdel"})
		Expect(err).To(HaveOccurred())
	})

	Context("argument building", func() {
		It("deletes by id", func() {
			args, err := core.BuildDeleteArgs(false, 3, 17, math.MaxUint32)
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"3,17,4294967295"}))
			args, err = core.BuildDeleteArgs(true, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"-f", "3"}))
		})

		It("scopes deletion by name to the owner", func() {
			args, err := core.BuildDeleteByNameArgs(false, "alice", "maint", "bin*")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"-u", "alice", "maint,bin*"}))
			args, err = core.BuildDeleteByNameArgs(true, "alice", "maint")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"-f", "-u", "alice", "maint"}))
		})

		DescribeTable("requires an owner naming exactly one user, even with GCS_VALIDATION=off",
			func(owner string) {
				GinkgoT().Setenv(validate.EnvVar, "off")
				_, err := core.BuildDeleteByNameArgs(false, owner, "maint")
				Expect(err).To(MatchError(ContainSubstring("owner")))
			},
			Entry("empty", ""),
			Entry("all users", "*"),
			Entry("pattern", "a*"),
			Entry("two users", "alice,bob"),
			Entry("flag", "-f"),
		)

		It("deletes by user, where * means all users", func() {
			args, err := core.BuildDeleteByUserArgs(true, "alice", "*")
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]string{"-f", "-u", "alice,*"}))
		})

		DescribeTable("refuses empty selections, which would otherwise be easy to get wrong",
			func(build func() ([]string, error)) {
				_, err := build()
				Expect(err).To(HaveOccurred())
			},
			Entry("ids", func() ([]string, error) { return core.BuildDeleteArgs(false) }),
			Entry("names", func() ([]string, error) { return core.BuildDeleteByNameArgs(false, "alice") }),
			Entry("users", func() ([]string, error) { return core.BuildDeleteByUserArgs(false) }),
		)

		DescribeTable("rejects ids qmaster cannot address",
			func(id int64) {
				_, err := core.BuildDeleteArgs(false, 1, id)
				Expect(err).To(MatchError(ContainSubstring("invalid advance reservation id")))
			},
			Entry("zero", int64(0)),
			Entry("negative", int64(-5)),
			Entry("wraps to another AR above 32 bit", int64(math.MaxUint32)+1),
		)

		DescribeTable("rejects injected names",
			func(name string) {
				_, err := core.BuildDeleteByNameArgs(false, "alice", name)
				Expect(err).To(HaveOccurred())
				Expect(core.ValidateReservationName(name)).To(HaveOccurred())
			},
			Entry("user flag", "-u"),
			Entry("force flag", "-f"),
			Entry("second selector", "mine,*"),
			Entry("space", "has space"),
			Entry("newline", "a\nb"),
			Entry("empty", ""),
			Entry("decimal id", "42"),
			Entry("signed id", "+42"),
			Entry("hex id", "0x2a"),
			Entry("octal id", "052"),
		)

		DescribeTable("rejects injected users",
			func(user string) {
				_, err := core.BuildDeleteByUserArgs(false, user)
				Expect(err).To(HaveOccurred())
				Expect(core.ValidateUsers([]string{user})).To(HaveOccurred())
				Expect(core.ValidateExactUsers([]string{user})).To(HaveOccurred())
			},
			Entry("flag", "-f"),
			Entry("second user", "alice,*"),
			Entry("control", "alice\x00"),
		)

		It("separates exact users from patterns", func() {
			for _, pattern := range []string{"*", "a*", "a?", "[ab]x"} {
				Expect(core.ValidateUsers([]string{pattern})).To(Succeed(), pattern)
				Expect(core.ValidateExactUsers([]string{pattern})).To(HaveOccurred(), pattern)
			}
			Expect(core.ValidateExactUsers([]string{"alice", "bob.smith"})).To(Succeed())
		})

		It("keeps the exported validators enforcing when GCS_VALIDATION=off", func() {
			GinkgoT().Setenv(validate.EnvVar, "off")
			Expect(core.ValidateReservationName("-u")).To(HaveOccurred())
			Expect(core.ValidateUsers([]string{"-f"})).To(HaveOccurred())
			Expect(core.ValidateExactUsers([]string{"*"})).To(HaveOccurred())
		})

		It("applies the object name charset in strict mode", func() {
			GinkgoT().Setenv(validate.EnvVar, "strict")
			_, err := core.BuildDeleteByNameArgs(false, "alice", "a:b")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("ParseDeleteOutput", func() {
		DescribeTable("reads the captured qrdel outputs",
			func(version, file string, want core.DeleteResult) {
				Expect(core.ParseDeleteOutput(fixture(version, file))).To(Equal(want))
			},
			Entry("9.1 deleted", "9.1", "qrdel_id.stdout", core.DeleteResult{Deleted: []int64{10}}),
			Entry("9.1 registered", "9.1", "qrdel_registered.stdout", core.DeleteResult{Registered: []int64{8}}),
			Entry("9.1 missing", "9.1", "qrdel_missing.stdout", core.DeleteResult{Missing: []string{"9999"}}),
			Entry("9.1 partial", "9.1", "qrdel_partial.stdout", core.DeleteResult{Deleted: []int64{11}, Missing: []string{"9998"}}),
			Entry("9.0 partial", "9.0", "qrdel_partial.stdout", core.DeleteResult{Deleted: []int64{4}, Missing: []string{"9998"}}),
			Entry("9.1 user without ARs", "9.1", "qrdel_baduser.stdout", core.DeleteResult{Missing: []string{"nosuchuser"}}),
		)

		It("reads privilege denials", func() {
			out := "tu1 has deleted advance_reservation 261\n" +
				"tu1 - you do not have the necessary privileges to delete the advance_reservation \"263\"\n"
			Expect(core.ParseDeleteOutput(out)).To(Equal(core.DeleteResult{Deleted: []int64{261}, Denied: []string{"263"}}))
		})
	})

	Context("dry run", func() {
		It("returns the command line, including -f from the config", func() {
			q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{DryRun: true, Force: true})
			Expect(err).NotTo(HaveOccurred())
			out, err := q.DeleteReservations(ctx, 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qrdel -f 1,2"))
			out, err = q.DeleteReservationsByName(ctx, "alice", "maint")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qrdel -f -u alice maint"))
			out, err = q.DeleteReservationsByUser(ctx, "alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qrdel -f -u alice"))
		})
	})

	Context("with a fake qrdel", func() {
		It("passes the validated selectors and returns the output", func() {
			exe, argv := fakeQrdel("testdata/9.1/qrdel_id.stdout", "0")
			q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			out, err := q.DeleteReservations(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("root has deleted advance_reservation 10\n"))
			recorded, err := os.ReadFile(argv)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(recorded)).To(Equal("10\n"))
		})

		It("deletes the ARs of users", func() {
			exe, argv := fakeQrdel("testdata/9.1/qrdel_user.stdout", "0")
			q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			out, err := q.DeleteReservationsByUser(ctx, "root")
			Expect(err).NotTo(HaveOccurred())
			Expect(core.ParseDeleteOutput(out).Deleted).NotTo(BeEmpty())
			recorded, err := os.ReadFile(argv)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(recorded)).To(Equal("-u\nroot\n"))
		})

		It("treats registered for deletion as success", func() {
			exe, _ := fakeQrdel("testdata/9.1/qrdel_registered.stdout", "0")
			q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			out, err := q.DeleteReservations(ctx, 8)
			Expect(err).NotTo(HaveOccurred())
			Expect(core.ParseDeleteOutput(out).Registered).To(Equal([]int64{8}))
		})

		DescribeTable("wraps ErrNotFound when nothing matched",
			func(version, file string) {
				exe, _ := fakeQrdel("testdata/"+version+"/"+file, "1")
				q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
				Expect(err).NotTo(HaveOccurred())
				_, err = q.DeleteReservations(ctx, 9999)
				Expect(err).To(MatchError(core.ErrNotFound))
				var cmdErr *core.CommandError
				Expect(errors.As(err, &cmdErr)).To(BeTrue())
				Expect(cmdErr.ExitCode).To(Equal(1))
			},
			Entry("9.0 id", "9.0", "qrdel_missing.stdout"),
			Entry("9.1 id", "9.1", "qrdel_missing.stdout"),
			Entry("9.1 user", "9.1", "qrdel_baduser.stdout"),
		)

		DescribeTable("returns a CommandError with the deleted ARs on partial failure",
			func(version string) {
				exe, _ := fakeQrdel("testdata/"+version+"/qrdel_partial.stdout", "1")
				q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
				Expect(err).NotTo(HaveOccurred())
				out, err := q.DeleteReservations(ctx, 11, 9998)
				var cmdErr *core.CommandError
				Expect(errors.As(err, &cmdErr)).To(BeTrue())
				Expect(cmdErr.ExitCode).To(Equal(1))
				Expect(errors.Is(err, core.ErrNotFound)).To(BeFalse())
				result := core.ParseDeleteOutput(out)
				Expect(result.Deleted).To(HaveLen(1))
				Expect(result.Missing).To(Equal([]string{"9998"}))
			},
			Entry("9.0", "9.0"),
			Entry("9.1", "9.1"),
		)

		It("reports an executable which cannot be started", func() {
			dir := GinkgoT().TempDir()
			exe := filepath.Join(dir, "qrdel")
			Expect(os.WriteFile(exe, []byte("#!/no/such/interpreter\n"), 0o700)).To(Succeed())
			q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.DeleteReservations(ctx, 1)
			Expect(err).To(MatchError(ContainSubstring("failed to run")))
			var cmdErr *core.CommandError
			Expect(errors.As(err, &cmdErr)).To(BeFalse())
		})

		It("never spawns qrdel for invalid input", func() {
			exe, argv := fakeQrdel("testdata/9.1/qrdel_id.stdout", "0")
			q, err := core.NewCommandLineQRDel(core.CommandLineQRDelConfig{Executable: exe})
			Expect(err).NotTo(HaveOccurred())
			_, err = q.DeleteReservationsByName(ctx, "alice", "-u")
			Expect(err).To(HaveOccurred())
			_, err = q.DeleteReservationsByName(ctx, "*", "maint")
			Expect(err).To(HaveOccurred())
			_, err = q.DeleteReservations(ctx, math.MaxUint32+1)
			Expect(err).To(HaveOccurred())
			_, err = q.NativeSpecification(ctx, []string{"1\n2"})
			Expect(err).To(HaveOccurred())
			_, statErr := os.Stat(argv)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
	})
})
