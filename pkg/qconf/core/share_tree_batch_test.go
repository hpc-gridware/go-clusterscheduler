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

// Offline tests for the subtree-batch API. The point of the batch is
// one qconf read + one qconf write for many edits; these specs pin
// that round-trip count via the fakeQConf harness.

package core_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("ApplySubtreeBatch (pure)", func() {
	var base *core.StructuredShareTree

	BeforeEach(func() {
		base = &core.StructuredShareTree{
			Root: &core.StructuredShareTreeNode{
				Name: "Root", Shares: 1,
				Children: []*core.StructuredShareTreeNode{
					{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
					{Name: "P1", Type: core.ShareTreeNodeProject, Shares: 100},
					{Name: "P2", Type: core.ShareTreeNodeProject, Shares: 100},
				},
			},
		}
	})

	It("applies a sequence of add, replace, delete and returns the final tree", func() {
		ops := []core.SubtreeOp{
			{
				Kind: core.SubtreeOpAdd,
				Path: "/Root",
				Subtree: &core.StructuredShareTreeNode{
					Name: "P3", Type: core.ShareTreeNodeProject, Shares: 50,
				},
			},
			{
				Kind: core.SubtreeOpReplace,
				Path: "/Root/P1",
				Subtree: &core.StructuredShareTreeNode{
					Name: "P1", Type: core.ShareTreeNodeProject, Shares: 999,
				},
			},
			{
				Kind: core.SubtreeOpDelete,
				Path: "/Root/P2",
			},
		}
		got, err := core.ApplySubtreeBatch(base, ops, nil)
		Expect(err).NotTo(HaveOccurred())

		names := []string{}
		for _, c := range got.Root.Children {
			names = append(names, c.Name)
		}
		Expect(names).To(ContainElement("default"))
		Expect(names).To(ContainElement("P3"))
		Expect(names).NotTo(ContainElement("P2"))

		p1, _, _ := core.FindNodeByPath(got.Root, "/Root/P1")
		Expect(p1).NotTo(BeNil())
		Expect(p1.Shares).To(Equal(999))
	})

	It("rejects a nil / rootless input tree with ErrNoShareTree", func() {
		_, err := core.ApplySubtreeBatch(nil, []core.SubtreeOp{{Kind: core.SubtreeOpDelete, Path: "/Root/P1"}}, nil)
		Expect(errors.Is(err, core.ErrNoShareTree)).To(BeTrue())
	})

	It("rejects unknown op kind", func() {
		_, err := core.ApplySubtreeBatch(base, []core.SubtreeOp{{Kind: 9999, Path: "/Root/P1"}}, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown op kind"))
	})

	It("propagates the first failing op with its index", func() {
		ops := []core.SubtreeOp{
			{Kind: core.SubtreeOpDelete, Path: "/Root/P1"},
			{Kind: core.SubtreeOpDelete, Path: "/Root/nonsense"},
		}
		_, err := core.ApplySubtreeBatch(base, ops, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("op[1]"))
	})
})

var _ = Describe("ApplyShareTreeBatch (qconf wrapper)", func() {
	It("issues exactly one -sstree read and one -Mstree write for a 3-op batch", func() {
		// First fake call answers ShowShareTreeStructured (sstree);
		// the wrapper then writes once via ModifyShareTreeStructured
		// (-Mstree). We serve both calls from the same fake stub: it
		// prints the same body for every invocation, which is fine
		// because the second call's body (-Mstree) is ignored by
		// the wrapper (it only cares about the exit code).
		tree := `id=0
name=Root
type=0
shares=1
childnodes=1,2,3
id=1
name=default
type=0
shares=10
childnodes=NONE
id=2
name=P1
type=1
shares=100
childnodes=NONE
id=3
name=P2
type=1
shares=100
childnodes=NONE
`
		f := newFakeQConf(tree, 0)
		defer f.Cleanup()

		qc := newQConfWith(f)
		ops := []core.SubtreeOp{
			{Kind: core.SubtreeOpDelete, Path: "/Root/P1"},
			{Kind: core.SubtreeOpAdd, Path: "/Root",
				Subtree: &core.StructuredShareTreeNode{
					Name: "P3", Type: core.ShareTreeNodeProject, Shares: 50,
				},
			},
			{Kind: core.SubtreeOpReplace, Path: "/Root/P2",
				Subtree: &core.StructuredShareTreeNode{
					Name: "P2", Type: core.ShareTreeNodeProject, Shares: 999,
				},
			},
		}
		Expect(qc.ApplyShareTreeBatch(ops)).To(Succeed())

		lines := f.AllArgvLines()
		Expect(lines).To(HaveLen(2),
			"batch must collapse 3 edits into exactly 1 read + 1 write, got %v", lines)
		Expect(lines[0]).To(Equal("-sstree"))
		// ModifyShareTree writes to a temp file and passes -Mstree <path>.
		Expect(lines[1]).To(ContainSubstring("-Mstree"))
	})

	It("refuses an empty op list", func() {
		f := newFakeQConf("", 0)
		defer f.Cleanup()
		qc := newQConfWith(f)
		Expect(qc.ApplyShareTreeBatch(nil)).To(HaveOccurred())
		Expect(f.AllArgvLines()).To(BeEmpty(),
			"qconf must not be invoked for empty batches")
	})
})
