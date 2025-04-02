/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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

package qconf_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

var _ = Describe("Helper", func() {

	Context("JoinListWithOverrides", func() {

		It("should join a list of elements with a separator", func() {
			Expect(qconf.JoinListWithOverrides(
				[]string{"pe1", "p2", "[host=p2]", "[master=pe1]"}, " ")).
				To(Equal("pe1 p2,[host=p2],[master=pe1]"))
			// different order
			Expect(qconf.JoinListWithOverrides(
				[]string{"p2", "[host=p2]", "p1", "[master=pe1]"}, " ")).
				To(Equal("p2 p1,[host=p2],[master=pe1]"))
			Expect(qconf.JoinListWithOverrides(
				[]string{"pe1", "p2"}, ",")).
				To(Equal("pe1,p2"))
			Expect(qconf.JoinListWithOverrides(
				[]string{"pe1", "p2"}, " ")).
				To(Equal("pe1 p2"))
			Expect(qconf.JoinListWithOverrides(
				[]string{}, ",")).
				To(Equal("NONE"))
		})

	})

	Context("ParseSpaceSeparatedValuesWithOverrides", func() {

		It("should parse a list of elements with a separator", func() {
			output := `qname                 all.q
hostlist              @allhosts
seq_no                0
load_thresholds       np_load_avg=1.75
suspend_thresholds    NONE
nsuspend              1
suspend_interval      00:05:00
priority              0
min_cpu_interval      00:05:00
processors            UNDEFINED
qtype                 BATCH INTERACTIVE
ckpt_list             NONE
pe_list               make test test2,[master=make test2],[global=test]
rerun                 FALSE
slots                 10,[master=14]`
			lines := strings.Split(output, "\n")

			Expect(qconf.ParseSpaceSeparatedValuesWithOverrides(
				lines, 12)).
				To(Equal([]string{"make", "test", "test2", "[master=make test2]", "[global=test]"}))
		})

	})
})
