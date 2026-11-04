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
	"os"
	"path/filepath"
	"strings"
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// jobDetailFixtures are the "qstat -j <id> -xml" captures of GCS 9.1.7 that
// hold a job (the unknown-job answer is tested on its own).
var jobDetailFixtures = []string{
	"hold", "dep", "time", "eqw", "never", "neverq", "matlab-one", "rqs", "alarm", "tc", "ad", "ar",
	"pe999", "running", "arr24", "fit", "eqw-root", "scoped", "sysophold", "arrhold", "with-messages",
}

func jobDetailFixture(name string) string {
	return "qstat-j-xml-" + name + "-9.1.7.xml"
}

func decodeJobDetail(name string) (*qstat.JobDetailXML, error) {
	f, err := os.Open(filepath.Join("testdata", jobDetailFixture(name)))
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	return qstat.DecodeJobDetailXML(f)
}

func mustDecodeJobDetail(name string) *qstat.JobDetailXML {
	job, err := decodeJobDetail(name)
	Expect(err).NotTo(HaveOccurred())
	Expect(job).NotTo(BeNil())
	return job
}

var _ = Describe("DecodeJobDetailXML (qstat -j -xml, 9.1.7 captures)", func() {

	DescribeTable("decodes the number, owner, task structure and times of every capture",
		func(name string) {
			job := mustDecodeJobDetail(name)
			Expect(job.JobNumber).NotTo(BeZero())
			Expect(job.Owner).NotTo(BeEmpty())
			Expect(job.Structure).NotTo(BeEmpty(), "every job has a task structure")
			Expect(job.SubmissionTime).To(HavePrefix("2026-"))
			Expect(job.Deadline).To(Equal("0"))
		},
		func(name string) string { return name },
		func() []TableEntry {
			entries := make([]TableEntry, 0, len(jobDetailFixtures))
			for _, name := range jobDetailFixtures {
				entries = append(entries, Entry(nil, name))
			}
			return entries
		}(),
	)

	It("reads the scalars and the user hold of a held plain job", func() {
		job := mustDecodeJobDetail("hold")
		Expect(job.JobNumber).To(Equal(uint64(3438)))
		Expect(job.Name).To(Equal("whyprobe_hold"))
		Expect(job.Owner).To(Equal("alice"))
		Expect(job.Group).To(Equal("alice"))
		Expect(job.Groups).To(Equal([]string{"alice"}))
		Expect(job.Project).To(Equal("research"))
		Expect(job.Department).To(Equal("dept_physics"))
		Expect(job.Account).To(Equal("sge"))
		Expect(job.CategoryID).To(Equal(uint64(7)))
		Expect(job.Reserve).To(BeFalse())
		Expect(job.Priority).To(Equal(int64(1024)))
		Expect(job.Restart).To(Equal(0))
		Expect(job.Type).To(Equal(uint64(64)))
		Expect(job.AR).To(BeZero())
		Expect(job.ExecutionTime).To(Equal("0"))
		Expect(job.Structure).To(Equal([]qstat.TaskRange{{Min: 1, Max: 1, Step: 1}}))
		Expect(job.UserHold).To(Equal([]qstat.TaskRange{{Min: 1, Max: 1, Step: 1}}))
		Expect(job.PendingTasks).To(BeNil(), "the held task is not pending without hold")
		Expect(job.OperatorHold).To(BeNil())
		Expect(job.SystemHold).To(BeNil())
		Expect(job.Successors).To(Equal([]uint64{3439}), "the dependent job waits on this one")
		Expect(job.Tasks).To(BeEmpty())
		Expect(job.Requests).To(BeEmpty(), "no -l, no -q: no request set")
	})

	It("reads the open predecessor and the requested id of a dependent job", func() {
		job := mustDecodeJobDetail("dep")
		Expect(job.Predecessors).To(Equal([]uint64{3438}))
		Expect(job.PredecessorRequests).To(Equal([]string{"3438"}))
		Expect(job.ArrayPredecessorRequests).To(BeNil())
		Expect(job.PendingTasks).To(Equal([]qstat.TaskRange{{Min: 1, Max: 1, Step: 1}}))
	})

	It("reads a requested start time and ParseXMLTime understands it", func() {
		job := mustDecodeJobDetail("time")
		Expect(job.ExecutionTime).To(Equal("2030-01-01T00:00:00.000000"))
		t, err := qstat.ParseXMLTime(job.ExecutionTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(t.Year()).To(Equal(2030))
		t, err = qstat.ParseXMLTime(job.Deadline)
		Expect(err).NotTo(HaveOccurred())
		Expect(t.IsZero()).To(BeTrue(), `"0" is an unset time`)
	})

	It("reads the error message of a task in Eqw, with entities decoded", func() {
		job := mustDecodeJobDetail("eqw")
		Expect(job.Tasks).To(HaveLen(1))
		task := job.Tasks[0]
		Expect(task.Number).To(Equal(uint64(1)))
		Expect(task.State).To(Equal(uint64(34880)))
		Expect(task.Messages).To(HaveLen(1))
		Expect(task.Messages[0]).To(ContainSubstring(`can't open output file "/nonexistent_dir_for_eqw/out"`))
		Expect(task.GrantedQueues).To(BeEmpty())
	})

	It("reads a global hard resource request", func() {
		job := mustDecodeJobDetail("never")
		Expect(job.Requests).To(HaveLen(1))
		Expect(job.Requests[0].Scope).To(Equal(qstat.ScopeGlobal))
		Expect(job.Requests[0].HardResources).To(Equal([]qstat.ResourceRequest{{Name: "gpu", Value: "99"}}))
		Expect(job.Requests[0].SoftResources).To(BeNil())
		Expect(job.Requests[0].HardQueues).To(BeNil())
		Expect(job.Requests[0].IgnoreSlaveRequestsOnMasterHost).To(BeFalse())
	})

	It("reads a global hard queue request", func() {
		job := mustDecodeJobDetail("neverq")
		Expect(job.Requests).To(HaveLen(1))
		Expect(job.Requests[0].HardQueues).To(Equal([]string{"gpu.q"}))
		Expect(job.Requests[0].HardResources).To(BeNil())
	})

	It("reads the master and slave request sets, PE range and reservation of a scoped parallel job", func() {
		job := mustDecodeJobDetail("scoped")
		Expect(job.Requests).To(HaveLen(2))
		Expect(job.Requests[0].Scope).To(Equal(qstat.ScopeMaster))
		Expect(job.Requests[0].HardResources).To(Equal([]qstat.ResourceRequest{{Name: "matlab", Value: "61"}}))
		Expect(job.Requests[1].Scope).To(Equal(qstat.ScopeSlave))
		Expect(job.Requests[1].HardQueues).To(Equal([]string{"all.q"}))
		Expect(job.PE).To(Equal("mpi"))
		Expect(job.PERange).To(Equal([]qstat.TaskRange{{Min: 2, Max: 4, Step: 1}}))
		Expect(job.Reserve).To(BeTrue())
	})

	It("reads the task concurrency, the running tasks and the pending ranges of an array job", func() {
		job := mustDecodeJobDetail("tc")
		Expect(job.TaskConcurrency).To(Equal(uint64(2)))
		Expect(job.Type).To(Equal(uint64(192)))
		Expect(job.Structure).To(Equal([]qstat.TaskRange{{Min: 1, Max: 10, Step: 1}}))
		Expect(job.PendingTasks).To(Equal([]qstat.TaskRange{{Min: 3, Max: 10, Step: 1}}))
		Expect(job.Tasks).To(HaveLen(2))
		// Document order, not task order.
		Expect(job.Tasks[0].Number).To(Equal(uint64(2)))
		Expect(job.Tasks[0].State).To(Equal(uint64(128)))
		Expect(job.Tasks[0].GrantedQueues).To(Equal([]qstat.GrantedQueueXML{{Queue: "all.q@ocs-worker1", Host: "ocs-worker1", Slots: 1}}))
		Expect(job.Tasks[1].Number).To(Equal(uint64(1)))
		Expect(job.Tasks[1].GrantedQueues).To(Equal([]qstat.GrantedQueueXML{{Queue: "all.q@ocs-master", Host: "ocs-master", Slots: 1}}))
	})

	It("reads an array dependency request", func() {
		job := mustDecodeJobDetail("ad")
		Expect(job.ArrayPredecessorRequests).To(Equal([]string{"3446"}))
		Expect(job.Type).To(Equal(uint64(192)))
		Expect(job.PredecessorRequests).To(BeNil())
	})

	It("reads the advance reservation id", func() {
		Expect(mustDecodeJobDetail("ar").AR).To(Equal(uint64(2)))
	})

	It("reads a fixed PE slot request", func() {
		job := mustDecodeJobDetail("pe999")
		Expect(job.PE).To(Equal("smp"))
		Expect(job.PERange).To(Equal([]qstat.TaskRange{{Min: 999, Max: 999, Step: 1}}))
	})

	It("reads the granted resources and queue of a running job", func() {
		job := mustDecodeJobDetail("running")
		Expect(job.Owner).To(Equal("bob"))
		Expect(job.Tasks).To(HaveLen(1))
		task := job.Tasks[0]
		Expect(task.State).To(Equal(uint64(128)))
		Expect(task.StartTime).To(Equal("2026-11-03T12:41:54.570596"))
		Expect(task.Restarted).To(BeZero())
		Expect(task.GrantedResources).To(Equal([]qstat.GrantedResourceXML{{Type: 0, Name: "matlab", Amount: "60.000000", Host: "global"}}))
		Expect(task.GrantedQueues).To(Equal([]qstat.GrantedQueueXML{{Queue: "bigmem.q@ocs-worker2", Host: "ocs-worker2", Slots: 1}}))
		Expect(task.Messages).To(BeNil())
	})

	It("reads all 24 enrolled tasks of an array job", func() {
		job := mustDecodeJobDetail("arr24")
		Expect(job.Tasks).To(HaveLen(24))
		Expect(job.Structure).To(Equal([]qstat.TaskRange{{Min: 1, Max: 24, Step: 1}}))
		seen := map[uint64]bool{}
		for _, t := range job.Tasks {
			seen[t.Number] = true
		}
		Expect(seen).To(HaveLen(24))
	})

	It("reads a system and an operator hold on the same task", func() {
		job := mustDecodeJobDetail("sysophold")
		Expect(job.SystemHold).To(Equal([]qstat.TaskRange{{Min: 1, Max: 1, Step: 1}}))
		Expect(job.OperatorHold).To(Equal([]qstat.TaskRange{{Min: 1, Max: 1, Step: 1}}))
		Expect(job.UserHold).To(BeNil())
		Expect(job.PendingTasks).To(BeNil())
	})

	It("reads several hold ranges with a step", func() {
		job := mustDecodeJobDetail("arrhold")
		Expect(job.Structure).To(Equal([]qstat.TaskRange{{Min: 1, Max: 20, Step: 2}}))
		Expect(job.UserHold).To(Equal([]qstat.TaskRange{
			{Min: 1, Max: 3, Step: 2}, {Min: 7, Max: 7, Step: 1}, {Min: 15, Max: 19, Step: 2},
		}))
		Expect(job.Tasks).To(HaveLen(4))
	})

	It("answers ErrNoSuchJob for the unknown-job document, from its root alone", func() {
		job, err := decodeJobDetail("unknown")
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
		Expect(job).To(BeNil())
	})

	It("decodes a document with a scheduler messages block and leaves the block unread", func() {
		job := mustDecodeJobDetail("with-messages")
		Expect(job.JobNumber).To(Equal(uint64(3030)))
		Expect(job.Owner).To(Equal("root"))
		Expect(job.Tasks).To(HaveLen(1))
		Expect(job.Tasks[0].Messages[0]).To(ContainSubstring("can't chdir to /no/such/dir"))
	})

	It("does not parse the messages block, even when it is not well-formed", func() {
		doc := readFixture(jobDetailFixture("hold"))
		i := strings.Index(doc, "<messages>")
		Expect(i).To(BeNumerically(">", 0))
		doc = doc[:i] + "<messages><element><job ids>3438</job ids><<<not xml at all</element></messages></detailed_job_info>"
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(doc))
		Expect(err).NotTo(HaveOccurred())
		Expect(job.JobNumber).To(Equal(uint64(3438)))
	})

	It("reports a document cut inside djob_info as incomplete, keeping what was decoded", func() {
		doc := readFixture(jobDetailFixture("hold"))
		i := strings.Index(doc, "<JB_deadline>")
		Expect(i).To(BeNumerically(">", 0))
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(doc[:i]))
		Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
		Expect(job).NotTo(BeNil())
		Expect(job.JobNumber).To(Equal(uint64(3438)))
		Expect(job.Project).To(Equal("research"), "decoded before the cut")
		Expect(job.Successors).To(Equal([]uint64{3439}), "the list before the cut survived")
		Expect(job.UserHold).To(BeNil(), "after the cut")
	})

	It("reports a document cut between the job and the messages as incomplete", func() {
		doc := readFixture(jobDetailFixture("hold"))
		i := strings.Index(doc, "<messages>")
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(doc[:i]))
		Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
		Expect(job.JobNumber).To(Equal(uint64(3438)))
	})

	It("reports empty input as incomplete and another document as malformed", func() {
		_, err := qstat.DecodeJobDetailXML(strings.NewReader(""))
		Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
		_, err = qstat.DecodeJobDetailXML(strings.NewReader("<job_info><queue_info/></job_info>"))
		Expect(err).To(MatchError(ContainSubstring("unexpected root element <job_info>")))
		// Non-strict decoding forgives a mismatched end tag; a bare "<<" is
		// still no XML.
		_, err = qstat.DecodeJobDetailXML(strings.NewReader("<detailed_job_info><djob_info><element><JB_job_number>1</JB_job_number><<"))
		Expect(err).To(MatchError(HavePrefix("malformed qstat -j -xml document")))
		_, err = qstat.DecodeJobDetailXML(strings.NewReader("<detailed_job_info><djob_info></djob_info></detailed_job_info>"))
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "a complete document without a job")
	})

	It("answers ErrNoSuchJob for a document with a messages block but no job", func() {
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(
			"<detailed_job_info><messages><element><SME_global_message_list/></element></messages></detailed_job_info>"))
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
		Expect(job).To(BeNil())
	})

	It("skips an unknown element next to djob_info", func() {
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(`<detailed_job_info>
		  <future_block><nested><deeper>x</deeper></nested></future_block>
		  <djob_info><element><JB_job_number>11</JB_job_number></element></djob_info>
		  <another_future_block/>
		</detailed_job_info>`))
		Expect(err).NotTo(HaveOccurred())
		Expect(job.JobNumber).To(Equal(uint64(11)))
	})

	It("reads the soft resource and queue requests of every scope", func() {
		// qsub -soft -l / -soft -q requests: the request set's soft lists
		// carry the same entries as the hard ones (CE_Type, QR_Type). No
		// live capture with -soft yet; shaped after the hard lists of the
		// "scoped" capture.
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(`<detailed_job_info><djob_info><element>
		  <JB_job_number>12</JB_job_number>
		  <JB_request_set_list>
		    <ulong_sublist>
		      <JRS_scope>0</JRS_scope>
		      <JRS_soft_resource_list>
		        <qstat_l_requests><CE_name>arch</CE_name><CE_valtype>3</CE_valtype><CE_stringval>lx-amd64</CE_stringval></qstat_l_requests>
		      </JRS_soft_resource_list>
		      <JRS_soft_queue_list><destin_ident_list><QR_name>fast.q</QR_name></destin_ident_list></JRS_soft_queue_list>
		    </ulong_sublist>
		    <ulong_sublist>
		      <JRS_scope>2</JRS_scope>
		      <JRS_hard_resource_list>
		        <qstat_l_requests><CE_name>gpu</CE_name><CE_stringval>1</CE_stringval></qstat_l_requests>
		      </JRS_hard_resource_list>
		      <JRS_soft_resource_list>
		        <qstat_l_requests><CE_name>mem_free</CE_name><CE_stringval>8G</CE_stringval></qstat_l_requests>
		        <qstat_l_requests><CE_name>h_rt</CE_name><CE_stringval>600</CE_stringval></qstat_l_requests>
		      </JRS_soft_resource_list>
		      <JRS_soft_queue_list>
		        <destin_ident_list><QR_name>gpu.q@n1</QR_name></destin_ident_list>
		        <destin_ident_list><QR_name>gpu.q@n2</QR_name></destin_ident_list>
		      </JRS_soft_queue_list>
		    </ulong_sublist>
		  </JB_request_set_list>
		</element></djob_info></detailed_job_info>`))
		Expect(err).NotTo(HaveOccurred())
		Expect(job.Requests).To(Equal([]qstat.RequestSet{
			{
				Scope:         qstat.ScopeGlobal,
				SoftResources: []qstat.ResourceRequest{{Name: "arch", Value: "lx-amd64"}},
				SoftQueues:    []string{"fast.q"},
			},
			{
				Scope:         qstat.ScopeSlave,
				HardResources: []qstat.ResourceRequest{{Name: "gpu", Value: "1"}},
				SoftResources: []qstat.ResourceRequest{{Name: "mem_free", Value: "8G"}, {Name: "h_rt", Value: "600"}},
				SoftQueues:    []string{"gpu.q@n1", "gpu.q@n2"},
			},
		}))
	})

	It("skips unknown elements and accepts any wrapper name for list entries", func() {
		const doc = `<?xml version='1.0'?>
<detailed_job_info>
  <djob_info>
    <element>
      <JB_job_number>42</JB_job_number>
      <JB_future_field><nested><deeper>x</deeper></nested></JB_future_field>
      <JB_owner>carol</JB_owner>
      <JB_env_list><job_sublist><VA_variable>HUGE</VA_variable><VA_value>...</VA_value></job_sublist></JB_env_list>
      <JB_request_set_list>
        <whatever_wrapper>
          <JRS_scope>2</JRS_scope>
          <JRS_hard_resource_list>
            <another_wrapper><CE_name>mem</CE_name><CE_unknown>1</CE_unknown><CE_stringval>4G</CE_stringval></another_wrapper>
            <another_wrapper><CE_name>h_rt</CE_name><CE_stringval>60</CE_stringval></another_wrapper>
          </JRS_hard_resource_list>
          <JRS_soft_queue_list><x><QR_name>soft.q</QR_name></x></JRS_soft_queue_list>
          <JRS_allocation_rule>$pe_slots</JRS_allocation_rule>
          <JRS_ignore_slave_requests_on_master_host>true</JRS_ignore_slave_requests_on_master_host>
        </whatever_wrapper>
      </JB_request_set_list>
      <JB_ja_u_h_ids>
        <range><RN_min>1</RN_min><RN_max>9</RN_max><RN_step>4</RN_step></range>
        <range><RN_min>20</RN_min><RN_max>20</RN_max><RN_step>1</RN_step></range>
      </JB_ja_u_h_ids>
      <JB_ja_a_h_ids><r><RN_min>30</RN_min><RN_max>31</RN_max><RN_step>1</RN_step></r></JB_ja_a_h_ids>
      <JB_jid_predecessor_list><p><JRE_job_number>7</JRE_job_number></p><p><JRE_job_number>8</JRE_job_number></p></JB_jid_predecessor_list>
      <JB_restart>2</JB_restart>
      <JB_priority>924</JB_priority>
      <JB_checkpoint_name>ckpt</JB_checkpoint_name>
      <JB_ja_tasks>
        <t>
          <JAT_task_number>1</JAT_task_number>
          <JAT_state>128</JAT_state>
          <JAT_future>1</JAT_future>
          <JAT_granted_destin_identifier_list>
            <g><JG_qname>a.q@h1</JG_qname><JG_qhostname>h1</JG_qhostname><JG_slots>4</JG_slots></g>
            <g><JG_qname>a.q@h2</JG_qname><JG_qhostname>h2</JG_qhostname><JG_slots>not a number</JG_slots></g>
          </JAT_granted_destin_identifier_list>
        </t>
      </JB_ja_tasks>
    </element>
  </djob_info>
</detailed_job_info>`
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(doc))
		Expect(err).NotTo(HaveOccurred())
		Expect(job.JobNumber).To(Equal(uint64(42)))
		Expect(job.Owner).To(Equal("carol"))
		Expect(job.Restart).To(Equal(2))
		Expect(job.Priority).To(Equal(int64(924)))
		Expect(job.Checkpoint).To(Equal("ckpt"))
		Expect(job.Requests).To(Equal([]qstat.RequestSet{{
			Scope:                           qstat.ScopeSlave,
			HardResources:                   []qstat.ResourceRequest{{Name: "mem", Value: "4G"}, {Name: "h_rt", Value: "60"}},
			SoftQueues:                      []string{"soft.q"},
			AllocationRule:                  "$pe_slots",
			IgnoreSlaveRequestsOnMasterHost: true,
		}}))
		Expect(job.UserHold).To(Equal([]qstat.TaskRange{{Min: 1, Max: 9, Step: 4}, {Min: 20, Max: 20, Step: 1}}))
		Expect(job.ArrayHold).To(Equal([]qstat.TaskRange{{Min: 30, Max: 31, Step: 1}}))
		Expect(job.Predecessors).To(Equal([]uint64{7, 8}))
		Expect(job.Tasks).To(HaveLen(1))
		Expect(job.Tasks[0].GrantedQueues).To(Equal([]qstat.GrantedQueueXML{
			{Queue: "a.q@h1", Host: "h1", Slots: 4}, {Queue: "a.q@h2", Host: "h2", Slots: 0},
		}))
	})

	It("maps the job-level request lists of a 9.0 document to one global request set", func() {
		const doc = `<detailed_job_info><djob_info><element>
		  <JB_job_number>5</JB_job_number>
		  <JB_hard_resource_list><qstat_l_requests><CE_name>h_rt</CE_name><CE_stringval>10</CE_stringval></qstat_l_requests></JB_hard_resource_list>
		  <JB_soft_resource_list><qstat_l_requests><CE_name>mem</CE_name><CE_stringval>1G</CE_stringval></qstat_l_requests></JB_soft_resource_list>
		  <JB_hard_queue_list><destin_ident_list><QR_name>all.q</QR_name></destin_ident_list></JB_hard_queue_list>
		  <JB_soft_queue_list><destin_ident_list><QR_name>fast.q</QR_name></destin_ident_list></JB_soft_queue_list>
		</element></djob_info></detailed_job_info>`
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(doc))
		Expect(err).NotTo(HaveOccurred())
		Expect(job.Requests).To(Equal([]qstat.RequestSet{{
			Scope:         qstat.ScopeGlobal,
			HardResources: []qstat.ResourceRequest{{Name: "h_rt", Value: "10"}},
			SoftResources: []qstat.ResourceRequest{{Name: "mem", Value: "1G"}},
			HardQueues:    []string{"all.q"},
			SoftQueues:    []string{"fast.q"},
		}}))
	})

	It("returns the first job when several were asked for", func() {
		const doc = `<detailed_job_info><djob_info>
		  <element><JB_job_number>1</JB_job_number></element>
		  <element><JB_job_number>2</JB_job_number></element>
		</djob_info></detailed_job_info>`
		job, err := qstat.DecodeJobDetailXML(strings.NewReader(doc))
		Expect(err).NotTo(HaveOccurred())
		Expect(job.JobNumber).To(Equal(uint64(1)))
	})
})

