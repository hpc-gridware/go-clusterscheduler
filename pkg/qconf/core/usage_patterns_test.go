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

	It("appends a usage_patterns entry when there is none", func() {
		merged := core.MergeUsagePatterns(base, "agent", "agent_*")
		Expect(merged).To(Equal([]string{"accounting=true", "reporting=false",
			"flush_time=00:00:01", "usage_patterns=agent:agent_*"}))
	})

	It("does not modify the input slice", func() {
		input := append(make([]string, 0, 8), base...)
		input = append(input, "usage_patterns=gpu:nvidia*")
		core.MergeUsagePatterns(input, "gpu", "amd*")
		core.MergeUsagePatterns(input[:3], "agent", "agent_*")
		Expect(input).To(Equal([]string{"accounting=true", "reporting=false",
			"flush_time=00:00:01", "usage_patterns=gpu:nvidia*"}))
	})

	It("joins several patterns of a new group with |", func() {
		merged := core.MergeUsagePatterns(nil, "gpu", "nvidia*", "amd*")
		Expect(merged).To(Equal([]string{"usage_patterns=gpu:nvidia*|amd*"}))
	})

	It("adds a group next to the groups of other tools", func() {
		merged := core.MergeUsagePatterns(
			[]string{"accounting=true", "usage_patterns=gpu:nvidia*|amd*", "joblog=false"},
			"agent", "agent_*")
		Expect(merged).To(Equal([]string{"accounting=true",
			"usage_patterns=gpu:nvidia*|amd*;agent:agent_*", "joblog=false"}))
	})

	It("adds only the missing patterns to an existing group", func() {
		merged := core.MergeUsagePatterns(
			[]string{"usage_patterns=power:power-*;gpu:nvidia*"},
			"gpu", "nvidia*", "amd*")
		Expect(merged).To(Equal([]string{"usage_patterns=power:power-*;gpu:nvidia*|amd*"}))
	})

	It("changes nothing when applied twice", func() {
		once := core.MergeUsagePatterns(base, "agent", "agent_*")
		twice := core.MergeUsagePatterns(once, "agent", "agent_*")
		Expect(twice).To(Equal(once))
	})

	It("does not mistake a group for another one with the same prefix", func() {
		merged := core.MergeUsagePatterns(
			[]string{"usage_patterns=agents:x*"}, "agent", "agent_*")
		Expect(merged).To(Equal([]string{"usage_patterns=agents:x*;agent:agent_*"}))
	})

	It("fills an empty usage_patterns entry", func() {
		merged := core.MergeUsagePatterns([]string{"usage_patterns="}, "agent", "agent_*")
		Expect(merged).To(Equal([]string{"usage_patterns=agent:agent_*"}))
	})

	It("fills a group that has no expression", func() {
		merged := core.MergeUsagePatterns([]string{"usage_patterns=agent:"}, "agent", "agent_*")
		Expect(merged).To(Equal([]string{"usage_patterns=agent:agent_*"}))
	})

	// qmaster keeps the last usage_patterns entry of reporting_params.
	It("changes the last usage_patterns entry, the one the scheduler uses", func() {
		merged := core.MergeUsagePatterns(
			[]string{"usage_patterns=gpu:nvidia*", "usage_patterns=power:p*"},
			"agent", "agent_*")
		Expect(merged).To(Equal([]string{"usage_patterns=gpu:nvidia*",
			"usage_patterns=power:p*;agent:agent_*"}))
	})

	It("changes nothing without a group or without patterns", func() {
		Expect(core.MergeUsagePatterns(base, "", "agent_*")).To(Equal(base))
		Expect(core.MergeUsagePatterns(base, "agent")).To(Equal(base))
	})
})
