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

// Integration specs for the streaming qstat client against a live cluster
// (the dev container, see the Makefile; run with OCS_VERSION=9.0.x and
// 9.1.x). They skip without a cluster and delete the jobs they submit.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// longJobName is longer than the 10 characters of qstat's text column and
// contains spaces, the two reasons the streaming client reads XML.
const longJobName = "stream integration job with a long name"

var submittedJobID = regexp.MustCompile(`Your job (\d+) `)

// streamClusterAvailable reports whether a live cluster is reachable. The
// probe "qstat -f" is read-only.
func streamClusterAvailable() bool {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return false
	}
	for _, binary := range []string{"qstat", "qsub", "qdel"} {
		if _, err := exec.LookPath(binary); err != nil {
			return false
		}
	}
	// A qmaster host that resolves but does not answer would otherwise
	// hold every BeforeEach for qstat's own connect timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "qstat", "-f").Run() == nil
}

// submitJob runs qsub with args and returns the job number. The job is
// deleted when the spec ends.
func submitJob(args ...string) uint64 {
	out, err := exec.Command("qsub", args...).CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(out))
	m := submittedJobID.FindStringSubmatch(string(out))
	Expect(m).NotTo(BeNil(), string(out))
	id, err := strconv.ParseUint(m[1], 10, 64)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		// Not fatal (the job may be gone already), but a refused delete
		// leaves a "sleep 300" behind, so say so.
		if out, err := exec.Command("qdel", "-f", m[1]).CombinedOutput(); err != nil {
			GinkgoWriter.Printf("qdel -f %s: %v: %s\n", m[1], err, out)
		}
	})
	return id
}

var _ = Describe("Streaming qstat against a live cluster", Label("integration"), func() {
	var impl *qstat.QStatImpl

	BeforeEach(func() {
		if !streamClusterAvailable() {
			Skip("no cluster available")
		}
		var err error
		impl, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
		Expect(err).NotTo(HaveOccurred())
	})

	findJob := func(ctx context.Context, opts qstat.JobStreamOptions, id uint64) (qstat.JobXML, qstat.Stats, error) {
		var found qstat.JobXML
		stats, err := impl.StreamJobsXML(ctx, nil, opts, func(j qstat.JobXML) error {
			if j.JobNumber == id {
				found = j
			}
			return nil
		})
		return found, stats, err
	}

	It("streams a running and a held job with their full names", func() {
		running := submitJob("-b", "y", "-N", longJobName, "sleep", "300")
		held := submitJob("-b", "y", "-h", "-N", longJobName+" held", "sleep", "300")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		// A dispatched job is listed as running in state "t" (transferring)
		// until the execd has started it, so wait for "r" itself.
		var job qstat.JobXML
		Eventually(func(g Gomega) {
			var stats qstat.Stats
			var err error
			job, stats, err = findJob(ctx, qstat.JobStreamOptions{}, running)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stats.Complete).To(BeTrue())
			g.Expect(job.ListState).To(Equal("running"))
			g.Expect(job.State).To(Equal("r"))
		}).WithTimeout(45 * time.Second).WithPolling(time.Second).Should(Succeed())

		Expect(job.Name).To(Equal(longJobName))
		Expect(job.QueueName).NotTo(BeEmpty())
		start, err := qstat.ParseXMLTime(job.StartTime)
		Expect(err).NotTo(HaveOccurred())
		// A start time in the wrong zone is hours off in either direction.
		Expect(time.Since(start)).To(BeNumerically("~", 0, 10*time.Minute), "the start time is in the local zone")

		job, _, err = findJob(ctx, qstat.JobStreamOptions{ExtraArgs: []string{"-s", "p"}}, held)
		Expect(err).NotTo(HaveOccurred())
		Expect(job.Name).To(Equal(longJobName + " held"))
		Expect(job.State).To(Equal("hqw"))
		Expect(job.SubmissionTime).NotTo(BeEmpty())

		job, _, err = findJob(ctx, qstat.JobStreamOptions{Extended: true}, held)
		Expect(err).NotTo(HaveOccurred())
		Expect(job.Department).NotTo(BeEmpty())
	})

	It("streams the queue instances", func() {
		// Collect first, assert afterwards: a failing Expect inside the
		// callback would panic out of the stream with qstat still running.
		var names []string
		total := uint32(0)
		stats, err := impl.StreamQueueInstancesXML(context.Background(), nil, func(q qstat.QueueInstanceXML) error {
			names = append(names, q.Name)
			total += q.SlotsTotal
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		Expect(stats.Decoded).To(BeNumerically(">", 0))
		Expect(names).To(HaveEach(ContainSubstring("@")))
		Expect(total).To(BeNumerically(">", 0))
	})

	It("reads a job with qstat -j and maps an unknown job to ErrNoSuchJob", func() {
		held := submitJob("-b", "y", "-h", "-N", longJobName, "sleep", "300")
		env := append(os.Environ(), "LC_ALL=C")

		jobs, err := impl.SchedulerJobInfoContext(context.Background(), env, strconv.FormatUint(held, 10))
		Expect(err).NotTo(HaveOccurred())
		Expect(jobs).To(HaveLen(1))
		Expect(jobs[0].JobName).To(Equal(longJobName))

		_, err = impl.SchedulerJobInfoContext(context.Background(), env, "99999999")
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
	})
})
