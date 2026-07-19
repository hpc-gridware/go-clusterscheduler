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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("ParseParallelEnvironmentConfigFromLines", func() {

	peText := `pe_name            openmpi
slots              999
user_lists         NONE
xuser_lists        NONE
start_proc_args    /bin/true
stop_proc_args     /bin/true
allocation_rule    $fill_up
control_slaves     TRUE
job_is_first_task  FALSE
urgency_slots      min
accounting_summary FALSE
ign_sreq_on_mhost  FALSE
master_forks_slaves FALSE
daemon_forks_slaves FALSE
`

	It("parses all typed fields and counts them", func() {
		cfg, recognized := core.ParseParallelEnvironmentConfigFromLines(
			strings.Split(peText, "\n"))
		Expect(cfg.Name).To(Equal("openmpi"))
		Expect(cfg.Slots).To(Equal(999))
		Expect(cfg.AllocationRule).To(Equal("$fill_up"))
		Expect(cfg.ControlSlaves).To(Equal("TRUE"))
		Expect(cfg.JobIsFirstTask).To(BeFalse())
		Expect(recognized).To(Equal(14))
		Expect(cfg.ExtraFields).To(BeEmpty())
	})

	It("captures unrecognised keys into ExtraFields", func() {
		lines := strings.Split(peText+"future_parameter some_value\n", "\n")
		cfg, recognized := core.ParseParallelEnvironmentConfigFromLines(lines)
		Expect(recognized).To(Equal(14))
		Expect(cfg.ExtraFields).To(HaveKeyWithValue(
			"future_parameter", "some_value"))
	})

	It("skips full-line comments without counting or capturing them", func() {
		commented := "#___INFO__MARK_BEGIN_NEW__\n" +
			"# Licensed under the Apache License, Version 2.0\n" +
			"#___INFO__MARK_END_NEW__\n" +
			"# A template description line\n" +
			peText
		cfg, recognized := core.ParseParallelEnvironmentConfigFromLines(
			strings.Split(commented, "\n"))
		Expect(recognized).To(Equal(14))
		Expect(cfg.Name).To(Equal("openmpi"))
		Expect(cfg.ExtraFields).To(BeEmpty())
	})

	It("tolerates CRLF line endings without leaking carriage returns", func() {
		crlf := strings.ReplaceAll(peText, "\n", "\r\n")
		cfg, recognized := core.ParseParallelEnvironmentConfigFromLines(
			strings.Split(crlf, "\n"))
		Expect(recognized).To(Equal(14))
		Expect(cfg.StartProcArgs).To(Equal("/bin/true"))
		Expect(cfg.AllocationRule).To(Equal("$fill_up"))
		for _, v := range []string{cfg.Name, cfg.StartProcArgs,
			cfg.StopProcArgs, cfg.AllocationRule, cfg.UrgencySlots} {
			Expect(v).NotTo(ContainSubstring("\r"))
		}
	})

	It("tolerates a missing final newline", func() {
		cfg, recognized := core.ParseParallelEnvironmentConfigFromLines(
			strings.Split(strings.TrimRight(peText, "\n"), "\n"))
		Expect(recognized).To(Equal(14))
		Expect(cfg.DaemonForksSlaves).To(BeFalse())
	})

	It("recognises nothing in a file that is not a PE definition", func() {
		junk := "This directory contains helper files.\n" +
			"See the README for details.\n"
		_, recognized := core.ParseParallelEnvironmentConfigFromLines(
			strings.Split(junk, "\n"))
		Expect(recognized).To(BeZero())
	})
})

var _ = Describe("ParseCalendarConfigFromLines", func() {

	calText := `calendar_name    day
year             1.1.1997,2.1.1997=on
week             mon-fri=6-20
`

	It("parses the typed fields and counts them", func() {
		cfg, recognized := core.ParseCalendarConfigFromLines(
			strings.Split(calText, "\n"))
		Expect(cfg.Name).To(Equal("day"))
		Expect(cfg.Year).To(Equal("1.1.1997,2.1.1997=on"))
		Expect(cfg.Week).To(Equal("mon-fri=6-20"))
		Expect(recognized).To(Equal(3))
	})

	It("preserves multiple space-separated ranges in year and week", func() {
		text := "calendar_name multi\n" +
			"year 1.1.2026=off 25.12.2026-26.12.2026=off\n" +
			"week mon-fri=20-6 sat-sun\n"
		cfg, recognized := core.ParseCalendarConfigFromLines(
			strings.Split(text, "\n"))
		Expect(recognized).To(Equal(3))
		Expect(cfg.Year).To(Equal("1.1.2026=off 25.12.2026-26.12.2026=off"))
		Expect(cfg.Week).To(Equal("mon-fri=20-6 sat-sun"))
	})

	It("skips full-line comments and captures unknown keys", func() {
		text := "# Off at night, on during the day\n" + calText +
			"future_key future_value\n"
		cfg, recognized := core.ParseCalendarConfigFromLines(
			strings.Split(text, "\n"))
		Expect(recognized).To(Equal(3))
		Expect(cfg.ExtraFields).To(HaveKeyWithValue(
			"future_key", "future_value"))
	})

	It("recognises nothing in a non-calendar file", func() {
		_, recognized := core.ParseCalendarConfigFromLines(
			strings.Split("just some text\n", "\n"))
		Expect(recognized).To(BeZero())
	})
})
