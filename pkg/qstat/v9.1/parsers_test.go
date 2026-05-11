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

		It("parses per-task details with structured exec_host_list", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())

			j := jobs[0]
			Expect(j.Tasks).To(HaveLen(4))

			t0 := j.Tasks[0]
			Expect(t0.TaskID).To(Equal(49))
			Expect(t0.State).To(Equal("r"))
			Expect(t0.QueueList).To(Equal("all.q@gcs-demo=1"))
			Expect(t0.StartTime).To(Equal("2026-03-14 13:45:31.248077"))
			Expect(t0.BindingList).To(Equal("NONE"))
			Expect(t0.ResourceMap).To(Equal("NONE"))

			// exec_host_list is now a structured slice
			Expect(t0.ExecHostList).To(HaveLen(1))
			Expect(t0.ExecHostList[0].Hostname).To(Equal("gcs-demo"))
			Expect(t0.ExecHostList[0].Slots).To(Equal(1))

			Expect(j.Tasks[1].TaskID).To(Equal(50))
			Expect(j.Tasks[1].State).To(Equal("r"))

			Expect(j.Tasks[2].TaskID).To(Equal(51))
			Expect(j.Tasks[3].TaskID).To(Equal(52))
		})

		It("parses per-task usage into structured TaskUsageDetail", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())

			u := jobs[0].Tasks[0].Usage
			Expect(u.WallClock).To(Equal("00:00:25.174140"))
			Expect(u.CPU).To(Equal("00:00:00.335190"))
			Expect(u.Mem).To(Equal("0.00000 GBs"))
			Expect(u.IO).To(Equal("0.00000"))
			Expect(u.IOOps).To(Equal("0"))
			Expect(u.IOW).To(Equal("00:00:00.000000"))
			Expect(u.VMem).To(Equal("N/A"))
			Expect(u.MaxVMem).To(Equal("N/A"))
			Expect(u.RSS).To(Equal("460.000K"))
			Expect(u.MaxRSS).To(Equal("3.168M"))
		})

		It("initialises granted_requests, granted_licenses, gpu_usage, cgroups_usage as empty slices", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())

			t0 := jobs[0].Tasks[0]
			Expect(t0.GrantedRequests).NotTo(BeNil())
			Expect(t0.GrantedRequests).To(BeEmpty())
			Expect(t0.GrantedLicenses).NotTo(BeNil())
			Expect(t0.GrantedLicenses).To(BeEmpty())
			Expect(t0.GPUUsage).NotTo(BeNil())
			Expect(t0.GPUUsage).To(BeEmpty())
			Expect(t0.CgroupsUsage).NotTo(BeNil())
			Expect(t0.CgroupsUsage).To(BeEmpty())
		})

		It("parses scheduling info", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJ)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].SchedulingInfo).To(ContainSubstring("Collecting of scheduler job information"))
		})

	})

	Context("ParseSchedulerJobInfo with -ac context vars", func() {

		qstatJWithContext := `==============================================================
job_number:                      2
owner:                           root
job_name:                        ctxtest
priority:                        0
jobshare:                        0
env_list:                        HOSTNAME=master
context:                         tag=alpha,priority=high
binding:                         NONE
job_state                   1:   r
exec_host_list              1:   sim117=1
start_time                  1:   2026-05-11 09:33:55.404696`

		qstatJWithoutContext := `==============================================================
job_number:                      3
owner:                           root
job_name:                        noctx
priority:                        0
jobshare:                        0
env_list:                        HOSTNAME=master
binding:                         NONE
job_state                   1:   r
exec_host_list              1:   sim117=1
start_time                  1:   2026-05-11 09:34:00.000000`

		It("captures the context line verbatim as a comma-separated K=V string", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJWithContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Context).To(Equal("tag=alpha,priority=high"))
		})

		It("leaves Context empty when the context line is absent (no NONE sentinel)", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJWithoutContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Context).To(Equal(""))
		})

	})

	Context("ParseSchedulerJobInfo Gap A fields (cwd, path lists, shell_list, merge, restart, predecessor)", func() {

		qstatJDecorated := `==============================================================
job_number:                      10
owner:                           root
job_name:                        decorated
priority:                        100
jobshare:                        5
cwd:                            /tmp
stderr_path_list:                NONE:NONE:/tmp/e.log
stdout_path_list:                NONE:NONE:/tmp/o.log
stdin_path_list:                 NONE:NONE:/dev/null
shell_list:                      NONE:/bin/bash
merge:                           y
restart:                         y
job_args:                        600
jid_predecessor_list (req):      99999
env_list:                        HOSTNAME=master
binding:                         NONE
job_state                   1:   r
exec_host_list              1:   sim181=1
start_time                  1:   2026-05-11 09:34:00.000000`

		It("captures cwd, the three path lists, and shell_list verbatim", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJDecorated)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			j := jobs[0]
			Expect(j.Cwd).To(Equal("/tmp"))
			Expect(j.StderrPathList).To(Equal("NONE:NONE:/tmp/e.log"))
			Expect(j.StdoutPathList).To(Equal("NONE:NONE:/tmp/o.log"))
			Expect(j.StdinPathList).To(Equal("NONE:NONE:/dev/null"))
			Expect(j.ShellList).To(Equal("NONE:/bin/bash"))
		})

		It("converts merge and restart y/n text to bool at the parser", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJDecorated)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].Merge).To(BeTrue())
			Expect(jobs[0].Restart).To(BeTrue())
		})

		It("captures job_args and jid_predecessor_list (req)", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJDecorated)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].JobArgs).To(Equal("600"))
			Expect(jobs[0].JIDPredecessorListReq).To(Equal("99999"))
		})

		It("leaves Gap A fields empty when their lines are absent", func() {
			minimal := `==============================================================
job_number:                      11
owner:                           root
job_name:                        minimal
priority:                        0
jobshare:                        0
env_list:                        HOSTNAME=master
binding:                         NONE
job_state                   1:   r
exec_host_list              1:   sim181=1
start_time                  1:   2026-05-11 10:00:00.000000`
			jobs, err := qstat.ParseSchedulerJobInfo(minimal)
			Expect(err).NotTo(HaveOccurred())
			j := jobs[0]
			Expect(j.Cwd).To(Equal(""))
			Expect(j.StderrPathList).To(Equal(""))
			Expect(j.StdoutPathList).To(Equal(""))
			Expect(j.StdinPathList).To(Equal(""))
			Expect(j.ShellList).To(Equal(""))
			Expect(j.Merge).To(BeFalse())
			Expect(j.Restart).To(BeFalse())
			Expect(j.JIDPredecessorListReq).To(Equal(""))
		})

	})

	Context("ParseSchedulerJobInfo with task_concurrency and granted_request", func() {

		qstatJArray := `==============================================================
job_number:                      1911580
owner:                           jdoe
job_name:                        my_array
job-array tasks:                 1-1000:1
task_concurrency:                300
job_state                  135:  r
exec_host_list             135:  someserver=8
usage                      135:  wallclock=00:36:41,cpu=03:16:20,mem=117504.22093 GBs,io=12.86688 GB,iow=55.310,ioops=2497350,vmem=20.134G,maxvmem=20.134G,rss=13.945G,maxrss=13.945G,pss=13.462G,smem=617.145M,pmem=13.343G,maxpss=13.462G
granted_request            135:  jobs_counter=1, m_mem_free=6.000G
granted_request            135:  m_mem_free=6.000G
granted_request            135:  m_mem_free=6.000G`

		It("parses task_concurrency at job level", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].TaskConcurrency).To(Equal("300"))
		})

		It("parses exec_host_list with multiple slots on one host", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJArray)
			Expect(err).NotTo(HaveOccurred())

			hosts := jobs[0].Tasks[0].ExecHostList
			Expect(hosts).To(HaveLen(1))
			Expect(hosts[0].Hostname).To(Equal("someserver"))
			Expect(hosts[0].Slots).To(Equal(8))
		})

		It("parses extended usage fields including pss, smem, pmem, maxpss", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJArray)
			Expect(err).NotTo(HaveOccurred())

			u := jobs[0].Tasks[0].Usage
			Expect(u.WallClock).To(Equal("00:36:41"))
			Expect(u.CPU).To(Equal("03:16:20"))
			Expect(u.Mem).To(Equal("117504.22093 GBs"))
			Expect(u.IO).To(Equal("12.86688 GB"))
			Expect(u.IOW).To(Equal("55.310"))
			Expect(u.IOOps).To(Equal("2497350"))
			Expect(u.VMem).To(Equal("20.134G"))
			Expect(u.MaxVMem).To(Equal("20.134G"))
			Expect(u.RSS).To(Equal("13.945G"))
			Expect(u.MaxRSS).To(Equal("13.945G"))
			Expect(u.PSS).To(Equal("13.462G"))
			Expect(u.SMem).To(Equal("617.145M"))
			Expect(u.PMem).To(Equal("13.343G"))
			Expect(u.MaxPSS).To(Equal("13.462G"))
		})

		It("parses granted_request lines as auto-incrementing ptg_id entries", func() {
			jobs, err := qstat.ParseSchedulerJobInfo(qstatJArray)
			Expect(err).NotTo(HaveOccurred())

			reqs := jobs[0].Tasks[0].GrantedRequests
			Expect(reqs).To(HaveLen(3))

			Expect(reqs[0].PTGID).To(Equal(0))
			Expect(reqs[0].GrantedReq).To(Equal("jobs_counter=1, m_mem_free=6.000G"))

			Expect(reqs[1].PTGID).To(Equal(1))
			Expect(reqs[1].GrantedReq).To(Equal("m_mem_free=6.000G"))

			Expect(reqs[2].PTGID).To(Equal(2))
		})

	})

	Context("parseExecHostList (via ParseSchedulerJobInfo)", func() {

		It("parses multiple hosts", func() {
			input := `==============================================================
job_number:                      9
exec_host_list             1:    host1=4,host2=8`
			jobs, err := qstat.ParseSchedulerJobInfo(input)
			Expect(err).NotTo(HaveOccurred())

			hosts := jobs[0].Tasks[0].ExecHostList
			Expect(hosts).To(HaveLen(2))
			Expect(hosts[0].Hostname).To(Equal("host1"))
			Expect(hosts[0].Slots).To(Equal(4))
			Expect(hosts[1].Hostname).To(Equal("host2"))
			Expect(hosts[1].Slots).To(Equal(8))
		})

		It("returns empty slice for NONE", func() {
			input := `==============================================================
job_number:                      9
exec_host_list             1:    NONE`
			jobs, err := qstat.ParseSchedulerJobInfo(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].Tasks[0].ExecHostList).To(BeEmpty())
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

	Context("ParseExtendedJobInfo with array jobs and task ranges", func() {

		qstatExtArray := `job-ID     prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
         4 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 23
         4 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 25
         4 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 27
         4 0.55500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 67-99:2
         5 0.55500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 1-99:2`

		It("parses running array tasks with NA cpu/mem/io", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(5))

			Expect(jobs[0].JobID).To(Equal(4))
			Expect(jobs[0].Priority).To(Equal(0.555))
			Expect(jobs[0].Ntckts).To(Equal(0.5))
			Expect(jobs[0].Name).To(Equal("sleep"))
			Expect(jobs[0].User).To(Equal("root"))
			Expect(jobs[0].Project).To(Equal("NA"))
			Expect(jobs[0].Department).To(Equal("defaultdep"))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].CPU).To(Equal("NA"))
			Expect(jobs[0].Memory).To(Equal(0.0))
			Expect(jobs[0].IO).To(Equal(0.0))
			Expect(jobs[0].Queue).To(Equal("all.q@sim7"))
			Expect(jobs[0].Slots).To(Equal(1))
			Expect(jobs[0].JATaskID).To(Equal("23"))
		})

		It("parses running tasks on different hosts", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtArray)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[1].Queue).To(Equal("all.q@sim1"))
			Expect(jobs[1].JATaskID).To(Equal("25"))

			Expect(jobs[2].Queue).To(Equal("all.q@sim6"))
			Expect(jobs[2].JATaskID).To(Equal("27"))
		})

		It("parses waiting array tasks with step ranges", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtArray)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[3].JobID).To(Equal(4))
			Expect(jobs[3].State).To(Equal("qw"))
			Expect(jobs[3].CPU).To(Equal(""))
			Expect(jobs[3].Queue).To(Equal(""))
			Expect(jobs[3].Slots).To(Equal(1))
			Expect(jobs[3].JATaskID).To(Equal("67-99:2"))

			Expect(jobs[4].JobID).To(Equal(5))
			Expect(jobs[4].State).To(Equal("qw"))
			Expect(jobs[4].JATaskID).To(Equal("1-99:2"))
		})

	})

	Context("ParseExtendedJobInfo with PE array jobs", func() {

		qstatExtPE := `job-ID     prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
         6 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 97
         6 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 99
         7 0.60500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                    10 1,51`

		It("parses running PE array tasks", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtPE)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(3))

			Expect(jobs[0].JobID).To(Equal(6))
			Expect(jobs[0].Priority).To(Equal(0.505))
			Expect(jobs[0].Queue).To(Equal("all.q@sim8"))
			Expect(jobs[0].Slots).To(Equal(1))
			Expect(jobs[0].JATaskID).To(Equal("97"))

			Expect(jobs[1].Queue).To(Equal("all.q@sim3"))
			Expect(jobs[1].JATaskID).To(Equal("99"))
		})

		It("parses waiting PE job with multi-slot and comma-separated task IDs", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtPE)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[2].JobID).To(Equal(7))
			Expect(jobs[2].Priority).To(Equal(0.605))
			Expect(jobs[2].State).To(Equal("qw"))
			Expect(jobs[2].Slots).To(Equal(10))
			Expect(jobs[2].JATaskID).To(Equal("1,51"))
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

	Context("ParseJobArrayTask with PE array jobs", func() {

		qstatGDPE := `job-ID     prior   name       user         state submit/start at     queue                          slots ja-task-ID
-----------------------------------------------------------------------------------------------------------------
         7 0.55500 sleep      root         r     2026-03-30 15:10:08 all.q@sim3                        10 1
         7 0.55500 sleep      root         r     2026-03-30 15:10:08 all.q@sim6                        10 51`

		It("parses running PE array tasks with multi-slot", func() {
			tasks, err := qstat.ParseJobArrayTask(qstatGDPE)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(2))

			Expect(tasks[0].JobID).To(Equal(7))
			Expect(tasks[0].Priority).To(Equal(0.555))
			Expect(tasks[0].Name).To(Equal("sleep"))
			Expect(tasks[0].User).To(Equal("root"))
			Expect(tasks[0].State).To(Equal("r"))
			Expect(tasks[0].Queue).To(Equal("all.q@sim3"))
			Expect(tasks[0].Slots).To(Equal(10))
			Expect(tasks[0].JaTaskIDs).To(Equal([]int64{1}))
			Expect(tasks[0].StartTime).To(Equal(
				time.Date(2026, 3, 30, 15, 10, 8, 0, time.UTC)))

			Expect(tasks[1].Queue).To(Equal("all.q@sim6"))
			Expect(tasks[1].Slots).To(Equal(10))
			Expect(tasks[1].JaTaskIDs).To(Equal([]int64{51}))
		})

	})

	Context("parseJaTaskIDs", func() {

		qstatGDComma := `job-ID     prior   name       user         state submit/start at     queue                          slots ja-task-ID
-----------------------------------------------------------------------------------------------------------------
         7 0.60500 sleep      root         qw    2026-03-30 15:09:00                                   10 1,51`

		It("handles comma-separated task IDs via ParseJobArrayTask", func() {
			tasks, err := qstat.ParseJobArrayTask(qstatGDComma)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))

			Expect(tasks[0].JobID).To(Equal(7))
			Expect(tasks[0].Slots).To(Equal(10))
			Expect(tasks[0].JaTaskIDs).To(Equal([]int64{1, 51}))
		})

	})

	Context("ParseExtendedJobInfo with task concurrency", func() {

		qstatExtTC := `job-ID     prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
         8 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 1
         8 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 2
         8 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 3
         8 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 10
         8 0.00000 0.00000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 11-100:1`

		It("parses running tasks at scheduled priority", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtTC)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(5))

			Expect(jobs[0].JobID).To(Equal(8))
			Expect(jobs[0].Priority).To(Equal(0.555))
			Expect(jobs[0].Ntckts).To(Equal(0.5))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].Queue).To(Equal("all.q@sim9"))
			Expect(jobs[0].JATaskID).To(Equal("1"))

			Expect(jobs[3].Queue).To(Equal("all.q@sim2"))
			Expect(jobs[3].JATaskID).To(Equal("10"))
		})

		It("parses waiting tasks at zero priority with task range", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtTC)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[4].JobID).To(Equal(8))
			Expect(jobs[4].Priority).To(Equal(0.0))
			Expect(jobs[4].Ntckts).To(Equal(0.0))
			Expect(jobs[4].State).To(Equal("qw"))
			Expect(jobs[4].Slots).To(Equal(1))
			Expect(jobs[4].JATaskID).To(Equal("11-100:1"))
		})

	})

	Context("ParseExtendedJobInfo with mixed PE and regular jobs", func() {

		qstatExtMixed := `job-ID     prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
        10 0.60500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                      10 1
        10 0.60500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                        10 51
         8 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 1
         8 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 2
         8 0.00000 0.00000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 11-100:`

		It("parses PE jobs with higher priority first", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtMixed)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(5))

			Expect(jobs[0].JobID).To(Equal(10))
			Expect(jobs[0].Priority).To(Equal(0.605))
			Expect(jobs[0].Queue).To(Equal("all.q@master"))
			Expect(jobs[0].Slots).To(Equal(10))
			Expect(jobs[0].JATaskID).To(Equal("1"))

			Expect(jobs[1].JobID).To(Equal(10))
			Expect(jobs[1].Queue).To(Equal("all.q@sim4"))
			Expect(jobs[1].Slots).To(Equal(10))
			Expect(jobs[1].JATaskID).To(Equal("51"))
		})

		It("parses interleaved regular jobs after PE jobs", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtMixed)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[2].JobID).To(Equal(8))
			Expect(jobs[2].Priority).To(Equal(0.505))
			Expect(jobs[2].Slots).To(Equal(1))
			Expect(jobs[2].JATaskID).To(Equal("1"))

			Expect(jobs[3].JobID).To(Equal(8))
			Expect(jobs[3].JATaskID).To(Equal("2"))
		})

		It("parses task range with trailing colon and no step", func() {
			jobs, err := qstat.ParseExtendedJobInfo(qstatExtMixed)
			Expect(err).NotTo(HaveOccurred())

			Expect(jobs[4].JobID).To(Equal(8))
			Expect(jobs[4].State).To(Equal("qw"))
			Expect(jobs[4].JATaskID).To(Equal("11-100:"))
		})

	})

	Context("ParseJobArrayTask with task concurrency", func() {

		qstatGDTC := `job-ID     prior   name       user         state submit/start at     queue                          slots ja-task-ID
-----------------------------------------------------------------------------------------------------------------
         8 0.55500 sleep      root         r     2026-03-30 15:17:43 all.q@sim1                         1 3
         8 0.55500 sleep      root         r     2026-03-30 15:17:43 all.q@sim9                         1 1
         8 0.55500 sleep      root         r     2026-03-30 15:17:43 all.q@sim2                         1 10
         8 0.00000 sleep      root         qw    2026-03-30 15:17:43                                    1 11
         8 0.00000 sleep      root         qw    2026-03-30 15:17:43                                    1 12
         8 0.00000 sleep      root         qw    2026-03-30 15:17:43                                    1 13`

		It("parses running tasks on different hosts", func() {
			tasks, err := qstat.ParseJobArrayTask(qstatGDTC)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(6))

			Expect(tasks[0].JobID).To(Equal(8))
			Expect(tasks[0].State).To(Equal("r"))
			Expect(tasks[0].Queue).To(Equal("all.q@sim1"))
			Expect(tasks[0].Slots).To(Equal(1))
			Expect(tasks[0].JaTaskIDs).To(Equal([]int64{3}))
			Expect(tasks[0].StartTime).To(Equal(
				time.Date(2026, 3, 30, 15, 17, 43, 0, time.UTC)))
		})

		It("parses individual waiting tasks with zero priority", func() {
			tasks, err := qstat.ParseJobArrayTask(qstatGDTC)
			Expect(err).NotTo(HaveOccurred())

			Expect(tasks[3].JobID).To(Equal(8))
			Expect(tasks[3].Priority).To(Equal(0.0))
			Expect(tasks[3].State).To(Equal("qw"))
			Expect(tasks[3].Queue).To(Equal(""))
			Expect(tasks[3].JaTaskIDs).To(Equal([]int64{11}))

			Expect(tasks[4].JaTaskIDs).To(Equal([]int64{12}))
			Expect(tasks[5].JaTaskIDs).To(Equal([]int64{13}))
		})

	})

	Context("ParseQstatFullExtendedOutput (qstat -f -ext)", func() {

		qstatFExt := `queuename                      qtype resv/used/tot. load_avg arch          states
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@master                   BIP   0/2/14         0.22     lx-amd64
         3 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 7
         5 0.60500 0.50000 pe_2_5     root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim1                     BIP   0/1/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim10                    BIP   0/2/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 1
         5 0.60500 0.50000 pe_2_5     root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim11                    BIP   0/2/10         0.22     lx-amd64
         3 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 9
         5 0.60500 0.50000 pe_2_5     root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim12                    BIP   0/1/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 2
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim2                     BIP   0/2/10         0.22     lx-amd64
         3 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 5
         5 0.60500 0.50000 pe_2_5     root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim3                     BIP   0/2/10         0.22     lx-amd64
         2 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 2
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim4                     BIP   0/2/10         0.22     lx-amd64
         3 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 1
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 2
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim5                     BIP   0/2/10         0.22     lx-amd64
         3 0.50500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 3
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 2
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim6                     BIP   0/1/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim7                     BIP   0/1/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 1
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim8                     BIP   0/1/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 2
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
all.q@sim9                     BIP   0/2/10         0.22     lx-amd64
         4 0.60500 0.50000 array_pe_2 root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1 1
         5 0.60500 0.50000 pe_2_5     root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00      1`

		It("parses all 13 queue sections", func() {
			full, err := qstat.ParseQstatFullExtendedOutput(qstatFExt)
			Expect(err).NotTo(HaveOccurred())
			Expect(full).To(HaveLen(13))
		})

		It("parses queue header fields", func() {
			full, err := qstat.ParseQstatFullExtendedOutput(qstatFExt)
			Expect(err).NotTo(HaveOccurred())

			q := full[0]
			Expect(q.QueueName).To(Equal("all.q@master"))
			Expect(q.QueueType).To(Equal("BIP"))
			Expect(q.Reserved).To(Equal(0))
			Expect(q.Used).To(Equal(2))
			Expect(q.Total).To(Equal(14))
			Expect(q.LoadAvg).To(BeNumerically("~", 0.22, 0.001))
			Expect(q.Arch).To(Equal("lx-amd64"))
		})

		It("parses two jobs in all.q@master", func() {
			full, err := qstat.ParseQstatFullExtendedOutput(qstatFExt)
			Expect(err).NotTo(HaveOccurred())

			jobs := full[0].Jobs
			Expect(jobs).To(HaveLen(2))

			Expect(jobs[0].JobID).To(Equal(3))
			Expect(jobs[0].Priority).To(BeNumerically("~", 0.505, 0.0001))
			Expect(jobs[0].Ntckts).To(BeNumerically("~", 0.5, 0.0001))
			Expect(jobs[0].Name).To(Equal("sleep"))
			Expect(jobs[0].User).To(Equal("root"))
			Expect(jobs[0].Project).To(Equal("NA"))
			Expect(jobs[0].Department).To(Equal("defaultdep"))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].CPU).To(Equal("NA"))
			Expect(jobs[0].Queue).To(Equal("all.q@master"))
			Expect(jobs[0].Slots).To(Equal(1))
			Expect(jobs[0].JATaskID).To(Equal("7"))

			Expect(jobs[1].JobID).To(Equal(5))
			Expect(jobs[1].Name).To(Equal("pe_2_5"))
			Expect(jobs[1].JATaskID).To(Equal(""))
		})

		It("parses single job in all.q@sim1 with array task ID 1", func() {
			full, err := qstat.ParseQstatFullExtendedOutput(qstatFExt)
			Expect(err).NotTo(HaveOccurred())

			jobs := full[1].Jobs
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].JobID).To(Equal(4))
			Expect(jobs[0].Name).To(Equal("array_pe_2"))
			Expect(jobs[0].JATaskID).To(Equal("1"))
			Expect(jobs[0].Queue).To(Equal("all.q@sim1"))
		})

		It("parses plain running job with no task ID in all.q@sim3", func() {
			full, err := qstat.ParseQstatFullExtendedOutput(qstatFExt)
			Expect(err).NotTo(HaveOccurred())

			// all.q@sim3 is index 6
			sim3 := full[6]
			Expect(sim3.QueueName).To(Equal("all.q@sim3"))
			Expect(sim3.Jobs).To(HaveLen(2))
			Expect(sim3.Jobs[0].JobID).To(Equal(2))
			Expect(sim3.Jobs[0].JATaskID).To(Equal(""))
			Expect(sim3.Jobs[1].JATaskID).To(Equal("2"))
		})

		It("parses the last queue section without a trailing separator", func() {
			full, err := qstat.ParseQstatFullExtendedOutput(qstatFExt)
			Expect(err).NotTo(HaveOccurred())

			last := full[len(full)-1]
			Expect(last.QueueName).To(Equal("all.q@sim9"))
			Expect(last.Jobs).To(HaveLen(2))
			Expect(last.Jobs[0].JATaskID).To(Equal("1"))
			Expect(last.Jobs[1].JATaskID).To(Equal(""))
		})

	})

})
