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

	It("runs with the caller's environment plus SGE_SINGLE_LINE, never the process environment", func() {
		GinkgoT().Setenv("QCONF_TEST_SECRET", "leak")
		c := contextQconf(fakeQconf(`echo "LC_ALL=$LC_ALL SINGLE=$SGE_SINGLE_LINE SECRET=$QCONF_TEST_SECRET"`), 0)
		out, err := c.RunCommandContext(context.Background(), []string{"LC_ALL=C", "PATH=" + os.Getenv("PATH")}, "-sc")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("LC_ALL=C SINGLE=true SECRET=\n"))
	})
})
