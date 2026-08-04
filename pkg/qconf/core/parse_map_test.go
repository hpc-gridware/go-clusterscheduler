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

var _ = Describe("Parse map helpers", func() {

	Describe("ParseIntoStringStringMap", func() {
		It("parses key=value pairs", func() {
			m, err := core.ParseIntoStringStringMap("slots=10,mem_free=1G", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(Equal(map[string]string{
				"slots":    "10",
				"mem_free": "1G",
			}))
		})

		It("returns an empty map for NONE and empty input", func() {
			m, err := core.ParseIntoStringStringMap("NONE", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(BeEmpty())

			m, err = core.ParseIntoStringStringMap("", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(BeEmpty())
		})

		It("keeps '=' inside values", func() {
			m, err := core.ParseIntoStringStringMap(
				"docker_options=--env=FOO,slots=4", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(HaveKeyWithValue("docker_options", "--env=FOO"))
			Expect(m).To(HaveKeyWithValue("slots", "4"))
		})

		It("trims whitespace around pairs and skips empty tokens", func() {
			m, err := core.ParseIntoStringStringMap(" a=1, b=2, ", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(Equal(map[string]string{"a": "1", "b": "2"}))
		})

		It("errors on a token without '=' instead of dropping it", func() {
			_, err := core.ParseIntoStringStringMap("a=1,broken,b=2", ",")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("broken"))
		})

		It("errors on an empty key", func() {
			_, err := core.ParseIntoStringStringMap("=1", ",")
			Expect(err).To(HaveOccurred())
		})

		It("round-trips through JoinStringStringMap", func() {
			in := map[string]string{"slots": "10", "gpu": "2"}
			m, err := core.ParseIntoStringStringMap(
				core.JoinStringStringMap(in, ","), ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(Equal(in))
		})
	})

	Describe("ParseIntoStringFloatMap", func() {
		It("parses key=value pairs", func() {
			m, err := core.ParseIntoStringFloatMap(
				"np_load_avg=0.500000,mem_free=2.0", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(Equal(map[string]float64{
				"np_load_avg": 0.5,
				"mem_free":    2.0,
			}))
		})

		It("returns an empty map for NONE and empty input", func() {
			m, err := core.ParseIntoStringFloatMap("NONE", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(BeEmpty())

			m, err = core.ParseIntoStringFloatMap("", ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(BeEmpty())
		})

		It("errors on a token without '=' instead of dropping it", func() {
			_, err := core.ParseIntoStringFloatMap("a=1.0,broken", ",")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("broken"))
		})

		It("errors on a non-numeric value instead of dropping it", func() {
			_, err := core.ParseIntoStringFloatMap("a=1.0,b=fast", ",")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("b=fast"))
		})

		It("round-trips through JoinStringFloatMap", func() {
			in := map[string]float64{"np_load_avg": 0.5, "mem_free": 2.0}
			m, err := core.ParseIntoStringFloatMap(
				core.JoinStringFloatMap(in, ","), ",")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(Equal(in))
		})
	})
})
