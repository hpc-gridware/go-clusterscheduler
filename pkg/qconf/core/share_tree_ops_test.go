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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// newSampleTree returns a fresh copy of a multi-department share tree used
// by multiple specs. Each spec gets its own copy so mutations from one
// spec never leak into another.
func newSampleTree() *core.StructuredShareTree {
	return &core.StructuredShareTree{
		Root: &core.StructuredShareTreeNode{
			Name: "Root", Shares: 1,
			Children: []*core.StructuredShareTreeNode{
				{Name: "IT", Type: core.ShareTreeNodeUser, Shares: 100,
					Children: []*core.StructuredShareTreeNode{
						{Name: "devel", Type: core.ShareTreeNodeUser, Shares: 90},
						{Name: "kurt", Type: core.ShareTreeNodeUser, Shares: 10},
					},
				},
				{Name: "Projects", Type: core.ShareTreeNodeUser, Shares: 200,
					Children: []*core.StructuredShareTreeNode{
						{Name: "Alpha", Type: core.ShareTreeNodeProject, Shares: 50,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser, Shares: 20},
								{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
							},
						},
					},
				},
			},
		},
	}
}

// asValidationErrs unwraps *ShareTreeValidationErrors from an error, or
// returns nil if the error is not a validation error.
func asValidationErrs(err error) *core.ShareTreeValidationErrors {
	var ve *core.ShareTreeValidationErrors
	if errors.As(err, &ve) {
		return ve
	}
	return nil
}

var _ = Describe("ApplySubtreeReplace", func() {
	It("replaces a leaf with a new subtree, preserving siblings", func() {
		tree := newSampleTree()
		replacement := &core.StructuredShareTreeNode{
			Name: "devel", Type: core.ShareTreeNodeUser, Shares: 500,
		}
		got, err := core.ApplySubtreeReplace(tree, "/Root/IT/devel", replacement, nil)
		Expect(err).NotTo(HaveOccurred())

		// Verify IT still has two children (devel + kurt) with devel's
		// new share value.
		it, _, _ := core.FindNodeByPath(got.Root, "/Root/IT")
		Expect(it.Children).To(HaveLen(2))
		devel, _, _ := core.FindNodeByPath(got.Root, "/Root/IT/devel")
		Expect(devel.Shares).To(Equal(500))

		// Original tree is unchanged.
		origDevel, _, _ := core.FindNodeByPath(tree.Root, "/Root/IT/devel")
		Expect(origDevel.Shares).To(Equal(90))
	})

	It("can swap in a subtree that adds new descendants", func() {
		tree := newSampleTree()
		rich := &core.StructuredShareTreeNode{
			Name: "IT", Type: core.ShareTreeNodeUser, Shares: 150,
			Children: []*core.StructuredShareTreeNode{
				{Name: "newuser", Type: core.ShareTreeNodeUser, Shares: 75},
			},
		}
		got, err := core.ApplySubtreeReplace(tree, "/Root/IT", rich, nil)
		Expect(err).NotTo(HaveOccurred())

		it, _, _ := core.FindNodeByPath(got.Root, "/Root/IT")
		Expect(it.Children).To(HaveLen(1))
		Expect(it.Children[0].Name).To(Equal("newuser"))
	})

	It("returns SHARE_PATH_NOT_FOUND for an unknown path", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeReplace(tree, "/Root/IT/nope",
			&core.StructuredShareTreeNode{Name: "x"}, nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodePathNotFound))
	})

	It("returns SHARE_DUPLICATE_PATH when replacement introduces duplicate siblings", func() {
		tree := newSampleTree()
		dup := &core.StructuredShareTreeNode{
			Name: "IT", Type: core.ShareTreeNodeUser, Shares: 100,
			Children: []*core.StructuredShareTreeNode{
				{Name: "devel", Type: core.ShareTreeNodeUser, Shares: 1},
				{Name: "devel", Type: core.ShareTreeNodeUser, Shares: 2},
			},
		}
		_, err := core.ApplySubtreeReplace(tree, "/Root/IT", dup, nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
	})

	It("returns ErrNoShareTree for an empty input tree", func() {
		_, err := core.ApplySubtreeReplace(nil, "/Root",
			&core.StructuredShareTreeNode{Name: "x"}, nil)
		Expect(err).To(MatchError(core.ErrNoShareTree))
	})
})

