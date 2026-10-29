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

package qmod_test

// Integration specs for qmod under a context against a live cluster (the
// dev container, see the Makefile). They disable one queue instance for a
// moment and enable it again; run the cluster-backed packages with -p 1.

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qmod "github.com/hpc-gridware/go-clusterscheduler/pkg/qmod/v9.0"
)

// firstQueueInstance returns one queue instance of the live cluster, or ""
// when there is no cluster.
func firstQueueInstance() string {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return ""
	}
	for _, binary := range []string{"qmod", "qselect"} {
		if _, err := exec.LookPath(binary); err != nil {
			return ""
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "qselect").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}

var _ = Describe("qmod under a context against a live cluster", Label("integration"), func() {
	It("reports changed, already-in-state and invalid targets in its answer lines", func() {
		instance := firstQueueInstance()
		if instance == "" {
			Skip("no cluster available")
		}
		q := newQmod(qmod.CommandLineQModConfig{})
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		env := append(os.Environ(), "LC_ALL=C")
		DeferCleanup(func() {
			if out, _, err := q.ActionContext(context.Background(), env, "-e", []string{instance}); err != nil {
				GinkgoWriter.Printf("qmod -e %s: %v: %s\n", instance, err, out)
			}
		})

		out, _, err := q.ActionContext(ctx, env, "-d", []string{instance})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("changed state of \"" + instance + "\" (disabled)"))

		out, _, err = q.ActionContext(ctx, env, "-d", []string{instance})
		Expect(err).NotTo(HaveOccurred(), "an instance already in the state is not an error")
		Expect(out).To(ContainSubstring("is already in the specified state"))

		out, _, err = q.ActionContext(ctx, env, "-e", []string{instance})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("(enabled)"))

		out, _, err = q.ActionContext(ctx, env, "-d", []string{"nosuch.q.integration@" + strings.SplitN(instance, "@", 2)[1]})
		Expect(err).To(HaveOccurred(), "an invalid queue exits 1")
		Expect(out).To(ContainSubstring("invalid queue"))
	})
})
