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
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	gojson "github.com/goccy/go-json"
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

// Record of GCS 9.1.7 with usage_patterns=...;gpu:nvidia*|amd*;custom:cst_*;usage:usg_*
// (shortened): custom groups which have the names JobUsage itself uses.
const reservedGroupsJSON = `{"job_number":3,"task_number":0,"usage":{"rusage":{"ru_wallclock":1,"ru_utime":0.058272},"eusage":{"wallclock":2.012916,"cpu":0.06634799999999999,"ioops":102.0,"maxrss":7098368.0},"gpu":{"nvidia_energy_consumed":12.5},"custom":{"cst_a":1.0},"usage":{"usg_a":2.0}}}`

// qacct -j output of GCS 9.1.7 (shortened; exit status changed to 3) with
// execd_params ENABLE_MEM_DETAILS=1 and
// usage_patterns=agent:agent_*;std:io*|exit_status|end_time|maxpss: the
// names the std group selects are printed a second time, with %.3f.
const repeatedNamesText = `==============================================================
jobname                            std                 
jobnumber                          17                  
taskid                             undefined
end_time                           2026-11-12 16:18:19.270440
exit_status                        3                   
ru_wallclock                       2            
wallclock                          3.023        
cpu                                0.067        
mem                                0.015             
io                                 0.000             
ioops                              102          
iow                                0.000             
maxvmem                            3250630656        
maxrss                             7102464           
maxpss                             3207168           
agent_tokens_in                    5.000             
agent_neg                          -1.500            
agent_big                          123456789012345.000
io                                 0.000             
ioops                              102.000           
iow                                0.000             
exit_status                        3.000             
end_time                           1794500299270440.000
maxpss                             3207168.000       
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

		It("fails on a broken standard group", func() {
			_, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"usage":{"cpu":"much"}}}`)
			Expect(err).To(MatchError(ContainSubstring("usage")))
		})

		// main failed the whole record on this: ioops was read into an
		// integer field.
		It("reads a floating point ioops of a 9.0 record", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":10,"usage":{"usage":{"ioops":334.0}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Usage.IOOps).To(Equal(int64(334)))
		})

		It("reads every extended usage value into its field", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"eusage":{"wallclock":1.5,"utime":2.5,"stime":3.5,"cpu":4.5,"mem":5.5,"io":6.5,"ioops":7.0,"iow":8.5,"maxvmem":9.5,"maxrss":10.5,"maxpss":11.5}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Usage).To(Equal(core.Usage{WallClock: 1.5, UserTime: 2.5,
				SystemTime: 3.5, CPU: 4.5, Memory: 5.5, IO: 6.5, IOOps: 7, IOWait: 8.5,
				MaxVMem: 9.5, MaxRSS: 10.5, MaxPSS: 11.5}))
			Expect(job.JobUsage.Custom).To(BeNil())
		})

		It("keeps an ioops outside of the integer range in range", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"eusage":{"ioops":1e30}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Usage.IOOps).To(Equal(int64(9223372036854775807)))
		})

		// As before custom usage was read: a member a later version adds
		// must not cost the record.
		It("ignores a member of the extended usage it has no field for", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"eusage":{"cpu":1.5,"newfield":"x"}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Usage.CPU).To(Equal(1.5))
		})

		It("keeps the numbers of a group which also has other members", func() {
			job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"agent":{"agent_model":"gpt","agent_tokens":5,"agent_none":null}}}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{"agent": {"agent_tokens": 5}}))
		})

		// A custom group named "usage" (GCS 9.1.7 accepts it). Which of the
		// two was taken for the extended usage depended on the map order.
		It("takes eusage for the extended usage when the record also has usage", func() {
			for i := 0; i < 50; i++ {
				job, err := core.ParseAccountingJSONLine(`{"job_number":1,"usage":{"rusage":{},"eusage":{"cpu":1.5},"usage":{"usg_a":2.0}}}`)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.JobUsage.Usage.CPU).To(Equal(1.5))
				Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{"usage": {"usg_a": 2}}))
			}
		})

		It("clears the usage of the value it reads into", func() {
			job := core.JobDetail{}
			Expect(json.Unmarshal([]byte(customUsageJSON), &job)).To(Succeed())
			Expect(json.Unmarshal([]byte(`{"job_number":2,"usage":{"rusage":{},"eusage":{}}}`), &job)).To(Succeed())
			Expect(job.JobUsage).To(Equal(core.JobUsage{}))
		})

		fromText := func() core.JobDetail {
			jobs, err := core.ParseQAcctJobOutput(customUsageText)
			Expect(err).NotTo(HaveOccurred())
			return jobs[0]
		}
		fromLine := func(line string) func() core.JobDetail {
			return func() core.JobDetail {
				job, err := core.ParseAccountingJSONLine(line)
				Expect(err).NotTo(HaveOccurred())
				return job
			}
		}
		type codec struct {
			marshal   func(any) ([]byte, error)
			unmarshal func([]byte, any) error
		}
		std := codec{json.Marshal, json.Unmarshal}
		goccy := codec{gojson.Marshal, gojson.Unmarshal}

		// Consumers put JobDetail on the wire with either package.
		DescribeTable("reads back what it marshals",
			func(source func() core.JobDetail, writer, reader codec) {
				job := source()
				Expect(job.JobUsage.Custom).NotTo(BeEmpty())
				data, err := writer.marshal(job)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(ContainSubstring(`"custom":{`))

				var again core.JobDetail
				Expect(reader.unmarshal(data, &again)).To(Succeed())
				Expect(again).To(Equal(job))
			},
			Entry("file record, encoding/json", fromLine(customUsageJSON), std, std),
			Entry("file record, goccy", fromLine(customUsageJSON), goccy, goccy),
			Entry("file record, encoding/json to goccy", fromLine(customUsageJSON), std, goccy),
			Entry("qacct -j record (no group), encoding/json", fromText, std, std),
			Entry("qacct -j record (no group), goccy", fromText, goccy, goccy),
			Entry("groups named custom and usage, encoding/json", fromLine(reservedGroupsJSON), std, std),
			Entry("groups named custom and usage, goccy", fromLine(reservedGroupsJSON), goccy, goccy),
			Entry("only a group named custom", fromLine(`{"job_number":1,"usage":{"eusage":{},"custom":{"cst_a":1.0}}}`), std, goccy),
		)

		It("reads a group named custom as a group", func() {
			job, err := core.ParseAccountingJSONLine(reservedGroupsJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.JobUsage.Usage.WallClock).To(Equal(2.012916))
			Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{
				"gpu":    {"nvidia_energy_consumed": 12.5},
				"custom": {"cst_a": 1},
				"usage":  {"usg_a": 2},
			}))
		})
	})

	// WatchFile is how accounting files are read; it decodes with goccy.
	Context("WatchFile", func() {

		It("delivers the extended and the custom usage of the records", func() {
			path := filepath.Join(GinkgoT().TempDir(), "accounting.jsonl")
			line90 := `{"job_number":10,"usage":{"rusage":{"ru_utime":0.355821},"usage":{"wallclock":2.022342,"ioops":7.0}}}`
			Expect(os.WriteFile(path, []byte(customUsageJSON+"\n"+line90+"\n"), 0o600)).To(Succeed())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			records, err := core.WatchFile(ctx, path, 2)
			Expect(err).NotTo(HaveOccurred())

			var first, second core.JobDetail
			Eventually(records, "5s").Should(Receive(&first))
			Eventually(records, "5s").Should(Receive(&second))

			expected, err := core.ParseAccountingJSONLine(customUsageJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(first).To(Equal(expected))
			Expect(first.JobUsage.Usage.WallClock).To(Equal(3.009902))
			Expect(first.JobUsage.Usage.IOOps).To(Equal(int64(334)))
			Expect(first.JobUsage.Custom).To(HaveKey("agent"))
			Expect(second.JobUsage.Usage.WallClock).To(Equal(2.022342))
			Expect(second.JobUsage.Usage.IOOps).To(Equal(int64(7)))
			Expect(second.JobUsage.Custom).To(BeNil())
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

		// Before the standard usage of the record ends nothing is custom
		// usage: a field a later qacct adds would otherwise show up here.
		It("does not take a numeric field of the record for custom usage", func() {
			jobs, err := core.ParseQAcctJobOutput("==============================================================\n" +
				"jobnumber                          7\n" +
				"some_new_field                     10\n" +
				"ru_newthing                        5\n" +
				"maxvmem                            21045248\n" +
				"maxrss                             10547200\n" +
				"arid                               4711\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].ArID).To(Equal("4711"))
			Expect(jobs[0].JobUsage.Custom).To(BeNil())
		})

		It("keeps the standard values when a pattern selects their names again", func() {
			jobs, err := core.ParseQAcctJobOutput(repeatedNamesText)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			job := jobs[0]
			Expect(job.EndTime).To(Equal(int64(1794500299270440)))
			Expect(job.ExitStatus).To(Equal(int64(3)))
			Expect(job.JobUsage.Usage.IOOps).To(Equal(int64(102)))
			Expect(job.JobUsage.Usage.MaxRSS).To(Equal(7102464.0))
			Expect(job.JobUsage.Usage.MaxPSS).To(Equal(3207168.0))
			Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{
				core.CustomUsageNoGroup: {"agent_tokens_in": 5, "agent_neg": -1.5,
					"agent_big": 123456789012345, "io": 0, "ioops": 102, "iow": 0,
					"exit_status": 3, "end_time": 1794500299270440, "maxpss": 3207168},
			}))
		})

		It("reads maxpss as standard usage", func() {
			jobs, err := core.ParseQAcctJobOutput("==============================================================\n" +
				"jobnumber                          7\n" +
				"maxrss                             7114752\n" +
				"maxpss                             3207168\n" +
				"nvidia_energy_consumed             12.500\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].JobUsage.Usage.MaxPSS).To(Equal(3207168.0))
			Expect(jobs[0].JobUsage.Custom).To(Equal(map[string]map[string]float64{
				core.CustomUsageNoGroup: {"nvidia_energy_consumed": 12.5},
			}))
		})

		// NaN and Inf cannot be marshalled to JSON: one such value would
		// cost a consumer the whole listing.
		It("takes only plain decimal numbers for custom usage", func() {
			jobs, err := core.ParseQAcctJobOutput("==============================================================\n" +
				"jobnumber                          7\n" +
				"maxrss                             100\n" +
				"agent_nan                          nan\n" +
				"agent_inf                          inf\n" +
				"agent_ninf                         -inf\n" +
				"agent_hex                          0x10\n" +
				"agent_text                         gpt\n" +
				"agent_ok                           -2.500\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs[0].JobUsage.Custom).To(Equal(map[string]map[string]float64{
				core.CustomUsageNoGroup: {"agent_ok": -2.5},
			}))
			_, err = json.Marshal(jobs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("keeps the custom usage of each record of the output", func() {
			jobs, err := core.ParseQAcctJobOutput("==============================================================\n" +
				"jobnumber                          7\n" +
				"maxrss                             100\n" +
				"agent_tokens_in                    10.000\n" +
				"==============================================================\n" +
				"jobnumber                          8\n" +
				"==============================================================\n" +
				"jobnumber                          9\n" +
				"maxrss                             100\n" +
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

		It("prefers the value without a group, then the first group by name", func() {
			usage := core.JobUsage{Custom: map[string]map[string]float64{
				"power": {"energy": 2, "watts": 7},
				"gpu":   {"energy": 1},
				"zeta":  {"watts": 8},
			}}
			for i := 0; i < 50; i++ {
				v, ok := usage.CustomValue("energy")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal(1.0))
				v, _ = usage.CustomValue("watts")
				Expect(v).To(Equal(7.0))
			}
			usage.Custom[core.CustomUsageNoGroup] = map[string]float64{"energy": 3}
			v, _ := usage.CustomValue("energy")
			Expect(v).To(Equal(3.0))
		})

		It("reports nothing for a record without custom usage", func() {
			_, ok := core.JobUsage{}.CustomValue("agent_cost_usd")
			Expect(ok).To(BeFalse())
		})
	})
})
