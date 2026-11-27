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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

// The files under testdata are lines of real accounting files:
// accounting-9.0.10.jsonl of GCS 9.0.10, accounting-9.1.7.jsonl of GCS
// 9.1.7 (chosen to hold one record of each kind the specs name).

func fixture(name string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())
	return data
}

func readAll(data []byte) core.ReadResult {
	res, err := core.ReadTail(context.Background(), bytes.NewReader(data), int64(len(data)), core.ReadOptions{})
	Expect(err).NotTo(HaveOccurred())
	return res
}

// A record in the shape of 9.0.0 to 9.0.5: the extended usage is named
// "usage", and there is no pe_taskid, category or deleted_by.
const record900 = `{"job_number":77,"task_number":0,"start_time":1728000000000000,"end_time":1728000005000000,"owner":"alice","group":"users","account":"sge","qname":"all.q","hostname":"node1","department":"defaultdepartment","slots":1,"job_name":"old","priority":0,"submission_time":1727999999000000,"submit_cmd_line":"qsub -b y sleep 5","failed":0,"exit_status":0,"usage":{"rusage":{"ru_wallclock":5,"ru_utime":0.1,"ru_stime":0.2,"ru_maxrss":100},"usage":{"wallclock":5.5,"cpu":0.3,"mem":0.01,"io":0.0,"iow":0.0,"maxvmem":1024.0,"maxrss":512.0}}}`

// A record of a job which ran in an advance reservation: qmaster writes
// arid and ar_submission_time as numbers (sge_rusage.cc).
const recordAR = `{"job_number":78,"task_number":0,"start_time":1,"end_time":2,"owner":"alice","qname":"all.q","hostname":"node1","slots":1,"arid":5,"job_name":"inar","priority":0,"submission_time":1,"ar_submission_time":9,"failed":0,"exit_status":0,"usage":{"rusage":{"ru_wallclock":1},"eusage":{"wallclock":1.0}}}`

