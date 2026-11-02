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
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// fakeQconf writes an executable shell script standing in for qconf.
func fakeQconf(body string) string {
	path := filepath.Join(GinkgoT().TempDir(), "qconf")
	Expect(os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755)).To(Succeed())
	return path
}

func contextQconf(exe string, timeout time.Duration) *core.CommandLineQConf {
	c, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: exe, Timeout: timeout})
	Expect(err).NotTo(HaveOccurred())
	return c
}

var _ = Describe("Reading configuration under the caller's context", func() {
	It("parses every set of a live qconf -srqs answer (9.1.7)", func() {
		data, err := os.ReadFile(filepath.Join("testdata", "qconf-srqs-all-9.1.7.txt"))
		Expect(err).NotTo(HaveOccurred())
		sets := core.ParseResourceQuotaSets(string(data))
		Expect(sets).To(HaveLen(3))
		Expect(sets[0]).To(Equal(core.ResourceQuotaSetConfig{Name: "max_slots_per_user",
			Description: `"Fair use: no single user takes more than 24 slots"`, Enabled: true,
			Limits: []string{"users {*} to slots=24"}}))
		Expect(sets[1].Name).To(Equal("license_limits"))
		Expect(sets[1].Limits).To(Equal([]string{"projects finance to matlab=5", "users {*} to ansys=4"}))
		Expect(sets[2].Limits).To(Equal([]string{"users {*} queues gpu.q to gpu=2"}))
	})

	It("leaves out a block without a name and text outside blocks", func() {
		Expect(core.ParseResourceQuotaSets("noise\n{\n enabled TRUE\n}\n{\n name a\n enabled FALSE\n limit to slots=1\n}\n")).
			To(Equal([]core.ResourceQuotaSetConfig{{Name: "a", Enabled: false, Limits: []string{"to slots=1"}}}))
		Expect(core.ParseResourceQuotaSets("")).To(BeEmpty())
	})

	It("reads all sets with one qconf -srqs and the complex table with one -sc", func() {
		data, err := os.ReadFile(filepath.Join("testdata", "qconf-srqs-all-9.1.7.txt"))
		Expect(err).NotTo(HaveOccurred())
		src := filepath.Join(GinkgoT().TempDir(), "srqs.txt")
		Expect(os.WriteFile(src, data, 0o644)).To(Succeed())
		exe := fakeQconf(`case "$*" in
"-srqs") cat ` + src + ` ;;
"-sc") printf '#name shortcut type relop requestable consumable default urgency\n#----\nslots s INT <= YES YES 1 1000\nh_rt h_rt TIME <= YES NO 0:0:0 0\n' ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
		c := contextQconf(exe, 0)
		sets, err := c.ShowAllResourceQuotaSetsContext(context.Background(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(sets).To(HaveLen(3))
		cx, err := c.ShowAllComplexesContext(context.Background(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(cx).To(HaveLen(2))
		Expect(cx[0].Shortcut).To(Equal("s"))
		Expect(cx[1].Consumable).To(Equal("NO"))
	})

	It("answers no set defined with an empty list, not an error", func() {
		c := contextQconf(fakeQconf(`echo "No resource quota set found" >&2; exit 1`), 0)
		sets, err := c.ShowAllResourceQuotaSetsContext(context.Background(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(sets).To(BeEmpty())
		Expect(sets).NotTo(BeNil())
	})

	It("reports any other failure, with qconf's words", func() {
		c := contextQconf(fakeQconf(`echo "error: commlib error: got select error (Connection refused)" >&2; exit 1`), 0)
		_, err := c.ShowAllResourceQuotaSetsContext(context.Background(), nil)
		Expect(err).To(MatchError(ContainSubstring("commlib error")))
	})

	It("ends with the caller's context, long before the configured timeout", func() {
		c := contextQconf(fakeQconf(`sleep 30`), time.Minute)
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := c.ShowAllResourceQuotaSetsContext(ctx, nil)
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	It("reads one execution host, the global one included, with its complex values (live 9.1.7 capture)", func() {
		src, err := filepath.Abs(filepath.Join("testdata", "qconf-se-global-9.1.7.txt"))
		Expect(err).NotTo(HaveOccurred())
		c := contextQconf(fakeQconf(`[ "$1 $2" = "-se global" ] || { echo "unexpected $*" >&2; exit 2; }
cat `+src), 0)
		cfg, err := c.ShowExecHostContext(context.Background(), nil, "global")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Name).To(Equal("global"))
		Expect(cfg.ComplexValues).To(HaveKeyWithValue("matlab", "60"))
		Expect(cfg.ComplexValues).To(HaveKeyWithValue("license", "100"))
		Expect(cfg.ComplexValues).To(HaveLen(8))
	})

	It("parses every cluster queue of one qconf -sq answer, with its complex values (live 9.1.7 capture)", func() {
		data, err := os.ReadFile(filepath.Join("testdata", "qconf-sq-all-9.1.7.txt"))
		Expect(err).NotTo(HaveOccurred())
		queues := core.ParseClusterQueues(string(data))
		Expect(queues).To(HaveLen(10))
		byName := map[string]core.ClusterQueueConfig{}
		for _, q := range queues {
			byName[q.Name] = q
		}
		Expect(byName).To(HaveKey("all.q"))
		Expect(strings.Join(byName["all.q"].ComplexValues, ",")).To(ContainSubstring("zz_q=3"))
		Expect(strings.Join(byName["all.q"].ComplexValues, ",")).To(ContainSubstring("ocs-worker1=zz_q=1"))
	})

	It("reads every cluster queue with one -sql and one -sq, and no queue as an empty list", func() {
		src, err := filepath.Abs(filepath.Join("testdata", "qconf-sq-all-9.1.7.txt"))
		Expect(err).NotTo(HaveOccurred())
		argsLog := filepath.Join(GinkgoT().TempDir(), "args")
		c := contextQconf(fakeQconf(`echo "$*" >> `+argsLog+`
case "$1" in
  -sql) grep '^qname' `+src+` | awk '{print $2}' ;;
  -sq) cat `+src+` ;;
  *) exit 2 ;;
esac`), 0)
		queues, err := c.ShowAllClusterQueuesContext(context.Background(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(queues).To(HaveLen(10))
		calls, _ := os.ReadFile(argsLog)
		lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
		Expect(lines).To(HaveLen(2), "two qconf calls whatever the number of queues")
		Expect(lines[1]).To(Equal("-sq alarm.q,all.q,bigmem.q,err.q,gpu.q,interactive.q,long.q,night.q,short.q,slurm.q"), "every queue -sql named, in one list")
		empty := contextQconf(fakeQconf(`echo "no cqueue list defined" >&2; exit 1`), 0)
		queues, err = empty.ShowAllClusterQueuesContext(context.Background(), nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(queues).To(BeEmpty())
	})

	It("returns a failed -sql as an error unless it is the no-queue answer", func() {
		c := contextQconf(fakeQconf(`echo "error: no permission, user root not defined as manager" >&2; exit 1`), 0)
		queues, err := c.ShowAllClusterQueuesContext(context.Background(), nil)
		Expect(err).To(HaveOccurred())
		Expect(queues).To(BeNil())
		_, err = c.ShowClusterQueues()
		Expect(err).To(HaveOccurred(), "the context-free reader matches the same exact message")
	})

	It("refuses a host operand that could read as a flag, without running qconf", func() {
		exe := fakeQconf(`echo ran > "$0.ran"`)
		c := contextQconf(exe, 0)
		_, err := c.ShowExecHostContext(context.Background(), nil, "-sc")
		Expect(err).To(HaveOccurred())
		Expect(exe+".ran").NotTo(BeAnExistingFile(), "qconf must not have been started")
	})

	It("ends a host read with the caller's context", func() {
		c := contextQconf(fakeQconf(`sleep 30`), time.Minute)
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_, err := c.ShowExecHostContext(ctx, nil, "global")
		Expect(err).To(MatchError(context.DeadlineExceeded))
	})

	It("runs with the caller's environment plus SGE_SINGLE_LINE, never the process environment", func() {
		GinkgoT().Setenv("QCONF_TEST_SECRET", "leak")
		c := contextQconf(fakeQconf(`echo "LC_ALL=$LC_ALL SINGLE=$SGE_SINGLE_LINE SECRET=$QCONF_TEST_SECRET"`), 0)
		out, err := c.RunCommandContext(context.Background(), []string{"LC_ALL=C", "PATH=" + os.Getenv("PATH")}, "-sc")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("LC_ALL=C SINGLE=true SECRET=\n"))
	})
})
