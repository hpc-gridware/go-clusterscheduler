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

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qhost "github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/v9.0"
)

func decodeHosts(doc string) ([]qhost.HostXML, qhost.HostStats, error) {
	var hosts []qhost.HostXML
	stats, err := qhost.DecodeHostsXML(strings.NewReader(doc), func(h qhost.HostXML) error {
		hosts = append(hosts, h)
		return nil
	})
	return hosts, stats, err
}

var _ = Describe("Streaming qhost XML", func() {

	Context("DecodeHostsXML on a 9.1.7 capture with one silent execd", func() {
		// Captured 2026-10-29 from a three-host cluster after sge_execd on
		// ocs-worker2 had been killed and max_unheard (5 min) had passed.
		var hosts []qhost.HostXML
		var stats qhost.HostStats

		BeforeEach(func() {
			b, err := os.ReadFile(filepath.Join("testdata", "qhost-xml-unreachable-9.1.7.xml"))
			Expect(err).NotTo(HaveOccurred())
			hosts, stats, err = decodeHosts(string(b))
			Expect(err).NotTo(HaveOccurred())
		})

		It("decodes every host element, the global pseudo-host included", func() {
			Expect(stats.Complete).To(BeTrue())
			Expect(stats.Decoded).To(Equal(4))
			Expect(hosts[0].Name).To(Equal("global"))
			Expect(hosts[0].IsGlobal()).To(BeTrue())
			Expect(hosts[0].Reporting()).To(BeFalse())
		})

		It("keeps the values of a reporting host as printed", func() {
			h := hosts[1]
			Expect(h.Name).To(Equal("ocs-master"))
			Expect(h.IsGlobal()).To(BeFalse())
			Expect(h.Reporting()).To(BeTrue())
			Expect(h.Arch).To(Equal("lx-amd64"))
			Expect(h.NumProc).To(Equal("14"))
			Expect(h.MemTotal).To(Equal("11.7G"))
			Expect(h.SwapUsed).To(Equal("0.0"))
		})

		It("reports a host whose execd is silent as not reporting, keeping its static values", func() {
			h := hosts[3]
			Expect(h.Name).To(Equal("ocs-worker2"))
			Expect(h.Reporting()).To(BeFalse())
			Expect(h.LoadAvg).To(Equal(qhost.NoValue))
			Expect(h.MemUsed).To(Equal(qhost.NoValue))
			Expect(h.Sockets).To(Equal(qhost.NoValue))
			// arch, num_proc and the totals survive the silence.
			Expect(h.Arch).To(Equal("lx-amd64"))
			Expect(h.NumProc).To(Equal("14"))
			Expect(h.MemTotal).To(Equal("11.7G"))
		})
	})

	Context("DecodeHostsXML edge cases", func() {
		It("treats a load of 0.00 as reporting", func() {
			hosts, _, err := decodeHosts(`<qhost><host name='idle'><hostvalue name='load_avg'>0.00</hostvalue></host></qhost>`)
			Expect(err).NotTo(HaveOccurred())
			Expect(hosts[0].Reporting()).To(BeTrue())
		})

		It("reads only direct hostvalue children, not queue or job values", func() {
			doc := `<qhost><host name='n1'>
			  <hostvalue name='load_avg'>1.50</hostvalue>
			  <queue name='all.q'><queuevalue qname='all.q' name='state_string'>d</queuevalue></queue>
			  <job name='12'><jobvalue jobid='12' name='job_name'>x</jobvalue></job>
			</host></qhost>`
			hosts, stats, err := decodeHosts(doc)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(1))
			Expect(hosts[0].LoadAvg).To(Equal("1.50"))
		})

		It("skips a host element without a name and counts it", func() {
			_, stats, err := decodeHosts(`<qhost><host name=''></host><host name='n1'></host></qhost>`)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(1))
			Expect(stats.Skipped).To(Equal(1))
		})

		It("reports a document cut short, with the hosts read so far", func() {
			hosts, stats, err := decodeHosts(`<qhost><host name='n1'></host><host name='n2'>`)
			Expect(errors.Is(err, qhost.ErrIncompleteHostDocument)).To(BeTrue())
			Expect(stats.Complete).To(BeFalse())
			Expect(hosts).To(HaveLen(1))
		})

		It("stops at the first callback error", func() {
			boom := errors.New("boom")
			n := 0
			_, err := qhost.DecodeHostsXML(strings.NewReader(`<qhost><host name='a'/><host name='b'/></qhost>`),
				func(qhost.HostXML) error { n++; return boom })
			Expect(err).To(MatchError(boom))
			Expect(n).To(Equal(1))
		})
	})

	Context("against a qhost process", func() {
		var impl *qhost.QHostImpl
		noop := func(qhost.HostXML) error { return nil }
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string {
			return append([]string{"PATH=" + os.Getenv("PATH")}, kv...)
		}
		capture := fixture("qhost-xml-unreachable-9.1.7.xml")

		BeforeEach(func() {
			var err error
			impl, err = qhost.NewCommandLineQhost(qhost.CommandLineQHostConfig{Executable: fixture("fake-qhost.sh")})
			Expect(err).NotTo(HaveOccurred())
		})

		It(`streams the hosts of "qhost -xml" with exactly the given environment`, func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			var names []string
			stats, err := impl.StreamHostsXML(context.Background(),
				env("FAKE_QHOST_ARGS_FILE="+argsFile, "FAKE_QHOST_FILE="+capture),
				func(h qhost.HostXML) error {
					names = append(names, h.Name)
					return nil
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats).To(Equal(qhost.HostStats{Decoded: 4, Complete: true}))
			Expect(names).To(Equal([]string{"global", "ocs-master", "ocs-worker1", "ocs-worker2"}))
			got, err := os.ReadFile(argsFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("-xml\n"))
		})

		It("returns the stderr of a failed run in the command-error shape", func() {
			stats, err := impl.StreamHostsXML(context.Background(),
				env("FAKE_QHOST_EXIT=1", "FAKE_QHOST_STDERR=error: unable to contact qmaster"), noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to run command (error: unable to contact qmaster)"))
			Expect(errors.Is(err, qhost.ErrIncompleteHostDocument)).To(BeFalse(), "the process failure wins")
			Expect(stats.Complete).To(BeFalse())
		})

		It("reports a successful exit without a root close tag as incomplete", func() {
			doc := filepath.Join(GinkgoT().TempDir(), "cut.xml")
			Expect(os.WriteFile(doc, []byte("<qhost><host name='n1'></host>"), 0o600)).To(Succeed())
			stats, err := impl.StreamHostsXML(context.Background(), env("FAKE_QHOST_FILE="+doc), noop)
			Expect(errors.Is(err, qhost.ErrIncompleteHostDocument)).To(BeTrue(), "got: %v", err)
			Expect(stats.Decoded).To(Equal(1))
		})

		It("reports malformed output as malformed", func() {
			doc := filepath.Join(GinkgoT().TempDir(), "broken.xml")
			Expect(os.WriteFile(doc, []byte("<qhost></wrong>"), 0o600)).To(Succeed())
			_, err := impl.StreamHostsXML(context.Background(), env("FAKE_QHOST_FILE="+doc), noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("malformed qhost XML after 0 hosts"))
		})

		It("is cancelled by its context while qhost is still silent", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			started := time.Now()
			stats, err := impl.StreamHostsXML(ctx, env("FAKE_QHOST_SLEEP=5", "FAKE_QHOST_FILE="+capture), noop)
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
			Expect(stats.Decoded).To(Equal(0))
		})

		It("kills qhost and returns the callback error when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			stats, err := impl.StreamHostsXML(context.Background(),
				env("FAKE_QHOST_FILE="+capture, "FAKE_QHOST_HANG=30"),
				func(qhost.HostXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(stats.Decoded).To(Equal(0))
			Expect(time.Since(started)).To(BeNumerically("<", 5*time.Second), "qhost was killed, not waited for")
		})

		It("does not wait for a grandchild that keeps the output open when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			_, err := impl.StreamHostsXML(context.Background(),
				env("FAKE_QHOST_FILE="+capture, "FAKE_QHOST_LINGER=8"),
				func(qhost.HostXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second), "the lingering child was waited for")
		})

		It("reports a qhost that cannot be started, with nothing decoded", func() {
			gone := filepath.Join(GinkgoT().TempDir(), "qhost")
			Expect(os.WriteFile(gone, []byte("#!/bin/sh\n"), 0o700)).To(Succeed())
			missing, err := qhost.NewCommandLineQhost(qhost.CommandLineQHostConfig{Executable: gone})
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Remove(gone)).To(Succeed())
			stats, err := missing.StreamHostsXML(context.Background(), nil, noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to start qhost: "))
			Expect(stats).To(Equal(qhost.HostStats{}))
		})

		It("runs nothing in dry-run mode", func() {
			dry, err := qhost.NewCommandLineQhost(qhost.CommandLineQHostConfig{DryRun: true})
			Expect(err).NotTo(HaveOccurred())
			stats, err := dry.StreamHostsXML(context.Background(), nil, func(qhost.HostXML) error {
				Fail("no host expected")
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			out, _, err := dry.RunContext(context.Background(), nil, "-h", "n1")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qhost [-h n1]"))
		})

		Context("RunContext", func() {
			It("returns stdout and stderr of a successful run", func() {
				out, stderr, err := impl.RunContext(context.Background(),
					env("FAKE_QHOST_FILE="+capture, "FAKE_QHOST_STDERR=warning"), "-xml")
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(ContainSubstring("<host name='ocs-worker2'>"))
				Expect(stderr).To(Equal("warning\n"))
			})

			It("keeps the output of a failed run and puts stderr in the error", func() {
				out, stderr, err := impl.RunContext(context.Background(),
					env("FAKE_QHOST_FILE="+capture, "FAKE_QHOST_EXIT=1",
						"FAKE_QHOST_STDERR=error: can't resolve hostname \"nosuchhost\""), "-h", "nosuchhost")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`failed to run command (error: can't resolve hostname "nosuchhost"): exit status 1`))
				Expect(out).NotTo(BeEmpty())
				Expect(stderr).To(ContainSubstring("nosuchhost"))
			})

			It("is cancelled by its context and says so", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
				defer cancel()
				started := time.Now()
				_, _, err := impl.RunContext(ctx, env("FAKE_QHOST_SLEEP=5"), "-xml")
				Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
				Expect(err.Error()).To(HavePrefix("failed to run command ("))
				Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
			})

			It("caps stderr without blocking qhost", func() {
				_, stderr, err := impl.RunContext(context.Background(), env("FAKE_QHOST_STDERR_BYTES=200000"), "-xml")
				Expect(err).NotTo(HaveOccurred())
				Expect(len(stderr)).To(Equal(64 * 1024))
			})

			It("refuses control characters before spawning anything", func() {
				argsFile := filepath.Join(GinkgoT().TempDir(), "args")
				_, _, err := impl.RunContext(context.Background(), env("FAKE_QHOST_ARGS_FILE="+argsFile), "-h", "n1\nn2")
				Expect(err).To(HaveOccurred())
				Expect(argsFile).NotTo(BeAnExistingFile())
			})
		})
	})
})
