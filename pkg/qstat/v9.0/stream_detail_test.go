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

	Context("on a whole-cluster 9.1.7 capture (-u nobodyx, so no nested jobs)", func() {
		It("reads every instance and the one load alarm", func() {
			f := openFixture("qstat-f-xml-explain-all-9.1.7.xml")
			defer f.Close()
			var details []qstat.QueueInstanceDetailXML
			stats, err := qstat.DecodeQueueInstanceDetailsXML(f, func(d qstat.QueueInstanceDetailXML) error {
				details = append(details, d)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats).To(Equal(qstat.Stats{Decoded: 22, Complete: true}))
			d := detailByName(details, "alarm.q@ocs-worker2")
			Expect(d.State).To(Equal("a"))
			Expect(d.LoadAlarmReasons).To(Equal([]string{"alarm hl:np_load_avg=0.058571 load-threshold=0.01"}))
			for _, d := range details {
				Expect(d.Jobs).To(BeEmpty(), d.Name)
			}
		})
	})

	It("carries the -r requests of nested jobs", func() {
		details, _, err := collectDetails(`<job_info><queue_info>
		  <Queue-List><name>gpu.q@n1</name><slots_total>2</slots_total>
		    <job_list state="running">
		      <JB_job_number>7</JB_job_number><JB_name>train</JB_name><JB_owner>alice</JB_owner><state>r</state>
		      <slots>2</slots>
		      <hard_request name="gpu" resource_contribution="1000.000000">1</hard_request>
		      <hard_request name="h_rt" resource_contribution="0.000000">3600</hard_request>
		      <requested_pe name="smp">2</requested_pe>
		      <hard_req_queue>gpu.q</hard_req_queue>
		    </job_list>
		  </Queue-List>
		</queue_info></job_info>`)
		Expect(err).NotTo(HaveOccurred())
		Expect(details[0].Jobs).To(HaveLen(1))
		job := details[0].Jobs[0]
		Expect(job.HardRequests).To(Equal([]qstat.ResourceRequestXML{{Name: "gpu", Value: "1"}, {Name: "h_rt", Value: "3600"}}))
		Expect(job.RequestedPE).To(Equal("smp"))
		Expect(job.HardRequestedQueues).To(Equal([]string{"gpu.q"}))
	})

	It("carries the master and slave scoped -r requests of nested jobs apart from the global ones", func() {
		// Element names and shapes from ocs_qstat_xml.cc
		// (qstat_xml_job_report_hard_resource,
		// qstat_xml_job_hard_requested_queue); not a live capture.
		details, _, err := collectDetails(`<job_info><queue_info>
		  <Queue-List><name>gpu.q@n1</name><slots_total>4</slots_total>
		    <job_list state="running">
		      <JB_job_number>8</JB_job_number><JB_name>mpi</JB_name><JB_owner>alice</JB_owner><state>r</state>
		      <slots>4</slots>
		      <hard_request name="h_rt" resource_contribution="0.000000">3600</hard_request>
		      <master_hard_request name="mem" resource_contribution="0.000000">4G</master_hard_request>
		      <slave_hard_request name="gpu" resource_contribution="1000.000000">1</slave_hard_request>
		      <slave_hard_request name="" resource_contribution="0.000000">x</slave_hard_request>
		      <requested_pe name="mpi">4</requested_pe>
		      <hard_req_queue>all.q</hard_req_queue>
		      <master_hard_req_queue> io.q </master_hard_req_queue>
		      <slave_hard_req_queue>gpu.q</slave_hard_req_queue>
		    </job_list>
		  </Queue-List>
		</queue_info></job_info>`)
		Expect(err).NotTo(HaveOccurred())
		job := details[0].Jobs[0]
		Expect(job.HardRequests).To(Equal([]qstat.ResourceRequestXML{{Name: "h_rt", Value: "3600"}}))
		Expect(job.MasterHardRequests).To(Equal([]qstat.ResourceRequestXML{{Name: "mem", Value: "4G"}}))
		Expect(job.SlaveHardRequests).To(Equal([]qstat.ResourceRequestXML{{Name: "gpu", Value: "1"}}), "a nameless request is dropped")
		Expect(job.HardRequestedQueues).To(Equal([]string{"all.q"}))
		Expect(job.MasterHardRequestedQueues).To(Equal([]string{"io.q"}))
		Expect(job.SlaveHardRequestedQueues).To(Equal([]string{"gpu.q"}))
		Expect(job.RequestedPE).To(Equal("mpi"))

		plain, _, err := collectDetails(`<job_info><queue_info>
		  <Queue-List><name>all.q@n1</name><slots_total>1</slots_total>
		    <job_list state="running"><JB_job_number>9</JB_job_number><slots>1</slots></job_list>
		  </Queue-List></queue_info></job_info>`)
		Expect(err).NotTo(HaveOccurred())
		Expect(plain[0].Jobs[0].MasterHardRequests).To(BeNil())
		Expect(plain[0].Jobs[0].SlaveHardRequests).To(BeNil())
		Expect(plain[0].Jobs[0].MasterHardRequestedQueues).To(BeNil())
		Expect(plain[0].Jobs[0].SlaveHardRequestedQueues).To(BeNil())
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

		It("refuses to run without a queue unless AllQueues is set", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env(), qstat.QueueDetailOptions{}, noop)
			Expect(errors.Is(err, qstat.ErrNoQueues)).To(BeTrue(), "got: %v", err)
			Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
		})

		It("refuses AllQueues together with a queue list", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env(),
				qstat.QueueDetailOptions{AllQueues: true, Queues: []string{"all.q"}}, noop)
			Expect(err).To(MatchError(ContainSubstring("mutually exclusive")))
			Expect(errors.Is(err, qstat.ErrNoQueues)).To(BeFalse())
			Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
		})

		It("asks for the whole cluster, without -q, when AllQueues is set", func() {
			n := 0
			stats, err := impl.StreamQueueInstanceDetailsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-f-xml-explain-all-9.1.7.xml")),
				qstat.QueueDetailOptions{AllQueues: true}, func(qstat.QueueInstanceDetailXML) error { n++; return nil })
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			Expect(n).To(Equal(22))
			Expect(recordedArgs()).To(Equal([]string{"-f", "-xml", "-explain", "aAcE", "-s", "r", "-u", "*"}))
		})

		It("adds -r before the user filter when the nested jobs should carry their requests", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-xml-empty-9.1.7.xml")),
				qstat.QueueDetailOptions{Queues: []string{"all.q"}, Requests: true}, noop)
			Expect(err).NotTo(HaveOccurred())
			Expect(recordedArgs()).To(Equal([]string{"-f", "-xml", "-explain", "aAcE", "-s", "r", "-r", "-u", "*", "-q", "all.q"}))
		})

		It("does not blame a -q list when there is none and qstat fails", func() {
			_, err := impl.StreamQueueInstanceDetailsXML(context.Background(),
				env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=error: unable to contact qmaster"),
				qstat.QueueDetailOptions{AllQueues: true}, noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to run command (error: unable to contact qmaster)"))
		})

		DescribeTable("refuses an empty queue entry whatever GCS_VALIDATION says, so it never becomes the whole cluster",
			func(mode string, queues []string) {
				GinkgoT().Setenv("GCS_VALIDATION", mode)
				_, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env(),
					qstat.QueueDetailOptions{Queues: queues}, noop)
				Expect(errors.Is(err, qstat.ErrNoQueues)).To(BeTrue(), "got: %v", err)
				Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
			},
			Entry("default", "", []string{""}),
			Entry("strict", "strict", []string{""}),
			Entry("warn", "warn", []string{""}),
			Entry("off", "off", []string{""}),
			Entry("off, blank next to a real queue", "off", []string{"all.q", " "}),
		)

		It("splits a -q list too long for one argument into runs, in order, with Stats added up", func() {
			queues := make([]string, 3000)
			for i := range queues {
				queues[i] = fmt.Sprintf("q%059d.q", i)
			}
			argsLog := filepath.Join(GinkgoT().TempDir(), "argslog")
			var names []string
			stats, err := impl.StreamQueueInstanceDetailsXML(context.Background(),
				env("FAKE_QSTAT_ARGS_LOG="+argsLog, "FAKE_QSTAT_FILE="+fixture("qstat-f-xml-explain-9.1.7.xml")),
				qstat.QueueDetailOptions{Queues: queues},
				func(d qstat.QueueInstanceDetailXML) error { names = append(names, d.Name); return nil })
			Expect(err).NotTo(HaveOccurred())
			logged, err := os.ReadFile(argsLog)
			Expect(err).NotTo(HaveOccurred())
			lines := strings.Split(strings.TrimSpace(string(logged)), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 2), "3,000 queues of 64 characters do not fit one argument")
			var asked []string
			for _, l := range lines {
				f := strings.Fields(l)
				Expect(f[:len(f)-2]).To(Equal([]string{"-f", "-xml", "-explain", "aAcE", "-s", "r", "-u", "*"}))
				Expect(f[len(f)-2]).To(Equal("-q"))
				Expect(len(f[len(f)-1])).To(BeNumerically("<=", qstat.MaxListArgument))
				asked = append(asked, strings.Split(f[len(f)-1], ",")...)
			}
			Expect(asked).To(Equal(queues), "every queue asked for exactly once, in order")

			perRun := stats.Decoded / len(lines)
			Expect(perRun).To(BeNumerically(">", 0))
			Expect(stats.Decoded).To(Equal(perRun*len(lines)), "each run's instances are counted")
			Expect(stats.Complete).To(BeTrue())
			Expect(names).To(HaveLen(stats.Decoded))
			for run := 1; run < len(lines); run++ {
				Expect(names[run*perRun:(run+1)*perRun]).To(Equal(names[:perRun]), "run %d in document order", run)
			}
		})

		It("stops at the first failing run and names its -q list", func() {
			queues := make([]string, 3000)
			for i := range queues {
				queues[i] = fmt.Sprintf("q%059d.q", i)
			}
			stats, err := impl.StreamQueueInstanceDetailsXML(context.Background(), env("FAKE_QSTAT_EXIT=1"),
				qstat.QueueDetailOptions{Queues: queues}, noop)
			Expect(err).To(MatchError(ContainSubstring(queues[0])))
			Expect(err.Error()).NotTo(ContainSubstring(queues[len(queues)-1]), "only the first run was made")
			Expect(stats.Complete).To(BeFalse())
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
