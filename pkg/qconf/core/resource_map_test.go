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

var _ = Describe("ParseResourceMap", func() {

	ids := func(in []core.ResourceMapInstance) []string {
		out := make([]string, 0, len(in))
		for _, i := range in {
			out = append(out, i.ID)
		}
		return out
	}

	It("parses bare ids", func() {
		amount, inst, err := core.ParseResourceMap("2(gpu0 gpu1)")
		Expect(err).NotTo(HaveOccurred())
		Expect(amount).To(Equal(2))
		Expect(ids(inst)).To(Equal([]string{"gpu0", "gpu1"}))
		Expect(inst[0].Characteristics).To(BeNil())
	})

	It("expands an integer range", func() {
		_, inst, err := core.ParseResourceMap("4(0-3)")
		Expect(err).NotTo(HaveOccurred())
		Expect(ids(inst)).To(Equal([]string{"0", "1", "2", "3"}))
	})

	It("names the instances 0..amount-1 when the id list is left out", func() {
		amount, inst, err := core.ParseResourceMap("3")
		Expect(err).NotTo(HaveOccurred())
		Expect(amount).To(Equal(3))
		Expect(ids(inst)).To(Equal([]string{"0", "1", "2"}))
	})

	It("parses a characteristics block per instance", func() {
		_, inst, err := core.ParseResourceMap(
			"2(gpu0[device=/dev/nvidia0,memory=80G] gpu1[device=/dev/nvidia1,memory=80G])")
		Expect(err).NotTo(HaveOccurred())
		Expect(ids(inst)).To(Equal([]string{"gpu0", "gpu1"}))
		Expect(inst[0].Characteristics).To(Equal(map[string]string{
			"device": "/dev/nvidia0", "memory": "80G",
		}))
		Expect(inst[1].Characteristics).To(HaveKeyWithValue("device", "/dev/nvidia1"))
	})

	It("keeps the device isolation list intact (';' and ':' are part of the value)", func() {
		_, inst, err := core.ParseResourceMap(
			"2(gpu0[devices=/dev/nvidia0:rw;/dev/nvidiactl:r] gpu1[devices=/dev/nvidia1:rw;/dev/nvidiactl:r])")
		Expect(err).NotTo(HaveOccurred())
		Expect(inst[0].Characteristics).To(HaveKeyWithValue("devices", "/dev/nvidia0:rw;/dev/nvidiactl:r"))
		Expect(inst[1].Characteristics).To(HaveKeyWithValue("devices", "/dev/nvidia1:rw;/dev/nvidiactl:r"))
	})

	It("mixes bare and annotated instances", func() {
		_, inst, err := core.ParseResourceMap("2(gpu0[devices=/dev/nvidia0:rw] gpu1)")
		Expect(err).NotTo(HaveOccurred())
		Expect(inst[0].Characteristics).To(HaveKey("devices"))
		Expect(inst[1].Characteristics).To(BeNil())
	})

	It("ignores whitespace inside a characteristics block, as a continuation leaves it", func() {
		_, inst, err := core.ParseResourceMap(
			"2(gpu0[device=/dev/nvidia0, memory=80G] gpu1[ device=/dev/nvidia1 ,memory=80G ])")
		Expect(err).NotTo(HaveOccurred())
		Expect(ids(inst)).To(Equal([]string{"gpu0", "gpu1"}))
		Expect(inst[0].Characteristics).To(HaveKeyWithValue("memory", "80G"))
		Expect(inst[1].Characteristics).To(HaveKeyWithValue("device", "/dev/nvidia1"))
	})

	It("keeps a duplicated id, which models a shared device", func() {
		amount, inst, err := core.ParseResourceMap("2(gpu0 gpu0)")
		Expect(err).NotTo(HaveOccurred())
		Expect(amount).To(Equal(2))
		Expect(ids(inst)).To(Equal([]string{"gpu0", "gpu0"}))
	})

	DescribeTable("rejects malformed values",
		func(value, want string) {
			_, _, err := core.ParseResourceMap(value)
			Expect(err).To(HaveOccurred())
			Expect(strings.ToLower(err.Error())).To(ContainSubstring(want))
		},
		Entry("non-numeric amount", "two(gpu0)", "amount"),
		Entry("unclosed id list", "2(gpu0 gpu1", ")"),
		Entry("unclosed characteristics block", "1(gpu0[device=/dev/nvidia0)", "]"),
		Entry("characteristic without '='", "1(gpu0[device])", "device"),
		Entry("characteristics on a range", "2(0-1[device=/dev/nvidia0])", "range"),
	)
})

var _ = Describe("ParseExecHostConfigFromLines with RSMAP characteristics", func() {
	It("keeps the whole RSMAP entry under its own key", func() {
		cfg, err := core.ParseExecHostConfigFromLines([]string{
			"hostname              node01",
			"complex_values        slots=24,gpu=2(gpu0[device=/dev/nvidia0,memory=24G] gpu1[device=/dev/nvidia1,memory=24G])",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ComplexValues).To(HaveLen(2))
		_, inst, err := core.ParseResourceMap(cfg.ComplexValues["gpu"])
		Expect(err).NotTo(HaveOccurred())
		Expect(inst).To(HaveLen(2))
		Expect(inst[1].Characteristics).To(HaveKeyWithValue("memory", "24G"))
	})
})
