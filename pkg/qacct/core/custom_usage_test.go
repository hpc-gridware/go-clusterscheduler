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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

// Record of OCS 9.1.6 with usage_patterns=agent:agent_* in reporting_params;
// the epilog of the job added the three agent_ values.
const customUsageJSON = `{"job_number":9599,"task_number":0,"start_time":1794406166743835,"end_time":1794406168842333,"owner":"openshell","group":"openshell","account":"sge","qname":"acct.q","hostname":"aro-master","department":"defaultdepartment","slots":1,"job_name":"acct3","priority":0,"submission_time":1794406166739447,"submit_cmd_line":"qsub -terse -q acct.q -b y -o /dev/null -j y -N acct3 /bin/sleep 2","pe_taskid":"NONE","category":"-U openshell -scope global -hard -q acct.q","failed":0,"exit_status":0,"usage":{"rusage":{"ru_wallclock":2,"ru_utime":0.047717,"ru_stime":0.051579,"ru_maxrss":7644,"ru_ixrss":0,"ru_ismrss":0,"ru_idrss":0,"ru_isrss":0,"ru_minflt":4987,"ru_majflt":0,"ru_nswap":0,"ru_inblock":0,"ru_oublock":8,"ru_msgsnd":0,"ru_msgrcv":0,"ru_nsignals":0,"ru_nvcsw":64,"ru_nivcsw":14},"eusage":{"wallclock":3.009902,"cpu":0.099296,"mem":0.00017642974853515625,"io":0.00015635136514902115,"ioops":334.0,"iow":0.0,"maxvmem":5439488.0,"maxrss":2482176.0},"agent":{"agent_tokens_in":5.0,"agent_cost_usd":0.42,"agent_model":0.0}}}`

// qacct -j output of the same job.
const customUsageText = `==============================================================
qname                              acct.q              
hostname                           aro-master          
group                              openshell           
owner                              openshell           
project                            NONE                
department                         defaultdepartment   
jobname                            acct3               
jobnumber                          9599                
taskid                             undefined
pe_taskid                          NONE
arid                               undefined
account                            sge                 
priority                           0                   
qsub_time                          2026-11-11 14:09:26.739447
submit_cmd_line                    qsub -terse -q acct.q -b y -o /dev/null -j y -N acct3 /bin/sleep 2
start_time                         2026-11-11 14:09:26.743835
end_time                           2026-11-11 14:09:28.842333
granted_pe                         NONE                
slots                              1                   
failed                             0    
exit_status                        0                   
ru_wallclock                       2            
ru_utime                           0.048        
ru_stime                           0.052        
ru_maxrss                          7644                
ru_nvcsw                           64                  
ru_nivcsw                          14                  
wallclock                          3.010        
cpu                                0.099        
mem                                0.000             
io                                 0.000             
ioops                              334          
iow                                0.000             
maxvmem                            5439488           
maxrss                             2482176           
agent_tokens_in                    5.000             
agent_cost_usd                     0.420             
agent_model                        0.000             
`

