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

var _ = Describe("ParseExecHostConfigFromLines", func() {

	// Representative qconf -se output. load_values and processors are
	// runtime attributes reported by the execd; qconf -Ae/-Me reject
	// them, so parsing must not let them reach ExtraFields.
	seOutput := `hostname              master
load_scaling          NONE
complex_values        test_mem=1024
load_values           arch=lx-amd64,num_proc=8,mem_total=15990.777344M
processors            8
user_lists            arusers deadlineusers
xuser_lists           NONE
projects              NONE
xprojects             NONE
usage_scaling         NONE
report_variables      NONE`

	It("parses typed fields from qconf -se output", func() {
		cfg, err := core.ParseExecHostConfigFromLines(strings.Split(seOutput, "\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Name).To(Equal("master"))
		Expect(cfg.ComplexValues).To(HaveKeyWithValue("test_mem", "1024"))
		Expect(cfg.UserLists).To(Equal([]string{"arusers", "deadlineusers"}))
	})

	It("does not round-trip read-only runtime attributes", func() {
		cfg, err := core.ParseExecHostConfigFromLines(strings.Split(seOutput, "\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ExtraFields).NotTo(HaveKey("load_values"))
		Expect(cfg.ExtraFields).NotTo(HaveKey("processors"))
		Expect(cfg.ExtraFields).To(BeEmpty())
	})

	It("still captures unknown settable attributes into ExtraFields", func() {
		lines := strings.Split(seOutput+"\nfuture_param 42", "\n")
		cfg, err := core.ParseExecHostConfigFromLines(lines)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ExtraFields).To(HaveKeyWithValue("future_param", "42"))
		Expect(cfg.ExtraFields).NotTo(HaveKey("load_values"))
	})

	It("rejects malformed complex_values instead of dropping entries", func() {
		lines := strings.Split(strings.Replace(seOutput,
			"complex_values        test_mem=1024",
			"complex_values        test_mem=1024,broken", 1), "\n")
		_, err := core.ParseExecHostConfigFromLines(lines)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("complex_values"))
		Expect(err.Error()).To(ContainSubstring("broken"))
	})

	It("rejects malformed load_scaling values", func() {
		lines := strings.Split(strings.Replace(seOutput,
			"load_scaling          NONE",
			"load_scaling          np_load_avg=abc", 1), "\n")
		_, err := core.ParseExecHostConfigFromLines(lines)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("load_scaling"))
	})
})