var _ = Describe("Accounting record", func() {

	Context("real files", func() {
		It("reads every line of a 9.0.10 file", func() {
			res := readAll(fixture("accounting-9.0.10.jsonl"))
			Expect(res.NotParsed).To(BeZero())
			Expect(res.Records).To(HaveLen(3))
			job := res.Records[0].JobDetail
			Expect(job.JobNumber).To(Equal(int64(1)))
			Expect(job.JobName).To(Equal("fin1"))
			Expect(job.QName).To(Equal("all.q"))
			Expect(job.StartTime).To(BeNumerically(">", int64(1700000000000000)))
			// 9.0.6 and later name the extended usage "eusage".
			Expect(job.JobUsage.Usage.WallClock).To(BeNumerically(">", 0))
			// 9.0 writes no pe_taskid for a job without a PE task.
			Expect(job.PETaskID).To(BeEmpty())
			Expect(job.Extra).To(BeEmpty())
		})

		It("reads every line of a 9.1.7 file", func() {
			res := readAll(fixture("accounting-9.1.7.jsonl"))
			Expect(res.NotParsed).To(BeZero())
			Expect(res.Records).To(HaveLen(13))
			for _, record := range res.Records {
				Expect(record.Extra).To(BeEmpty(), "job %d", record.JobNumber)
				Expect(record.Category).NotTo(BeEmpty(), "job %d", record.JobNumber)
			}
		})

		It("reads a 9.0.10 record of a job in an advance reservation", func() {
			// qmaster writes arid as a number; this record failed to parse
			// while the field was read as a string.
			res := readAll(fixture("accounting-9.0.10-ar.jsonl"))
			Expect(res.NotParsed).To(BeZero())
			Expect(res.Records).To(HaveLen(1))
			job := res.Records[0]
			Expect(job.JobName).To(Equal("lu_inar"))
			Expect(job.ArID).To(Equal("2"))
			Expect(job.ARSubmissionTime).To(BeNumerically(">", int64(1700000000000000)))
			Expect(job.Category).To(Equal("-ar 2"))
			Expect(job.Extra).To(BeEmpty())
		})

		It("reads the kinds of records a 9.1.7 file holds", func() {
			records := readAll(fixture("accounting-9.1.7.jsonl")).Records

			peTask := records[1]
			Expect(peTask.PETaskID).To(Equal("1.node4"))
			Expect(peTask.GrantedPE).To(Equal("demo_openmpi"))
			Expect(peTask.DeletedBy).To(Equal("root@node1"))
			Expect(peTask.Failed).To(Equal(int64(100)))
			Expect(peTask.ExitStatus).To(Equal(int64(137)))

			plain := records[5]
			Expect(plain.PETaskID).To(Equal("NONE"))
			Expect(plain.GrantedPE).To(BeEmpty())
			Expect(plain.DeletedBy).To(BeEmpty())
			Expect(plain.Failed).To(BeZero())

			arrayTask := records[6]
			Expect(arrayTask.JobNumber).To(Equal(int64(239)))
			Expect(arrayTask.TaskID).To(Equal(int64(4)))

			// A job which never started: no start and no end time.
			neverStarted := records[7]
			Expect(neverStarted.Failed).To(Equal(int64(8)))
			Expect(neverStarted.StartTime).To(BeZero())
			Expect(neverStarted.EndTime).To(BeZero())
			// One job, one task, two records: it was run twice.
			Expect(records[8].JobNumber).To(Equal(neverStarted.JobNumber))
			Expect(records[8].TaskID).To(Equal(neverStarted.TaskID))

			Expect(records[9].Project).To(Equal("research"))
			Expect(records[12].Failed).To(Equal(int64(37)))
		})
	})

	Context("shapes of other releases", func() {
		It("reads a record of 9.0.0 to 9.0.5 (extended usage named usage)", func() {
			job, err := core.ParseAccountingRecord([]byte(record900))
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Usage.WallClock).To(Equal(5.5))
			Expect(job.JobUsage.Usage.CPU).To(Equal(0.3))
			Expect(job.JobUsage.RUsage.RuWallclock).To(Equal(int64(5)))
			Expect(job.JobUsage.Custom).To(BeNil())
		})

		It("reads a record of a job in an advance reservation (arid is a number)", func() {
			job, err := core.ParseAccountingRecord([]byte(recordAR))
			Expect(err).NotTo(HaveOccurred())
			Expect(job.ArID).To(Equal("5"))
			Expect(job.ARSubmissionTime).To(Equal(int64(9)))
			Expect(job.Extra).To(BeEmpty())
		})

		It("reads what JobDetail marshals to (arid is a string there)", func() {
			job, err := core.ParseAccountingRecord([]byte(recordAR))
			Expect(err).NotTo(HaveOccurred())
			marshalled, err := json.Marshal(job)
			Expect(err).NotTo(HaveOccurred())
			var again core.JobDetail
			Expect(json.Unmarshal(marshalled, &again)).To(Succeed())
			Expect(again.ArID).To(Equal("5"))
			Expect(again.JobNumber).To(Equal(job.JobNumber))
			Expect(again.JobUsage.Usage.WallClock).To(Equal(job.JobUsage.Usage.WallClock))
		})

		It("keeps a member a newer release added", func() {
			job, err := core.ParseAccountingRecord([]byte(
				`{"job_number":1,"binding":"0,1","limits":{"h_rt":60},"usage":{"rusage":{}}}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Extra).To(HaveLen(2))
			Expect(string(job.Extra["binding"])).To(Equal(`"0,1"`))
			Expect(string(job.Extra["limits"])).To(Equal(`{"h_rt":60}`))
		})

		It("keeps the record when a known member has another type", func() {
			job, err := core.ParseAccountingRecord([]byte(
				`{"job_number":4,"priority":"high","slots":2,"arid":{"id":1},"owner":"bob"}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Owner).To(Equal("bob"))
			Expect(job.Slots).To(Equal(int64(2)))
			Expect(job.Priority).To(BeZero())
			Expect(string(job.Extra["priority"])).To(Equal(`"high"`))
			Expect(job.Extra).To(HaveKey("arid"))
		})

		It("leaves a field alone when its value does not fit, even partly", func() {
			// The decoder writes the digits it has read before it fails:
			// decoded straight into the field, 1e3 left 1 there.
			job, err := core.ParseAccountingRecord([]byte(
				`{"job_number":7,"task_number":4.5,"end_time":1.7e15,"exit_status":1.5,"slots":2e0,"failed":"8"}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(job.TaskID).To(BeZero())
			Expect(job.EndTime).To(BeZero())
			Expect(job.ExitStatus).To(BeZero())
			Expect(job.Slots).To(BeZero())
			Expect(job.Failed).To(BeZero())
			for _, key := range []string{"task_number", "end_time", "exit_status", "slots", "failed"} {
				Expect(job.Extra).To(HaveKey(key))
			}
		})

		It("does not marshal Extra", func() {
			job, err := core.ParseAccountingRecord([]byte(`{"job_number":1,"binding":"0,1"}`))
			Expect(err).NotTo(HaveOccurred())
			marshalled, err := json.Marshal(job)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(marshalled)).NotTo(ContainSubstring("binding"))
			Expect(string(marshalled)).NotTo(ContainSubstring("category"))
		})
	})

	DescribeTable("a line which is no record",
		func(line string) {
			_, err := core.ParseAccountingRecord([]byte(line))
			Expect(err).To(HaveOccurred())
		},
		Entry("not JSON", `job 4711 ended`),
		Entry("cut in the middle", `{"job_number":4711,"task_nu`),
		Entry("an array", `[1,2]`),
		Entry("an empty object", `{}`),
		Entry("no job_number", `{"owner":"alice"}`),
		Entry("a job_number which is no number", `{"job_number":"x"}`),
		Entry("job_number 0", `{"job_number":0}`),
		Entry("a job_number in exponent form", `{"job_number":1e3}`),
		Entry("a job_number with a fraction", `{"job_number":4711.5}`),
		Entry("a job_number beyond 64 bits", `{"job_number":99999999999999999999}`),
		Entry("a standard usage group which cannot be read", `{"job_number":1,"usage":{"rusage":{"ru_nswap":"many"}}}`),
	)

	Context("every usage value", func() {
		It("lists the values of a real record, eusage first", func() {
			line := bytes.Split(fixture("accounting-9.1.7.jsonl"), []byte("\n"))[5]
			values, err := core.AccountingUsageValues(line)
			Expect(err).NotTo(HaveOccurred())
			Expect(values[0].Group).To(Equal("eusage"))
			Expect(values[0].Name).To(Equal("cpu"))
			Expect(values[0].Numeric).To(BeTrue())
			groups := map[string]int{}
			for _, v := range values {
				groups[v.Group]++
				Expect(v.Numeric).To(BeTrue(), "%s.%s", v.Group, v.Name)
			}
			Expect(groups).To(Equal(map[string]int{"eusage": 8, "rusage": 18}))
		})

		It("reports the extended usage of 9.0.0 as eusage", func() {
			values, err := core.AccountingUsageValues([]byte(record900))
			Expect(err).NotTo(HaveOccurred())
			Expect(values[0]).To(Equal(core.UsageValue{Group: "eusage", Name: "cpu", Number: 0.3, Numeric: true}))
		})

		It("keeps what has no field: new members, text, groups", func() {
			values, err := core.AccountingUsageValues([]byte(`{"job_number":1,"usage":{` +
				`"rusage":{"ru_wallclock":3},` +
				`"eusage":{"wallclock":3.5,"energy":7.25},` +
				`"usage":{"x":1},` +
				`"gpu":{"model":"A100","count":2,"ok":true,"none":null,"per_device":[1,2],"big":1e3},` +
				`"note":"text"}}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal([]core.UsageValue{
				{Group: "eusage", Name: "energy", Number: 7.25, Numeric: true},
				{Group: "eusage", Name: "wallclock", Number: 3.5, Numeric: true},
				{Group: "rusage", Name: "ru_wallclock", Number: 3, Numeric: true},
				{Group: "gpu", Name: "big", Number: 1000, Numeric: true},
				{Group: "gpu", Name: "count", Number: 2, Numeric: true},
				{Group: "gpu", Name: "model", Text: "A100"},
				{Group: "gpu", Name: "ok", Text: "true"},
				{Group: "gpu", Name: "per_device", Text: "[1,2]"},
				{Group: "note", Name: "", Text: "text"},
				// With an eusage group, "usage" is a usage_patterns group.
				{Group: "usage", Name: "x", Number: 1, Numeric: true},
			}))
		})

		It("returns nothing for a record without usage", func() {
			values, err := core.AccountingUsageValues([]byte(`{"job_number":1}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(BeEmpty())
		})
	})

	DescribeTable("the text of a failed code",
		func(code int64, text string) {
			Expect(core.FailedText(code)).To(Equal(text))
		},
		Entry("ran and ended by itself", int64(0), ""),
		Entry("prolog", int64(8), "in prolog"),
		Entry("rescheduling", int64(25), "rescheduling"),
		Entry("h_rt enforced by qmaster", int64(37), "qmaster enforced h_rt limit"),
		Entry("after the job", int64(100), "assumedly after job"),
		Entry("unused code", int64(31), "invalid execution state"),
		Entry("unknown code", int64(4711), "invalid execution state"),
	)
})
