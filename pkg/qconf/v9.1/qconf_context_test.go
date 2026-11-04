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

package qconf

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("v9.1 ShowGlobalConfigurationContext", func() {
	BeforeEach(func() {
		if runtime.GOOS == "windows" {
			Skip("the fake qconf is a shell script")
		}
	})

	// fakeQconf writes a shell script standing in for qconf that logs
	// its argv and prints body.
	fakeQconf := func(body string) (exe, logPath string) {
		dir := GinkgoT().TempDir()
		logPath = filepath.Join(dir, "argv.log")
		exe = filepath.Join(dir, "qconf")
		script := "#!/bin/sh\necho \"$*\" >> " + logPath + "\n" + body + "\n"
		Expect(os.WriteFile(exe, []byte(script), 0o755)).To(Succeed())
		return exe, logPath
	}

	It("promotes the v9.1 keys like the context-free method", func() {
		exe, logPath := fakeQconf(`printf 'execd_spool_dir   /opt/cs/default/spool\njsv_params        -verbose\ntopology_file     /etc/topo\nmail_tag          NONE\ngdi_request_limits NONE\nbinding_params    strategy=linear,mode=auto\nfuture_key        1\n'`)
		c, err := NewCommandLineQConf(CommandLineQConfConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())

		cfg, err := c.ShowGlobalConfigurationContext(context.Background(), nil)
		Expect(err).NotTo(HaveOccurred())
		raw, err := os.ReadFile(logPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(raw)).To(Equal("-sconf global\n"))

		Expect(cfg.ExecdSpoolDir).To(Equal("/opt/cs/default/spool"))
		Expect(cfg.JsvParams).To(Equal("-verbose"))
		Expect(cfg.TopologyFile).To(Equal("/etc/topo"))
		Expect(cfg.BindingParams).To(Equal(map[string]string{"strategy": "linear", "mode": "auto"}))
		Expect(cfg.ExtraFields).To(Equal(map[string]string{"future_key": "1"}))

		plain, err := c.ShowGlobalConfiguration()
		Expect(err).NotTo(HaveOccurred())
		Expect(plain).To(Equal(cfg))
	})

	It("returns a failing qconf as an error carrying stderr only", func() {
		exe, _ := fakeQconf(`echo "execd_spool_dir /secret/spool"; echo "error: commlib error: got select error (Connection refused)" >&2; exit 1`)
		c, err := NewCommandLineQConf(CommandLineQConfConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())
		cfg, err := c.ShowGlobalConfigurationContext(context.Background(), nil)
		Expect(cfg).To(BeNil())
		Expect(err).To(MatchError(ContainSubstring("commlib error")))
		Expect(err.Error()).NotTo(ContainSubstring("/secret/spool"))
	})

	It("ends with the caller's context", func() {
		exe, _ := fakeQconf(`sleep 30`)
		c, err := NewCommandLineQConf(CommandLineQConfConfig{Executable: exe, Timeout: time.Minute})
		Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_, err = c.ShowGlobalConfigurationContext(ctx, nil)
		Expect(err).To(MatchError(context.DeadlineExceeded))
	})
})
