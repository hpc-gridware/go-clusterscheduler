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

package qhost_test

// Integration specs for the streaming qhost client against a live cluster
// (the dev container, see the Makefile). They skip without a cluster and
// change nothing.

import (
	"context"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qhost "github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/v9.0"
)

// qhostClusterAvailable reports whether a live cluster answers qhost.
func qhostClusterAvailable() bool {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return false
	}
	if _, err := exec.LookPath("qhost"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "qhost").Run() == nil
}

var _ = Describe("Streaming qhost against a live cluster", Label("integration"), func() {
	var impl *qhost.QHostImpl

	BeforeEach(func() {
		if !qhostClusterAvailable() {
			Skip("no cluster available")
		}
		var err error
		impl, err = qhost.NewCommandLineQhost(qhost.CommandLineQHostConfig{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("streams the global pseudo-host first and at least one reporting execution host", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		env := append(os.Environ(), "LC_ALL=C")
		Eventually(func(g Gomega) {
			var hosts []qhost.HostXML
			stats, err := impl.StreamHostsXML(ctx, env, func(h qhost.HostXML) error {
				hosts = append(hosts, h)
				return nil
			})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stats.Complete).To(BeTrue())
			g.Expect(stats.Skipped).To(Equal(0))
			g.Expect(hosts).NotTo(BeEmpty())
			g.Expect(hosts[0].IsGlobal()).To(BeTrue())
			g.Expect(hosts[0].Reporting()).To(BeFalse())
			g.Expect(hosts[1:]).To(ContainElement(And(
				HaveField("Reporting()", BeTrue()),
				HaveField("Arch", Not(BeEmpty())),
				HaveField("NumProc", Not(Equal(qhost.NoValue))))))
		}).WithTimeout(45 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
	})

	It("puts qhost's error text into the error of a failed run", func() {
		_, stderr, err := impl.RunContext(context.Background(), nil, "-h", "nosuchhost.integration.invalid")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("nosuchhost.integration.invalid"))
		Expect(err.Error()).To(ContainSubstring("nosuchhost.integration.invalid"))
	})
})
