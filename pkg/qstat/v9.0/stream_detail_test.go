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
	"io"
	"os"
	"path/filepath"
	"strings"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func collectDetails(doc string) ([]qstat.QueueInstanceDetailXML, qstat.Stats, error) {
	var out []qstat.QueueInstanceDetailXML
	stats, err := qstat.DecodeQueueInstanceDetailsXML(strings.NewReader(doc), func(d qstat.QueueInstanceDetailXML) error {
		out = append(out, d)
		return nil
	})
	return out, stats, err
}

// detailByName returns the instance named name, failing the spec when it is
// missing.
func detailByName(details []qstat.QueueInstanceDetailXML, name string) qstat.QueueInstanceDetailXML {
	for _, d := range details {
		if d.Name == name {
			return d
		}
	}
	Fail("no queue instance " + name)
	return qstat.QueueInstanceDetailXML{}
}

var _ = Describe("Queue instance detail XML", func() {

	Context("on a 9.1.7 capture (one silent execd, one load alarm)", func() {
		// Captured 2026-10-29: every instance of ocs-worker2 after its execd
		// had been silent for max_unheard, plus cs2964.q@ocs-worker1 with an
		// impossible load threshold.
		var details []qstat.QueueInstanceDetailXML

		BeforeEach(func() {
			f := openFixture("qstat-f-xml-explain-9.1.7.xml")
			defer f.Close()
			b, err := io.ReadAll(f)
			Expect(err).NotTo(HaveOccurred())
			var stats qstat.Stats
			details, stats, err = collectDetails(string(b))
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
		})

		It("keeps the instance fields of the plain queue listing", func() {
			Expect(details[0].Name).To(Equal("all.q@ocs-worker2"))
			Expect(details[0].State).To(Equal("u"))
			Expect(details[0].SlotsTotal).To(Equal(uint32(14)))
			Expect(details[0].LoadAlarmReasons).To(BeEmpty())
		})

		It("reads the load alarm reason the scheduler gives for an unknown host", func() {
			d := detailByName(details, "bigmem.q@ocs-worker2")
			Expect(d.State).To(Equal("au"))
			Expect(d.LoadAlarmReasons).To(ConsistOf(
				`error: no value for "np_load_avg" because execd is in unknown state`))
		})

		It("reads a real load alarm reason", func() {
			d := detailByName(details, "cs2964.q@ocs-worker1")
			Expect(d.State).To(Equal("a"))
			Expect(d.LoadAlarmReasons).To(HaveLen(1))
			Expect(d.LoadAlarmReasons[0]).To(HavePrefix("alarm hl:np_load_avg="))
		})

		It("has no load value: the XML writer never prints a real one", func() {
			for _, d := range details {
				Expect(d.LoadAvg).To(BeEmpty(), d.Name)
			}
		})
	})

	Context("reasons and nested jobs", func() {
		const doc = `<?xml version='1.0'?>
<job_info>
  <queue_info>
    <Queue-List>
      <name>all.q@node01</name>
      <qtype>BIP</qtype>
      <slots_used>3</slots_used>
      <slots_resv>0</slots_resv>
      <slots_total>4</slots_total>
      <arch>lx-amd64</arch>
      <state>aAE</state>
      <load-alarm-reason>alarm hl:np_load_avg=1.2 load-threshold=1.0</load-alarm-reason>
      <suspend-alarm-reason>suspend alarm hl:np_load_avg=1.2 suspend-threshold=1.1</suspend-alarm-reason>
      <message>queue all.q marked QERROR as result of job 7's failure at host node01</message>
      <job_list state="running">
        <JB_job_number>7</JB_job_number>
        <JAT_prio>0.55500</JAT_prio>
        <JB_name>mpi job</JB_name>
        <JB_owner>alice</JB_owner>
        <state>r</state>
        <JAT_start_time>2026-10-29T10:00:00.000000</JAT_start_time>
        <slots>2</slots>
      </job_list>
      <job_list state="running">
        <JB_job_number>8</JB_job_number>
        <JAT_prio>0.50000</JAT_prio>
        <JB_name>array</JB_name>
        <JB_owner>bob</JB_owner>
        <state>r</state>
        <JAT_start_time>2026-10-29T10:01:00.000000</JAT_start_time>
        <slots>1</slots>
        <tasks>3</tasks>
      </job_list>
    </Queue-List>
  </queue_info>
  <job_info>
  </job_info>
</job_info>`

		It("collects every reason kind and the running jobs, naming their queue", func() {
			details, stats, err := collectDetails(doc)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(1))
			d := details[0]
			Expect(d.LoadAlarmReasons).To(HaveLen(1))
			Expect(d.SuspendAlarmReasons).To(HaveLen(1))
			Expect(d.Messages).To(ConsistOf(ContainSubstring("marked QERROR")))
			Expect(d.Jobs).To(HaveLen(2))
			Expect(d.Jobs[0].JobNumber).To(Equal(uint64(7)))
			Expect(d.Jobs[0].Name).To(Equal("mpi job"))
			Expect(d.Jobs[0].QueueName).To(Equal("all.q@node01"))
			Expect(d.Jobs[1].Tasks).To(Equal("3"))
		})
	})

	It("skips an instance it cannot type and keeps going", func() {
		details, stats, err := collectDetails(`<job_info><queue_info>
		  <Queue-List><name>bad.q@n1</name><slots_used>x</slots_used></Queue-List>
		  <Queue-List><name>all.q@n1</name><slots_total>2</slots_total>
		    <job_list state="running"><JB_job_number>not a number</JB_job_number></job_list>
		  </Queue-List>
		</queue_info></job_info>`)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats).To(Equal(qstat.Stats{Decoded: 1, Skipped: 1, Complete: true}))
		Expect(details[0].Name).To(Equal("all.q@n1"))
		Expect(details[0].Jobs).To(BeEmpty(), "a job it cannot type is dropped")
	})

	Context("StreamQueueInstanceDetailsXML", func() {
		var impl *qstat.QStatImpl
		var argsFile string
		noop := func(qstat.QueueInstanceDetailXML) error { return nil }
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string {
			return append([]string{"PATH=" + os.Getenv("PATH"), "FAKE_QSTAT_ARGS_FILE=" + argsFile}, kv...)
		}
		recordedArgs := func() []string {
			got, err := os.ReadFile(argsFile)
			Expect(err).NotTo(HaveOccurred())
			return strings.Split(strings.TrimSuffix(string(got), "\n"), "\n")
		}

		BeforeEach(func() {
			var err error
			impl, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{Executable: fixture("fake-qstat.sh")})
			Expect(err).NotTo(HaveOccurred())
			argsFile = filepath.Join(GinkgoT().TempDir(), "args")
		})

		It("asks for the explained, running-only listing of the given instances for every user", func() {
			n := 0
			stats, err := impl.StreamQueueInstanceDetailsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-f-xml-explain-9.1.7.xml")),
				qstat.QueueDetailOptions{Queues: []string{"*@ocs-worker2", "cs2964.q@ocs-worker1"}},
				func(qstat.QueueInstanceDetailXML) error { n++; return nil })
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			Expect(n).To(Equal(stats.Decoded))
			Expect(recordedArgs()).To(Equal([]string{"-f", "-xml", "-explain", "aAcE", "-s", "r",
				"-u", "*", "-q", "*@ocs-worker2,cs2964.q@ocs-worker1"}))
		})

		It("restricts the nested jobs to the given users", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-empty-9.1.7.xml")),
				qstat.QueueDetailOptions{Queues: []string{"all.q"}, Users: []string{"alice", "bob"}}, noop)
			Expect(err).NotTo(HaveOccurred())
			Expect(recordedArgs()).To(ContainElements("-u", "alice,bob"))
		})

		It("names the -q list when qstat fails silently, as it does for an unknown queue", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env("FAKE_QSTAT_EXIT=1"),
				qstat.QueueDetailOptions{Queues: []string{"nosuch.q"}}, noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`"nosuch.q"`))
			Expect(err.Error()).To(ContainSubstring("failed to run command ()"))
		})

		It("refuses to run without a queue", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env(), qstat.QueueDetailOptions{}, noop)
			Expect(errors.Is(err, qstat.ErrNoQueues)).To(BeTrue(), "got: %v", err)
			Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
		})

		DescribeTable("refuses an entry that could forge an option or another entry",
			func(opts qstat.QueueDetailOptions) {
				_, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env(), opts, noop)
				Expect(err).To(HaveOccurred())
				Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
			},
			Entry("a queue that is an option", qstat.QueueDetailOptions{Queues: []string{"-u"}}),
			Entry("two queues in one entry", qstat.QueueDetailOptions{Queues: []string{"a.q,b.q"}}),
			Entry("a user that is an option", qstat.QueueDetailOptions{Queues: []string{"all.q"}, Users: []string{"-x"}}),
			Entry("a user that widens the list", qstat.QueueDetailOptions{Queues: []string{"all.q"}, Users: []string{"alice,*"}}),
		)
	})
})
