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
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

var _ = Describe("Watching the accounting file", func() {
	var (
		path   string
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		path = filepath.Join(GinkgoT().TempDir(), "accounting.jsonl")
		ctx, cancel = context.WithCancel(context.Background())
		DeferCleanup(func() { cancel() })
	})

	write := func(text string) {
		Expect(os.WriteFile(path, []byte(text), 0o644)).To(Succeed())
	}
	appendText := func(text string) {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
		Expect(err).NotTo(HaveOccurred())
		_, err = f.WriteString(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
	}
	watch := func(o core.WatchOptions) <-chan core.JobDetail {
		if o.PollInterval == 0 {
			o.PollInterval = 10 * time.Millisecond
		}
		if o.BufferSize == 0 {
			o.BufferSize = 100
		}
		jobs, err := core.WatchFileFrom(ctx, path, o)
		Expect(err).NotTo(HaveOccurred())
		return jobs
	}
	// next returns the job numbers of the next n records.
	next := func(jobs <-chan core.JobDetail, n int) []int64 {
		var ids []int64
		for len(ids) < n {
			var job core.JobDetail
			Eventually(jobs, "3s").Should(Receive(&job))
			ids = append(ids, job.JobNumber)
		}
		return ids
	}

	It("emits what is in the file, then what is written to it", func() {
		write(acctLine(1, 0) + "\n" + acctLine(2, 0) + "\n")
		jobs := watch(core.WatchOptions{})
		Expect(next(jobs, 2)).To(Equal([]int64{1, 2}))
		appendText(acctLine(3, 0) + "\n")
		Expect(next(jobs, 1)).To(Equal([]int64{3}))
		Consistently(jobs, "50ms").ShouldNot(Receive())
	})

	It("emits only what is written after the call with FromEnd", func() {
		// The last line is one qmaster is still writing: it belongs to
		// the watch.
		write(acctLine(1, 0) + "\n" + acctLine(2, 0)[:30])
		jobs := watch(core.WatchOptions{FromEnd: true})
		Consistently(jobs, "50ms").ShouldNot(Receive())
		appendText(acctLine(2, 0)[30:] + "\n" + acctLine(3, 0) + "\n")
		Expect(next(jobs, 2)).To(Equal([]int64{2, 3}))
	})

	It("goes on after a line longer than 64 KiB", func() {
		write(acctLine(10, 0) + "\n" + acctLine(11, 70000) + "\n" + acctLine(12, 0) + "\n")
		Expect(next(watch(core.WatchOptions{}), 3)).To(Equal([]int64{10, 11, 12}))
	})

	It("passes over a line which is no record, and says so", func() {
		var reported atomic.Int32
		write(acctLine(10, 0) + "\nnot a record\n" + acctLine(11, 5000) + "\n" + acctLine(12, 0) + "\n")
		jobs := watch(core.WatchOptions{MaxLineBytes: 2000, OnError: func(error) { reported.Add(1) }})
		Expect(next(jobs, 2)).To(Equal([]int64{10, 12}))
		Expect(reported.Load()).To(BeNumerically(">=", 1))
	})

	It("waits for a record which is written in two parts", func() {
		second := acctLine(21, 0) + "\n"
		write(acctLine(20, 0) + "\n" + second[:60])
		jobs := watch(core.WatchOptions{})
		Expect(next(jobs, 1)).To(Equal([]int64{20}))
		Consistently(jobs, "50ms").ShouldNot(Receive())
		appendText(second[60:] + acctLine(22, 0) + "\n")
		Expect(next(jobs, 2)).To(Equal([]int64{21, 22}))
	})

	It("follows a rotation: the rest of the old file, then the new one", func() {
		write(acctLine(30, 0) + "\n")
		jobs := watch(core.WatchOptions{PollInterval: 30 * time.Millisecond})
		Expect(next(jobs, 1)).To(Equal([]int64{30}))

		// qmaster writes once more, logchecker renames the file, and
		// qmaster creates a new one with its next flush.
		appendText(acctLine(31, 0) + "\n")
		Expect(os.Rename(path, path+".0")).To(Succeed())
		write(acctLine(32, 0) + "\n")
		Expect(next(jobs, 2)).To(Equal([]int64{31, 32}))

		appendText(acctLine(33, 0) + "\n")
		Expect(next(jobs, 1)).To(Equal([]int64{33}))
	})

	It("keeps watching while the file is missing during a rotation", func() {
		write(acctLine(40, 0) + "\n")
		jobs := watch(core.WatchOptions{})
		Expect(next(jobs, 1)).To(Equal([]int64{40}))
		Expect(os.Rename(path, path+".0")).To(Succeed())
		Consistently(jobs, "60ms").ShouldNot(Receive())
		write(acctLine(41, 0) + "\n")
		Expect(next(jobs, 1)).To(Equal([]int64{41}))
	})

	It("reads a file which got shorter from its start", func() {
		write(acctLine(50, 0) + "\n" + acctLine(51, 0) + "\n")
		jobs := watch(core.WatchOptions{})
		Expect(next(jobs, 2)).To(Equal([]int64{50, 51}))
		Expect(os.Truncate(path, 0)).To(Succeed())
		appendText(acctLine(52, 0) + "\n")
		Expect(next(jobs, 1)).To(Equal([]int64{52}))
	})

	It("stops and closes the channel when the context ends, also when nobody reads", func() {
		text := ""
		for id := 1; id <= 50; id++ {
			text += acctLine(id, 0) + "\n"
		}
		write(text)
		// A buffer of one and no reader: the watch blocks in its send.
		jobs, err := core.WatchFileFrom(ctx, path, core.WatchOptions{BufferSize: 1, PollInterval: 10 * time.Millisecond})
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(50 * time.Millisecond)
		cancel()
		Eventually(func() bool {
			for {
				select {
				case _, open := <-jobs:
					if !open {
						return true
					}
				default:
					return false
				}
			}
		}, "3s", "10ms").Should(BeTrue())
	})

	It("returns an error for a file which does not exist", func() {
		_, err := core.WatchFileFrom(ctx, path+".missing", core.WatchOptions{})
		Expect(err).To(HaveOccurred())
		_, err = core.WatchFile(ctx, path+".missing", 1)
		Expect(err).To(HaveOccurred())
	})

	It("emits a whole large file in batches through WatchFile", func() {
		text := ""
		for id := 1; id <= 2500; id++ {
			text += acctLine(id, 0) + "\n"
		}
		write(text)
		jobs, err := core.WatchFile(ctx, path, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(next(jobs, 2500)).To(Equal(ints(1, 2500)))
	})
})
