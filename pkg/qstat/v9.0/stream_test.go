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

		It("skips queue instances it cannot type instead of failing the stream", func() {
			queue := func(name, used, resv, total string) string {
				return "<Queue-List><name>" + name + "</name><qtype>BIP</qtype><slots_used>" + used +
					"</slots_used><slots_resv>" + resv + "</slots_resv><slots_total>" + total + "</slots_total></Queue-List>"
			}
			doc := "<job_info><queue_info>" +
				queue("", "0", "0", "1") +
				queue("bad.q@used", "x", "0", "1") +
				queue("bad.q@resv", "0", "-1", "1") +
				queue("bad.q@total", "0", "0", "99999999999") +
				queue("all.q@ok", "", "", "4") +
				"</queue_info></job_info>"
			var qis []qstat.QueueInstanceXML
			stats, err := qstat.DecodeQueueInstancesXML(strings.NewReader(doc), func(q qstat.QueueInstanceXML) error {
				qis = append(qis, q)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats).To(Equal(qstat.Stats{Decoded: 1, Skipped: 4, Complete: true}))
			Expect(qis).To(Equal([]qstat.QueueInstanceXML{
				{Name: "all.q@ok", QType: "BIP", SlotsTotal: 4},
			}), "an empty slot count is 0")
		})

		It("stops when the callback returns an error", func() {
			f := openFixture("qstat-f-xml-9.1.7.xml")
			defer f.Close()
			stop := errors.New("enough")
			stats, err := qstat.DecodeQueueInstancesXML(f, func(qstat.QueueInstanceXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(stats.Decoded).To(Equal(0))
		})
	})

	// The same job mix captured from live 9.0.12, 9.1.0 and 9.1.7 clusters:
	// a running job with a space in its name, two running array tasks, a
	// held job with a hostile name, a pending array range, a job with a
	// future start time and (not in the 9.1.7 capture) a job in Eqw.
	Context("live captures across releases", func() {
		byKey := func(jobs []qstat.JobXML) map[string]qstat.JobXML {
			m := make(map[string]qstat.JobXML, len(jobs))
			for _, j := range jobs {
				m[fmt.Sprintf("%d.%s", j.JobNumber, j.Tasks)] = j
			}
			return m
		}

		DescribeTable("qstat -xml -u '*'",
			func(file string, wantElements int) {
				jobs, stats, err := collectJobs(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats).To(Equal(qstat.Stats{Decoded: wantElements, Complete: true}))

				m := byKey(jobs)
				Expect(m["1."].Name).To(Equal("plain job"))
				Expect(m["1."].ListState).To(Equal("running"))
				Expect(m["1."].QueueName).To(HavePrefix("all.q@"))
				Expect(m["4.1"].State).To(Equal("r"))
				Expect(m["4.2"].State).To(Equal("r"))
				Expect(m["2."].Name).To(Equal("held %lt;img src=x onerror=alert(1)> & co"))
				Expect(m["2."].State).To(Equal("hqw"))
				Expect(m["3.6001-100000:3"].ListState).To(Equal("pending"))
				Expect(m["5."].Name).To(Equal("future"))

				for _, j := range jobs {
					Expect(j.Owner).To(Equal("root"))
					running := j.ListState == "running"
					Expect(j.StartTime != "").To(Equal(running), "job %d: start time only when running", j.JobNumber)
					Expect(j.SubmissionTime != "").To(Equal(!running), "job %d: submission time only when pending", j.JobNumber)
					_, err := qstat.ParseXMLTime(j.StartTime + j.SubmissionTime)
					Expect(err).NotTo(HaveOccurred())
				}
			},
			Entry("9.0.12", "qstat-xml-jobs-9.0.12.xml", 7),
			Entry("9.1.0", "qstat-xml-jobs-9.1.0.xml", 7),
			Entry("9.1.7", "qstat-xml-jobs-9.1.7.xml", 6),
		)

		DescribeTable("a job in error state is a pending element",
			func(file string) {
				jobs, _, err := collectJobs(file)
				Expect(err).NotTo(HaveOccurred())
				bad := byKey(jobs)["6."]
				Expect(bad.ListState).To(Equal("pending"))
				Expect(bad.State).To(Equal("Eqw"))
				Expect(bad.Name).To(Equal("badwd"))
			},
			Entry("9.0.12", "qstat-xml-jobs-9.0.12.xml"),
			Entry("9.1.0", "qstat-xml-jobs-9.1.0.xml"),
		)

		DescribeTable("qstat -xml -ext -u '*'",
			func(file string) {
				jobs, stats, err := collectJobs(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats).To(Equal(qstat.Stats{Decoded: 7, Complete: true}))
				for _, j := range jobs {
					Expect(j.Department).To(Equal("defaultdepartment"))
					Expect(j.Project).To(BeEmpty())
					// -ext replaces the time columns with the ticket columns.
					Expect(j.StartTime).To(BeEmpty())
					Expect(j.SubmissionTime).To(BeEmpty())
				}
				Expect(byKey(jobs)["1."].Name).To(Equal("plain job"))
			},
			Entry("9.0.12", "qstat-xml-ext-jobs-9.0.12.xml"),
			Entry("9.1.0", "qstat-xml-ext-jobs-9.1.0.xml"),
		)

		DescribeTable("qstat -f -xml",
			func(file string) {
				f := openFixture(file)
				defer f.Close()
				var qis []qstat.QueueInstanceXML
				stats, err := qstat.DecodeQueueInstancesXML(f, func(q qstat.QueueInstanceXML) error {
					qis = append(qis, q)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
				// The pending jobs follow the queues as top-level job_list
				// elements; they are neither decoded nor counted as skipped.
				Expect(stats).To(Equal(qstat.Stats{Decoded: 1, Complete: true}))
				Expect(qis).To(Equal([]qstat.QueueInstanceXML{{
					Name: "all.q@master", QType: "BIP", SlotsUsed: 3, SlotsTotal: 14, Arch: "lx-amd64",
				}}))
			},
			Entry("9.0.12", "qstat-f-xml-9.0.12.xml"),
			Entry("9.1.0", "qstat-f-xml-9.1.0.xml"),
		)

		DescribeTable("an empty cluster",
			func(file string) {
				jobs, stats, err := collectJobs(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs).To(BeEmpty())
				Expect(stats).To(Equal(qstat.Stats{Complete: true}))
			},
			Entry("9.0.12", "qstat-xml-empty-9.0.12.xml"),
			Entry("9.1.0", "qstat-xml-empty-9.1.0.xml"),
			Entry("9.1.7", "qstat-xml-empty-9.1.7.xml"),
		)
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
			names := map[uint64]string{}
			stats, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-jobs-9.1.7.xml")),
				qstat.JobStreamOptions{}, func(j qstat.JobXML) error {
					names[j.JobNumber] = j.Name
					return nil
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(6))
			Expect(stats.Complete).To(BeTrue())
			Expect(names).To(HaveKeyWithValue(uint64(2), "held %lt;img src=x onerror=alert(1)> & co"))
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

		DescribeTable("asks qstat for what the options select",
			func(opts qstat.JobStreamOptions, want []string) {
				argsFile := filepath.Join(GinkgoT().TempDir(), "args")
				_, err := impl.StreamJobsXML(context.Background(),
					env("FAKE_QSTAT_ARGS_FILE="+argsFile, "FAKE_QSTAT_FILE="+fixture("qstat-xml-empty-9.1.7.xml")),
					opts, func(qstat.JobXML) error { return nil })
				Expect(err).NotTo(HaveOccurred())
				got, err := os.ReadFile(argsFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Split(strings.TrimSuffix(string(got), "\n"), "\n")).To(Equal(want))
			},
			Entry("every user by default, not only the caller",
				qstat.JobStreamOptions{}, []string{"-xml", "-u", "*"}),
			Entry("a user filter joined with commas",
				qstat.JobStreamOptions{Users: []string{"alice", "bob"}}, []string{"-xml", "-u", "alice,bob"}),
			Entry("-ext before the filter, extra switches last",
				qstat.JobStreamOptions{Extended: true, ExtraArgs: []string{"-s", "p"}},
				[]string{"-xml", "-ext", "-u", "*", "-s", "p"}),
		)

		It("asks qstat for the full queue listing", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			_, err := impl.StreamQueueInstancesXML(context.Background(),
				env("FAKE_QSTAT_ARGS_FILE="+argsFile, "FAKE_QSTAT_FILE="+fixture("qstat-f-xml-9.1.7.xml")),
				func(qstat.QueueInstanceXML) error { return nil })
			Expect(err).NotTo(HaveOccurred())
			got, err := os.ReadFile(argsFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("-f\n-xml\n"))
		})

		It("refuses control characters in extra switches before spawning anything", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			_, err := impl.StreamJobsXML(context.Background(), env("FAKE_QSTAT_ARGS_FILE="+argsFile),
				qstat.JobStreamOptions{ExtraArgs: []string{"-s", "p\n"}}, func(qstat.JobXML) error { return nil })
			Expect(err).To(HaveOccurred())
			Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
		})

		It("does not wait for a grandchild that keeps the output open when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			_, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-jobs-9.1.7.xml"), "FAKE_QSTAT_LINGER=8"),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second), "the lingering child was waited for")
		})

		It("returns a complete listing even when the deadline fires while qstat exits", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			seen := 0
			stats, err := impl.StreamJobsXML(ctx,
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-jobs-9.1.7.xml")),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error {
					seen++
					if seen == 6 {
						// qstat has printed everything and exited by now; the
						// root close tag is still unread in the pipe.
						time.Sleep(300 * time.Millisecond)
						cancel()
					}
					return nil
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			Expect(stats.Decoded).To(Equal(6))
		})

		It("reports a qstat that cannot be started, with nothing decoded", func() {
			// The constructor checks the executable, so remove it afterwards:
			// an installation that went away under a long-running poller.
			gone := filepath.Join(GinkgoT().TempDir(), "qstat")
			Expect(os.WriteFile(gone, []byte("#!/bin/sh\n"), 0o700)).To(Succeed())
			missing, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{Executable: gone})
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Remove(gone)).To(Succeed())
			stats, err := missing.StreamJobsXML(context.Background(), nil,
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return nil })
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to start qstat: "))
			Expect(stats).To(Equal(qstat.Stats{}))
		})

		It("kills qstat and returns the callback error when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			seen := 0
			stats, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-jobs-9.1.7.xml"), "FAKE_QSTAT_HANG=30"),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error {
					seen++
					return stop
				})
			Expect(err).To(BeIdenticalTo(stop), "the callback error is returned unchanged, not the kill")
			Expect(seen).To(Equal(1))
			Expect(stats.Decoded).To(Equal(0))
			Expect(time.Since(started)).To(BeNumerically("<", 5*time.Second), "qstat was killed, not waited for")
		})

		It("reports malformed output of a successful qstat as malformed, not as a process failure", func() {
			doc := filepath.Join(GinkgoT().TempDir(), "broken.xml")
			Expect(os.WriteFile(doc, []byte("<job_info><queue_info></wrong>"), 0o600)).To(Succeed())
			_, err := impl.StreamJobsXML(context.Background(), env("FAKE_QSTAT_FILE="+doc),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return nil })
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("malformed qstat XML after 0 elements"))
		})

		It("prefers the process failure over the incomplete document it leaves behind", func() {
			stats, err := impl.StreamJobsXML(context.Background(),
				env("FAKE_QSTAT_EXIT=2", "FAKE_QSTAT_STDERR=error: qmaster went away",
					"FAKE_QSTAT_FILE="+fixture("qstat-xml-cut-between-elements.xml")),
				qstat.JobStreamOptions{}, func(qstat.JobXML) error { return nil })
			Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeFalse(), "got: %v", err)
			Expect(err.Error()).To(HavePrefix("failed to run command (error: qmaster went away)"))
			Expect(stats.Decoded).To(Equal(2), "what was decoded before the failure is still reported")
			Expect(stats.Complete).To(BeFalse())
		})

		It("echoes the command in dry-run mode and runs nothing", func() {
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

		It("is cancelled by its context in RunContext and says so", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			started := time.Now()
			_, _, err := impl.RunContext(ctx, env("FAKE_QSTAT_SLEEP=5"), "-j", "1")
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(HavePrefix("failed to run command ("))
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
		})

		It("keeps only the first 64 KiB of stderr and does not block a chatty qstat", func() {
			_, stderr, err := impl.RunContext(context.Background(),
				env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=error: first line", "FAKE_QSTAT_STDERR_BYTES=300000"),
				"-j", "1")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(HaveLen(64 * 1024))
			Expect(stderr).To(HavePrefix("error: first line\nxxx"))
		})

		Context("SchedulerJobInfoContext on 9.0.12 captures", func() {
			It("parses a held job with a hostile name", func() {
				argsFile := filepath.Join(GinkgoT().TempDir(), "args")
				jobs, err := impl.SchedulerJobInfoContext(context.Background(),
					env("FAKE_QSTAT_ARGS_FILE="+argsFile, "FAKE_QSTAT_FILE="+fixture("qstat-j-held-9.0.12.txt")), "2")
				Expect(err).NotTo(HaveOccurred())
				got, err := os.ReadFile(argsFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(got)).To(Equal("-j\n2\n"))
				Expect(jobs).To(HaveLen(1))
				Expect(jobs[0].JobNumber).To(Equal(2))
				Expect(jobs[0].JobName).To(Equal("held <img src=x onerror=alert(1)> & co"))
				Expect(jobs[0].Owner).To(Equal("root"))
				Expect(jobs[0].SubmitCmdLine).To(ContainSubstring("-h"))
				Expect(jobs[0].SchedulingInfo).To(ContainSubstring("Collecting of scheduler job information is turned off"))
			})

			It("parses the header of a running array job", func() {
				jobs, err := impl.SchedulerJobInfoContext(context.Background(),
					env("FAKE_QSTAT_FILE="+fixture("qstat-j-array-9.0.12.txt")), "4")
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs).To(HaveLen(1))
				Expect(jobs[0].JobNumber).To(Equal(4))
				Expect(jobs[0].JobName).To(Equal("smallarray"))
				// 9.0.12 already prints the per-task block ("usage 1:",
				// "job_state 1:"); the 9.0 parser has no field for it.
				Expect(jobs[0].Usage).To(BeEmpty())
			})

			It("parses a job in error state", func() {
				jobs, err := impl.SchedulerJobInfoContext(context.Background(),
					env("FAKE_QSTAT_FILE="+fixture("qstat-j-error-9.0.12.txt")), "6")
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs).To(HaveLen(1))
				Expect(jobs[0].JobName).To(Equal("badwd"))
				Expect(jobs[0].Cwd).To(Equal("/nonexistent/dir"))
			})

			It("maps the captured unknown-job answer to ErrNoSuchJob", func() {
				stderr, err := os.ReadFile(fixture("qstat-j-unknown-9.0.12.stderr"))
				Expect(err).NotTo(HaveOccurred())
				_, err = impl.SchedulerJobInfoContext(context.Background(),
					env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR="+string(stderr)), "99999999")
				Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
			})

			It("refuses an empty id list without running qstat", func() {
				_, err := impl.SchedulerJobInfoContext(context.Background(), nil)
				Expect(err).To(MatchError("no job id given"))
			})
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
