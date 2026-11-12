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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("MergeUsagePatterns", func() {

	base := []string{"accounting=true", "reporting=false", "flush_time=00:00:01"}

	merge := func(params []string, group string, patterns ...string) []string {
		merged, err := core.MergeUsagePatterns(params, group, patterns...)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		return merged
	}

	It("does not modify the input slice", func() {
		input := append(make([]string, 0, 8), base...)
		input = append(input, "usage_patterns=gpu:nvidia*")
		merge(input, "gpu", "amd*")
		merge(input[:3], "agent", "agent_*")
		Expect(input).To(Equal([]string{"accounting=true", "reporting=false",
			"flush_time=00:00:01", "usage_patterns=gpu:nvidia*"}))
	})

	DescribeTable("merges the patterns into the group",
		func(params []string, group string, patterns []string, expected []string) {
			merged := merge(params, group, patterns...)
			Expect(merged).To(Equal(expected))
			// Installers run repeatedly: a second run changes nothing.
			Expect(merge(merged, group, patterns...)).To(Equal(expected))
		},
		Entry("appends an entry when there is none",
			base, "agent", []string{"agent_*"},
			[]string{"accounting=true", "reporting=false", "flush_time=00:00:01",
				"usage_patterns=agent:agent_*"}),
		Entry("joins several patterns of a new group with |",
			nil, "gpu", []string{"nvidia*", "amd*"},
			[]string{"usage_patterns=gpu:nvidia*|amd*"}),
		Entry("adds a group next to the groups of other tools",
			[]string{"accounting=true", "usage_patterns=gpu:nvidia*|amd*", "joblog=false"},
			"agent", []string{"agent_*"},
			[]string{"accounting=true", "usage_patterns=gpu:nvidia*|amd*;agent:agent_*", "joblog=false"}),
		Entry("adds only the missing patterns to an existing group",
			[]string{"usage_patterns=power:power-*;gpu:nvidia*"},
			"gpu", []string{"nvidia*", "amd*"},
			[]string{"usage_patterns=power:power-*;gpu:nvidia*|amd*"}),
		Entry("does not mistake a group for another one with the same prefix",
			[]string{"usage_patterns=agents:x*"}, "agent", []string{"agent_*"},
			[]string{"usage_patterns=agents:x*;agent:agent_*"}),
		Entry("fills an empty entry",
			[]string{"usage_patterns="}, "agent", []string{"agent_*"},
			[]string{"usage_patterns=agent:agent_*"}),
		Entry("fills a group that has no expression",
			[]string{"usage_patterns=agent:"}, "agent", []string{"agent_*"},
			[]string{"usage_patterns=agent:agent_*"}),
		// qmaster keeps the last usage_patterns entry of reporting_params.
		Entry("changes the last entry, the one qmaster uses",
			[]string{"usage_patterns=gpu:nvidia*", "usage_patterns=power:p*"},
			"agent", []string{"agent_*"},
			[]string{"usage_patterns=gpu:nvidia*", "usage_patterns=power:p*;agent:agent_*"}),
		Entry("takes a pattern argument that is a list of alternatives apart",
			[]string{"usage_patterns=gpu:nvidia*|amd*"}, "gpu", []string{"nvidia*|amd*|intel*"},
			[]string{"usage_patterns=gpu:nvidia*|amd*|intel*"}),
		Entry("drops repeated and empty pattern arguments",
			nil, "gpu", []string{"a*", "", "a*", " b* ", "|"},
			[]string{"usage_patterns=gpu:a*|b*"}),
		// qmaster reads the parameter name without regard to case. A
		// second, lower case entry would replace the groups of this one.
		Entry("finds an entry whose name is not in lower case and keeps the spelling",
			[]string{"USAGE_PATTERNS=gpu:nvidia*"}, "agent", []string{"agent_*"},
			[]string{"USAGE_PATTERNS=gpu:nvidia*;agent:agent_*"}),
		Entry("does not keep the empty group of a trailing separator",
			[]string{"usage_patterns=gpu:x*;"}, "agent", []string{"agent_*"},
			[]string{"usage_patterns=gpu:x*;agent:agent_*"}),
		Entry("finds a group written with blanks",
			[]string{"usage_patterns=gpu : x*"}, "gpu", []string{"x*"},
			[]string{"usage_patterns=gpu : x*"}),
		Entry("trims the group name",
			[]string{"usage_patterns=agent:a*"}, " agent ", []string{"b*"},
			[]string{"usage_patterns=agent:a*|b*"}),
	)

	It("changes nothing without patterns", func() {
		Expect(merge(base, "agent")).To(Equal(base))
		Expect(merge(base, "agent", "", " ")).To(Equal(base))
	})

	DescribeTable("refuses what qmaster would read differently",
		func(group string, pattern string, message string) {
			merged, err := core.MergeUsagePatterns(base, group, pattern)
			Expect(err).To(MatchError(ContainSubstring(message)))
			Expect(merged).To(Equal(base))
		},
		Entry("no group", "", "agent_*", "need a group name"),
		Entry("group with a colon", "a:b", "x*", `group "a:b"`),
		Entry("group with an alternative", "a|b", "x*", `group "a|b"`),
		Entry("group with a blank", "a b", "x*", `group "a b"`),
		Entry("pattern which adds a group", "gpu", "x*;evil:*", `pattern "x*;evil:*"`),
		Entry("pattern which adds a parameter", "gpu", "x*,accounting=false", `pattern "x*,accounting=false"`),
		Entry("pattern with a blank", "gpu", "y z", `pattern "y z"`),
		// Seen on GCS 9.1.7: the record has the key twice and qacct -j
		// prints the standard usage of the job as 0.
		Entry("group named rusage", "rusage", "x*", "standard usage group"),
		Entry("group named eusage", "eusage", "x*", "standard usage group"),
		Entry("group named usage", "usage", "x*", "standard usage group"),
	)
})
