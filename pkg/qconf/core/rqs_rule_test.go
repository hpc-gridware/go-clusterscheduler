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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("ParseRQSRule", func() {

	slots := func(v string) []core.RQSLimit {
		return []core.RQSLimit{{Resource: "slots", Value: v}}
	}

	DescribeTable("parses valid limit lines",
		func(limit string, want core.RQSRule) {
			rule, err := core.ParseRQSRule(limit)
			Expect(err).NotTo(HaveOccurred())
			Expect(rule).To(Equal(want))
		},
		Entry("to only", "to slots=100",
			core.RQSRule{Limits: slots("100")}),
		Entry("leading limit keyword", "limit to slots=100",
			core.RQSRule{Limits: slots("100")}),
		Entry("users all", "users * to slots=24",
			core.RQSRule{Users: core.RQSScope{Names: []string{"*"}}, Limits: slots("24")}),
		Entry("users expanded all", "users {*} to slots=24",
			core.RQSRule{Users: core.RQSScope{Names: []string{"*"}, Expand: true}, Limits: slots("24")}),
		Entry("named rule with expanded users", "name max_slots_per_user users {*} to slots=24",
			core.RQSRule{Name: "max_slots_per_user",
				Users: core.RQSScope{Names: []string{"*"}, Expand: true}, Limits: slots("24")}),
		Entry("expanded single user", "users {carol} to slots=8",
			core.RQSRule{Users: core.RQSScope{Names: []string{"carol"}, Expand: true}, Limits: slots("8")}),
		Entry("project filter", "projects finance to matlab=5",
			core.RQSRule{Projects: core.RQSScope{Names: []string{"finance"}},
				Limits: []core.RQSLimit{{Resource: "matlab", Value: "5"}}}),
		Entry("acl and hostgroup names", "users @power_users hosts {@gpuhosts} to gpu=2",
			core.RQSRule{Users: core.RQSScope{Names: []string{"@power_users"}},
				Hosts:  core.RQSScope{Names: []string{"@gpuhosts"}, Expand: true},
				Limits: []core.RQSLimit{{Resource: "gpu", Value: "2"}}}),
		Entry("exclusion, several pes, dynamic value",
			"users * pes mpi,smp queues !short.q to slots=$num_proc*2",
			core.RQSRule{Users: core.RQSScope{Names: []string{"*"}},
				PEs:    core.RQSScope{Names: []string{"mpi", "smp"}},
				Queues: core.RQSScope{Excluded: []string{"short.q"}},
				Limits: slots("$num_proc*2")}),
		Entry("exclusion with braces outside", "hosts !{a,b} to slots=1",
			core.RQSRule{Hosts: core.RQSScope{Excluded: []string{"a", "b"}, Expand: true}, Limits: slots("1")}),
		Entry("exclusion with braces inside, mixed", "queues {!short.q,long.q} to slots=1",
			core.RQSRule{Queues: core.RQSScope{Names: []string{"long.q"}, Excluded: []string{"short.q"}, Expand: true},
				Limits: slots("1")}),
		Entry("several users, xusers and several resources",
			"name rule2 users alice,bob,!charlie to h_vmem=4G,slots=10",
			core.RQSRule{Name: "rule2",
				Users:  core.RQSScope{Names: []string{"alice", "bob"}, Excluded: []string{"charlie"}},
				Limits: []core.RQSLimit{{Resource: "h_vmem", Value: "4G"}, {Resource: "slots", Value: "10"}}}),
		Entry("extra whitespace", "  name   r1   users   {*}   to   slots=24 ,  h_vmem=4G  ",
			core.RQSRule{Name: "r1", Users: core.RQSScope{Names: []string{"*"}, Expand: true},
				Limits: []core.RQSLimit{{Resource: "slots", Value: "24"}, {Resource: "h_vmem", Value: "4G"}}}),
		Entry("all five filters", "users u projects p pes e queues q hosts h to slots=1",
			core.RQSRule{
				Users:    core.RQSScope{Names: []string{"u"}},
				Projects: core.RQSScope{Names: []string{"p"}},
				PEs:      core.RQSScope{Names: []string{"e"}},
				Queues:   core.RQSScope{Names: []string{"q"}},
				Hosts:    core.RQSScope{Names: []string{"h"}},
				Limits:   slots("1")}),
	)

	DescribeTable("rejects invalid limit lines",
		func(limit, wantErr string) {
			_, err := core.ParseRQSRule(limit)
			Expect(err).To(MatchError(ContainSubstring(wantErr)))
		},
		Entry("empty string", "", "empty"),
		Entry("only whitespace", "   ", "empty"),
		Entry("only limit keyword", "limit", "empty"),
		Entry("missing to", "users alice slots=10", "unknown keyword"),
		Entry("filters but no to", "users alice", "missing \"to\""),
		Entry("unknown keyword", "user alice to slots=1", "unknown keyword \"user\""),
		Entry("keyword without value", "users", "without value"),
		Entry("duplicate keyword", "users a users b to slots=1", "given twice"),
		Entry("to without limits", "users alice to", "no limits"),
		Entry("limit without equals", "to slots", "want resource=value"),
		Entry("limit without value", "to slots=", "want resource=value"),
		Entry("limit with two equals", "to slots=1=2", "want resource=value"),
		Entry("unterminated braces", "users {a,b to slots=1", "unterminated"),
		Entry("empty braces", "users {} to slots=1", "empty scope"),
		Entry("empty name in list", "users a,,b to slots=1", "invalid name"),
		Entry("bare exclamation mark", "users ! to slots=1", "invalid name"),
	)

	It("reports Filtered only for scopes with names", func() {
		Expect(core.RQSScope{}.Filtered()).To(BeFalse())
		Expect(core.RQSScope{Names: []string{"*"}}.Filtered()).To(BeTrue())
		Expect(core.RQSScope{Excluded: []string{"a"}}.Filtered()).To(BeTrue())
	})
})

var _ = Describe("ParseRQSRules", func() {

	It("numbers unnamed rules by their 1-based position and keeps explicit names", func() {
		rules, err := core.ParseRQSRules([]string{
			"users {*} to slots=24",
			"name gpus users {*} to gpu=2",
			"to slots=100",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(HaveLen(3))
		Expect(rules[0].Name).To(Equal("1"))
		Expect(rules[1].Name).To(Equal("gpus"))
		Expect(rules[2].Name).To(Equal("3"))
	})

	It("returns an empty slice for no limits", func() {
		rules, err := core.ParseRQSRules(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})

	It("fails on the first invalid rule", func() {
		_, err := core.ParseRQSRules([]string{"to slots=1", "users alice"})
		Expect(err).To(MatchError(ContainSubstring("missing \"to\"")))
	})

	It("parses every limit of the qconf -srqs fixture", func() {
		data, err := os.ReadFile(filepath.Join("testdata", "qconf-srqs-all-9.1.7.txt"))
		Expect(err).NotTo(HaveOccurred())
		sets := core.ParseResourceQuotaSets(string(data))
		Expect(sets).NotTo(BeEmpty())
		for _, set := range sets {
			rules, err := core.ParseRQSRules(set.Limits)
			Expect(err).NotTo(HaveOccurred(), set.Name)
			Expect(rules).To(HaveLen(len(set.Limits)), set.Name)
			for _, rule := range rules {
				Expect(rule.Limits).NotTo(BeEmpty(), set.Name)
			}
		}
	})
})
