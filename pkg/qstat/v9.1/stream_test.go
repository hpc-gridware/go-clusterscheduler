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

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeStreamer answers RunContext from canned output, so the v9.1 parser can
// be exercised on live 9.1.7 captures without a cluster.
type fakeStreamer struct {
	stdout, stderr string
	err            error
	args           []string
}

func (f *fakeStreamer) RunContext(_ context.Context, _ []string, args ...string) (string, string, error) {
	f.args = args
	return f.stdout, f.stderr, f.err
}

func (f *fakeStreamer) StreamJobsXML(context.Context, []string, qstat.JobStreamOptions, func(qstat.JobXML) error) (qstat.Stats, error) {
	return qstat.Stats{}, nil
}

func (f *fakeStreamer) StreamQueueInstancesXML(context.Context, []string, func(qstat.QueueInstanceXML) error) (qstat.Stats, error) {
	return qstat.Stats{}, nil
}

func readFixture(name string) string {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())
	return string(b)
}

var _ = Describe("SchedulerJobInfoContext (9.1 parser over the shared runner)", func() {
	It("parses a held job with a hostile name from a 9.1.7 capture", func() {
		s := &fakeStreamer{stdout: readFixture("qstat-j-held-9.1.7.txt")}
		jobs, err := qstat.SchedulerJobInfoContext(context.Background(), nil, s, "2")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.args).To(Equal([]string{"-j", "2"}))
		Expect(jobs).To(HaveLen(1))
		j := jobs[0]
		Expect(j.JobNumber).To(Equal(2))
		Expect(j.JobName).To(Equal("held <img src=x onerror=alert(1)> & co"))
		Expect(j.Owner).To(Equal("root"))
		Expect(j.Department).To(Equal("defaultdepartment"))
		Expect(j.PendingTasks).To(Equal(1))
		Expect(j.SchedulingInfo).To(ContainSubstring("Collecting of scheduler job information is turned off"))
		Expect(j.SubmissionTime).To(Equal("2026-10-22 13:16:41.669137"))
	})

	It("parses the running tasks of an array job", func() {
		s := &fakeStreamer{stdout: readFixture("qstat-j-array-head-9.1.7.txt")}
		jobs, err := qstat.SchedulerJobInfoContext(context.Background(), nil, s, "3")
		Expect(err).NotTo(HaveOccurred())
		Expect(jobs).To(HaveLen(1))
		Expect(jobs[0].JobArrayTasks).To(Equal("1-100000:3"))
		Expect(jobs[0].PendingTasks).To(Equal(31334))
		Expect(jobs[0].Tasks).NotTo(BeEmpty())
		Expect(jobs[0].Tasks[0].TaskID).To(Equal(1))
		Expect(jobs[0].Tasks[0].State).To(Equal("r"))
		Expect(jobs[0].Tasks[0].ExecHostList).To(HaveLen(1))
		Expect(jobs[0].Tasks[0].ExecHostList[0].Hostname).To(Equal("sim117"))
	})

	It("reads the requested start time and the hard requests of a 9.1.7 capture", func() {
		s := &fakeStreamer{stdout: readFixture("qstat-j-future-9.1.7.txt")}
		jobs, err := qstat.SchedulerJobInfoContext(context.Background(), nil, s, "5")
		Expect(err).NotTo(HaveOccurred())
		Expect(jobs[0].ExecutionTime).To(Equal("2030-01-01 00:00:00.000000"))
		Expect(jobs[0].HardResourceList).To(Equal("h_rt=10"))
		Expect(jobs[0].ARID).To(Equal(0))
	})

	It("reads the advance reservation and per-task error reasons", func() {
		out := "==============================================================\n" +
			"job_number:                      9\n" +
			"owner:                           alice\n" +
			"ar_id:                           19\n" +
			"job_state                   1:   Eqw\n" +
			"error reason          1:   10/22/2026 13:20:01 [0:42]: error: can't chdir to /scratch/x: No such file or directory\n" +
			"error reason          1:   second line\n"
		s := &fakeStreamer{stdout: out}
		jobs, err := qstat.SchedulerJobInfoContext(context.Background(), nil, s, "9")
		Expect(err).NotTo(HaveOccurred())
		Expect(jobs[0].ARID).To(Equal(19))
		Expect(jobs[0].Tasks).To(HaveLen(1))
		Expect(jobs[0].Tasks[0].State).To(Equal("Eqw"))
		Expect(jobs[0].Tasks[0].ErrorReasons).To(Equal([]string{
			"10/22/2026 13:20:01 [0:42]: error: can't chdir to /scratch/x: No such file or directory",
			"second line",
		}))
	})

	It("maps an unknown job to ErrNoSuchJob", func() {
		s := &fakeStreamer{stderr: "Following jobs do not exist: \n99999999\n", err: errors.New("failed to run command (Following jobs do not exist): exit status 1")}
		_, err := qstat.SchedulerJobInfoContext(context.Background(), nil, s, "99999999")
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue())
	})

	It("passes other failures through unchanged", func() {
		boom := errors.New("failed to run command (error: unable to send message to qmaster): exit status 1")
		s := &fakeStreamer{stderr: "error: unable to send message to qmaster", err: boom}
		_, err := qstat.SchedulerJobInfoContext(context.Background(), nil, s, "1")
		Expect(errors.Is(err, boom)).To(BeTrue())
	})
})
