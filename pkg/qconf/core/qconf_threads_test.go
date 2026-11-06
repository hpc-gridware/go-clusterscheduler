/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core/internal/fakeqconf"
)

// Captured from GCS 9.1.7 (LC_ALL=C) while the scheduler was stopped with
// "qconf -kt scheduler" and started again with "qconf -at scheduler".
const (
	stlRunning = "POOL            SIZE\n--------------------\nlistener        4\nreader          4\nscheduler       1\nworker          4\n"
	stlStopped = "POOL            SIZE\n--------------------\nlistener        4\nreader          4\nworker          4\n"
)

var _ = Describe("qmaster threads", func() {
	DescribeTable("reads qmaster's answer to -at and -kt, never the exit code (all exit 0)",
		func(out string, want core.ThreadAnswer) {
			got, ok := core.ParseThreadAnswer(out)
			Expect(ok).To(BeTrue())
			Expect(got).To(Equal(want))
		},
		Entry("-at, stopped", "scheduler has been started\n", core.ThreadStarted),
		Entry("-at, running", "scheduler thread is already running\n", core.ThreadAlreadyRunning),
		Entry("-kt, running", "scheduler thread terminated\n", core.ThreadTerminated),
		Entry("-kt, stopped", "scheduler thread is not running\n", core.ThreadNotRunning),
	)

	It("does not read an unknown answer as one of them", func() {
		_, ok := core.ParseThreadAnswer("denied: host \"node9\" is neither submit nor admin host\n")
		Expect(ok).To(BeFalse())
	})

	It("tells a running scheduler from a stopped one by its -stl pool", func() {
		Expect(core.SchedulerInThreads(core.ParseThreadList(stlRunning))).To(BeTrue())
		Expect(core.SchedulerInThreads(core.ParseThreadList(stlStopped))).To(BeFalse())
	})

	Context("through qconf", func() {
		BeforeEach(func() {
			if !fakeqconf.Available() {
				Skip("fakeqconf uses a bash script; skip on this platform")
			}
		})

		// qmaster answers -at and -kt on stderr, stdout empty, exit 0 (9.1.7).
		run := func(stderr string, rc int, call func(*core.CommandLineQConf) (core.ThreadAnswer, error)) (core.ThreadAnswer, []string, error) {
			f := fakeqconf.NewStderr(GinkgoT(), stderr, rc)
			defer f.Cleanup()
			qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: f.Path()})
			Expect(err).NotTo(HaveOccurred())
			answer, err := call(qc)
			return answer, f.Argv(), err
		}

		It("starts the scheduler with -at and reports the answer", func() {
			answer, argv, err := run("scheduler has been started\n", 0, func(qc *core.CommandLineQConf) (core.ThreadAnswer, error) {
				return qc.StartQmasterThreadContext(context.Background(), nil, core.SchedulerThread)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(argv).To(Equal([]string{"-at", "scheduler"}))
			Expect(answer).To(Equal(core.ThreadStarted))
		})

		It("kills the scheduler with -kt and reports a refusal as an answer, not as success", func() {
			answer, argv, err := run("scheduler thread is not running\n", 0, func(qc *core.CommandLineQConf) (core.ThreadAnswer, error) {
				return qc.KillQmasterThreadContext(context.Background(), nil, core.SchedulerThread)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(argv).To(Equal([]string{"-kt", "scheduler"}))
			Expect(answer).To(Equal(core.ThreadNotRunning))
		})

		It("turns an unknown answer into an error carrying it", func() {
			_, _, err := run("denied: only managers may do this\n", 0, func(qc *core.CommandLineQConf) (core.ThreadAnswer, error) {
				return qc.StartQmasterThreadContext(context.Background(), nil, core.SchedulerThread)
			})
			Expect(err).To(MatchError(ContainSubstring("denied: only managers may do this")))
		})

		It("reads a running scheduler from -stl's exit-1 answer", func() {
			f := fakeqconf.New(GinkgoT(), stlRunning, 1)
			defer f.Cleanup()
			qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: f.Path()})
			Expect(err).NotTo(HaveOccurred())
			running, err := qc.SchedulerRunningContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(running).To(BeTrue())
		})

		It("fails on an empty -stl answer with exit 0 (qmaster down), never reads it as stopped", func() {
			f := fakeqconf.New(GinkgoT(), "", 0)
			defer f.Cleanup()
			qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: f.Path()})
			Expect(err).NotTo(HaveOccurred())
			_, err = qc.SchedulerRunningContext(context.Background(), nil)
			Expect(err).To(MatchError(ContainSubstring("no thread list")))
		})

		It("still fails on -stl output without a thread list", func() {
			f := fakeqconf.New(GinkgoT(), "error: commlib error: can't connect to service\n", 1)
			defer f.Cleanup()
			qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: f.Path()})
			Expect(err).NotTo(HaveOccurred())
			_, err = qc.SchedulerRunningContext(context.Background(), nil)
			Expect(err).To(HaveOccurred())
		})

		It("refuses a thread name that is an option", func() {
			qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).NotTo(HaveOccurred())
			_, err = qc.StartQmasterThreadContext(context.Background(), nil, "-kt")
			Expect(err).To(HaveOccurred())
		})

		// qconf -stl exits 1 even when it prints the whole list (9.1.7).
		It("reads the scheduler state from -stl, which exits 1 with a valid list", func() {
			f := fakeqconf.New(GinkgoT(), stlStopped, 1)
			defer f.Cleanup()
			qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: f.Path()})
			Expect(err).NotTo(HaveOccurred())
			running, err := qc.SchedulerRunningContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(running).To(BeFalse())
			Expect(f.Argv()).To(Equal([]string{"-stl"}))
		})
	})
})
