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

var _ = Describe("Join map helpers", func() {

	Describe("JoinStringStringMap", func() {
		It("returns NONE for an empty map", func() {
			Expect(core.JoinStringStringMap(nil, ",")).To(Equal("NONE"))
			Expect(core.JoinStringStringMap(map[string]string{}, ",")).To(Equal("NONE"))
		})

		It("emits entries in sorted key order", func() {
			m := map[string]string{
				"slots":    "10",
				"mem_free": "1G",
				"arch":     "lx-amd64",
			}
			Expect(core.JoinStringStringMap(m, ",")).To(
				Equal("arch=lx-amd64,mem_free=1G,slots=10"))
		})

		It("is deterministic across repeated calls", func() {
			m := map[string]string{
				"h_vmem": "4G", "s_vmem": "3G", "slots": "8",
				"mem_free": "2G", "arch": "lx-amd64", "num_proc": "16",
			}
			first := core.JoinStringStringMap(m, ",")
			for i := 0; i < 100; i++ {
				Expect(core.JoinStringStringMap(m, ",")).To(Equal(first))
			}
		})
	})

	Describe("JoinStringFloatMap", func() {
		It("returns NONE for an empty map", func() {
			Expect(core.JoinStringFloatMap(nil, ",")).To(Equal("NONE"))
			Expect(core.JoinStringFloatMap(map[string]float64{}, ",")).To(Equal("NONE"))
		})

		It("emits entries in sorted key order", func() {
			m := map[string]float64{
				"np_load_avg": 0.5,
				"mem_free":    2,
				"load_avg":    1.25,
			}
			Expect(core.JoinStringFloatMap(m, ",")).To(
				Equal("load_avg=1.250000,mem_free=2.000000,np_load_avg=0.500000"))
		})

		It("is deterministic across repeated calls", func() {
			m := map[string]float64{
				"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6,
			}
			first := core.JoinStringFloatMap(m, ",")
			for i := 0; i < 100; i++ {
				Expect(core.JoinStringFloatMap(m, ",")).To(Equal(first))
			}
		})
	})
})
