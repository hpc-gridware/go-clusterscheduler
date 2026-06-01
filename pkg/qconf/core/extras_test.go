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
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("ExtraFields helpers", func() {

	Describe("CaptureExtraField", func() {
		It("captures a single-line unknown key/value", func() {
			lines := []string{"port_range 50100-50200"}
			extras := map[string]string{}
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).To(HaveKeyWithValue("port_range", "50100-50200"))
		})

		It("preserves multiple words in the value", func() {
			lines := []string{"some_param foo bar baz"}
			extras := map[string]string{}
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).To(HaveKeyWithValue("some_param", "foo bar baz"))
		})

		It("lazily initialises the map when nil", func() {
			lines := []string{"new_key value"}
			var extras map[string]string
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).NotTo(BeNil())
			Expect(extras).To(HaveKeyWithValue("new_key", "value"))
		})

		It("skips continuation lines (two-space prefix)", func() {
			lines := []string{"  more values on continuation"}
			extras := map[string]string{}
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).To(BeEmpty())
		})

		It("skips empty lines", func() {
			lines := []string{""}
			extras := map[string]string{}
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).To(BeEmpty())
		})

		DescribeTable("rejects keys that violate the snake_case allowlist",
			func(line string) {
				extras := map[string]string{}
				core.CaptureExtraField(&extras, []string{line}, 0)
				Expect(extras).To(BeEmpty(),
					"expected key on line %q to be rejected", line)
			},
			Entry("backslash key (continuation injection)", `\ value`),
			Entry("leading #", `#comment_like value`),
			Entry("equals in key", `weird=key value`),
			Entry("uppercase letter", `MaxJobs value`),
			Entry("leading digit", `1bad value`),
			Entry("leading underscore", `_bad value`),
			Entry("hyphen", `bad-key value`),
			Entry("dot", `bad.key value`),
		)

		It("accepts ordinary snake_case unknown keys", func() {
			lines := []string{"port_range 50100-50200"}
			extras := map[string]string{}
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).To(HaveKeyWithValue("port_range", "50100-50200"))
		})

		It("refuses to capture backslash-continued multi-line unknowns", func() {
			// ParseMultiLineValue joins continuation lines without
			// preserving the original delimiter, which would silently
			// corrupt a round trip. The helper drops the key rather
			// than store a malformed value. See
			// qontrol/todos/185-* for the long-term fix that would
			// preserve multi-line unknowns byte-faithfully.
			lines := []string{
				"future_param accounting=true \\",
				"  reporting=false finished_jobs=0",
			}
			extras := map[string]string{}
			core.CaptureExtraField(&extras, lines, 0)
			Expect(extras).To(BeEmpty())
		})
	})

	Describe("SanitizeExtraValue", func() {
		It("accepts ordinary single-line values", func() {
			out, ok := core.SanitizeExtraValue("50100-50200")
			Expect(ok).To(BeTrue())
			Expect(out).To(Equal("50100-50200"))
		})

		It("accepts values with tabs and printable special chars", func() {
			out, ok := core.SanitizeExtraValue("foo\tbar=baz,quux")
			Expect(ok).To(BeTrue())
			Expect(out).To(Equal("foo\tbar=baz,quux"))
		})

		It("rejects values containing newline", func() {
			_, ok := core.SanitizeExtraValue("foo\nbar")
			Expect(ok).To(BeFalse())
		})

		It("rejects values containing carriage return", func() {
			_, ok := core.SanitizeExtraValue("foo\rbar")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("WriteExtraFields", func() {
		It("emits sorted key/value lines", func() {
			extras := map[string]string{
				"zeta":  "z",
				"alpha": "a",
				"mid":   "m",
			}
			buf := &bytes.Buffer{}
			Expect(core.WriteExtraFields(buf, extras, nil)).To(Succeed())
			Expect(buf.String()).To(Equal("alpha a\nmid m\nzeta z\n"))
		})

		It("drops keys that collide with typed-field keys", func() {
			extras := map[string]string{
				"keep":  "yes",
				"drop":  "no",
				"other": "val",
			}
			typed := map[string]struct{}{"drop": {}}
			buf := &bytes.Buffer{}
			Expect(core.WriteExtraFields(buf, extras, typed)).To(Succeed())
			Expect(buf.String()).To(Equal("keep yes\nother val\n"))
		})

		It("is a no-op when extras is empty or nil", func() {
			buf := &bytes.Buffer{}
			Expect(core.WriteExtraFields(buf, nil, nil)).To(Succeed())
			Expect(buf.Len()).To(BeZero())

			Expect(core.WriteExtraFields(buf, map[string]string{}, nil)).To(Succeed())
			Expect(buf.Len()).To(BeZero())
		})
	})

	Describe("TypedKeysOf", func() {
		It("returns json tag names for top-level struct fields", func() {
			keys := core.TypedKeysOf(core.CalendarConfig{})
			Expect(keys).To(HaveKey("calendar_name"))
			Expect(keys).To(HaveKey("year"))
			Expect(keys).To(HaveKey("week"))
		})

		It("skips fields tagged json:\"-\"", func() {
			keys := core.TypedKeysOf(core.CalendarConfig{})
			Expect(keys).NotTo(HaveKey("-"))
			Expect(keys).NotTo(HaveKey(""))
		})

		It("recurses into embedded structs", func() {
			type wrapper struct {
				core.CalendarConfig
				Extra string `json:"extra"`
			}
			keys := core.TypedKeysOf(wrapper{})
			Expect(keys).To(HaveKey("calendar_name"))
			Expect(keys).To(HaveKey("extra"))
		})

		It("strips ,omitempty and other tag options", func() {
			type s struct {
				X string `json:"x,omitempty"`
			}
			keys := core.TypedKeysOf(s{})
			Expect(keys).To(HaveKey("x"))
			Expect(keys).NotTo(HaveKey("x,omitempty"))
		})

		It("accepts both struct values and pointers", func() {
			byVal := core.TypedKeysOf(core.CalendarConfig{})
			byPtr := core.TypedKeysOf(&core.CalendarConfig{})
			Expect(byPtr).To(Equal(byVal))
		})
	})
})

var _ = Describe("ExtraFields round-trip via ParseGlobalConfigFromLines", func() {

	// renderExtras serialises only the ExtraFields entries of cfg to a
	// string using WriteExtraFields. It is the test-time analogue of
	// what ModifyGlobalConfig does on its way to a temp file; isolating
	// it here lets us assert the emit contract without needing a real
	// qmaster.
	renderExtras := func(cfg core.GlobalConfig) string {
		buf := &bytes.Buffer{}
		Expect(core.WriteExtraFields(buf, cfg.ExtraFields, core.TypedKeysOf(cfg))).To(Succeed())
		return buf.String()
	}

	It("captures port_range as an unknown field while keeping known fields typed", func() {
		input := strings.Split(`execd_spool_dir /opt/sge/spool
mailer /usr/bin/mail
port_range 50100-50200
max_jobs 1000`, "\n")

		cfg := core.ParseGlobalConfigFromLines(input)

		Expect(cfg.ExecdSpoolDir).To(Equal("/opt/sge/spool"))
		Expect(cfg.Mailer).To(Equal("/usr/bin/mail"))
		Expect(cfg.MaxJobs).To(Equal(1000))
		Expect(cfg.ExtraFields).To(HaveKeyWithValue("port_range", "50100-50200"))
		Expect(cfg.ExtraFields).NotTo(HaveKey("execd_spool_dir"))
	})

	It("emits the captured unknown field on the modify side via WriteExtraFields", func() {
		input := strings.Split(`max_jobs 1000
port_range 50100-50200`, "\n")
		cfg := core.ParseGlobalConfigFromLines(input)
		Expect(renderExtras(cfg)).To(Equal("port_range 50100-50200\n"))
	})

	It("emits multiple unknown keys in sorted order", func() {
		input := strings.Split(`max_jobs 1000
zeta_param z
alpha_param a
mid_param m`, "\n")
		cfg := core.ParseGlobalConfigFromLines(input)
		Expect(renderExtras(cfg)).To(Equal("alpha_param a\nmid_param m\nzeta_param z\n"))
	})

	It("drops collisions where typed-field key matches an extras key", func() {
		// Simulate a malformed input where the same key appears as both
		// a typed field and via direct ExtraFields injection (only
		// reachable through reflection-of-Map construction by a buggy
		// caller). The typed field's emit must win and the extras entry
		// must be dropped.
		cfg := core.GlobalConfig{MaxJobs: 1000}
		cfg.ExtraFields = map[string]string{
			"max_jobs":   "shadow",
			"port_range": "50100-50200",
		}
		Expect(renderExtras(cfg)).To(Equal("port_range 50100-50200\n"))
	})

	It("treats parsing of input without any unknown keys as a no-op for extras", func() {
		input := strings.Split(`execd_spool_dir /opt/sge/spool
max_jobs 1000`, "\n")
		cfg := core.ParseGlobalConfigFromLines(input)
		Expect(cfg.ExtraFields).To(BeEmpty())
		Expect(renderExtras(cfg)).To(BeEmpty())
	})

	It("does not capture a continuation line of a known multi-line value as an extra", func() {
		// qmaster_params has a backslash continuation; the second line
		// is its continuation, not a fresh key called "param2".
		input := strings.Split(`qmaster_params param1 \
  param2 param3
max_jobs 1000`, "\n")
		cfg := core.ParseGlobalConfigFromLines(input)
		// The continuation line "param2 param3" must NOT land in extras.
		_, present := cfg.ExtraFields["param2"]
		Expect(present).To(BeFalse(),
			"continuation line was misclassified as an unknown key; ExtraFields: %v",
			cfg.ExtraFields)
	})

	It("drops values containing embedded newlines on capture", func() {
		// SanitizeExtraValue refuses CR/LF, so we have to construct a
		// scenario where ParseMultiLineValue could synthesise one.
		// Direct injection is the only path: confirm the helper rejects.
		_, ok := core.SanitizeExtraValue("foo\nbar")
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ExtraFields round-trip via ShowSchedulerConfiguration parse switch", func() {

	// The ShowSchedulerConfiguration parser is not exposed as a
	// standalone function, but its inline switch follows the same
	// "default: CaptureExtraField" contract. The path here exercises
	// that contract by constructing a CalendarConfig (simpler shape,
	// same Capture/Write helpers) and confirming the round trip.

	It("captures unknown calendar keys and emits them sorted", func() {
		input := strings.Split(`calendar_name peak
year 1.1.2026,31.12.2026
week mon-fri
future_field future_value
another_unknown another_value`, "\n")

		// We can't call ShowCalendar without a cluster, but we can
		// exercise the parser logic by simulating the same key dispatch
		// directly through CaptureExtraField.
		cfg := core.CalendarConfig{Name: "peak"}
		for i, line := range input {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			switch fields[0] {
			case "calendar_name":
				cfg.Name = fields[1]
			case "year":
				cfg.Year = fields[1]
			case "week":
				cfg.Week = fields[1]
			default:
				core.CaptureExtraField(&cfg.ExtraFields, input, i)
			}
		}

		Expect(cfg.Name).To(Equal("peak"))
		Expect(cfg.Year).To(Equal("1.1.2026,31.12.2026"))
		Expect(cfg.Week).To(Equal("mon-fri"))
		Expect(cfg.ExtraFields).To(HaveKeyWithValue("future_field", "future_value"))
		Expect(cfg.ExtraFields).To(HaveKeyWithValue("another_unknown", "another_value"))

		buf := &bytes.Buffer{}
		Expect(core.WriteExtraFields(buf, cfg.ExtraFields, core.TypedKeysOf(cfg))).To(Succeed())
		Expect(buf.String()).To(Equal("another_unknown another_value\nfuture_field future_value\n"))
	})
})