var _ = Describe("ApplySubtreeAdd", func() {
	It("appends a new child under the parent", func() {
		tree := newSampleTree()
		addition := &core.StructuredShareTreeNode{
			Name: "Administration", Type: core.ShareTreeNodeUser, Shares: 50,
		}
		got, err := core.ApplySubtreeAdd(tree, "/Root", addition, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Root.Children).To(HaveLen(3))
	})

	It("returns SHARE_DUPLICATE_PATH when name collides with existing sibling", func() {
		tree := newSampleTree()
		collision := &core.StructuredShareTreeNode{
			Name: "IT", Type: core.ShareTreeNodeUser,
		}
		_, err := core.ApplySubtreeAdd(tree, "/Root", collision, nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		codes := []string{}
		for _, e := range ve.Errs {
			codes = append(codes, e.Code)
		}
		Expect(codes).To(ContainElement(core.ShareCodeDuplicatePath))
	})

	It("returns SHARE_PATH_NOT_FOUND when parent does not exist", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeAdd(tree, "/Root/MissingDept",
			&core.StructuredShareTreeNode{Name: "x"}, nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodePathNotFound))
	})
})

var _ = Describe("ApplySubtreeDelete", func() {
	It("removes a leaf and returns the trimmed tree", func() {
		tree := newSampleTree()
		got, err := core.ApplySubtreeDelete(tree, "/Root/IT/kurt")
		Expect(err).NotTo(HaveOccurred())
		it, _, _ := core.FindNodeByPath(got.Root, "/Root/IT")
		Expect(it.Children).To(HaveLen(1))
		Expect(it.Children[0].Name).To(Equal("devel"))
	})

	It("rejects deleting the root with SHARE_ROOT_DELETE", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeDelete(tree, "/Root")
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodeRootDelete))
	})

	It("returns SHARE_PATH_NOT_FOUND for an unknown path", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeDelete(tree, "/Root/Nope")
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodePathNotFound))
	})

	It("removes an entire subtree with all descendants", func() {
		tree := newSampleTree()
		got, err := core.ApplySubtreeDelete(tree, "/Root/Projects")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Root.Children).To(HaveLen(1))
		Expect(got.Root.Children[0].Name).To(Equal("IT"))
	})
})

var _ = Describe("ApplySubtreeMove", func() {
	It("relocates a subtree and detaches it from the old parent", func() {
		tree := newSampleTree()
		got, err := core.ApplySubtreeMove(tree, "/Root/IT/kurt", "/Root/Projects", nil)
		Expect(err).NotTo(HaveOccurred())

		it, _, _ := core.FindNodeByPath(got.Root, "/Root/IT")
		Expect(it.Children).To(HaveLen(1))

		moved, _, _ := core.FindNodeByPath(got.Root, "/Root/Projects/kurt")
		Expect(moved).NotTo(BeNil())
		Expect(moved.Shares).To(Equal(10))
	})

	It("rejects moving a node into itself (SHARE_CYCLE)", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeMove(tree, "/Root/IT", "/Root/IT", nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodeCycle))
	})

	It("rejects moving a node into its own descendant (SHARE_CYCLE)", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeMove(tree, "/Root/IT", "/Root/IT", nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodeCycle))
	})

	It("rejects moving the root (SHARE_ROOT_DELETE)", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeMove(tree, "/Root", "/Root/IT", nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodeRootDelete))
	})

	It("returns SHARE_PATH_NOT_FOUND for missing source", func() {
		tree := newSampleTree()
		_, err := core.ApplySubtreeMove(tree, "/Root/Nope", "/Root/IT", nil)
		ve := asValidationErrs(err)
		Expect(ve).NotTo(BeNil())
		Expect(ve.Errs[0].Code).To(Equal(core.ShareCodePathNotFound))
	})
})
