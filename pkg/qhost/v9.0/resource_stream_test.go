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

func decodeResources(doc string) ([]qhost.HostResourcesXML, qhost.HostStats, error) {
	var hosts []qhost.HostResourcesXML
	stats, err := qhost.DecodeHostResourcesXML(strings.NewReader(doc), func(h qhost.HostResourcesXML) error {
		hosts = append(hosts, h)
		return nil
	})
	return hosts, stats, err
}

func resourceCapture(name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

// valuesOf maps a host's resource values as "dominance:value".
func valuesOf(h qhost.HostResourcesXML) map[string]string {
	out := map[string]string{}
	for _, v := range h.Resources {
		out[v.Name] = v.Dominance + ":" + v.Value
	}
	return out
}

var _ = Describe("Streaming qhost resource values", func() {
	Context("decoding live captures (9.1.7)", func() {
		It("reads the global pools first and every host's values with their dominance", func() {
			hosts, stats, err := decodeResources(resourceCapture("qhost-F-xml-9.1.7.xml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stats).To(Equal(qhost.HostStats{Decoded: 4, Complete: true}))
			Expect(hosts[0].IsGlobal()).To(BeTrue())
			Expect(valuesOf(hosts[0])).To(Equal(map[string]string{
				"zz_lic": "gl:3", "zz_both": "gc:9", "matlab": "gc:60"}),
				"a load report that is smaller wins (gl); a pool with a job booked shows what is left")
			Expect(hosts[1].Name).To(Equal("ocs-master"))
			w1 := valuesOf(hosts[2])
			Expect(hosts[2].Name).To(Equal("ocs-worker1"))
			Expect(w1["zz_both"]).To(Equal("hc:1"), "a host's own, smaller capacity wins over the global one")
			Expect(w1["zz_gpu"]).To(Equal("hc:3"), "an RSMAP is a count left")
			Expect(w1["gpu"]).To(Equal("hc:4"))
			Expect(w1["scratch"]).To(Equal("hc:500.000G"), "memory keeps its suffix")
			Expect(valuesOf(hosts[3])["zz_both"]).To(Equal("gc:9"), "a host without its own capacity repeats the global value")
			Expect(valuesOf(hosts[1])).NotTo(HaveKey("gpu"), "a host without a value prints nothing for it")
		})

		It("reads the pools once when the hosts are restricted to one", func() {
			hosts, stats, err := decodeResources(resourceCapture("qhost-F-global-xml-9.1.7.xml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(2))
			Expect(hosts[0].IsGlobal()).To(BeTrue())
			Expect(valuesOf(hosts[0])).To(HaveKeyWithValue("license", "gc:100"))
			Expect(hosts[1].Name).To(Equal("ocs-worker1"))
		})

		It("reads a busy cluster's values as what is left", func() {
			hosts, _, err := decodeResources(resourceCapture("qhost-F-busy-xml-9.1.7.xml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(valuesOf(hosts[1])).To(HaveKeyWithValue("slots", "hc:0"), "14 of 14 slots booked")
		})
	})

	It("skips a host without a name and a value without a name", func() {
		hosts, stats, err := decodeResources(`<qhost><host name=''><resourcevalue name='a' dominance='hc'>1</resourcevalue></host>` +
			`<host name='n1'><resourcevalue name='' dominance='hc'>1</resourcevalue><resourcevalue name='b' dominance='hc'> 2 </resourcevalue></host></qhost>`)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats).To(Equal(qhost.HostStats{Decoded: 1, Skipped: 1, Complete: true}))
		Expect(hosts[0].Resources).To(Equal([]qhost.ResourceValueXML{{Name: "b", Dominance: "hc", Value: "2"}}))
	})

	It("reports a cut document as incomplete and malformed output as malformed", func() {
		_, stats, err := decodeResources(`<qhost><host name='n1'></host>`)
		Expect(errors.Is(err, qhost.ErrIncompleteHostDocument)).To(BeTrue(), "got: %v", err)
		Expect(stats.Decoded).To(Equal(1))
		_, _, err = decodeResources(`<qhost></wrong>`)
		Expect(err).To(MatchError(HavePrefix("malformed qhost XML after 0 hosts")))
	})

	Context("against a fake qhost process", func() {
		var impl *qhost.QHostImpl
		noop := func(qhost.HostResourcesXML) error { return nil }
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string {
			return append([]string{"PATH=" + os.Getenv("PATH")}, kv...)
		}
		capture := fixture("qhost-F-xml-9.1.7.xml")

		BeforeEach(func() {
			var err error
			impl, err = qhost.NewCommandLineQhost(qhost.CommandLineQHostConfig{Executable: fixture("fake-qhost.sh")})
			Expect(err).NotTo(HaveOccurred())
		})

		It("asks for exactly the listed resources, and for the listed hosts when given", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			stats, err := impl.StreamHostResourcesXML(context.Background(),
				env("FAKE_QHOST_ARGS_FILE="+argsFile, "FAKE_QHOST_FILE="+capture), []string{"matlab", "gpu"}, nil, noop)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			got, _ := os.ReadFile(argsFile)
			Expect(string(got)).To(Equal("-xml\n-F\nmatlab,gpu\n"))
			_, err = impl.StreamHostResourcesXML(context.Background(),
				env("FAKE_QHOST_ARGS_FILE="+argsFile, "FAKE_QHOST_FILE="+capture), []string{"matlab"}, []string{"ocs-worker1"}, noop)
			Expect(err).NotTo(HaveOccurred())
			got, _ = os.ReadFile(argsFile)
			Expect(string(got)).To(Equal("-xml\n-F\nmatlab\n-h\nocs-worker1\n"))
		})

		It("refuses an empty resource list without running qhost", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			_, err := impl.StreamHostResourcesXML(context.Background(), env("FAKE_QHOST_ARGS_FILE="+argsFile), nil, nil, noop)
			Expect(err).To(MatchError(qhost.ErrNoResources))
			Expect(argsFile).NotTo(BeAnExistingFile())
		})

		It("refuses a resource name that is not a plain argument", func() {
			_, err := impl.StreamHostResourcesXML(context.Background(), env(), []string{"a\nb"}, nil, noop)
			Expect(err).To(HaveOccurred())
		})

		It("returns the stderr of a failed run in the command-error shape", func() {
			_, err := impl.StreamHostResourcesXML(context.Background(),
				env("FAKE_QHOST_EXIT=1", "FAKE_QHOST_STDERR=error: unable to contact qmaster"), []string{"matlab"}, nil, noop)
			Expect(err).To(MatchError(HavePrefix("failed to run command (error: unable to contact qmaster)")))
		})

		It("is cancelled by its context while qhost is still silent", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			started := time.Now()
			_, err := impl.StreamHostResourcesXML(ctx, env("FAKE_QHOST_SLEEP=5", "FAKE_QHOST_FILE="+capture), []string{"matlab"}, nil, noop)
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
		})

		It("kills qhost and returns the callback error when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			_, err := impl.StreamHostResourcesXML(context.Background(),
				env("FAKE_QHOST_FILE="+capture, "FAKE_QHOST_HANG=30"), []string{"matlab"}, nil,
				func(qhost.HostResourcesXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(time.Since(started)).To(BeNumerically("<", 5*time.Second))
		})
	})
})
