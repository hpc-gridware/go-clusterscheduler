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

package core

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Fixtures are verbatim GCS 9.1.4 stdout. Every case below exits 0, so
// the output is the only signal that the change was not applied.
var _ = Describe("checkAttrModification", func() {

	Describe("output that means nothing was changed", func() {

		It("flags an add whose key is already present", func() {
			// The dangerous one: slots=99 was requested, the existing
			// slots value stays, and qconf exits 0.
			out := `No modification because "slots" already exists in "complex_values" of "exechost"`
			err := checkAttrModification(out)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrNoModification)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("slots"))
		})

		It("flags a delete of a value that is not there", func() {
			out := `"nosuchpe" does not exist in "pe_list" of "all.q"` + "\n" +
				`root@master modified "all.q" in cluster queue list`
			err := checkAttrModification(out)
			Expect(errors.Is(err, ErrNoModification)).To(BeTrue())
		})

		It("flags a no-op even when qconf also claims it modified the object", func() {
			// qconf prints both lines; the "modified" line alone would
			// be a misleading success signal.
			out := `No modification because "master" already exists in "hostlist" of "hostgroup"` + "\n" +
				`root@master modified "@allhosts" in host group list`
			Expect(errors.Is(checkAttrModification(out), ErrNoModification)).To(BeTrue())
		})
	})

	Describe("output that means the change was applied", func() {

		It("accepts a plain modification", func() {
			out := `root@master modified "all.q" in cluster queue list`
			Expect(checkAttrModification(out)).To(Succeed())
		})

		It("accepts an empty sublist being populated", func() {
			// Same INFO family as the no-op notices, but this one did
			// change the object -- it must not be flagged.
			out := `"pe_list" of "all.q" is empty - Adding new element(s).` + "\n" +
				`root@master modified "all.q" in cluster queue list`
			Expect(checkAttrModification(out)).To(Succeed())
		})

		It("accepts a modify that fell back to adding a new element", func() {
			out := `Unable to find "make" in "pe_list" of "all.q" - Adding new element.` + "\n" +
				`root@master modified "all.q" in cluster queue list`
			Expect(checkAttrModification(out)).To(Succeed())
		})

		It("accepts empty output, as produced in dry-run mode", func() {
			Expect(checkAttrModification("")).To(Succeed())
		})
	})

	It("lets callers opt into idempotent semantics", func() {
		err := checkAttrModification(
			`"nosuchhost" does not exist in "hostlist" of "hostgroup"`)
		// The end state a caller asked for (value absent) does hold
		// here, so idempotent cleanup code can ignore exactly this.
		if err != nil && !errors.Is(err, ErrNoModification) {
			Fail("unexpected error kind: " + err.Error())
		}
		Expect(errors.Is(err, ErrNoModification)).To(BeTrue())
	})
})
