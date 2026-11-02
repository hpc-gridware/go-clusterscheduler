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
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The captures come from a disabled probe queue on a live 9.1.7 cluster:
//
//	slots           6,[ocs-master=2],[@gpu=3]
//	complex_values  matlab=5,scratch=100G,[ocs-worker2=matlab=1],
//	                [@gpu=gpu=2,matlab=4],[@bigmem=scratch=500G]
//	h_vmem          8G,[@bigmem=64G]
//
// qmaster stores the group overrides before the host overrides, whatever
// order they were written in. "qconf -sq zzdom.q@ocs-worker2" printed the
// RESOLVED instance: slots 6, complex_values matlab=1 (the host override
// replaces the whole list and wins over @bigmem), h_vmem 64G (@bigmem).
var _ = Describe("Queue overrides for hosts and host groups (queue domains)", func() {
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		Expect(err).NotTo(HaveOccurred())
		return string(data)
	}
	queue := func(queues []core.ClusterQueueConfig, name string) core.ClusterQueueConfig {
		for _, q := range queues {
			if q.Name == name {
				return q
			}
		}
		Fail("no queue " + name)
		return core.ClusterQueueConfig{}
	}

	It("keeps each override, host or host group, as one token, the multi-value one whole", func() {
		q := queue(core.ParseClusterQueues(read("qconf-sq-domains-9.1.7.txt")), "zzdom.q")
		Expect(q.ComplexValues).To(Equal([]string{"matlab=5", "scratch=100G",
			"[@gpu=gpu=2,matlab=4]", "[@bigmem=scratch=500G]", "[ocs-worker2=matlab=1]"}))
		Expect(q.Slots).To(Equal([]string{"6", "[@gpu=3]", "[ocs-master=2]"}))
		Expect(q.HVmem).To(Equal([]string{"8G", "[@bigmem=64G]"}))
		Expect(q.HostList).To(Equal([]string{"@allhosts"}))
	})

	It("parses the other queue of the same answer independently", func() {
		queues := core.ParseClusterQueues(read("qconf-sq-domains-9.1.7.txt"))
		Expect(queues).To(HaveLen(2))
		all := queue(queues, "all.q")
		Expect(all.ComplexValues).To(BeEmpty(), "all.q has complex_values NONE")
		Expect(all.Slots).NotTo(ContainElement(ContainSubstring("@gpu")), "no override leaks across queues")
	})

	It("writes the overrides back as qmaster stores them", func() {
		q := queue(core.ParseClusterQueues(read("qconf-sq-domains-9.1.7.txt")), "zzdom.q")
		Expect(core.JoinListWithOverrides(q.ComplexValues, ",")).To(Equal(
			"matlab=5,scratch=100G,[@gpu=gpu=2,matlab=4],[@bigmem=scratch=500G],[ocs-worker2=matlab=1]"))
		Expect(core.JoinListWithOverrides(q.Slots, ",")).To(Equal("6,[@gpu=3],[ocs-master=2]"))
	})

	It("reads a queue instance's resolved configuration without overrides", func() {
		q := queue(core.ParseClusterQueues(read("qconf-sq-instance-9.1.7.txt")), "zzdom.q")
		Expect(q.ComplexValues).To(Equal([]string{"matlab=1"}), "the host override replaced the whole list")
		Expect(q.Slots).To(Equal([]string{"6"}))
		Expect(q.HVmem).To(Equal([]string{"64G"}))
	})

	DescribeTable("the override grammar",
		func(line string, want []string) {
			queues := core.ParseClusterQueues("qname t.q\n" + line + "\n")
			Expect(queues).To(HaveLen(1))
			Expect(queues[0].ComplexValues).To(Equal(want))
		},
		Entry("NONE", "complex_values NONE", []string(nil)),
		Entry("only a host-group override", "complex_values [@gpu=gpu=2]", []string{"[@gpu=gpu=2]"}),
		Entry("a nested host group name with dashes", "complex_values a=1,[@gpu-a100=gpu=8,a=2]",
			[]string{"a=1", "[@gpu-a100=gpu=8,a=2]"}),
		Entry("a fully qualified host", "complex_values a=1,[node1.example.com=a=3]",
			[]string{"a=1", "[node1.example.com=a=3]"}),
		Entry("an RSMAP with characteristics inside an override",
			"complex_values [@gpu=gpu=2(gpu0[devices=/dev/nvidia0] gpu1)]",
			[]string{"[@gpu=gpu=2(gpu0[devices=/dev/nvidia0] gpu1)]"}),
	)

	It("splits blocks only at qname lines", func() {
		out := read("qconf-sq-domains-9.1.7.txt")
		Expect(strings.Count(out, "\nqname ") + 1).To(Equal(len(core.ParseClusterQueues(out))))
	})
})
