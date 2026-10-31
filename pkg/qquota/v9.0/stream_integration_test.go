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

package qquota_test

// Integration specs for the streaming qquota client against a live cluster
// (the dev container, see the Makefile). They skip without a cluster. The
// first only reads; the second adds two resource quota sets and one short
// job, and removes both again.

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qquota "github.com/hpc-gridware/go-clusterscheduler/pkg/qquota/v9.0"
)

// qquotaClusterAvailable reports whether a live cluster answers qquota.
func qquotaClusterAvailable() bool {
	for _, binary := range []string{"qquota", "qconf", "qsub", "qdel"} {
		if _, err := exec.LookPath(binary); err != nil {
			return false
		}
	}
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "qquota", "-xml").Run() == nil
}

// liveProbeSets are the rule sets the second spec adds: a counting rule, a
// non-consumable rule, and a disabled set that must not show up.
var liveProbeSets = map[string]string{
	"zz_probe_live": `{
   name         zz_probe_live
   description  "go-clusterscheduler qquota live spec"
   enabled      TRUE
   limit        users {*} to slots=1000
   limit        users {*} to h_vmem=64G
}
`,
	"zz_probe_live_off": `{
   name         zz_probe_live_off
   description  "go-clusterscheduler qquota live spec"
   enabled      FALSE
   limit        to slots=1
}
`,
}

func runCommand(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "%s %v: %s", name, args, out)
	return strings.TrimSpace(string(out))
}

func streamAll(impl *qquota.QQuotaImpl) ([]qquota.QuotaRuleXML, qquota.QuotaStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var rules []qquota.QuotaRuleXML
	stats, err := impl.StreamQuotasXML(ctx, nil, func(r qquota.QuotaRuleXML) error {
		rules = append(rules, r)
		return nil
	})
	return rules, stats, err
}

var _ = Describe("qquota against a live cluster", Label("integration"), func() {
	BeforeEach(func() {
		if !qquotaClusterAvailable() {
			Skip("no cluster answers qquota")
		}
	})

	It("streams every rule row as a complete document", func() {
		impl, err := qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{})
		Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stats, err := impl.StreamQuotasXML(ctx, nil, func(r qquota.QuotaRuleXML) error {
			Expect(r.Name).To(ContainSubstring("/"), "every row names <set>/<rule>")
			for _, l := range r.Limits {
				Expect(l.Resource).NotTo(BeEmpty())
				Expect(l.Limit).NotTo(BeEmpty())
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
	})

	It("streams the rows of rule sets it added, with the usage of a running job", func() {
		me, err := user.Current()
		Expect(err).NotTo(HaveOccurred())
		dir := GinkgoT().TempDir()
		for name, body := range liveProbeSets {
			file := filepath.Join(dir, name)
			Expect(os.WriteFile(file, []byte(body), 0o600)).To(Succeed())
			runCommand("qconf", "-Arqs", file)
			DeferCleanup(func() { _ = exec.Command("qconf", "-drqs", name).Run() })
		}
		jobID := runCommand("qsub", "-terse", "-b", "y", "-o", "/dev/null", "-j", "y", "sleep", "300")
		DeferCleanup(func() { _ = exec.Command("qdel", jobID).Run() })

		impl, err := qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{})
		Expect(err).NotTo(HaveOccurred())

		// The counting rule has a row only once the job runs.
		var rules []qquota.QuotaRuleXML
		Eventually(func(g Gomega) {
			var stats qquota.QuotaStats
			rules, stats, err = streamAll(impl)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stats.Complete).To(BeTrue())
			g.Expect(ruleNamed(rules, "zz_probe_live/1")).NotTo(BeEmpty())
		}).WithTimeout(90 * time.Second).WithPolling(2 * time.Second).Should(Succeed())

		counting := ruleNamed(rules, "zz_probe_live/1")
		Expect(counting).To(HaveLen(1), "one row per user with usage")
		Expect(counting[0].Users).To(Equal([]string{me.Username}))
		Expect(counting[0].Limits).To(HaveLen(1))
		slots := counting[0].Limits[0]
		Expect(slots.Resource).To(Equal("slots"))
		Expect(slots.Limit).To(Equal("1000"))
		Expect(slots.HasValue).To(BeTrue())
		inUse, err := strconv.Atoi(slots.Value)
		Expect(err).NotTo(HaveOccurred())
		Expect(inUse).To(BeNumerically(">=", 1), "at least the probe job; other jobs of the user count too")

		static := ruleNamed(rules, "zz_probe_live/2")
		Expect(static).To(HaveLen(1))
		Expect(static[0].Users).To(Equal([]string{"*"}))
		Expect(static[0].Limits).To(Equal([]qquota.LimitXML{{Resource: "h_vmem", Limit: "64G"}}))

		for _, r := range rules {
			Expect(r.RuleSet()).NotTo(Equal("zz_probe_live_off"), "a disabled set has no row")
		}
	})
})
