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

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qquota "github.com/hpc-gridware/go-clusterscheduler/pkg/qquota/v9.0"
)

func decodeQuotas(doc string) ([]qquota.QuotaRuleXML, qquota.QuotaStats, error) {
	var rules []qquota.QuotaRuleXML
	stats, err := qquota.DecodeQuotasXML(strings.NewReader(doc), func(r qquota.QuotaRuleXML) error {
		rules = append(rules, r)
		return nil
	})
	return rules, stats, err
}

func decodeFile(name string) ([]qquota.QuotaRuleXML, qquota.QuotaStats) {
	b, err := os.ReadFile(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())
	rules, stats, err := decodeQuotas(string(b))
	Expect(err).NotTo(HaveOccurred())
	return rules, stats
}

func ruleNamed(rules []qquota.QuotaRuleXML, name string) []qquota.QuotaRuleXML {
	var out []qquota.QuotaRuleXML
	for _, r := range rules {
		if r.Name == name {
			out = append(out, r)
		}
	}
	return out
}

var _ = Describe("Streaming qquota XML", func() {

	Context("DecodeQuotasXML on 9.1.7 captures (2026-10-31, three-host demo cluster)", func() {
		It("decodes a per-user consumable and a project row with their scope and usage", func() {
			// qquota -u '*' -xml with the demo's three rule sets.
			rules, stats := decodeFile("qquota-xml-9.1.7.xml")
			Expect(stats.Complete).To(BeTrue())
			Expect(stats.Skipped).To(BeZero())
			Expect(stats.Decoded).To(Equal(len(rules)))
			project := ruleNamed(rules, "license_limits/1")
			Expect(project).To(HaveLen(1))
			Expect(project[0].Projects).To(Equal([]string{"finance"}))
			Expect(project[0].Users).To(BeEmpty())
			Expect(project[0].Limits).To(Equal([]qquota.LimitXML{{Resource: "matlab", Limit: "5", Value: "1", HasValue: true}}))
			Expect(project[0].RuleSet()).To(Equal("license_limits"))
			Expect(project[0].Rule()).To(Equal("1"))
		})

		It("decodes a dynamic per-host limit evaluated per host, and a static limit without a value", func() {
			// A probe rule set: "users {*} hosts {*} to slots=$num_proc*4"
			// and "users {*} to h_vmem=64G" (h_vmem is not consumable).
			rules, _ := decodeFile("qquota-xml-dynamic-static-9.1.7.xml")
			dynamic := ruleNamed(rules, "zz_probe/1")
			Expect(dynamic).NotTo(BeEmpty())
			for _, r := range dynamic {
				Expect(r.Users).To(HaveLen(1), "one row per user and host")
				Expect(r.Hosts).To(HaveLen(1))
				Expect(r.Limits[0].Resource).To(Equal("slots"))
				Expect(r.Limits[0].Limit).To(Equal("56"), "$num_proc*4 evaluated for a 14-core host")
				Expect(r.Limits[0].HasValue).To(BeTrue())
			}
			static := ruleNamed(rules, "zz_probe/2")
			Expect(static).To(HaveLen(1))
			Expect(static[0].Users).To(Equal([]string{"*"}), `"{*}" arrives as "*"`)
			Expect(static[0].Limits).To(Equal([]qquota.LimitXML{{Resource: "h_vmem", Limit: "64G"}}),
				"a non-consumable limit has no value: absent, not zero")
			Expect(static[0].Limits[0].HasValue).To(BeFalse())
		})

		It("decodes a memory consumable as printed, and a named rule with an excluded user", func() {
			// "users {*} to scratch=100G" with one job requesting 1536M, and
			// "name probe_x users !bob queues all.q to slots=1000".
			rules, _ := decodeFile("qquota-xml-memory-named-9.1.7.xml")
			memory := ruleNamed(rules, "zz_probe_mem/1")
			Expect(memory).To(HaveLen(1))
			Expect(memory[0].Users).To(Equal([]string{"alice"}))
			Expect(memory[0].Limits[0]).To(Equal(qquota.LimitXML{Resource: "scratch", Limit: "100.000G", Value: "1.500G", HasValue: true}),
				"memory is printed 1024-based with three decimals: 1536M reads 1.500G")
			named := ruleNamed(rules, "zz_probe_named/probe_x")
			Expect(named).To(HaveLen(1))
			Expect(named[0].RuleSet()).To(Equal("zz_probe_named"))
			Expect(named[0].Rule()).To(Equal("probe_x"), "a named rule keeps its name")
			Expect(named[0].XUsers).To(Equal([]string{"bob"}))
			Expect(named[0].Users).To(BeEmpty())
			Expect(named[0].Queues).To(Equal([]string{"all.q"}))
		})
	})

	Context("DecodeQuotasXML on a 9.0.12 capture (2026-10-31, single-host dev container)", func() {
		It("decodes the same shapes as 9.1.7: dynamic, static, named with exclusion, disabled set absent", func() {
			// Sets: "users {*} hosts {*} to slots=$num_proc*2" and
			// "users {*} to h_vmem=64G"; "name probe_x users !bob queues all.q
			// to slots=1000"; a disabled "to slots=1". One job by root running.
			// OCS 9.1.0 and 9.1.7 and GCS 9.1.4 and 9.1.7 printed this document
			// byte for byte.
			rules, stats := decodeFile("qquota-xml-9.0.12.xml")
			Expect(stats).To(Equal(qquota.QuotaStats{Decoded: 3, Complete: true}))
			Expect(rules).To(Equal([]qquota.QuotaRuleXML{
				{
					Name: "zz_probe/1", Users: []string{"root"}, Hosts: []string{"master"},
					Limits: []qquota.LimitXML{{Resource: "slots", Limit: "28", Value: "1", HasValue: true}},
				},
				{
					Name: "zz_probe/2", Users: []string{"*"},
					Limits: []qquota.LimitXML{{Resource: "h_vmem", Limit: "64G"}},
				},
				{
					Name: "zz_probe_named/probe_x", XUsers: []string{"bob"}, Queues: []string{"all.q"},
					Limits: []qquota.LimitXML{{Resource: "slots", Limit: "1000", Value: "1", HasValue: true}},
				},
			}), "the disabled set has no row")
		})
	})

	Context("DecodeQuotasXML edge cases", func() {
		It("answers an empty result as complete, with no rule", func() {
			rules, stats, err := decodeQuotas(`<?xml version='1.0'?><qquota_result xmlns="x"></qquota_result>`)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats).To(Equal(qquota.QuotaStats{Complete: true}))
			Expect(rules).To(BeEmpty())
		})

		It("keeps several limits of one rule in order", func() {
			rules, _, err := decodeQuotas(`<qquota_result><qquota_rule name='s/1'><users>a</users>
			  <limit resource='slots' limit='4' value='1'/><limit resource='h_rt' limit='1:00:00'/></qquota_rule></qquota_result>`)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules[0].Limits).To(Equal([]qquota.LimitXML{
				{Resource: "slots", Limit: "4", Value: "1", HasValue: true},
				{Resource: "h_rt", Limit: "1:00:00"},
			}))
		})

		It("reads every scope list and its exclusions", func() {
			rules, _, err := decodeQuotas(`<qquota_result><qquota_rule name='s/r'>
			  <users>a</users><xusers>b</xusers><projects>p</projects><xprojects>xp</xprojects>
			  <pes>mpi</pes><xpes>smp</xpes><queues>q</queues><xqueues>xq</xqueues>
			  <hosts>@gpu</hosts><xhosts>n9</xhosts><limit resource='slots' limit='1' value='0'/>
			</qquota_rule></qquota_result>`)
			Expect(err).NotTo(HaveOccurred())
			r := rules[0]
			Expect([][]string{r.Users, r.XUsers, r.Projects, r.XProjects, r.PEs, r.XPEs, r.Queues, r.XQueues, r.Hosts, r.XHosts}).
				To(Equal([][]string{{"a"}, {"b"}, {"p"}, {"xp"}, {"mpi"}, {"smp"}, {"q"}, {"xq"}, {"@gpu"}, {"n9"}}))
		})

		It("tells a value of 0 from no value", func() {
			rules, _, err := decodeQuotas(`<qquota_result><qquota_rule name='s/1'><limit resource='slots' limit='4' value='0'/></qquota_rule></qquota_result>`)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules[0].Limits[0].HasValue).To(BeTrue())
			Expect(rules[0].Limits[0].Value).To(Equal("0"))
		})

		It("skips a rule element without a name and counts it", func() {
			_, stats, err := decodeQuotas(`<qquota_result><qquota_rule name=''/><qquota_rule name='s/1'/></qquota_result>`)
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Decoded).To(Equal(1))
			Expect(stats.Skipped).To(Equal(1))
		})

		It("reports a document cut short, with the rules read so far", func() {
			rules, stats, err := decodeQuotas(`<qquota_result><qquota_rule name='s/1'></qquota_rule><qquota_rule name='s/2'>`)
			Expect(errors.Is(err, qquota.ErrIncompleteQuotaDocument)).To(BeTrue(), "got: %v", err)
			Expect(stats.Complete).To(BeFalse())
			Expect(rules).To(HaveLen(1), "a prefix, never presented as the whole list")
		})

		It("reports an empty output as incomplete, never as no quotas", func() {
			_, stats, err := decodeQuotas("")
			Expect(errors.Is(err, qquota.ErrIncompleteQuotaDocument)).To(BeTrue())
			Expect(stats.Complete).To(BeFalse())
		})

		It("stops at the first callback error", func() {
			boom := errors.New("boom")
			n := 0
			_, err := qquota.DecodeQuotasXML(strings.NewReader(`<qquota_result><qquota_rule name='a/1'/><qquota_rule name='b/1'/></qquota_result>`),
				func(qquota.QuotaRuleXML) error { n++; return boom })
			Expect(err).To(MatchError(boom))
			Expect(n).To(Equal(1))
		})

		It("names a rule set without a rule part as the whole name", func() {
			r := qquota.QuotaRuleXML{Name: "odd"}
			Expect(r.RuleSet()).To(Equal("odd"))
			Expect(r.Rule()).To(Equal(""))
		})
	})

	Context("against a qquota process", func() {
		var impl *qquota.QQuotaImpl
		noop := func(qquota.QuotaRuleXML) error { return nil }
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string {
			return append([]string{"PATH=" + os.Getenv("PATH")}, kv...)
		}
		capture := fixture("qquota-xml-memory-named-9.1.7.xml")

		BeforeEach(func() {
			var err error
			impl, err = qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{Executable: fixture("fake-qquota.sh")})
			Expect(err).NotTo(HaveOccurred())
		})

		It(`streams the rules of "qquota -u * -xml", "*" passed literally, with exactly the given environment`, func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			var names []string
			stats, err := impl.StreamQuotasXML(context.Background(),
				env("FAKE_QQUOTA_ARGS_FILE="+argsFile, "FAKE_QQUOTA_FILE="+capture),
				func(r qquota.QuotaRuleXML) error {
					names = append(names, r.Name)
					return nil
				})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			Expect(names).To(ContainElements("zz_probe_mem/1", "zz_probe_named/probe_x"))
			got, err := os.ReadFile(argsFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("-u\n*\n-xml\n"), "without -u only the caller's rows would be printed")
		})

		It("returns the stderr of a failed run in the command-error shape", func() {
			// qquota with qmaster unreachable exits 1 with the reason on stderr.
			stats, err := impl.StreamQuotasXML(context.Background(),
				env("FAKE_QQUOTA_EXIT=1", "FAKE_QQUOTA_STDERR=error: commlib error: got select error (Connection refused)"), noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to run command (error: commlib error: got select error (Connection refused))"))
			Expect(errors.Is(err, qquota.ErrIncompleteQuotaDocument)).To(BeFalse(), "the process failure wins")
			Expect(stats.Complete).To(BeFalse())
		})

		It("reports a successful exit without a root close tag as incomplete", func() {
			doc := filepath.Join(GinkgoT().TempDir(), "cut.xml")
			Expect(os.WriteFile(doc, []byte("<qquota_result><qquota_rule name='s/1'></qquota_rule>"), 0o600)).To(Succeed())
			stats, err := impl.StreamQuotasXML(context.Background(), env("FAKE_QQUOTA_FILE="+doc), noop)
			Expect(errors.Is(err, qquota.ErrIncompleteQuotaDocument)).To(BeTrue(), "got: %v", err)
			Expect(stats.Decoded).To(Equal(1))
		})

		It("reports malformed output as malformed", func() {
			doc := filepath.Join(GinkgoT().TempDir(), "broken.xml")
			Expect(os.WriteFile(doc, []byte("<qquota_result></wrong>"), 0o600)).To(Succeed())
			_, err := impl.StreamQuotasXML(context.Background(), env("FAKE_QQUOTA_FILE="+doc), noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("malformed qquota XML after 0 rules"))
		})

		It("caps the stderr a failing qquota leaves behind", func() {
			_, err := impl.StreamQuotasXML(context.Background(),
				env("FAKE_QQUOTA_EXIT=1", "FAKE_QQUOTA_STDERR=start", "FAKE_QQUOTA_STDERR_BYTES=1000000"), noop)
			Expect(err).To(HaveOccurred())
			Expect(len(err.Error())).To(BeNumerically("<", 70*1024))
		})

		It("is cancelled by its context while qquota is still silent", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			started := time.Now()
			stats, err := impl.StreamQuotasXML(ctx, env("FAKE_QQUOTA_SLEEP=5", "FAKE_QQUOTA_FILE="+capture), noop)
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second))
			Expect(stats.Decoded).To(Equal(0))
		})

		It("kills qquota and returns the callback error when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			stats, err := impl.StreamQuotasXML(context.Background(),
				env("FAKE_QQUOTA_FILE="+capture, "FAKE_QQUOTA_HANG=30"),
				func(qquota.QuotaRuleXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(stats.Decoded).To(Equal(0))
			Expect(time.Since(started)).To(BeNumerically("<", 5*time.Second), "qquota was killed, not waited for")
		})

		It("does not wait for a grandchild that keeps the output open when the callback stops the stream", func() {
			stop := errors.New("enough")
			started := time.Now()
			_, err := impl.StreamQuotasXML(context.Background(),
				env("FAKE_QQUOTA_FILE="+capture, "FAKE_QQUOTA_LINGER=8"),
				func(qquota.QuotaRuleXML) error { return stop })
			Expect(err).To(BeIdenticalTo(stop))
			Expect(time.Since(started)).To(BeNumerically("<", 3*time.Second), "the lingering child was waited for")
		})

		It("reports a qquota that cannot be started, with nothing decoded", func() {
			gone := filepath.Join(GinkgoT().TempDir(), "qquota")
			Expect(os.WriteFile(gone, []byte("#!/bin/sh\n"), 0o700)).To(Succeed())
			missing, err := qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{Executable: gone})
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Remove(gone)).To(Succeed())
			stats, err := missing.StreamQuotasXML(context.Background(), nil, noop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to start qquota: "))
			Expect(stats).To(Equal(qquota.QuotaStats{}))
		})

		It("refuses an executable that is not on the PATH at construction", func() {
			_, err := qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{Executable: "/nonexistent/qquota"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("executable not found"))
		})

		It("runs nothing in dry-run mode", func() {
			dry, err := qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{DryRun: true})
			Expect(err).NotTo(HaveOccurred())
			stats, err := dry.StreamQuotasXML(context.Background(), nil, func(qquota.QuotaRuleXML) error {
				Fail("no rule expected")
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Complete).To(BeTrue())
			out, _, err := dry.RunContext(context.Background(), nil, "-u", "alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("Dry run: qquota [-u alice]"))
		})

		Context("RunContext", func() {
			It("returns stdout and stderr of a successful run", func() {
				out, stderr, err := impl.RunContext(context.Background(),
					env("FAKE_QQUOTA_FILE="+capture, "FAKE_QQUOTA_STDERR=warning"), "-xml")
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(ContainSubstring("<qquota_rule name='zz_probe_mem/1'>"))
				Expect(stderr).To(Equal("warning\n"))
			})

			It("returns the stderr of a failed run in the error", func() {
				_, _, err := impl.RunContext(context.Background(), env("FAKE_QQUOTA_EXIT=1", "FAKE_QQUOTA_STDERR=denied"), "-xml")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(HavePrefix("failed to run command (denied)"))
			})

			It("refuses an argument with a control character before running anything", func() {
				_, _, err := impl.RunContext(context.Background(), nil, "-u", "a\nb")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
