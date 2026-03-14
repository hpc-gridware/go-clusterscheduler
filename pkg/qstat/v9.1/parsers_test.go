/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parsers v9.1", func() {

	Context("ParseSchedulerJobInfo (qstat -j)", func() {

		qstatJ := `==============================================================
job_number:                      23
category_id:                     1
exec_file:                       job_scripts/23
submission_time:                 2026-03-14 13:37:31.653513
submit_cmd_line:                 qsub -b y -t 1-1000 sleep 30
effective_submit_cmd_line:       qsub -A sge -b yes -M gcsdemo@gcs-demo -N sleep -pty no -r yes -t 1-1000:1 sleep 30
owner:                           gcsdemo
uid:                             1003
group:                           gcsdemo
gid:                             1004
groups:                          1004(gcsdemo),27(sudo)
sge_o_home:                      /home/gcsdemo
sge_o_log_name:                  gcsdemo
sge_o_path:                      /opt/gcs/bin/lx-amd64:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
sge_o_shell:                     /bin/bash
sge_o_workdir:                   /home/gcsdemo
sge_o_host:                      gcs-demo
account:                         sge
mail_list:                       gcsdemo@gcs-demo
notify:                          FALSE
job_name:                        sleep
priority:                        0
jobshare:                        0
env_list:                        NONE
job_args:                        30
script_file:                     sleep
department:                      defaultdepartment
sync_options:                    n
job-array tasks:   i              1-1000:1
binding:                         NONE
job_state                  49:   r
usage                      49:   wallclock=00:00:25.174140,cpu=00:00:00.335190,mem=0.00000 GBs,io=0.00000,ioops=0,iow=00:00:00.000000,vmem=N/A,maxvmem=N/A,rss=460.000K,maxrss=3.168M
exec_binding_list          49:   NONE
exec_queue_list            49:   all.q@gcs-demo=1
exec_host_list             49:   gcs-demo=1
start_time                 49:   2026-03-14 13:45:31.248077
resource_map               49:   NONE
job_state                  50:   r
usage                      50:   wallclock=00:00:25.171116,cpu=00:00:00.330582,mem=0.00000 GBs,io=0.00000,ioops=0,iow=00:00:00.000000,vmem=N/A,maxvmem=N/A,rss=460.000K,maxrss=3.012M
exec_binding_list          50:   NONE
exec_queue_list            50:   all.q@gcs-demo=1
exec_host_list             50:   gcs-demo=1
start_time                 50:   2026-03-14 13:45:31.249055
resource_map               50:   NONE
job_state                  51:   r
usage                      51:   wallclock=00:00:17.113799,cpu=00:00:00.290752,mem=0.00000 GBs,io=0.00000,ioops=0,iow=00:00:00.000000,vmem=N/A,maxvmem=N/A,rss=452.000K,maxrss=3.785M
exec_binding_list          51:   NONE
exec_queue_list            51:   all.q@gcs-demo=1
exec_host_list             51:   gcs-demo=1
start_time                 51:   2026-03-14 13:45:39.308831
resource_map               51:   NONE
job_state                  52:   r
usage                      52:   wallclock=00:00:06.031501,cpu=00:00:00.288341,mem=0.00000 GBs,io=0.00000,ioops=0,iow=00:00:00.000000,vmem=N/A,maxvmem=N/A,rss=756.000K,maxrss=3.969M
exec_binding_list          52:   NONE
exec_queue_list            52:   all.q@gcs-demo=1
exec_host_list             52:   gcs-demo=1
start_time                 52:   2026-03-14 13:45:50.390741
resource_map               52:   NONE
scheduling info:                 (Collecting of scheduler job information is turned off - use qalter -w p job_id to verify if the job can be scheduled)`

		It("parses header fields correctly", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))

			j := jobs[0]
			Expect(j.JobNumber).To(Equal(23))
			Expect(j.CategoryID).To(Equal(1))
			Expect(j.ExecFile).To(Equal("job_scripts/23"))
			Expect(j.SubmissionTime).To(Equal("2026-03-14 13:37:31.653513"))
			Expect(j.SubmitCmdLine).To(Equal("qsub -b y -t 1-1000 sleep 30"))
			Expect(j.EffectiveSubmitCmdLine).To(HavePrefix("qsub -A sge"))
			Expect(j.Owner).To(Equal("gcsdemo"))
			Expect(j.UID).To(Equal(1003))
			Expect(j.Group).To(Equal("gcsdemo"))
			Expect(j.GID).To(Equal(1004))
			Expect(j.Groups).To(Equal("1004(gcsdemo),27(sudo)"))
			Expect(j.SgeOLogName).To(Equal("gcsdemo"))
			Expect(j.SgeOShell).To(Equal("/bin/bash"))
			Expect(j.SgeOHost).To(Equal("gcs-demo"))
			Expect(j.Account).To(Equal("sge"))
			Expect(j.Notify).To(BeFalse())
			Expect(j.JobName).To(Equal("sleep"))
			Expect(j.Priority).To(Equal(0))
			Expect(j.JobShare).To(Equal(0))
			Expect(j.Department).To(Equal("defaultdepartment"))
			Expect(j.SyncOptions).To(Equal("n"))
			Expect(j.Binding).To(Equal("NONE"))
		})

		It("parses job-array tasks field with leading 'i'", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].JobArrayTasks).To(Equal("i              1-1000:1"))
		})

		It("parses per-task details", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())

			j := jobs[0]
			Expect(j.Tasks).To(HaveLen(4))

			Expect(j.Tasks[0].TaskID).To(Equal(49))
			Expect(j.Tasks[0].State).To(Equal("r"))
			Expect(j.Tasks[0].QueueList).To(Equal("all.q@gcs-demo=1"))
			Expect(j.Tasks[0].HostList).To(Equal("gcs-demo=1"))
			Expect(j.Tasks[0].StartTime).To(Equal("2026-03-14 13:45:31.248077"))
			Expect(j.Tasks[0].BindingList).To(Equal("NONE"))
			Expect(j.Tasks[0].ResourceMap).To(Equal("NONE"))
			Expect(j.Tasks[0].Usage).To(ContainSubstring("wallclock="))

			Expect(j.Tasks[1].TaskID).To(Equal(50))
			Expect(j.Tasks[1].State).To(Equal("r"))

			Expect(j.Tasks[2].TaskID).To(Equal(51))
			Expect(j.Tasks[3].TaskID).To(Equal(52))
		})

		It("parses scheduling info", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].SchedulingInfo).To(ContainSubstring("Collecting of scheduler job information"))
		})

	})

	Context("ParseExtendedJobInfo (qstat -ext)", func() {

		qstatExt := `job-ID     prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
        23 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep r     0:00:00:00 0.00000 0.00000     0     0     0     0     0 0.00  all.q@gcs-demo                     1 109
        23 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep r     0:00:00:00 0.00000 0.00000     0     0     0     0     0 0.00  all.q@gcs-demo                     1 110
        23 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep r     0:00:00:00 0.00000 0.00000     0     0     0     0     0 0.00  all.q@gcs-demo                     1 111
        23 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@gcs-demo                     1 112
        23 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 113-1000:1
        24 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 1-1000:1
        25 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1
        28 0.55500 0.50000 sleep      gcsdemo      NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1`

		It("parses running jobs with duration cpu field", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExt)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(jobs)).To(BeNumerically(">=", 8))

			Expect(jobs[0].JobID).To(Equal(23))
			Expect(jobs[0].Priority).To(Equal(0.555))
			Expect(jobs[0].Ntckts).To(Equal(0.5))
			Expect(jobs[0].Name).To(Equal("sleep"))
			Expect(jobs[0].User).To(Equal("gcsdemo"))
			Expect(jobs[0].Project).To(Equal("NA"))
			Expect(jobs[0].Department).To(Equal("defaultdep"))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].CPU).To(Equal("0:00:00:00"))
			Expect(jobs[0].Queue).To(Equal("all.q@gcs-demo"))
			Expect(jobs[0].Slots).To(Equal(1))
			Expect(jobs[0].JATaskID).To(Equal("109"))
		})

		It("parses running jobs with NA cpu field", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExt)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[3].JobID).To(Equal(23))
			Expect(jobs[3].State).To(Equal("r"))
			Expect(jobs[3].CPU).To(Equal("NA"))
			Expect(jobs[3].JATaskID).To(Equal("112"))
		})

		It("parses waiting jobs with task ranges", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExt)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[4].JobID).To(Equal(23))
			Expect(jobs[4].State).To(Equal("qw"))
			Expect(jobs[4].JATaskID).To(Equal("113-1000:1"))

			Expect(jobs[5].JobID).To(Equal(24))
			Expect(jobs[5].JATaskID).To(Equal("1-1000:1"))
		})

		It("parses waiting jobs without task ID", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExt)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[6].JobID).To(Equal(25))
			Expect(jobs[6].State).To(Equal("qw"))
			Expect(jobs[6].JATaskID).To(Equal(""))

			Expect(jobs[7].JobID).To(Equal(28))
			Expect(jobs[7].Slots).To(Equal(1))
		})

	})

	Context("ParseGroupByTask (qstat -g t)", func() {

		qstatGT := `job-ID     prior   name       user         state submit/start at     queue                          master ja-task-ID
------------------------------------------------------------------------------------------------------------------
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:53:38 all.q@gcs-demo                 MASTER 112
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:53:50 all.q@gcs-demo                 MASTER 113
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:53:50 all.q@gcs-demo                 MASTER 114
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:53:58 all.q@gcs-demo                 MASTER 115
        23 0.55500 sleep      gcsdemo      qw    2026-03-14 13:37:31                                       116-1000:1
        24 0.55500 sleep      gcsdemo      qw    2026-03-14 13:37:34                                       1-1000:1
        25 0.55500 sleep      gcsdemo      qw    2026-03-14 13:38:11
        28 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27
        29 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27
        30 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27
        31 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27
        32 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27`

		It("parses running jobs with MASTER and task ID", func() {
			jobs, err := qstat.ParseGroupByTask(qstatGT)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(12))

			Expect(jobs[0].JobID).To(Equal(23))
			Expect(jobs[0].Priority).To(Equal(0.555))
			Expect(jobs[0].Name).To(Equal("sleep"))
			Expect(jobs[0].User).To(Equal("gcsdemo"))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].Queue).To(Equal("all.q@gcs-demo"))
			Expect(jobs[0].Master).To(Equal("MASTER"))
			Expect(jobs[0].JaTaskIDs).To(Equal([]int64{112}))
			Expect(jobs[0].StartTime.Year()).To(Equal(2026))
			Expect(jobs[0].StartTime.Month()).To(Equal(time.March))
			Expect(jobs[0].StartTime.Day()).To(Equal(14))
		})

		It("parses waiting jobs with task ranges", func() {
			jobs, err := qstat.ParseGroupByTask(qstatGT)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[4].JobID).To(Equal(23))
			Expect(jobs[4].State).To(Equal("qw"))
			Expect(jobs[4].SubmitTime.Year()).To(Equal(2026))
			Expect(jobs[4].JaTaskIDs).To(HaveLen(885))
			Expect(jobs[4].JaTaskIDs[0]).To(Equal(int64(116)))
			Expect(jobs[4].JaTaskIDs[len(jobs[4].JaTaskIDs)-1]).To(Equal(int64(1000)))
		})

		It("parses waiting jobs with full task range", func() {
			jobs, err := qstat.ParseGroupByTask(qstatGT)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[5].JobID).To(Equal(24))
			Expect(jobs[5].State).To(Equal("qw"))
			Expect(jobs[5].JaTaskIDs).To(HaveLen(1000))
		})

		It("parses waiting jobs without task IDs", func() {
			jobs, err := qstat.ParseGroupByTask(qstatGT)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[6].JobID).To(Equal(25))
			Expect(jobs[6].State).To(Equal("qw"))
			Expect(jobs[6].JaTaskIDs).To(BeNil())

			Expect(jobs[11].JobID).To(Equal(32))
		})

	})

	Context("ParseQstatFullOutput (qstat -f)", func() {

		qstatF := `queuename                      qtype resv/used/tot. load_avg arch          states
---------------------------------------------------------------------------------
all.q@gcs-demo                 BIP   0/4/16         0.06     lx-amd64
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:54:09     1 116
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:54:22     1 117
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:54:22     1 118
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:54:30     1 119
---------------------------------------------------------------------------------
test.q@gcs-demo                BIP   0/0/16         0.06     lx-amd64

############################################################################
 - PENDING JOBS - PENDING JOBS - PENDING JOBS - PENDING JOBS - PENDING JOBS
############################################################################
        23 0.55500 sleep      gcsdemo      qw    2026-03-14 13:37:31     1 120-1000:1
        24 0.55500 sleep      gcsdemo      qw    2026-03-14 13:37:34     1 1-1000:1
        25 0.55500 sleep      gcsdemo      qw    2026-03-14 13:38:11     1
        28 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27     1
        29 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27     1
        30 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27     1
        31 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27     1
        32 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27     1
        33 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27     1`

		It("parses queue headers correctly", func() {
			full, err := qstat.ParseQstatFullOutput(qstatF)
			Expect(err).NotTo(HaveOccurred())
			Expect(full).To(HaveLen(2))

			Expect(full[0].QueueName).To(Equal("all.q@gcs-demo"))
			Expect(full[0].QueueType).To(Equal("BIP"))
			Expect(full[0].Reserved).To(Equal(0))
			Expect(full[0].Used).To(Equal(4))
			Expect(full[0].Total).To(Equal(16))
			Expect(full[0].LoadAvg).To(Equal(0.06))
			Expect(full[0].Arch).To(Equal("lx-amd64"))

			Expect(full[1].QueueName).To(Equal("test.q@gcs-demo"))
			Expect(full[1].Used).To(Equal(0))
			Expect(full[1].Total).To(Equal(16))
		})

		It("parses running jobs in queues", func() {
			full, err := qstat.ParseQstatFullOutput(qstatF)
			Expect(err).NotTo(HaveOccurred())

			Expect(full[0].Jobs).To(HaveLen(4))
			Expect(full[0].Jobs[0].JobID).To(Equal(23))
			Expect(full[0].Jobs[0].State).To(Equal("r"))
			Expect(full[0].Jobs[0].StartTime).To(Equal(
				time.Date(2026, 3, 14, 13, 54, 9, 0, time.UTC)))
			Expect(full[0].Jobs[0].JaTaskIDs).To(Equal([]int64{116}))

			Expect(full[0].Jobs[3].JaTaskIDs).To(Equal([]int64{119}))
		})

		It("stops before pending section", func() {
			full, err := qstat.ParseQstatFullOutput(qstatF)
			Expect(err).NotTo(HaveOccurred())

			Expect(full[1].Jobs).To(HaveLen(0))
		})

	})

	Context("ParseClusterQueueSummary (qstat -g c)", func() {

		qstatGC := `CLUSTER QUEUE                   CQLOAD   USED    RES  AVAIL  TOTAL aoACDS  cdsuE
--------------------------------------------------------------------------------
all.q                             0.01      4      0     12     16      0      0
test.q                            0.01      0      0     16     16      0      0`

		It("parses cluster queue summary", func() {
			summary, err := qstat.ParseClusterQueueSummary(qstatGC)
			Expect(err).NotTo(HaveOccurred())
			Expect(summary).To(HaveLen(2))

			Expect(summary[0].ClusterQueue).To(Equal("all.q"))
			Expect(summary[0].CQLoad).To(Equal(0.01))
			Expect(summary[0].Used).To(Equal(4))
			Expect(summary[0].Reserved).To(Equal(0))
			Expect(summary[0].Available).To(Equal(12))
			Expect(summary[0].Total).To(Equal(16))

			Expect(summary[1].ClusterQueue).To(Equal("test.q"))
			Expect(summary[1].Used).To(Equal(0))
			Expect(summary[1].Available).To(Equal(16))
		})

	})

	Context("ParseJobArrayTask (qstat -g d)", func() {

		qstatGD := `job-ID     prior   name       user         state submit/start at     queue                          slots ja-task-ID
-----------------------------------------------------------------------------------------------------------------
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:54:53 all.q@gcs-demo                     1 121
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:54:53 all.q@gcs-demo                     1 122
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:55:01 all.q@gcs-demo                     1 123
        23 0.55500 sleep      gcsdemo      r     2026-03-14 13:55:12 all.q@gcs-demo                     1 124
        23 0.55500 sleep      gcsdemo      qw    2026-03-14 13:37:31                                    1 125-1000:1
        24 0.55500 sleep      gcsdemo      qw    2026-03-14 13:37:34                                    1 1-1000:1
        25 0.55500 sleep      gcsdemo      qw    2026-03-14 13:38:11
        28 0.55500 sleep      gcsdemo      qw    2026-03-14 13:41:27`

		It("parses running array tasks", func() {
			tasks, err := qstat.ParseJobArrayTask(qstatGD)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(8))

			Expect(tasks[0].JobID).To(Equal(23))
			Expect(tasks[0].State).To(Equal("r"))
			Expect(tasks[0].Queue).To(Equal("all.q@gcs-demo"))
			Expect(tasks[0].JaTaskIDs).To(Equal([]int64{121}))
			Expect(tasks[0].StartTime.Year()).To(Equal(2026))
		})

		It("parses waiting tasks without queue", func() {
			tasks, err := qstat.ParseJobArrayTask(qstatGD)
			Expect(err).NotTo(HaveOccurred())

			Expect(tasks[6].JobID).To(Equal(25))
			Expect(tasks[6].State).To(Equal("qw"))
			Expect(tasks[6].Queue).To(Equal(""))
			Expect(tasks[6].SubmitTime.Year()).To(Equal(2026))

			Expect(tasks[7].JobID).To(Equal(28))
		})

	})

})
