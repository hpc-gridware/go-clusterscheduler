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
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

func TestShareTree(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Share Tree Suite")
}

var _ = Describe("ParseShareTreeText", func() {

	Context("when the tree is empty or missing", func() {
		It("returns ErrNoShareTree for an empty input", func() {
			_, err := core.ParseShareTreeText("")
			Expect(err).To(MatchError(core.ErrNoShareTree))
		})

		It("returns ErrNoShareTree when qconf reports no sharetree", func() {
			_, err := core.ParseShareTreeText("no sharetree defined")
			Expect(err).To(MatchError(core.ErrNoShareTree))
		})

		It("returns ErrNoShareTree for whitespace-only input", func() {
			_, err := core.ParseShareTreeText("   \n\t  \n")
			Expect(err).To(MatchError(core.ErrNoShareTree))
		})
	})

	Context("with the man-page reference tree", func() {
		var tree *core.StructuredShareTree

		BeforeEach(func() {
			raw, err := os.ReadFile("testdata/share_tree_basic.txt")
			Expect(err).NotTo(HaveOccurred())
			tree, err = core.ParseShareTreeText(string(raw))
			Expect(err).NotTo(HaveOccurred())
		})

		It("produces a Root with three children", func() {
			Expect(tree.Root).NotTo(BeNil())
			Expect(tree.Root.Name).To(Equal("Root"))
			Expect(tree.Root.Type).To(Equal(core.ShareTreeNodeUser))
			Expect(tree.Root.Shares).To(Equal(1))
			Expect(tree.Root.Children).To(HaveLen(3))
		})

		It("preserves node types (user=0, project=1)", func() {
			names := map[string]core.ShareTreeNodeType{}
			for _, c := range tree.Root.Children {
				names[c.Name] = c.Type
			}
			Expect(names).To(HaveKeyWithValue("P1", core.ShareTreeNodeProject))
			Expect(names).To(HaveKeyWithValue("P2", core.ShareTreeNodeProject))
			Expect(names).To(HaveKeyWithValue("default", core.ShareTreeNodeUser))
		})

		It("preserves share values", func() {
			shares := map[string]int{}
			for _, c := range tree.Root.Children {
				shares[c.Name] = c.Shares
			}
			Expect(shares).To(HaveKeyWithValue("P1", 50))
			Expect(shares).To(HaveKeyWithValue("P2", 50))
			Expect(shares).To(HaveKeyWithValue("default", 10))
		})

		It("marks all leaves as having no children", func() {
			for _, c := range tree.Root.Children {
				Expect(c.Children).To(BeEmpty())
			}
		})
	})

	Context("with a deeper nested tree", func() {
		var tree *core.StructuredShareTree

		BeforeEach(func() {
			raw, err := os.ReadFile("testdata/share_tree_nested.txt")
			Expect(err).NotTo(HaveOccurred())
			tree, err = core.ParseShareTreeText(string(raw))
			Expect(err).NotTo(HaveOccurred())
		})

		It("resolves IT/devel as a grand-child of Root", func() {
			var it *core.StructuredShareTreeNode
			for _, c := range tree.Root.Children {
				if c.Name == "IT" {
					it = c
				}
			}
			Expect(it).NotTo(BeNil())
			Expect(it.Children).To(HaveLen(2))
			names := []string{it.Children[0].Name, it.Children[1].Name}
			Expect(names).To(ContainElement("devel"))
			Expect(names).To(ContainElement("kurt"))
		})

		It("puts default + alice under ProjectX", func() {
			var px *core.StructuredShareTreeNode
			for _, c := range tree.Root.Children {
				if c.Name == "ProjectX" {
					px = c
				}
			}
			Expect(px).NotTo(BeNil())
			Expect(px.Type).To(Equal(core.ShareTreeNodeProject))
			Expect(px.Children).To(HaveLen(2))
		})
	})

	Context("with malformed input", func() {
		It("rejects a name line before any id", func() {
			_, err := core.ParseShareTreeText("name=Root\nshares=1\n")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("before id"))
		})

		It("rejects a non-numeric id", func() {
			_, err := core.ParseShareTreeText("id=abc\nname=X\n")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid id"))
		})

		It("rejects duplicate ids", func() {
			_, err := core.ParseShareTreeText("id=0\nname=A\nid=0\nname=B\n")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate id"))
		})

		It("rejects references to unknown child ids", func() {
			_, err := core.ParseShareTreeText("id=0\nname=Root\ntype=0\nshares=1\nchildnodes=99\n")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown child"))
		})
	})
})

