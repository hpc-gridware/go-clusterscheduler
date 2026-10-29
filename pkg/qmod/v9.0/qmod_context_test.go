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

package qmod_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qmod "github.com/hpc-gridware/go-clusterscheduler/pkg/qmod/v9.0"
)

// fakeQmod writes a qmod stand-in that records its arguments, runs body
// and exits with rc. It returns the script and the file of the recorded
// arguments.
func fakeQmod(body, rc string) (string, string) {
	dir := GinkgoT().TempDir()
	args := filepath.Join(dir, "args")
	path := filepath.Join(dir, "qmod")
	script := "#!/bin/sh\necho \"$*\" > " + args + "\n" + body + "\nexit " + rc + "\n"
	Expect(os.WriteFile(path, []byte(script), 0o755)).To(Succeed())
	return path, args
}

func newQmod(config qmod.CommandLineQModConfig) *qmod.CommandLineQMod {
	q, err := qmod.NewCommandLineQMod(config)
	Expect(err).NotTo(HaveOccurred())
	return q
}

func recordedArgs(file string) string {
	b, err := os.ReadFile(file)
	Expect(err).NotTo(HaveOccurred())
	return string(b)
}

var _ = Describe("qmod under a context", func() {
	It("hands over every answer line and the exit error, never just one of them", func() {
		// The answer shapes of a live qmod (9.0.12 to 9.1.7): a mixed run exits 0.
		path, args := fakeQmod(`echo 'Queue instance "a.q@n1" is already in the specified state: disabled'
echo 'root@master changed state of "a.q@n2" (disabled)'`, "0")
		var runner qmod.ContextRunner = newQmod(qmod.CommandLineQModConfig{Executable: path})
		out, _, err := runner.ActionContext(context.Background(), nil, "-d", []string{"a.q@n1", "a.q@n2"})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("already in the specified state"))
		Expect(out).To(ContainSubstring("changed state of"))
		Expect(recordedArgs(args)).To(Equal("-d a.q@n1,a.q@n2\n"))
	})

	It("keeps stdout when qmod exits 1, with stderr in the error", func() {
		path, _ := fakeQmod(`echo 'eve - you have no permission to modify queue "a.q@n1"'
echo 'some warning' >&2`, "1")
		q := newQmod(qmod.CommandLineQModConfig{Executable: path})
		out, stderr, err := q.ActionContext(context.Background(), nil, "-d", []string{"a.q@n1"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("failed to run command (some warning): exit status 1"))
		Expect(out).To(ContainSubstring("no permission"))
		Expect(stderr).To(Equal("some warning\n"))
	})

	It("passes -f when Force is set, and the chosen environment", func() {
		path, args := fakeQmod(`echo "$QMOD_TEST"`, "0")
		q := newQmod(qmod.CommandLineQModConfig{Executable: path, Force: true})
		out, _, err := q.ActionContext(context.Background(), []string{"QMOD_TEST=scrubbed"}, "-e", []string{"*@n1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("scrubbed\n"))
		Expect(recordedArgs(args)).To(Equal("-f -e *@n1\n"))
	})

	It("builds the same arguments as the context-free actions", func() {
		path, args := fakeQmod("", "0")
		q := newQmod(qmod.CommandLineQModConfig{Executable: path, Force: true})
		_, err := q.Disable([]string{"a.q@n1", "b.q@n2"})
		Expect(err).NotTo(HaveOccurred())
		plain := recordedArgs(args)
		_, _, err = q.ActionContext(context.Background(), nil, "-d", []string{"a.q@n1", "b.q@n2"})
		Expect(err).NotTo(HaveOccurred())
		Expect(recordedArgs(args)).To(Equal(plain))
	})

	DescribeTable("refuses targets that could forge an option or a second target, before spawning qmod",
		func(targets []string) {
			path, args := fakeQmod("", "0")
			q := newQmod(qmod.CommandLineQModConfig{Executable: path})
			_, _, err := q.ActionContext(context.Background(), nil, "-d", targets)
			Expect(err).To(HaveOccurred())
			Expect(args).NotTo(BeAnExistingFile())
		},
		Entry("an option", []string{"-u"}),
		Entry("two targets in one", []string{"a.q@n1,b.q@n2"}),
		Entry("no target", nil),
	)

	It("returns the whole answer however long it is", func() {
		// 30000 answer lines, about 1.8 MB: a wildcard target on a large cluster.
		path, _ := fakeQmod(`i=0; while [ $i -lt 30000 ]; do echo "root@master changed state of \"all.q@node$i\" (disabled)"; i=$((i+1)); done`, "0")
		q := newQmod(qmod.CommandLineQModConfig{Executable: path})
		out, _, err := q.ActionContext(context.Background(), nil, "-d", []string{"*"})
		Expect(err).NotTo(HaveOccurred())
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		Expect(lines).To(HaveLen(30000))
		Expect(lines[29999]).To(ContainSubstring(`"all.q@node29999"`))
	})

	It("caps stderr without blocking qmod", func() {
		path, _ := fakeQmod("head -c 200000 /dev/zero | tr '\\0' x >&2", "0")
		q := newQmod(qmod.CommandLineQModConfig{Executable: path})
		_, stderr, err := q.RunContext(context.Background(), nil, "-d", "a.q@n1")
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(HaveLen(64 * 1024))
	})

	It("echoes the command in dry-run mode and runs nothing", func() {
		q := newQmod(qmod.CommandLineQModConfig{DryRun: true, Force: true})
		out, stderr, err := q.ActionContext(context.Background(), nil, "-e", []string{"all.q@n1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qmod [-f -e all.q@n1]"))
		Expect(stderr).To(BeEmpty())
	})

	It("ends the run when the context ends", func() {
		path, _ := fakeQmod("sleep 5", "0")
		q := newQmod(qmod.CommandLineQModConfig{Executable: path})
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, _, err := q.ActionContext(ctx, nil, "-d", []string{"a.q@n1"})
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(HavePrefix("failed to run command ("))
		Expect(time.Since(start)).To(BeNumerically("<", 3*time.Second))
	})
})
