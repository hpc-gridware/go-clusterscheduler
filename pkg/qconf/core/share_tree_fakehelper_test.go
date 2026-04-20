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

// Reuse demonstration for the extracted fakeqconf helper.
//
// share_tree_impl_offline_test.go consumes the helper via the local
// newFakeQConf/newQConfWith adapters. This file exercises the helper
// package directly so a second test site proves the API is importable
// and stable on its own terms.

package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core/internal/fakeqconf"
)

var _ = Describe("fakeqconf helper (reuse)", func() {
	BeforeEach(func() {
		if !fakeqconf.Available() {
			Skip("fakeqconf uses a bash script; skip on this platform")
		}
	})

	It("captures multi-call argv and feeds stdout back through CommandLineQConf", func() {
		f := fakeqconf.New(GinkgoT(), "/=1\n/P1=100\n", 0)
		defer f.Cleanup()

		qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{
			Executable: f.Path(),
		})
		Expect(err).NotTo(HaveOccurred())

		// Two successive calls should both hit the stub and land in
		// the argv log in call order.
		_, err = qc.ShowShareTreeNodes(nil)
		Expect(err).NotTo(HaveOccurred())
		_, err = qc.ShowShareTreeNodes([]string{"/P1"})
		Expect(err).NotTo(HaveOccurred())

		lines := f.AllArgvLines()
		Expect(lines).To(HaveLen(2))
		Expect(lines[0]).To(Equal("-sstnode /"))
		Expect(lines[1]).To(Equal("-sstnode /P1"))

		// Argv() returns the first call (backwards-compatible with the
		// original helper semantics).
		Expect(f.Argv()).To(Equal([]string{"-sstnode", "/"}))
	})
})
