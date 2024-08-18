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

type mockQacctCli struct {
	mockOutput string
	mockErr    error
}

func (m *mockQacctCli) RunCommand(args ...string) (string, error) {
	return m.mockOutput, m.mockErr
}

var _ = Describe("Qacct", func() {

	var (
		qacctClient qacct.QAcct
	)

	mockQacctCli := &mockQacctCli{}

	BeforeEach(func() {
		// Use the real qacct client but override the RunCommand method with the mock
		var err error
		qacctClient, err = qacct.NewCommandLineQAcct("qacct")
		Expect(err).To(BeNil())
		qacctClient.(*qacct.CommandLineQAcct).WithRunCommand(mockQacctCli.RunCommand)
	})

	Context("ListAdvanceReservations", func() {
		// @TODO CS-486
		It("should list advance reservations correctly", func() {
			mockQacctCli.mockOutput = "AR       WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW\n" +
				"========================================================================================================================\n" +
				"    1             0         0.667         0.400         1.067              0.046              0.000              0.000\n"

			reservations, err := qacctClient.ListAdvanceReservations("1")
			Expect(err).To(BeNil())
			Expect(reservations).To(HaveLen(1))
			Expect(reservations[0].ArID).To(Equal("1"))
			Expect(reservations[0].Usage.WallClock).To(Equal(float64(0.0)))
			Expect(reservations[0].Usage.UserTime).To(Equal(float64(0.667)))
			Expect(reservations[0].Usage.SystemTime).To(Equal(float64(0.400)))
			Expect(reservations[0].Usage.CPU).To(Equal(float64(1.067)))
			Expect(reservations[0].Usage.Memory).To(Equal(0.046))
			Expect(reservations[0].Usage.IO).To(Equal(float64(0.000)))
			Expect(reservations[0].Usage.IOWait).To(Equal(float64(0.000)))
		})
	})

	Context("JobsAccountedTo", func() {
		It("should list jobs accounted to a specific account", func() {
			mockQacctCli.mockOutput = `Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
          132         4.520         4.112         8.632              1.375              0.000              0.000\n
`

			usage, err := qacctClient.JobsAccountedTo("account123")
			Expect(err).To(BeNil())
			Expect(usage.CPU).To(Equal(8.632))
		})
	})

	Context("JobsStartedAfter", func() {
		It("should list jobs started after a specific time", func() {
			mockQacctCli.mockOutput = `Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
          132         4.520         4.112         8.632              1.375              0.000              0.000`

			usage, err := qacctClient.JobsStartedAfter("202208160000")
			Expect(err).To(BeNil())
			Expect(usage.Memory).To(Equal(1.375))
		})
	})

	Context("JobsStartedLastDays", func() {
		It("should list jobs started in the last N days", func() {
			mockQacctCli.mockOutput = `Total System Usage
    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
================================================================================================================
            9         4.544         4.178         8.721              1.374              0.000              0.000`
			usage, err := qacctClient.JobsStartedLastDays(1)
			Expect(err).To(BeNil())
			Expect(usage.UserTime).To(Equal(4.544))
		})
	})

	Context("ShowJobDetails", func() {
		It("should show job details correctly", func() {
			mockQacctCli.mockOutput = `==============================================================
qname                              all.q
hostname                           master
group                              root
owner                              root
project                            NONE
department                         defaultdepartment
jobname                            memhog
jobnumber                          22
taskid                             undefined
pe_taskid                          NONE
account                            accountingstring
priority                           0
qsub_time                          2024-08-18 08:13:15.547136
submit_cmd_line                    qsub -b y -A accountingstring memhog 1g
start_time                         2024-08-18 08:13:15.995298
end_time                           2024-08-18 08:13:16.551426
granted_pe                         NONE
slots                              1
failed                             0
exit_status                        0
ru_wallclock                       0
ru_utime                           0.336
ru_stime                           0.202
ru_maxrss                          1050068
ru_ixrss                           0
ru_ismrss                          0
ru_idrss                           0
ru_isrss                           0
ru_minflt                          1051
ru_majflt                          0
ru_nswap                           0
ru_inblock                         0
ru_oublock                         24
ru_msgsnd                          0
ru_msgrcv                          0
ru_nsignals                        0
ru_nvcsw                           200
ru_nivcsw                          0
wallclock                          1.004
cpu                                0.539
mem                                0.000
io                                 0.000
iow                                0.000
maxvmem                            0
maxrss                             0
arid                               undefined`

			detail, err := qacctClient.ShowJobDetails(123)
			Expect(err).To(BeNil())
			Expect(detail.QName).To(Equal("all.q"))
			Expect(detail.HostName).To(Equal("master"))
			Expect(detail.JobNumber).To(Equal(int64(22)))
			Expect(detail.RuUTime).To(Equal(0.336))
			Expect(detail.WallClock).To(Equal(1.004))
		})
	})

	Context("ListDepartment", func() {
		It("should list department usage correctly", func() {
			mockQacctCli.mockOutput = "DEPARTMENT WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"dept 1234 100 50 150 200GB 50 10\n"

			departments, err := qacctClient.ListDepartment("dept")
			Expect(err).To(BeNil())
			Expect(departments).To(HaveLen(1))
			Expect(departments[0].Department).To(Equal("dept"))
			Expect(departments[0].Usage.WallClock).To(Equal(1234.0))
		})
	})

	Context("ListHost", func() {
		It("should list host usage correctly", func() {
			mockQacctCli.mockOutput = "HOST WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"host1 1234 100 50 150 200GB 50 10\n"

			hosts, err := qacctClient.ListHost("host1")
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(1))
			Expect(hosts[0].HostName).To(Equal("host1"))
			Expect(hosts[0].Usage.WallClock).To(Equal(1234.0))
		})
	})

	Context("ListParallelEnvironment", func() {
		It("should list parallel environment usage correctly", func() {
			mockQacctCli.mockOutput = "PE WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"pe1 1234 100 50 150 200GB 50 10\n"

			peUsages, err := qacctClient.ListParallelEnvironment("pe1")
			Expect(err).To(BeNil())
			Expect(peUsages).To(HaveLen(1))
			Expect(peUsages[0].Pename).To(Equal("pe1"))
			Expect(peUsages[0].Usage.WallClock).To(Equal(1234.0))
		})
	})

	Context("ListProject", func() {
		It("should list project usage correctly", func() {
			mockQacctCli.mockOutput = "PROJECT WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"project1 1234 100 50 150 200GB 50 10\n"

			projects, err := qacctClient.ListProject("project1")
			Expect(err).To(BeNil())
			Expect(projects).To(HaveLen(1))
			Expect(projects[0].ProjectName).To(Equal("project1"))
			Expect(projects[0].Usage.WallClock).To(Equal(1234.0))
		})
	})

	Context("ListQueue", func() {
		It("should list queue usage correctly", func() {
			mockQacctCli.mockOutput = "HOST CLUSTER QUEUE WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"host1 queue1 1234 100 50 150 200GB 50 10\n"

			queueUsages, err := qacctClient.ListQueue("queue1")
			Expect(err).To(BeNil())
			Expect(queueUsages).To(HaveLen(1))
			Expect(queueUsages[0].QueueName).To(Equal("queue1"))
			Expect(queueUsages[0].Usage.WallClock).To(Equal(1234.0))
		})
	})

	Context("ListOwner", func() {
		It("should list owner usage correctly", func() {
			mockQacctCli.mockOutput = "OWNER WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"root 1234 100 50 150 200GB 50 10\n"

			owners, err := qacctClient.ListOwner("root")
			Expect(err).To(BeNil())
			Expect(owners).To(HaveLen(1))
			Expect(owners[0].OwnerName).To(Equal("root"))
			Expect(owners[0].Usage.WallClock).To(Equal(1234.0))
		})
	})

	Context("ShowHelp", func() {
		It("should show help text", func() {
			mockQacctCli.mockOutput = "usage: qacct [options]"
			help, err := qacctClient.ShowHelp()
			Expect(err).To(BeNil())
			Expect(help).To(ContainSubstring("usage: qacct"))
		})
	})

	Context("ShowTotalSystemUsage", func() {
		It("should show total system usage", func() {
			mockQacctCli.mockOutput = "WALLCLOCK UTIME STIME CPU MEMORY IO IOW\n" +
				"=================\n" +
				"1234 100 50 150 200GB 50 10\n"
			usage, err := qacctClient.ShowTotalSystemUsage()
			Expect(err).To(BeNil())
			Expect(usage.WallClock).To(Equal(1234.0))
		})
	})
})
