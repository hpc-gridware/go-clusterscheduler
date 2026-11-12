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

// Integration spec for custom usage (usage_patterns). It needs a live
// cluster (the dev container, see the Makefile) and skips otherwise. Every
// change it makes is undone: the usage_patterns group it adds to the global
// configuration and the epilog of all.q.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// clusterAvailable reports whether a live cluster is reachable. The probe
// "qconf -sconf global" is read-only.
func clusterAvailable() bool {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return false
	}
	for _, binary := range []string{"qconf", "qsub", "qacct"} {
		if _, err := exec.LookPath(binary); err != nil {
			return false
		}
	}
	return exec.Command("qconf", "-sconf", "global").Run() == nil
}

func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "%s %v: %s", name, args, out)
	return strings.TrimSpace(string(out))
}

// accountingRecord returns the record of the job from the accounting file.
func accountingRecord(jobID int64) (core.JobDetail, error) {
	file, err := os.Open(core.GetDefaultQacctFile())
	if err != nil {
		return core.JobDetail{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		job, err := core.ParseAccountingJSONLine(scanner.Text())
		if err != nil {
			return core.JobDetail{}, err
		}
		if job.JobNumber == jobID {
			return job, nil
		}
	}
	return core.JobDetail{}, fmt.Errorf("job %d is not in the accounting file yet", jobID)
}

var _ = Describe("Custom usage of a job", Ordered, Label("integration"), func() {
	const (
		queue = "all.q"
		group = "gotest"
	)
	var jobID int64

	BeforeAll(func() {
		if !clusterAvailable() {
			Skip("no reachable cluster; run inside the dev container (make run)")
		}
		qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{})
		Expect(err).NotTo(HaveOccurred())

		// The epilog reports the custom usage: it appends name=value lines
		// to the usage file of the job. One value is not selected by the
		// patterns, one is not a number.
		dir, err := os.MkdirTemp("", "gotest-usage")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, dir)
		Expect(os.Chmod(dir, 0o755)).To(Succeed())
		epilog := filepath.Join(dir, "epilog.sh")
		script := "#!/bin/sh\n" +
			"cat >> \"$SGE_JOB_SPOOL_DIR/usage\" <<EOF\n" +
			"gotest_tokens=5\n" +
			"gotest_cost=0.42\n" +
			"gotest_model=gpt\n" +
			"gotest_unselected=99\n" +
			"EOF\n"
		Expect(os.WriteFile(epilog, []byte(script), 0o755)).To(Succeed())

		previousEpilog := "NONE"
		for _, line := range strings.Split(run("qconf", "-sq", queue), "\n") {
			if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "epilog" {
				previousEpilog = fields[1]
			}
		}
		run("qconf", "-mattr", "queue", "epilog", epilog, queue)
		DeferCleanup(func() {
			run("qconf", "-mattr", "queue", "epilog", previousEpilog, queue)
		})

		// The group also selects ioops, which qacct -j then prints twice.
		config, err := qc.ShowGlobalConfiguration()
		Expect(err).NotTo(HaveOccurred())
		previous := *config
		merged, err := qconf.MergeUsagePatterns(config.ReportingParams, group,
			"gotest_tok*|gotest_cost", "gotest_model", "ioops")
		Expect(err).NotTo(HaveOccurred())
		config.ReportingParams = merged
		Expect(qc.ModifyGlobalConfig(*config)).To(Succeed())
		DeferCleanup(func() {
			Expect(qc.ModifyGlobalConfig(previous)).To(Succeed())
		})

		// qmaster keeps what MergeUsagePatterns wrote, and merging what
		// qconf prints again changes nothing.
		config, err = qc.ShowGlobalConfiguration()
		Expect(err).NotTo(HaveOccurred())
		Expect(config.ReportingParams).To(Equal(merged))
		again, err := qconf.MergeUsagePatterns(config.ReportingParams, group,
			"gotest_tok*", "gotest_cost", "gotest_model", "ioops")
		Expect(err).NotTo(HaveOccurred())
		Expect(again).To(Equal(merged))

		out := run("qsub", "-terse", "-sync", "y", "-b", "y", "-q", queue,
			"-o", "/dev/null", "-j", "y", "/bin/sleep", "1")
		jobID, err = strconv.ParseInt(strings.Fields(out)[0], 10, 64)
		Expect(err).NotTo(HaveOccurred())
	})

	It("is in the record of the accounting file, in its group", func() {
		var job core.JobDetail
		Eventually(func() (err error) {
			job, err = accountingRecord(jobID)
			return err
		}, "30s", "500ms").Should(Succeed())

		Expect(job.JobUsage.Custom[group]).To(Equal(map[string]float64{
			"gotest_tokens": 5,
			"gotest_cost":   0.42,
			"gotest_model":  0, // not a number: qmaster accounts 0
			"ioops":         float64(job.JobUsage.Usage.IOOps),
		}))
		v, ok := job.JobUsage.CustomValue("gotest_tokens")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(5.0))
		_, ok = job.JobUsage.CustomValue("gotest_unselected")
		Expect(ok).To(BeFalse())

		// The standard usage is next to it (eusage of 9.1).
		Expect(job.JobUsage.Usage.WallClock).To(BeNumerically(">", 0))
		Expect(job.JobUsage.RUsage.RuWallclock).To(BeNumerically(">", 0))
		Expect(job.EndTime).To(BeNumerically(">", job.StartTime))
	})

	It("is in the qacct -j output, which leaves the standard values alone", func() {
		qa, err := core.NewCommandLineQAcct(core.CommandLineQAcctConfig{})
		Expect(err).NotTo(HaveOccurred())
		var jobs []core.JobDetail
		Eventually(func() (err error) {
			jobs, err = qa.ShowJobDetails([]int64{jobID})
			if err == nil && len(jobs) != 1 {
				err = fmt.Errorf("%d records for job %d", len(jobs), jobID)
			}
			return err
		}, "30s", "500ms").Should(Succeed())
		job := jobs[0]

		fromFile, err := accountingRecord(jobID)
		Expect(err).NotTo(HaveOccurred())

		Expect(job.JobUsage.Custom).To(Equal(map[string]map[string]float64{
			core.CustomUsageNoGroup: {
				"gotest_tokens": 5,
				"gotest_cost":   0.42,
				"gotest_model":  0,
				"ioops":         float64(fromFile.JobUsage.Usage.IOOps),
			},
		}))
		Expect(job.JobUsage.Usage.IOOps).To(Equal(fromFile.JobUsage.Usage.IOOps))
		Expect(job.JobUsage.Usage.MaxRSS).To(Equal(fromFile.JobUsage.Usage.MaxRSS))
		Expect(job.JobUsage.Usage.WallClock).To(BeNumerically("~", fromFile.JobUsage.Usage.WallClock, 0.001))
		Expect(job.EndTime).To(Equal(fromFile.EndTime))
		Expect(job.ExitStatus).To(Equal(int64(0)))
	})
})
