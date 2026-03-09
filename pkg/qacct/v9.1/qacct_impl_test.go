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

package qacct_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.1"
)

var _ = Describe("QAcct v9.1", func() {

	Context("Native specification", func() {

		It("should return the help output using the native specification", func() {
			q, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())

			result, err := q.NativeSpecification([]string{"-help"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).
				To(ContainSubstring("usage: qacct [options]"))
		})

		It("should return an error if the command does not exist", func() {
			q, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "nonexistent-command",
				})
			Expect(err).To(HaveOccurred())
			Expect(q).To(BeNil())
		})

	})

	Context("Parsing", func() {

		It("should parse qacct output correctly", func() {
			sampleOutput := `==============================================================
qname                              all.q
hostname                           master
group                              root
owner                              root
project                            NONE
department                         defaultdepartment
jobname                            sleep
jobnumber                          8
taskid                             97
pe_taskid                          NONE
account                            sge
priority                           0
qsub_time                          2024-09-27 07:41:44.421951
submit_cmd_line                    qsub -b y -t 1-100:2 sleep 0
start_time                         2024-09-27 07:42:07.272221
end_time                           2024-09-27 07:42:08.801865
granted_pe                         NONE
slots                              1
failed                             0
exit_status                        0
ru_wallclock                       1
ru_utime                           0.492
ru_stime                           0.234
ru_maxrss                          10300
ru_ixrss                           0
ru_ismrss                          0
ru_idrss                           0
ru_isrss                           0
ru_minflt                          572
ru_majflt                          0
ru_nswap                           0
ru_inblock                         0
ru_oublock                         3
ru_msgsnd                          0
ru_msgrcv                          0
ru_nsignals                        0
ru_nvcsw                           471
ru_nivcsw                          568
wallclock                          3.487
cpu                                0.726
mem                                0.002
io                                 0.000
iow                                0.000
maxvmem                            21045248
maxrss                             10547200
arid                               undefined`

			jobs, err := qacct.ParseQAcctJobOutput(sampleOutput)
			Expect(err).To(BeNil())
			Expect(jobs).To(HaveLen(1))

			job := jobs[0]
			Expect(job.QName).To(Equal("all.q"))
			Expect(job.HostName).To(Equal("master"))
			Expect(job.JobNumber).To(Equal(int64(8)))
			Expect(job.TaskID).To(Equal(int64(97)))
			Expect(job.JobUsage.Usage.WallClock).To(Equal(3.487))
		})

		It("should parse summary output", func() {
			input := `Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
           72        25.861        12.399        38.259              0.284              0.000              0.000`

			usage, err := qacct.ParseSummaryOutput(input)
			Expect(err).To(BeNil())
			Expect(usage.WallClock).To(Equal(72.0))
			Expect(usage.CPU).To(Equal(38.259))
		})

		It("should parse accounting JSON line", func() {
			sampleOutput := `{"job_number":10,"task_number":1,"start_time":1730532913429415,"end_time":1730532913979016,"owner":"root","group":"root","account":"sge","qname":"all.q","hostname":"master","department":"defaultdepartment","slots":1,"job_name":"echo","priority":0,"submission_time":1730532912874519,"submit_cmd_line":"qsub -b y -terse echo 'job 1'","category":"","failed":0,"exit_status":0,"usage":{"rusage":{"ru_wallclock":0,"ru_utime":0.355821,"ru_stime":0.161309,"ru_maxrss":10284,"ru_ixrss":0,"ru_ismrss":0,"ru_idrss":0,"ru_isrss":0,"ru_minflt":504,"ru_majflt":0,"ru_nswap":0,"ru_inblock":0,"ru_oublock":11,"ru_msgsnd":0,"ru_msgrcv":0,"ru_nsignals":0,"ru_nvcsw":248,"ru_nivcsw":14},"usage":{"wallclock":2.022342,"cpu":0.51713,"mem":0.0043125152587890625,"io":0.000008341856300830841,"iow":0.0,"maxvmem":21049344.0,"maxrss":10530816.0}}}`

			job, err := qacct.ParseAccountingJSONLine(sampleOutput)
			Expect(err).To(BeNil())
			Expect(job.JobNumber).To(Equal(int64(10)))
			Expect(job.JobName).To(Equal("echo"))
		})
	})

	Context("Builders", func() {

		It("should build summary query with filters", func() {
			qa, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())

			builder := qa.Summary().
				LastDays(7).
				Owner("root").
				Queue("all.q")
			Expect(builder).NotTo(BeNil())
		})

		It("should build jobs query with filters", func() {
			qa, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					Executable: "qacct",
				})
			Expect(err).NotTo(HaveOccurred())

			builder := qa.Jobs().
				Owner("root").
				Host("master").
				LastDays(1)
			Expect(builder).NotTo(BeNil())
		})
	})

	Context("Dry run", func() {

		It("should return dry run output", func() {
			q, err := qacct.NewCommandLineQAcct(
				qacct.CommandLineQAcctConfig{
					DryRun: true,
				})
			Expect(err).NotTo(HaveOccurred())

			result, err := q.NativeSpecification([]string{"-j", "123"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("Dry run"))
			Expect(result).To(ContainSubstring("-j"))
			Expect(result).To(ContainSubstring("123"))
		})
	})
})