var _ = Describe("Custom usage (usage_patterns)", func() {

	Context("accounting file (JSON)", func() {

		It("reads the custom usage groups of a 9.1 record", func() {
			job, err := core.ParseAccountingJSONLine(customUsageJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobNumber).To(Equal(int64(9599)))
			Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{
				"agent": {"agent_tokens_in": 5, "agent_cost_usd": 0.42, "agent_model": 0},
			}))
		})

		It("reads the extended usage of a 9.1 record (eusage)", func() {
			job, err := core.ParseAccountingJSONLine(customUsageJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.RUsage.RuMaxrss).To(Equal(int64(7644)))
			Expect(job.JobUsage.Usage.WallClock).To(Equal(3.009902))
			Expect(job.JobUsage.Usage.CPU).To(Equal(0.099296))
			Expect(job.JobUsage.Usage.IOOps).To(Equal(int64(334)))
			Expect(job.JobUsage.Usage.MaxVMem).To(Equal(5439488.0))
			Expect(job.JobUsage.Usage.MaxRSS).To(Equal(2482176.0))
		})

		It("reads the extended usage of a 9.0 record (usage)", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":10,"usage":{"rusage":{"ru_utime":0.355821},"usage":{"wallclock":2.022342,"cpu":0.51713,"maxrss":10530816.0}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.RUsage.RuUtime).To(Equal(0.355821))
			Expect(job.JobUsage.Usage.WallClock).To(Equal(2.022342))
			Expect(job.JobUsage.Usage.MaxRSS).To(Equal(10530816.0))
			Expect(job.JobUsage.Custom).To(BeNil())
		})

		It("keeps several groups apart", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"rusage":{},"eusage":{},"gpu":{"nvidia_energy_consumed":12.5},"agent":{"agent_tool_calls":7}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{
				"gpu":   {"nvidia_energy_consumed": 12.5},
				"agent": {"agent_tool_calls": 7},
			}))
		})

		It("has no custom usage for a record without it or with an empty group", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"rusage":{},"eusage":{},"agent":{}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Custom).To(BeNil())
		})

		It("ignores a usage member that is not a group of numbers", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"rusage":{"ru_nswap":3},"future":"x","other":{"a":"b"},"agent":{"agent_x":1}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.RUsage.RuNswap).To(Equal(int64(3)))
			Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{"agent": {"agent_x": 1}}))
		})

		It("fails on a broken standard group", func() {
			_, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"rusage":{"ru_nswap":"many"}}}`)
			Expect(err).To(MatchError(ContainSubstring("rusage")))
			_, err = core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"eusage":[1]}}`)
			Expect(err).To(MatchError(ContainSubstring("eusage")))
		})

		It("reads back what it marshals", func() {
			job, err := core.ParseAccountingJSONLine(customUsageJSON)
			Expect(err).NotTo(HaveOccurred())
			data, err := json.Marshal(job)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"custom":{"agent":{`))

			var again core.JobDetail
			Expect(json.Unmarshal(data, &again)).To(Succeed())
			Expect(again).To(Equal(job))
		})
	})

	Context("qacct -j output", func() {

		It("collects the values after the standard usage, without a group", func() {
			jobs, err := core.ParseQAcctJobOutput(customUsageText)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].JobUsage.Usage.MaxRSS).To(Equal(2482176.0))
			Expect(jobs[0].JobUsage.Usage.IOOps).To(Equal(int64(334)))
			Expect(jobs[0].JobUsage.Custom).To(Equal(map[string]map[string]float64{
				core.CustomUsageNoGroup: {"agent_tokens_in": 5, "agent_cost_usd": 0.42, "agent_model": 0},
			}))
		})

		It("does not take the fields of the record for custom usage", func() {
			jobs, err := core.ParseQAcctJobOutput("==============================================================\n" +
				"jobnumber                          7\n" +
				"taskid                             undefined\n" +
				"some_new_field                     NONE\n" +
				"exit_status                        0\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].JobUsage.Custom).To(BeNil())
		})

		It("keeps the custom usage of each record of the output", func() {
			jobs, err := core.ParseQAcctJobOutput("==============================================================\n" +
				"jobnumber                          7\n" +
				"agent_tokens_in                    10.000\n" +
				"==============================================================\n" +
				"jobnumber                          8\n" +
				"==============================================================\n" +
				"jobnumber                          9\n" +
				"agent_tokens_in                    30.000\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(3))
			Expect(jobs[0].JobUsage.Custom[core.CustomUsageNoGroup]).To(Equal(map[string]float64{"agent_tokens_in": 10}))
			Expect(jobs[1].JobUsage.Custom).To(BeNil())
			Expect(jobs[2].JobUsage.Custom[core.CustomUsageNoGroup]).To(Equal(map[string]float64{"agent_tokens_in": 30}))
		})
	})

	Context("CustomValue", func() {

		It("finds a value in the accounting file and in the qacct output alike", func() {
			fromFile, err := core.ParseAccountingJSONLine(customUsageJSON)
			Expect(err).NotTo(HaveOccurred())
			fromQacct, err := core.ParseQAcctJobOutput(customUsageText)
			Expect(err).NotTo(HaveOccurred())

			for _, usage := range []core.JobUsage{fromFile.JobUsage, fromQacct[0].JobUsage} {
				v, ok := usage.CustomValue("agent_cost_usd")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal(0.42))
				_, ok = usage.CustomValue("agent_missing")
				Expect(ok).To(BeFalse())
			}
		})

		It("reports nothing for a record without custom usage", func() {
			_, ok := core.JobUsage{}.CustomValue("agent_cost_usd")
			Expect(ok).To(BeFalse())
		})
	})
})
