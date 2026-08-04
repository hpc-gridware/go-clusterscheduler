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

package core

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The fixtures below are verbatim GCS 9.1.4 output. Each pair was
// captured twice from the same object -- once with SGE_SINGLE_LINE=true
// and once without -- so folding the wrapped form must reproduce the
// single-line form byte for byte.
var _ = Describe("joinContinuationLines", func() {

	// qconf -sq: breaks after a comma, so the comma is the separator
	// and the space before the backslash is only padding.
	wrappedQueue := []string{
		`qname                 wraptest.q`,
		`complex_values        slots=10,mem_free=1G,h_vmem=2G,virtual_free=3G, \`,
		`                      num_proc=4,h_rt=3600,s_rt=3000,h_cpu=100,s_cpu=90, \`,
		`                      h_fsize=10G`,
		`projects              NONE`,
	}
	singleQueue := `complex_values        slots=10,mem_free=1G,h_vmem=2G,virtual_free=3G,num_proc=4,h_rt=3600,s_rt=3000,h_cpu=100,s_cpu=90,h_fsize=10G`

	// qconf -scal: breaks after a space, so the whitespace in front of
	// the backslash IS the separator and must survive the fold.
	wrappedCalendar := []string{
		`calendar_name    wraptest`,
		`year             1.1.2026-2.1.2026=on 3.1.2026-4.1.2026=on \`,
		`                 5.1.2026-6.1.2026=on 7.1.2026-8.1.2026=on 9.1.2026-10.1.2026=on`,
		`week             NONE`,
	}
	singleCalendar := `year             1.1.2026-2.1.2026=on 3.1.2026-4.1.2026=on 5.1.2026-6.1.2026=on 7.1.2026-8.1.2026=on 9.1.2026-10.1.2026=on`

	It("reproduces the single-line form of a comma-wrapped value", func() {
		got := joinContinuationLines(wrappedQueue)
		Expect(got).To(HaveLen(3))
		Expect(got[1]).To(Equal(singleQueue))
	})

	It("reproduces the single-line form of a space-wrapped value", func() {
		got := joinContinuationLines(wrappedCalendar)
		Expect(got).To(HaveLen(3))
		Expect(got[1]).To(Equal(singleCalendar))
	})

	It("leaves already single-line input untouched", func() {
		in := []string{"qname all.q", "slots 10,[master=14]", "projects NONE"}
		Expect(joinContinuationLines(in)).To(Equal(in))
	})

	It("folds a value wrapped across many lines", func() {
		in := []string{`k a, \`, `   b, \`, `   c, \`, `   d`}
		Expect(joinContinuationLines(in)).To(Equal([]string{"k a,b,c,d"}))
	})

	It("keeps the accumulated value when input ends on a continuation", func() {
		// Truncated file: better to keep what is there than to drop the key.
		in := []string{`complex_values a=1, \`}
		Expect(joinContinuationLines(in)).To(Equal([]string{"complex_values a=1,"}))
	})

	It("tolerates CRLF via normalizeConfigLines", func() {
		in := []string{"year 1.1.2026=on \\\r", "     2.1.2026=on\r"}
		Expect(normalizeConfigLines(in)).To(
			Equal([]string{"year 1.1.2026=on 2.1.2026=on"}))
	})
})

// Regression specs for the actual corruption: before continuations were
// folded, a wrapped line lost every value after the break and kept a
// literal backslash as data.
var _ = Describe("Parsing wrapped config input", func() {

	wrappedQueue := []string{
		`qname                 wraptest.q`,
		`complex_values        slots=10,mem_free=1G,h_vmem=2G,virtual_free=3G, \`,
		`                      num_proc=4,h_rt=3600,s_rt=3000,h_cpu=100,s_cpu=90, \`,
		`                      h_fsize=10G`,
	}

	It("recovers every complex_values entry from a wrapped queue config", func() {
		lines := normalizeConfigLines(wrappedQueue)
		var values []string
		for i, line := range lines {
			if strings.HasPrefix(line, "complex_values") {
				values = ParseCommaSeparatedValuesWithOverrides(lines, i)
			}
		}
		Expect(values).To(HaveLen(10))
		Expect(values).To(ContainElement("h_fsize=10G"))
		Expect(values).NotTo(ContainElement(`\`),
			"a literal backslash must never survive as a value")
	})

	It("parses a wrapped exec host complex_values into the typed map", func() {
		lines := []string{
			`hostname              master`,
			`load_scaling          NONE`,
			`complex_values        slots=10,mem_free=1G, \`,
			`                      h_vmem=2G`,
			`usage_scaling         NONE`,
		}
		cfg, err := ParseExecHostConfigFromLines(lines)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ComplexValues).To(HaveKeyWithValue("slots", "10"))
		Expect(cfg.ComplexValues).To(HaveKeyWithValue("mem_free", "1G"))
		Expect(cfg.ComplexValues).To(HaveKeyWithValue("h_vmem", "2G"))
	})

	It("parses a wrapped global config value", func() {
		lines := []string{
			`execd_spool_dir      /opt/spool`,
			`qmaster_params       ENABLE_FORCED_QDEL=true, \`,
			`                     ENABLE_RESCHEDULE_KILL=true`,
		}
		cfg := ParseGlobalConfigFromLines(lines)
		Expect(cfg.QmasterParams).To(ContainElement("ENABLE_FORCED_QDEL=true"))
		Expect(cfg.QmasterParams).To(ContainElement("ENABLE_RESCHEDULE_KILL=true"))
	})

	It("parses a wrapped calendar year across the fold", func() {
		lines := []string{
			`calendar_name    wraptest`,
			`year             1.1.2026-2.1.2026=on 3.1.2026-4.1.2026=on \`,
			`                 5.1.2026-6.1.2026=on`,
			`week             NONE`,
		}
		cfg, recognized := ParseCalendarConfigFromLines(lines)
		Expect(recognized).To(BeNumerically(">=", 2))
		Expect(cfg.Year).To(ContainSubstring("1.1.2026-2.1.2026=on"))
		Expect(cfg.Year).To(ContainSubstring("5.1.2026-6.1.2026=on"))
		Expect(cfg.Year).NotTo(ContainSubstring(`\`))
	})
})
