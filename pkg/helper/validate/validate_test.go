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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

var _ = Describe("validate", func() {

	Describe("NoControl", func() {
		DescribeTable("rejects control characters and invalid UTF-8",
			func(in string) {
				Expect(validate.NoControl(in)).To(HaveOccurred())
			},
			Entry("nul", "node\x00rm"),
			Entry("newline", "node\nfoo"),
			Entry("carriage return", "node\rfoo"),
			Entry("tab", "node\tfoo"),
			Entry("vertical tab", "node\vfoo"),
			Entry("form feed", "node\ffoo"),
			Entry("escape", "node\x1bfoo"),
			Entry("del", "node\x7f"),
			Entry("c1 control", "node\x85foo"),
			Entry("invalid utf-8", "node\xfffoo"),
		)

		DescribeTable("accepts printable values including multi-byte UTF-8",
			func(in string) {
				Expect(validate.NoControl(in)).ToNot(HaveOccurred())
			},
			Entry("plain", "master"),
			Entry("comma joined list", "all.q,gpu.q"),
			Entry("name value", "slots=8"),
			Entry("space", "job name"),
			Entry("leading dash ok for NoControl", "-help"),
			Entry("multibyte utf-8", "n\u00f6de"),
		)
	})

	Describe("Operand", func() {
		DescribeTable("rejects empty, leading dash, and control characters",
			func(in string) {
				Expect(validate.Operand(in)).To(HaveOccurred())
			},
			Entry("empty", ""),
			Entry("leading hyphen", "-help"),
			Entry("leading hyphen subcommand", "-ke"),
			Entry("newline", "node\nfoo"),
			Entry("nul", "node\x00"),
		)

		DescribeTable("accepts real cluster identifiers",
			func(in string) {
				Expect(validate.Operand(in)).ToNot(HaveOccurred())
			},
			Entry("plain host", "master"),
			Entry("underscore host", "node_07"),
			Entry("fqdn", "node07.cluster.example.com"),
			Entry("queue", "all.q"),
			Entry("user with dot", "first.last"),
			Entry("hostgroup", "@allhosts"),
			Entry("queue instance", "all.q@node07"),
			Entry("mid hyphen", "pe-2024"),
			Entry("equals in name is allowed", "slots=8"),
		)
	})

	Describe("ListElement", func() {
		DescribeTable("rejects list separators, leading dash, and controls",
			func(in string) {
				Expect(validate.ListElement(in)).To(HaveOccurred())
			},
			Entry("comma", "node,foo"),
			Entry("space", "node foo"),
			Entry("leading hyphen", "-ke"),
			Entry("empty", ""),
			Entry("newline", "node\nfoo"),
		)

		DescribeTable("accepts single list elements",
			func(in string) {
				Expect(validate.ListElement(in)).ToNot(HaveOccurred())
			},
			Entry("host", "node07"),
			Entry("queue instance", "all.q@node07"),
			Entry("hostgroup instance", "all.q@@lx-hosts"),
		)
	})

	Describe("SplitAndValidateList", func() {
		It("accepts a legitimate multi-object comma list", func() {
			Expect(validate.SplitAndValidateList("all.q,gpu.q")).ToNot(HaveOccurred())
		})
		It("rejects a forged leading-dash element", func() {
			// "goodq,-ke somehost" would forge a -ke subcommand.
			Expect(validate.SplitAndValidateList("goodq,-ke somehost")).To(HaveOccurred())
		})
		It("rejects an element containing a space", func() {
			Expect(validate.SplitAndValidateList("goodq -ke host")).To(HaveOccurred())
		})
	})

	Describe("List", func() {
		It("accepts clean elements", func() {
			Expect(validate.List("queue", []string{"all.q", "gpu.q@node01"})).ToNot(HaveOccurred())
		})
		It("rejects a comma-injected element", func() {
			Expect(validate.List("queue", []string{"all.q,evil"})).To(HaveOccurred())
		})
		It("rejects a leading-dash element", func() {
			Expect(validate.List("host", []string{"-x"})).To(HaveOccurred())
		})
	})

	Describe("JobTaskList", func() {
		DescribeTable("accepts job id / range / name / wildcard tokens",
			func(in string) {
				Expect(validate.JobTaskList(in)).ToNot(HaveOccurred())
			},
			Entry("single id", "42"),
			Entry("id with task range", "42.1-10:2"),
			Entry("comma list", "1,2,3"),
			Entry("space list", "1 2 3"),
			Entry("job name", "myjob"),
			Entry("wildcard", "job*"),
			Entry("boolean wildcard", "(lx*|sol*)&*64*"),
			Entry("all jobs", "*"),
			Entry("empty", ""),
		)
		DescribeTable("rejects forged flag elements and control chars",
			func(in string) {
				Expect(validate.JobTaskList(in)).To(HaveOccurred())
			},
			Entry("leading-dash element after comma", "1,-t"),
			Entry("leading-dash element after space", "1 -f"),
			Entry("newline", "1\n2"),
		)
	})

	Describe("NameValueKey / NameValueValue", func() {
		DescribeTable("key rejects '=', ',', and leading dash",
			func(in string) {
				Expect(validate.NameValueKey(in)).To(HaveOccurred())
			},
			Entry("equals", "sl=ots"),
			Entry("comma", "slots,x"),
			Entry("leading hyphen", "-l"),
			Entry("control", "slots\n"),
		)
		It("key accepts a plain resource name", func() {
			Expect(validate.NameValueKey("slots")).ToNot(HaveOccurred())
		})
		It("value allows '=' but rejects ','", func() {
			Expect(validate.NameValueValue("a=b")).ToNot(HaveOccurred())
			Expect(validate.NameValueValue("a,b")).To(HaveOccurred())
		})
	})

	Describe("LocalFileName", func() {
		DescribeTable("rejects traversal and separators",
			func(in string) {
				Expect(validate.LocalFileName(in)).To(HaveOccurred())
			},
			Entry("dotdot", ".."),
			Entry("slash", "a/b"),
			Entry("backslash", "a\\b"),
			Entry("absolute", "/etc/passwd"),
			Entry("empty", ""),
			Entry("nul", "a\x00b"),
		)
		DescribeTable("accepts plain object names",
			func(in string) {
				Expect(validate.LocalFileName(in)).ToNot(HaveOccurred())
			},
			Entry("queue", "all.q"),
			Entry("hostgroup", "@allhosts"),
			Entry("user", "root"),
		)
	})

	Describe("Args (Layer 1)", func() {
		It("passes assembled tokens with commas and equals", func() {
			Expect(validate.Args("-mattr", "queue", "slots", "8", "all.q,gpu.q")).ToNot(HaveOccurred())
		})
		It("rejects a token containing a control character", func() {
			err := validate.Args("-mattr", "queue", "slots", "8\nrm", "all.q")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("argument 3"))
		})
	})

	Describe("Enforce (mode gate)", func() {
		var sentinel = errors.New("boom")

		AfterEach(func() {
			GinkgoT().Setenv(validate.EnvVar, "")
		})

		It("returns the error in the default (enforce) mode", func() {
			GinkgoT().Setenv(validate.EnvVar, "")
			Expect(validate.Enforce(sentinel)).To(MatchError(sentinel))
		})
		It("returns nil for a nil input", func() {
			Expect(validate.Enforce(nil)).ToNot(HaveOccurred())
		})
		It("suppresses the error and logs in warn mode", func() {
			var logged []string
			orig := validate.WarnLogger
			validate.WarnLogger = func(msg string) { logged = append(logged, msg) }
			defer func() { validate.WarnLogger = orig }()

			GinkgoT().Setenv(validate.EnvVar, "warn")
			Expect(validate.Enforce(sentinel)).ToNot(HaveOccurred())
			Expect(logged).To(HaveLen(1))
			Expect(logged[0]).To(ContainSubstring("boom"))
		})
		It("suppresses the error in off mode", func() {
			GinkgoT().Setenv(validate.EnvVar, "off")
			Expect(validate.Enforce(sentinel)).ToNot(HaveOccurred())
		})
		It("blocks (like enforce) in strict mode until the charset lands", func() {
			GinkgoT().Setenv(validate.EnvVar, "strict")
			Expect(validate.Enforce(sentinel)).To(MatchError(sentinel))
		})
	})
})
