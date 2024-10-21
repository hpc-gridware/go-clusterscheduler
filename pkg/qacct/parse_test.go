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

package qacct_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct"
)

var _ = Describe("Parse", func() {
	var sampleOutput string

	BeforeEach(func() {
		sampleOutput = `==============================================================
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
arid                               undefined
==============================================================
qname                              all.q
hostname                           master
group                              root
owner                              root
project                            NONE
department                         defaultdepartment
jobname                            sleep
jobnumber                          8
taskid                             99
pe_taskid                          NONE
account                            sge
priority                           0
qsub_time                          2024-09-27 07:41:44.421951
submit_cmd_line                    qsub -b y -t 1-100:2 sleep 0
start_time                         2024-09-27 07:42:07.265733
end_time                           2024-09-27 07:42:08.796845
granted_pe                         NONE
slots                              1
failed                             0
exit_status                        0
ru_wallclock                       1
ru_utime                           0.487
ru_stime                           0.240
ru_maxrss                          10348
ru_ixrss                           0
ru_ismrss                          0
ru_idrss                           0
ru_isrss                           0
ru_minflt                          567
ru_majflt                          0
ru_nswap                           0
ru_inblock                         0
ru_oublock                         3
ru_msgsnd                          0
ru_msgrcv                          0
ru_nsignals                        0
ru_nvcsw                           434
ru_nivcsw                          519
wallclock                          3.464
cpu                                0.726
mem                                0.002
io                                 0.000
iow                                0.000
maxvmem                            21045248
maxrss                             10596352
arid                               undefined
Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
           72        25.861        12.399        38.259              0.284              0.000              0.000`
	})

	Describe("ParseQAcctOutput", func() {
		It("should parse qacct output correctly", func() {
			jobs, err := qacct.ParseQAcctJobOutput(sampleOutput)

			Expect(err).To(BeNil())
			Expect(jobs).To(HaveLen(2))

			job1 := jobs[0]
			Expect(job1.QName).To(Equal("all.q"))
			Expect(job1.HostName).To(Equal("master"))
			Expect(job1.Group).To(Equal("root"))
			Expect(job1.Owner).To(Equal("root"))
			Expect(job1.Project).To(Equal("NONE"))
			Expect(job1.Department).To(Equal("defaultdepartment"))
			Expect(job1.JobName).To(Equal("sleep"))
			Expect(job1.JobNumber).To(Equal(int64(8)))
			Expect(job1.TaskID).To(Equal(int64(97)))
			Expect(job1.Account).To(Equal("sge"))
			Expect(job1.QSubTime).To(Equal("2024-09-27 07:41:44.421951"))
			Expect(job1.StartTime).To(Equal("2024-09-27 07:42:07.272221"))
			Expect(job1.EndTime).To(Equal("2024-09-27 07:42:08.801865"))
			Expect(job1.Failed).To(Equal(int64(0)))
			Expect(job1.ExitStatus).To(Equal(int64(0)))
			Expect(job1.RuWallClock).To(Equal(1.0))
			Expect(job1.RuUTime).To(Equal(0.492))
			Expect(job1.RuSTime).To(Equal(0.234))
			Expect(job1.RuMaxRSS).To(Equal(int64(10300)))
			Expect(job1.MaxVMem).To(Equal(int64(21045248)))
			Expect(job1.MaxRSS).To(Equal(int64(10547200)))

			job2 := jobs[1]
			Expect(job2.QName).To(Equal("all.q"))
			Expect(job2.HostName).To(Equal("master"))
			Expect(job2.JobNumber).To(Equal(int64(8)))
			Expect(job2.TaskID).To(Equal(int64(99)))
			Expect(job2.QSubTime).To(Equal("2024-09-27 07:41:44.421951"))
			Expect(job2.StartTime).To(Equal("2024-09-27 07:42:07.265733"))
			Expect(job2.EndTime).To(Equal("2024-09-27 07:42:08.796845"))
		})

		It("should handle empty input", func() {
			jobs, err := qacct.ParseQAcctJobOutput("")
			Expect(err).To(BeNil())
			Expect(jobs).To(BeEmpty())
		})

		It("should handle input with only total system usage", func() {
			input := `Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
           72        25.861        12.399        38.259              0.284              0.000              0.000`

			jobs, err := qacct.ParseQAcctJobOutput(input)
			Expect(err).To(BeNil())
			Expect(jobs).To(BeEmpty())
		})

	})
})