var _ = Describe("FormatShareTreeText", func() {

	Context("with nil input", func() {
		It("returns an error for a nil tree", func() {
			_, err := core.FormatShareTreeText(nil)
			Expect(err).To(HaveOccurred())
		})

		It("returns an error for a tree with nil Root", func() {
			_, err := core.FormatShareTreeText(&core.StructuredShareTree{})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("round-trip with the man-page tree", func() {
		It("parses, formats, and re-parses to an equivalent tree", func() {
			raw, err := os.ReadFile("testdata/share_tree_basic.txt")
			Expect(err).NotTo(HaveOccurred())

			first, err := core.ParseShareTreeText(string(raw))
			Expect(err).NotTo(HaveOccurred())

			formatted, err := core.FormatShareTreeText(first)
			Expect(err).NotTo(HaveOccurred())
			Expect(formatted).NotTo(BeEmpty())

			second, err := core.ParseShareTreeText(formatted)
			Expect(err).NotTo(HaveOccurred())

			expectTreesEqual(first.Root, second.Root)
		})

		It("produces output that ends with a newline for clean qconf input", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root", Shares: 1,
				},
			}
			out, err := core.FormatShareTreeText(tree)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(HaveSuffix("\n"))
			Expect(out).To(ContainSubstring("childnodes=NONE"))
		})
	})

	Context("round-trip with the nested tree", func() {
		It("preserves structure and share values", func() {
			raw, err := os.ReadFile("testdata/share_tree_nested.txt")
			Expect(err).NotTo(HaveOccurred())

			first, err := core.ParseShareTreeText(string(raw))
			Expect(err).NotTo(HaveOccurred())

			formatted, err := core.FormatShareTreeText(first)
			Expect(err).NotTo(HaveOccurred())

			second, err := core.ParseShareTreeText(formatted)
			Expect(err).NotTo(HaveOccurred())

			expectTreesEqual(first.Root, second.Root)
		})
	})

	Context("re-numbers nodes canonically", func() {
		It("assigns contiguous ids starting at 0 in pre-order", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					ID: 99, Name: "Root", Shares: 1,
					Children: []*core.StructuredShareTreeNode{
						{ID: 42, Name: "A", Shares: 10},
						{ID: 7, Name: "B", Shares: 20},
					},
				},
			}
			out, err := core.FormatShareTreeText(tree)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring("id=0"))
			Expect(out).To(ContainSubstring("id=1"))
			Expect(out).To(ContainSubstring("id=2"))
			Expect(out).NotTo(ContainSubstring("id=42"))
			Expect(out).NotTo(ContainSubstring("id=99"))
		})
	})
})

// expectTreesEqual is a recursive structural equality helper that ignores the
// ID field (IDs are canonically re-generated by the formatter).
func expectTreesEqual(a, b *core.StructuredShareTreeNode) {
	ExpectWithOffset(1, b).NotTo(BeNil())
	ExpectWithOffset(1, a.Name).To(Equal(b.Name))
	ExpectWithOffset(1, a.Type).To(Equal(b.Type))
	ExpectWithOffset(1, a.Shares).To(Equal(b.Shares))
	ExpectWithOffset(1, a.Children).To(HaveLen(len(b.Children)))
	// Children order is preserved by the parser/formatter.
	for i := range a.Children {
		expectTreesEqual(a.Children[i], b.Children[i])
	}
}