var _ = Describe("JobDetailXMLContext against a qstat process", func() {
	var impl *qstat.QStatImpl
	var argsFile string
	fixture := func(parts ...string) string {
		abs, err := filepath.Abs(filepath.Join(parts...))
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
		impl, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{
			Executable: fixture("..", "v9.0", "testdata", "fake-qstat.sh"),
		})
		Expect(err).NotTo(HaveOccurred())
		argsFile = filepath.Join(GinkgoT().TempDir(), "args")
	})

	It("returns the typed job and asks qstat for exactly that job", func() {
		job, err := impl.JobDetailXMLContext(context.Background(),
			env("FAKE_QSTAT_FILE="+fixture("testdata", jobDetailFixture("scoped"))), "3456")
		Expect(err).NotTo(HaveOccurred())
		Expect(job.JobNumber).To(Equal(uint64(3456)))
		Expect(job.PE).To(Equal("mpi"))
		Expect(job.Requests).To(HaveLen(2))
		Expect(recordedArgs()).To(Equal([]string{"-j", "3456", "-xml"}))
	})

	It("maps the unknown-jobs document to ErrNoSuchJob even when qstat exits 1", func() {
		job, err := impl.JobDetailXMLContext(context.Background(),
			env("FAKE_QSTAT_FILE="+fixture("testdata", jobDetailFixture("unknown")), "FAKE_QSTAT_EXIT=1"), "999999999")
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("999999999"))
		Expect(job).To(BeNil())
	})

	It("maps the stderr answer to ErrNoSuchJob", func() {
		_, err := impl.JobDetailXMLContext(context.Background(),
			env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=Following jobs do not exist: \n99999999"), "99999999")
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
	})

	It("passes another failure through in the command-error shape", func() {
		_, err := impl.JobDetailXMLContext(context.Background(),
			env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=error: unable to send message to qmaster"), "1")
		Expect(err).To(MatchError(HavePrefix("failed to run command (error: unable to send message to qmaster")))
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeFalse())
	})

	It("is cancelled by its context while qstat is still silent", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		started := time.Now()
		_, err := impl.JobDetailXMLContext(ctx,
			env("FAKE_QSTAT_SLEEP=5", "FAKE_QSTAT_FILE="+fixture("testdata", jobDetailFixture("hold"))), "3438")
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
		Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
	})

	It("reports a document that ends before the messages block as incomplete", func() {
		doc := readFixture(jobDetailFixture("hold"))
		cut := filepath.Join(GinkgoT().TempDir(), "cut.xml")
		Expect(os.WriteFile(cut, []byte(doc[:strings.Index(doc, "<JB_deadline>")]), 0o600)).To(Succeed())
		_, err := impl.JobDetailXMLContext(context.Background(), env("FAKE_QSTAT_FILE="+cut), "3438")
		Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
	})

	It("lets a qstat with a messages block far larger than a pipe finish", func() {
		doc := readFixture(jobDetailFixture("hold"))
		i := strings.Index(doc, "<messages>")
		var big strings.Builder
		big.WriteString(doc[:i])
		big.WriteString("<messages><element><SME_global_message_list>")
		for len(big.String()) < 2<<20 {
			big.WriteString("<element><MES_message_number>39</MES_message_number><MES_message>queue instance dropped because it is disabled</MES_message></element>\n")
		}
		big.WriteString("</SME_global_message_list></element></messages></detailed_job_info>\n")
		file := filepath.Join(GinkgoT().TempDir(), "big.xml")
		Expect(os.WriteFile(file, []byte(big.String()), 0o600)).To(Succeed())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		job, err := impl.JobDetailXMLContext(ctx, env("FAKE_QSTAT_FILE="+file), "3438")
		Expect(err).NotTo(HaveOccurred())
		Expect(job.JobNumber).To(Equal(uint64(3438)))
	})

	DescribeTable("refuses a job id that could forge an option, before spawning anything",
		func(id string) {
			_, err := impl.JobDetailXMLContext(context.Background(), env(), id)
			Expect(err).To(HaveOccurred())
			Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
		},
		Entry("an option", "-u"),
		Entry("a control character", "1\n-u *"),
		Entry("nothing", ""),
		Entry("blanks", "  "),
		Entry("a comma list", "1,2"),
		Entry("a blank list", "1 2"),
		Entry("a wildcard", "*"),
		Entry("a name pattern", "job*"),
		Entry("a job name", "myjob"),
	)

	DescribeTable("accepts one numeric job id, with or without a task range",
		func(id string) {
			_, err := impl.JobDetailXMLContext(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("testdata", jobDetailFixture("hold"))), id)
			Expect(err).NotTo(HaveOccurred())
			Expect(recordedArgs()).To(Equal([]string{"-j", id, "-xml"}))
		},
		Entry("a job id", "42"),
		Entry("a task range", "42.1-10:2"),
	)

	It("rejects a job list even when GCS_VALIDATION is off", func() {
		GinkgoT().Setenv("GCS_VALIDATION", "off")
		_, err := impl.JobDetailXMLContext(context.Background(), env(), "1,2")
		Expect(err).To(HaveOccurred())
		Expect(argsFile).NotTo(BeAnExistingFile(), "qstat must not have been started")
	})

	It("returns an empty job in dry-run mode", func() {
		dry, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		job, err := dry.JobDetailXMLContext(context.Background(), nil, "1")
		Expect(err).NotTo(HaveOccurred())
		Expect(job).To(Equal(&qstat.JobDetailXML{}))
	})
})
