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

package qstat_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func openFixture(name string) *os.File {
	f, err := os.Open(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())
	return f
}

func collectJobs(name string) ([]qstat.JobXML, qstat.Stats, error) {
	f := openFixture(name)
	defer f.Close()
	var jobs []qstat.JobXML
	stats, err := qstat.DecodeJobsXML(f, func(j qstat.JobXML) error {
		jobs = append(jobs, j)
		return nil
	})
	return jobs, stats, err
}

var _ = Describe("Streaming qstat XML", func() {

	Context("DecodeJobsXML", func() {
		It("decodes one element per running task and per pending range, with full names", func() {
			jobs, stats, err := collectJobs("qstat-xml-jobs-9.1.7.xml")
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			Expect(stats.Decoded).To(Equal(6))
			Expect(stats.Skipped).To(Equal(0))

			// Running first, in document order.
			Expect(jobs[0].ListState).To(Equal("running"))
			Expect(jobs[0].JobNumber).To(Equal(uint64(1)))
			Expect(jobs[0].Name).To(Equal("plain job")) // a name with a space survives
			Expect(jobs[0].Owner).To(Equal("root"))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].QueueName).To(Equal("all.q@sim128"))
			Expect(jobs[0].Slots).To(Equal(uint32(1)))
			Expect(jobs[0].Tasks).To(BeEmpty())
			Expect(jobs[0].StartTime).To(Equal("2026-10-22T13:16:41.647365"))
			Expect(jobs[0].SubmissionTime).To(BeEmpty())
			Expect(jobs[0].Priority).To(BeNumerically("~", 0.555, 0.0001))

			// A running array task carries its single task id.
			Expect(jobs[1].JobNumber).To(Equal(uint64(4)))
			Expect(jobs[1].Tasks).To(Equal("1"))
			Expect(jobs[2].Tasks).To(Equal("2"))

			// Pending: the hostile name is data. Note the scheduler's own
			// escaping of '<' as "%lt;" (a bug upstream of this library);
			// '>' and '&' are proper entities and decode back.
			Expect(jobs[3].ListState).To(Equal("pending"))
			Expect(jobs[3].JobNumber).To(Equal(uint64(2)))
			Expect(jobs[3].Name).To(Equal("held %lt;img src=x onerror=alert(1)> & co"))
			Expect(jobs[3].State).To(Equal("hqw"))
			Expect(jobs[3].QueueName).To(BeEmpty())
			Expect(jobs[3].StartTime).To(BeEmpty())
			Expect(jobs[3].SubmissionTime).To(Equal("2026-10-22T13:16:41.669137"))

			// A pending array job is ONE element with its remaining range.
			Expect(jobs[4].JobNumber).To(Equal(uint64(3)))
			Expect(jobs[4].Tasks).To(Equal("6001-100000:3"))
			Expect(jobs[5].Name).To(Equal("future"))
		})

		It("reports an empty cluster as a complete document with no elements", func() {
			jobs, stats, err := collectJobs("qstat-xml-empty-9.1.7.xml")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(BeEmpty())
			Expect(stats).To(Equal(qstat.Stats{Complete: true}))
		})

		It("reports a document cut between elements as incomplete, keeping what was decoded", func() {
			jobs, stats, err := collectJobs("qstat-xml-cut-between-elements.xml")
			Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(HavePrefix("after 2 elements"))
			Expect(jobs).To(HaveLen(2))
			Expect(stats.Decoded).To(Equal(2))
			Expect(stats.Complete).To(BeFalse(), "the root close tag was never seen")
		})

		It("reports a document cut inside an element as incomplete too", func() {
			jobs, stats, err := collectJobs("qstat-xml-cut-inside-element.xml")
			Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
			Expect(jobs).To(HaveLen(2))
			Expect(stats.Complete).To(BeFalse())
		})

		It("reports a document with broken markup as malformed", func() {
			_, err := qstat.DecodeJobsXML(strings.NewReader("<job_info><queue_info><job_list state=\"running\"><JB_job_number>1</JB_job_number></job_list></queue_info></wrong>"), func(qstat.JobXML) error { return nil })
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("malformed qstat XML after 1 elements"))
		})

		It("skips an element it cannot type instead of failing the stream", func() {
			jobs, stats, err := collectJobs("qstat-xml-skipped-element.xml")
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(1))
			Expect(stats.Skipped).To(Equal(1))
			Expect(stats.Complete).To(BeTrue())
			Expect(jobs[0].JobNumber).To(Equal(uint64(7)))
			Expect(jobs[0].Slots).To(Equal(uint32(4)))
		})

		It("stops when the callback returns an error", func() {
			f := openFixture("qstat-xml-jobs-9.1.7.xml")
			defer f.Close()
			seen := 0
			stop := errors.New("enough")
			stats, err := qstat.DecodeJobsXML(f, func(qstat.JobXML) error {
				seen++
				if seen == 2 {
					return stop
				}
				return nil
			})
			Expect(errors.Is(err, stop)).To(BeTrue())
			Expect(seen).To(Equal(2))
			Expect(stats.Decoded).To(Equal(1), "the element whose callback failed is not counted")
			Expect(stats.Complete).To(BeFalse())
		})
	})

	Context("DecodeQueueInstancesXML", func() {
		It("decodes one element per queue instance and skips the nested jobs", func() {
			f := openFixture("qstat-f-xml-9.1.7.xml")
			defer f.Close()
			var qis []qstat.QueueInstanceXML
			stats, err := qstat.DecodeQueueInstancesXML(f, func(q qstat.QueueInstanceXML) error {
				qis = append(qis, q)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			Expect(stats.Decoded).To(Equal(3))
			Expect(qis[0]).To(Equal(qstat.QueueInstanceXML{
				Name: "all.q@master", QType: "BIP", SlotsUsed: 10, SlotsReserved: 0, SlotsTotal: 14,
				Arch: "lx-amd64",
			}))
			Expect(qis[1].Name).To(Equal("night.q@sim100"))
			Expect(qis[1].State).To(Equal("D"))
			Expect(qis[1].LoadAvg).To(BeEmpty(), "simulated hosts report no load")
			Expect(qis[2]).To(Equal(qstat.QueueInstanceXML{
				Name: "gpu.q@node41", QType: "BIP", SlotsUsed: 8, SlotsReserved: 2, SlotsTotal: 8,
				LoadAvg: "12.43000", Arch: "lx-amd64", State: "au",
			}))
		})
	})

	Context("ParseXMLTime", func() {
		It("reads both the microsecond and the whole-second form in the local zone", func() {
			t, err := qstat.ParseXMLTime("2026-10-22T13:16:41.647365")
			Expect(err).NotTo(HaveOccurred())
			Expect(t).To(Equal(time.Date(2026, 10, 22, 13, 16, 41, 647365000, time.Local)))

			t, err = qstat.ParseXMLTime("2024-10-28T07:21:41")
			Expect(err).NotTo(HaveOccurred())
			Expect(t).To(Equal(time.Date(2024, 10, 28, 7, 21, 41, 0, time.Local)))

			t, err = qstat.ParseXMLTime("")
			Expect(err).NotTo(HaveOccurred())
			Expect(t.IsZero()).To(BeTrue())

			_, err = qstat.ParseXMLTime("yesterday")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("against a qstat process", func() {
		var impl *qstat.QStatImpl
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string {
			return append([]string{"PATH=" + os.Getenv("PATH")}, kv...)
		}

		BeforeEach(func() {
			var err error
			impl, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{
				Executable: fixture("fake-qstat.sh"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("streams the jobs document with the caller's environment", func() {
			var jobs []qstat.JobXML
			stats, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-jobs-9.1.7.xml")),
				qstat.JobStreamOptions{}, func(j qstat.JobXML) error {
					jobs = append(jobs, j)
					return nil
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(6))
			Expect(stats.Complete).To(BeTrue())
			Expect(jobs[3].Name).To(ContainSubstring("onerror"))
		})

		It("streams the queue document", func() {
			n := 0
			stats, err := impl.StreamQueueInstancesXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-f-xml-9.1.7.xml")),
				func(qstat.QueueInstanceXML) error { n++; return nil })
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(3))
			Expect(stats.Complete).To(BeTrue())
		})

		It("reports a non-zero exit with the scheduler's stderr in the command-error shape", func() {
			stats, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=error: unable to send message to qmaster using port 6444 on host \"master\": got send error"),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return nil })
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to run command (error: unable to send message to qmaster"))
			Expect(stats.Decoded).To(Equal(0))
			Expect(stats.Complete).To(BeFalse())
		})

		It("reports a successful exit without a root close tag as incomplete", func() {
			_, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-cut-between-elements.xml")),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return nil })
			Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue())
		})

		It("is cancelled by its context while qstat is still silent", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			started := time.Now()
			stats, err := impl.StreamJobsXML(ctx,
				env("FAKE_QSTAT_SLEEP=5", "FAKE_QSTAT_FILE="+fixture("qstat-xml-jobs-9.1.7.xml")),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return nil })
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
			Expect(stats.Decoded).To(Equal(0))
			Expect(stats.Complete).To(BeFalse())
		})

		It("passes the user filter and extra switches through", func() {
			// The fake prints nothing, so only the argv matters: a dry-run
			// adapter echoes it.
			dry, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{Executable: "qstat", DryRun: true})
			Expect(err).NotTo(HaveOccurred())
			out, _, err := dry.RunContext(context.Background(), nil, "-j", "42")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qstat [-j 42]"))
			stats, err := dry.StreamJobsXML(context.Background(), nil, qstat.JobStreamOptions{}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
		})

		It("refuses control characters in arguments before spawning anything", func() {
			_, _, err := impl.RunContext(context.Background(), nil, "-j", "42\n43")
			Expect(err).To(HaveOccurred())
		})

		It("maps the scheduler's unknown-job answer to ErrNoSuchJob", func() {
			_, err := qstat.RunSchedulerJobInfo(context.Background(),
				env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=Following jobs do not exist: \n99999999"),
				impl, "99999999")
			Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
		})

		It("keeps stdout and stderr of a failed run available to the caller", func() {
			stdout, stderr, err := impl.RunContext(context.Background(),
				env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=Following jobs do not exist: ", "FAKE_QSTAT_FILE="+fixture("qstat-xml-empty-9.1.7.xml")),
				"-j", "1")
			Expect(err).To(HaveOccurred())
			Expect(stdout).To(ContainSubstring("<job_info"))
			Expect(stderr).To(ContainSubstring("Following jobs do not exist"))
		})
	})
})

// generatedJobs writes a synthetic "qstat -xml" document with n running
// elements and n/10 pending ranges into w, in the live 9.1 shape.
func generatedJobs(w io.Writer, n int) {
	fmt.Fprint(w, "<?xml version='1.0'?>\n<job_info>\n  <queue_info>\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(w, "    <job_list state=\"running\">\n      <JB_job_number>%d</JB_job_number>\n      <JAT_prio>0.55500</JAT_prio>\n      <JB_name>simulation step %d</JB_name>\n      <JB_owner>user%03d</JB_owner>\n      <state>r</state>\n      <JAT_start_time>2026-10-22T13:16:41.647365</JAT_start_time>\n      <queue_name>all.q@node%04d</queue_name>\n      <slots>1</slots>\n      <tasks>%d</tasks>\n    </job_list>\n",
			i/8+1, i, i%200, i%2000, i%8+1)
	}
	fmt.Fprint(w, "  </queue_info>\n  <job_info>\n")
	for i := 1; i <= n/10; i++ {
		fmt.Fprintf(w, "    <job_list state=\"pending\">\n      <JB_job_number>%d</JB_job_number>\n      <JAT_prio>0.00000</JAT_prio>\n      <JB_name>pending %d</JB_name>\n      <JB_owner>user%03d</JB_owner>\n      <state>qw</state>\n      <JAT_submission_time>2026-10-22T13:16:41.669137</JAT_submission_time>\n      <queue_name></queue_name>\n      <slots>1</slots>\n      <tasks>1-1000:1</tasks>\n    </job_list>\n",
			n+i, i, i%200)
	}
	fmt.Fprint(w, "  </job_info>\n</job_info>\n")
}

var _ = Describe("Streaming a generated 200k-element document", Label("slow"), func() {
	It("decodes it in bounded memory without buffering the document", func() {
		if os.Getenv("QSTAT_STREAM_SCALE") == "" {
			Skip("set QSTAT_STREAM_SCALE=1 to run the 200k-element decode (about a second)")
		}
		pr, pw := io.Pipe()
		go func() {
			generatedJobs(pw, 200000)
			pw.Close()
		}()
		var owners = map[string]int{}
		start := time.Now()
		stats, err := qstat.DecodeJobsXML(pr, func(j qstat.JobXML) error {
			owners[j.Owner]++
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		Expect(stats.Decoded).To(Equal(220000))
		Expect(owners).To(HaveLen(200))
		GinkgoWriter.Printf("decoded %d elements in %s (%s)\n", stats.Decoded, time.Since(start),
			strings.TrimSpace(fmt.Sprintf("%.0f elements/s", float64(stats.Decoded)/time.Since(start).Seconds())))
	})
})
