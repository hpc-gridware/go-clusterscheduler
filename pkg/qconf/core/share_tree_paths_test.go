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

var _ = Describe("NormalizeSharePath", func() {
	It("accepts the canonical form unchanged", func() {
		got, err := core.NormalizeSharePath("/Root/IT/devel")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/Root/IT/devel"))
	})

	It("accepts a missing leading slash", func() {
		got, err := core.NormalizeSharePath("Root/IT/devel")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/Root/IT/devel"))
	})

	It("prepends Root when missing", func() {
		got, err := core.NormalizeSharePath("IT/devel")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/Root/IT/devel"))
	})

	It("treats / as the root", func() {
		got, err := core.NormalizeSharePath("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/Root"))
	})

	It("accepts Root alone", func() {
		got, err := core.NormalizeSharePath("Root")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/Root"))
	})

	It("strips a trailing slash", func() {
		got, err := core.NormalizeSharePath("/Root/IT/")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/Root/IT"))
	})

	It("rejects empty input", func() {
		_, err := core.NormalizeSharePath("")
		Expect(err).To(HaveOccurred())
	})

	It("rejects doubled slashes", func() {
		_, err := core.NormalizeSharePath("/Root//IT")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("SplitSharePath", func() {
	It("returns segments starting with Root", func() {
		Expect(core.SplitSharePath("/Root/IT/devel")).To(Equal([]string{"Root", "IT", "devel"}))
	})

	It("accepts the short form", func() {
		Expect(core.SplitSharePath("IT/devel")).To(Equal([]string{"Root", "IT", "devel"}))
	})

	It("returns [Root] for the root itself", func() {
		Expect(core.SplitSharePath("/")).To(Equal([]string{"Root"}))
	})

	It("returns nil on invalid input", func() {
		Expect(core.SplitSharePath("")).To(BeNil())
	})
})

var _ = Describe("FindNodeByPath", func() {
	var root *core.StructuredShareTreeNode

	BeforeEach(func() {
		root = &core.StructuredShareTreeNode{
			Name: "Root", Shares: 1,
			Children: []*core.StructuredShareTreeNode{
				{Name: "IT", Shares: 100, Children: []*core.StructuredShareTreeNode{
					{Name: "devel", Shares: 90},
					{Name: "kurt", Shares: 10},
				}},
				{Name: "Projects", Shares: 200, Type: core.ShareTreeNodeUser, Children: []*core.StructuredShareTreeNode{
					{Name: "Alpha", Shares: 50, Type: core.ShareTreeNodeProject},
				}},
			},
		}
	})

	It("resolves the root with nil parent", func() {
		node, parent, err := core.FindNodeByPath(root, "/Root")
		Expect(err).NotTo(HaveOccurred())
		Expect(node).To(BeIdenticalTo(root))
		Expect(parent).To(BeNil())
	})

	It("resolves a second-level interior node", func() {
		node, parent, err := core.FindNodeByPath(root, "/Root/IT")
		Expect(err).NotTo(HaveOccurred())
		Expect(node.Name).To(Equal("IT"))
		Expect(parent).To(BeIdenticalTo(root))
	})

	It("resolves a leaf deep in the tree", func() {
		node, parent, err := core.FindNodeByPath(root, "/Root/IT/devel")
		Expect(err).NotTo(HaveOccurred())
		Expect(node.Name).To(Equal("devel"))
		Expect(parent.Name).To(Equal("IT"))
	})

	It("resolves a project node under a parent", func() {
		node, parent, err := core.FindNodeByPath(root, "/Root/Projects/Alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(node.Name).To(Equal("Alpha"))
		Expect(node.Type).To(Equal(core.ShareTreeNodeProject))
		Expect(parent.Name).To(Equal("Projects"))
	})

	It("returns ErrShareTreeNodeNotFound for an unknown segment", func() {
		_, _, err := core.FindNodeByPath(root, "/Root/IT/nope")
		Expect(err).To(MatchError(core.ErrShareTreeNodeNotFound))
	})

	It("returns ErrShareTreeNodeNotFound for nil root", func() {
		_, _, err := core.FindNodeByPath(nil, "/Root")
		Expect(err).To(MatchError(core.ErrShareTreeNodeNotFound))
	})

	It("returns ErrShareTreeNodeNotFound for invalid path", func() {
		_, _, err := core.FindNodeByPath(root, "")
		Expect(err).To(MatchError(core.ErrShareTreeNodeNotFound))
	})
})

var _ = Describe("CloneShareTreeSubtree", func() {
	It("returns nil for nil input", func() {
		Expect(core.CloneShareTreeSubtree(nil)).To(BeNil())
	})

	It("produces an independent deep copy", func() {
		src := &core.StructuredShareTreeNode{
			Name: "IT", Shares: 100,
			Children: []*core.StructuredShareTreeNode{
				{Name: "devel", Shares: 90},
			},
		}
		clone := core.CloneShareTreeSubtree(src)

		Expect(clone).NotTo(BeIdenticalTo(src))
		Expect(clone.Name).To(Equal("IT"))
		Expect(clone.Shares).To(Equal(100))
		Expect(clone.Children).To(HaveLen(1))
		Expect(clone.Children[0]).NotTo(BeIdenticalTo(src.Children[0]))

		// Mutating the clone does not affect the original.
		clone.Name = "Other"
		clone.Children[0].Shares = 9999
		Expect(src.Name).To(Equal("IT"))
		Expect(src.Children[0].Shares).To(Equal(90))
	})

	It("zeroes the ID on the clone", func() {
		src := &core.StructuredShareTreeNode{ID: 42, Name: "X"}
		clone := core.CloneShareTreeSubtree(src)
		Expect(clone.ID).To(Equal(0))
	})
})

var _ = Describe("IsDescendant", func() {
	var (
		root, it, devel, other *core.StructuredShareTreeNode
	)
	BeforeEach(func() {
		devel = &core.StructuredShareTreeNode{Name: "devel"}
		it = &core.StructuredShareTreeNode{Name: "IT", Children: []*core.StructuredShareTreeNode{devel}}
		other = &core.StructuredShareTreeNode{Name: "Other"}
		root = &core.StructuredShareTreeNode{Name: "Root", Children: []*core.StructuredShareTreeNode{it, other}}
	})

	It("recognizes direct children", func() {
		Expect(core.IsDescendant(root, it)).To(BeTrue())
	})

	It("recognizes transitive descendants", func() {
		Expect(core.IsDescendant(root, devel)).To(BeTrue())
	})

	It("rejects unrelated nodes", func() {
		Expect(core.IsDescendant(it, other)).To(BeFalse())
	})

	It("rejects self", func() {
		Expect(core.IsDescendant(root, root)).To(BeFalse())
	})

	It("handles nil safely", func() {
		Expect(core.IsDescendant(nil, it)).To(BeFalse())
		Expect(core.IsDescendant(root, nil)).To(BeFalse())
	})
})
