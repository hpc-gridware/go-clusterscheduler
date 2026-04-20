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

// Offline tests for CommandLineQConf share-tree wrappers.
//
// These specs exercise argument construction and stdout-parsing logic
// without touching a cluster. A per-spec fake "qconf" script captures
// argv and emits a fixture body, letting us assert:
//
//   - /Root prefix stripping for -sstnode / -astnode / -mstnode / -dstnode,
//   - -mstnode partial-success detection (stdout "Unable to locate"
//     must produce an error even when rc=0),
//   - ShowShareTree "no sharetree element" mapping to ErrNoShareTree,
//   - parseShareTreeNodes behaviour on the canonical /=1 format.

package core_test

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core/internal/fakeqconf"
)

// newFakeQConf wraps fakeqconf.New so specs keep their short call site
// and get an automatic Skip when the platform does not support the
// bash-script stub.
func newFakeQConf(stdout string, rc int) *fakeqconf.Fake {
	if !fakeqconf.Available() {
		Skip("fakeqconf uses a bash script; skip on this platform")
	}
	return fakeqconf.New(GinkgoT(), stdout, rc)
}

// newQConfWith builds a CommandLineQConf that shells out to the fake
// script instead of a real qconf binary.
func newQConfWith(f *fakeqconf.Fake) *core.CommandLineQConf {
	qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{
		Executable: f.Path(),
	})
	Expect(err).NotTo(HaveOccurred())
	return qc
}

var _ = Describe("CommandLineQConf share-tree wrappers (offline)", func() {

	Describe("ShowShareTreeNodes", func() {
		It("passes '/' when no paths are given and parses the /=N output", func() {
			f := newFakeQConf("/=1\n/default=10\n/P1=100\n/P2=100\n", 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			nodes, err := qc.ShowShareTreeNodes(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Argv()).To(Equal([]string{"-sstnode", "/"}))
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P1", Share: 100}))
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/default", Share: 10}))
		})

		It("strips /Root prefix from each input path before invoking qconf", func() {
			f := newFakeQConf("/P1=100\n", 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			_, err := qc.ShowShareTreeNodes([]string{"/Root/P1", "Root/P2", "/P3"})
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Argv()).To(Equal([]string{"-sstnode", "/P1", "/P2", "/P3"}))
		})
	})

	Describe("AddShareTreeNode", func() {
		It("emits -astnode with the /Root prefix removed", func() {
			f := newFakeQConf("modified sharetree\n", 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			Expect(qc.AddShareTreeNode(core.ShareTreeNode{
				Node: "/Root/P1", Share: 42,
			})).To(Succeed())
			Expect(f.Argv()).To(Equal([]string{"-astnode", "/P1=42"}))
		})
	})

	Describe("ModifyShareTreeNodes", func() {
		It("packs all entries into a comma-separated -mstnode arg", func() {
			f := newFakeQConf("setting \n/P1=1\n/P2=2\nmodified sharetree\n", 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			Expect(qc.ModifyShareTreeNodes([]core.ShareTreeNode{
				{Node: "/Root/P1", Share: 1},
				{Node: "/P2", Share: 2},
			})).To(Succeed())
			Expect(f.Argv()).To(Equal([]string{"-mstnode", "/P1=1,/P2=2"}))
		})

		It("returns an error when qconf reports 'Unable to locate' even with rc=0", func() {
			// Live qmaster emits this when one of the paths in a bulk
			// update is missing; rc is still 0 because the other writes
			// went through.
			f := newFakeQConf("setting \n/P1=1\nUnable to locate /bogus in sharetree\nmodified sharetree\n", 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			err := qc.ModifyShareTreeNodes([]core.ShareTreeNode{
				{Node: "/P1", Share: 1},
				{Node: "/bogus", Share: 2},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("share tree node not found"))
		})

		It("refuses empty input", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)
			Expect(qc.ModifyShareTreeNodes(nil)).To(HaveOccurred())
		})

		It("rejects comma/equal injection in Node before argv assembly", func() {
			// Without validation, Node="/P1=999,/victim" with Share=1
			// would produce argv `-mstnode /P1=999,/victim=1` and
			// silently modify /victim. The wrapper must refuse this
			// BEFORE shelling out, so the fake qconf is never invoked.
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)
			err := qc.ModifyShareTreeNodes([]core.ShareTreeNode{
				{Node: "/P1=999,/victim", Share: 1},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid character"))
			Expect(f.AllArgvLines()).To(BeEmpty(),
				"qconf must not be invoked when a path is tainted")
		})
	})

	Describe("AddShareTreeNode path validation", func() {
		It("rejects '=' in the node path", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)
			err := qc.AddShareTreeNode(core.ShareTreeNode{
				Node: "/P1=evil", Share: 1,
			})
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("rejects newline in the node path", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)
			err := qc.AddShareTreeNode(core.ShareTreeNode{
				Node: "/P1\n/victim", Share: 1,
			})
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})
	})

	Describe("DeleteShareTreeNodes", func() {
		It("strips /Root from each path and passes them as separate args", func() {
			f := newFakeQConf("removed nodes\n", 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			Expect(qc.DeleteShareTreeNodes([]string{"/Root/P1", "/P2"})).To(Succeed())
			Expect(f.Argv()).To(Equal([]string{"-dstnode", "/P1", "/P2"}))
		})

		It("refuses empty input", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)
			Expect(qc.DeleteShareTreeNodes(nil)).To(HaveOccurred())
		})
	})

	Describe("ShowShareTree", func() {
		It("returns the raw sstree output on success", func() {
			raw, err := os.ReadFile("testdata/share_tree_real_sstree.txt")
			Expect(err).NotTo(HaveOccurred())
			f := newFakeQConf(string(raw), 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			got, err := qc.ShowShareTree()
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(ContainSubstring("name=Root"))
			Expect(got).To(ContainSubstring("childnodes=1,2,3"))
			Expect(f.Argv()).To(Equal([]string{"-sstree"}))
		})

		It("maps 'no sharetree element' to a 'no sharetree' error", func() {
			f := newFakeQConf("no sharetree element\n", 1)
			defer f.Cleanup()

			qc := newQConfWith(f)
			_, err := qc.ShowShareTree()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no sharetree"))
		})
	})

	Describe("ShowShareTreeStructured", func() {
		It("parses the real -sstree capture into a StructuredShareTree", func() {
			raw, err := os.ReadFile("testdata/share_tree_real_sstree.txt")
			Expect(err).NotTo(HaveOccurred())
			f := newFakeQConf(string(raw), 0)
			defer f.Cleanup()

			qc := newQConfWith(f)
			tree, err := qc.ShowShareTreeStructured()
			Expect(err).NotTo(HaveOccurred())
			Expect(tree.Root.Name).To(Equal("Root"))
			names := []string{}
			for _, c := range tree.Root.Children {
				names = append(names, c.Name)
			}
			Expect(names).To(ContainElement("default"))
			Expect(names).To(ContainElement("P1"))
			Expect(names).To(ContainElement("P2"))
		})

		It("returns ErrNoShareTree when qconf reports no sharetree element", func() {
			f := newFakeQConf("no sharetree element\n", 1)
			defer f.Cleanup()
			qc := newQConfWith(f)
			_, err := qc.ShowShareTreeStructured()
			Expect(errors.Is(err, core.ErrNoShareTree)).To(BeTrue())
		})
	})
})
