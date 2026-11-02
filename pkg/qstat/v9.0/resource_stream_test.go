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

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Streaming qstat queue resource values", func() {
	decode := func(doc string) ([]qstat.QueueResourcesXML, qstat.Stats, error) {
		var out []qstat.QueueResourcesXML
		stats, err := qstat.DecodeQueueResourcesXML(strings.NewReader(doc), func(q qstat.QueueResourcesXML) error {
			out = append(out, q)
			return nil
		})
		return out, stats, err
	}

	It("reads every queue instance with its state and its qc values (live 9.1.7 capture)", func() {
		// all.q sets zz_q=3 with an override of 1 on ocs-worker1; a job
		// booked 2 of ocs-master's 3; gpu.q sets nothing and is disabled
		// on ocs-worker1; the caller's pending jobs follow the instances.
		data, err := os.ReadFile(filepath.Join("testdata", "qstat-F-queue-xml-9.1.7.xml"))
		Expect(err).NotTo(HaveOccurred())
		qis, stats, err := decode(string(data))
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		byName := map[string]qstat.QueueResourcesXML{}
		for _, q := range qis {
			byName[q.Name] = q
		}
		Expect(byName["all.q@ocs-master"].Resources).To(Equal([]qstat.QueueResourceValueXML{{Name: "zz_q", Dominance: "qc", Value: "1"}}))
		Expect(byName["all.q@ocs-worker1"].Resources).To(Equal([]qstat.QueueResourceValueXML{{Name: "zz_q", Dominance: "qc", Value: "1"}}),
			"the per-host override")
		Expect(byName["all.q@ocs-worker2"].Resources[0].Value).To(Equal("3"))
		Expect(byName["gpu.q@ocs-worker1"].State).To(Equal("d"))
		Expect(byName["gpu.q@ocs-worker1"].Resources).To(BeEmpty(), "a queue that does not set it prints nothing for it")
		for _, q := range qis {
			Expect(q.State).NotTo(Equal("Eqw"), "a pending job's state is not an instance's")
		}
	})

	It("reads the value qmaster resolved per instance from host and host-group overrides (live 9.1.7)", func() {
		// complex_values matlab=5,scratch=100G,[@gpu=gpu=2,matlab=4],
		// [@bigmem=scratch=500G],[ocs-worker2=matlab=1]; @gpu is
		// ocs-worker1, @bigmem ocs-worker2. An override replaces the whole
		// list, and the host override wins over the group one.
		data, err := os.ReadFile(filepath.Join("testdata", "qstat-F-queue-domains-xml-9.1.7.xml"))
		Expect(err).NotTo(HaveOccurred())
		qis, stats, err := decode(string(data))
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		// Where the instance's queue list does not set a resource, qstat
		// prints the instance's effective value from the host (hc) instead.
		got := map[string]map[string]string{}
		for _, q := range qis {
			Expect(q.State).To(Equal("d"))
			got[q.Name] = map[string]string{}
			for _, v := range q.Resources {
				got[q.Name][v.Name] = v.Dominance + ":" + v.Value
			}
		}
		Expect(got).To(Equal(map[string]map[string]string{
			"zzdom.q@ocs-master":  {"matlab": "qc:5", "scratch": "qc:100.000G"},
			"zzdom.q@ocs-worker1": {"matlab": "qc:4", "gpu": "qc:2", "scratch": "hc:500.000G"},
			"zzdom.q@ocs-worker2": {"matlab": "qc:1", "scratch": "hc:2048.000G"},
		}))
	})

	It("skips an instance without a name and reports a cut document as incomplete", func() {
		qis, stats, err := decode(`<job_info><queue_info><Queue-List><name></name></Queue-List>` +
			`<Queue-List><name>a.q@n1</name><resource name='r' type='qc'> 2 </resource></Queue-List></queue_info></job_info>`)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Skipped).To(Equal(1))
		Expect(qis).To(Equal([]qstat.QueueResourcesXML{{Name: "a.q@n1", Resources: []qstat.QueueResourceValueXML{{Name: "r", Dominance: "qc", Value: "2"}}}}))
		_, _, err = decode(`<job_info><queue_info><Queue-List><name>a.q@n1</name></Queue-List>`)
		Expect(errors.Is(err, qstat.ErrIncompleteDocument)).To(BeTrue(), "got: %v", err)
	})

	Context("against a qstat process", func() {
		var impl *qstat.QStatImpl
		noop := func(qstat.QueueResourcesXML) error { return nil }
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string { return append([]string{"PATH=" + os.Getenv("PATH")}, kv...) }
		BeforeEach(func() {
			var err error
			impl, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{Executable: fixture("fake-qstat.sh")})
			Expect(err).NotTo(HaveOccurred())
		})

		It("asks for exactly the listed resources in exactly the listed queues", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			stats, err := impl.StreamQueueResourcesXML(context.Background(),
				env("FAKE_QSTAT_ARGS_FILE="+argsFile, "FAKE_QSTAT_FILE="+fixture("qstat-F-queue-xml-9.1.7.xml")),
				[]string{"zz_q", "lic"}, []string{"all.q", "gpu.q"}, noop)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(4))
			got, _ := os.ReadFile(argsFile)
			Expect(string(got)).To(Equal("-F\nzz_q,lic\n-q\nall.q,gpu.q\n-xml\n"))
		})

		It("refuses no resource, no queue, or an element that is not a plain list element, without running qstat", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			e := env("FAKE_QSTAT_ARGS_FILE=" + argsFile)
			_, err := impl.StreamQueueResourcesXML(context.Background(), e, nil, []string{"all.q"}, noop)
			Expect(err).To(MatchError(qstat.ErrNoQueueResources))
			_, err = impl.StreamQueueResourcesXML(context.Background(), e, []string{"r"}, nil, noop)
			Expect(err).To(MatchError(qstat.ErrNoQueueResources))
			for _, bad := range [][2][]string{{{"-u"}, {"all.q"}}, {{"a,b"}, {"all.q"}}, {{"r"}, {"all.q,*"}}, {{"r"}, {"a q"}}} {
				_, err = impl.StreamQueueResourcesXML(context.Background(), e, bad[0], bad[1], noop)
				Expect(err).To(HaveOccurred(), "%v", bad)
			}
			Expect(argsFile).NotTo(BeAnExistingFile())
		})

		It("returns the stderr of a failed run in the command-error shape", func() {
			_, err := impl.StreamQueueResourcesXML(context.Background(),
				env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=error: unable to contact qmaster"), []string{"zz_q"}, []string{"all.q"}, noop)
			Expect(err).To(MatchError(HavePrefix("failed to run command (error: unable to contact qmaster)")))
		})

		It("kills qstat and returns the callback error when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			_, err := impl.StreamQueueResourcesXML(context.Background(),
				env("FAKE_QSTAT_FILE="+fixture("qstat-F-queue-xml-9.1.7.xml"), "FAKE_QSTAT_HANG=30"), []string{"zz_q"}, []string{"all.q"},
				func(qstat.QueueResourcesXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(time.Since(started)).To(BeNumerically("<", 5*time.Second))
		})

		It("splits a queue list too long for one argument into calls, and leaves out a too long name list", func() {
			queues := make([]string, 3000)
			for i := range queues {
				queues[i] = fmt.Sprintf("q%059d.q", i)
			}
			argsLog := filepath.Join(GinkgoT().TempDir(), "args")
			calls := 0
			stats, err := impl.StreamQueueResourcesXML(context.Background(),
				env("FAKE_QSTAT_ARGS_LOG="+argsLog, "FAKE_QSTAT_FILE="+fixture("qstat-F-queue-xml-9.1.7.xml")),
				[]string{"zz_q"}, queues, func(qstat.QueueResourcesXML) error { calls++; return nil })
			Expect(err).NotTo(HaveOccurred())
			logged, _ := os.ReadFile(argsLog)
			lines := strings.Split(strings.TrimSpace(string(logged)), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 2), "3,000 queues of 64 characters do not fit one argument")
			seen := 0
			for _, l := range lines {
				f := strings.Fields(l)
				Expect(f[0:2]).To(Equal([]string{"-F", "zz_q"}))
				Expect(len(f[3])).To(BeNumerically("<=", qstat.MaxListArgument))
				seen += len(strings.Split(f[3], ","))
			}
			Expect(seen).To(Equal(3000), "every queue asked for exactly once")
			Expect(stats.Decoded).To(Equal(4*len(lines)), "each call's instances are counted")
			Expect(stats.Complete).To(BeTrue())

			names := make([]string, 2000)
			for i := range names {
				names[i] = fmt.Sprintf("r%063d", i)
			}
			argsFile := filepath.Join(GinkgoT().TempDir(), "argsfile")
			_, err = impl.StreamQueueResourcesXML(context.Background(),
				env("FAKE_QSTAT_ARGS_FILE="+argsFile, "FAKE_QSTAT_FILE="+fixture("qstat-F-queue-xml-9.1.7.xml")),
				names, []string{"all.q"}, noop)
			Expect(err).NotTo(HaveOccurred())
			got, _ := os.ReadFile(argsFile)
			Expect(string(got)).To(Equal("-q\nall.q\n-xml\n-F\n"))
		})

		It("is cancelled by its context while qstat is still silent", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			started := time.Now()
			_, err := impl.StreamQueueResourcesXML(ctx, env("FAKE_QSTAT_SLEEP=5", "FAKE_QSTAT_FILE="+fixture("qstat-F-queue-xml-9.1.7.xml")),
				[]string{"zz_q"}, []string{"all.q"}, noop)
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
		})
	})
})
