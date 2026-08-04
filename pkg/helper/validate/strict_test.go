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

package validate_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

var _ = Describe("strict charset validators", func() {

	Describe("ObjectName (KEY_TABLE)", func() {
		DescribeTable("accepts documented object names and @-forms",
			func(in string) { Expect(validate.ObjectName(in)).ToNot(HaveOccurred()) },
			Entry("queue", "all.q"),
			Entry("underscore", "my_queue"),
			Entry("hyphen", "pe-2024"),
			Entry("plus", "proj+1"),
			Entry("equals", "slots=8"),
			Entry("host group", "@allhosts"),
			Entry("queue instance", "all.q@master"),
			Entry("hostgroup instance", "all.q@@lx-hosts"),
		)
		DescribeTable("rejects forbidden characters and reserved words",
			func(in string) { Expect(validate.ObjectName(in)).To(HaveOccurred()) },
			Entry("slash", "a/b"),
			Entry("comma", "a,b"),
			Entry("space", "all q"),
			Entry("leading dot", ".ssh"),
			Entry("leading hash", "#c"),
			Entry("reserved NONE", "NONE"),
			Entry("reserved all", "all"),
			Entry("empty", ""),
			Entry("only at signs", "@@"),
			Entry("segment with slash after @", "cq@a/b"),
			Entry("too long", strings.Repeat("a", 513)),
		)
		It("accepts a 512-char name (KEY_TABLE uses '>' so 512 is allowed)", func() {
			Expect(validate.ObjectName(strings.Repeat("a", 512))).ToNot(HaveOccurred())
		})
	})

	Describe("HostName", func() {
		DescribeTable("accepts real host names",
			func(in string) { Expect(validate.HostName(in)).ToNot(HaveOccurred()) },
			Entry("plain", "master"),
			Entry("underscore", "node_07"),
			Entry("fqdn", "node07.cluster.example.com"),
		)
		DescribeTable("rejects '@' and space",
			func(in string) { Expect(validate.HostName(in)).To(HaveOccurred()) },
			Entry("at", "host@x"),
			Entry("space", "a b"),
		)
	})

	Describe("JobName (QSUB_TABLE)", func() {
		DescribeTable("accepts permissive job names",
			func(in string) { Expect(validate.JobName(in)).ToNot(HaveOccurred()) },
			Entry("plain", "myjob"),
			Entry("with space", "my job"),
			Entry("with dot", "job.1"),
		)
		DescribeTable("rejects leading digit and forbidden chars",
			func(in string) { Expect(validate.JobName(in)).To(HaveOccurred()) },
			Entry("leading digit", "1job"),
			Entry("slash", "a/b"),
			Entry("at", "a@b"),
			Entry("star", "a*b"),
		)
		It("rejects a 512-char job name (qmaster uses '>=', so max is 511)", func() {
			Expect(validate.JobName(strings.Repeat("a", 512))).To(HaveOccurred())
			Expect(validate.JobName(strings.Repeat("a", 511))).ToNot(HaveOccurred())
		})
	})

	Describe("Strict* gates", func() {
		AfterEach(func() { GinkgoT().Setenv(validate.EnvVar, "") })

		It("are no-ops outside strict mode", func() {
			GinkgoT().Setenv(validate.EnvVar, "")
			Expect(validate.StrictObjectName("all.q@host")).ToNot(HaveOccurred())
			Expect(validate.StrictHostName("a b")).ToNot(HaveOccurred())
			Expect(validate.StrictJobName("1job")).ToNot(HaveOccurred())
		})
		It("apply their charset in strict mode", func() {
			GinkgoT().Setenv(validate.EnvVar, "strict")
			Expect(validate.StrictObjectName("all q")).To(HaveOccurred())
			Expect(validate.StrictObjectName("all.q")).ToNot(HaveOccurred())
			Expect(validate.StrictObjectName("@allhosts")).ToNot(HaveOccurred())
			Expect(validate.StrictHostName("host@x")).To(HaveOccurred())
			Expect(validate.StrictHostName("node_07")).ToNot(HaveOccurred())
			Expect(validate.StrictJobName("1job")).To(HaveOccurred())
			Expect(validate.StrictJobName("my job")).ToNot(HaveOccurred())
		})
	})
})
