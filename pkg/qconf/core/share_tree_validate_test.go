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

// codes returns the slice of validation codes from a list of errors so
// specs can assert on membership without caring about message wording.
func codes(es []core.ShareTreeValidationError) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.Code)
	}
	return out
}

var _ = Describe("ValidateShareTree", func() {
	It("accepts nil or empty trees without errors", func() {
		Expect(core.ValidateShareTree(nil, nil)).To(BeEmpty())
		Expect(core.ValidateShareTree(&core.StructuredShareTree{}, nil)).To(BeEmpty())
	})

	It("accepts the man-page reference tree", func() {
		tree := &core.StructuredShareTree{
			Root: &core.StructuredShareTreeNode{
				Name: "Root", Shares: 1,
				Children: []*core.StructuredShareTreeNode{
					{Name: "P1", Type: core.ShareTreeNodeProject, Shares: 50},
					{Name: "P2", Type: core.ShareTreeNodeProject, Shares: 50},
					{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
				},
			},
		}
		Expect(core.ValidateShareTree(tree, nil)).To(BeEmpty())
	})

	Context("structural rules", func() {
		It("flags duplicate sibling names (SHARE_DUPLICATE_PATH)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "A", Type: core.ShareTreeNodeUser},
						{Name: "A", Type: core.ShareTreeNodeUser},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeDuplicatePath))
		})

		It("flags a project appearing twice (SHARE_PROJECT_DUPLICATE)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "P1", Type: core.ShareTreeNodeProject},
						{Name: "P1", Type: core.ShareTreeNodeProject},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeProjectDuplicate))
		})

		It("flags projects nested under projects (SHARE_PROJECT_NESTED)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "P1", Type: core.ShareTreeNodeProject,
							Children: []*core.StructuredShareTreeNode{
								{Name: "P2", Type: core.ShareTreeNodeProject},
							},
						},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeProjectNested))
		})

		It("flags a user appearing twice under the same project (SHARE_USER_DUPLICATE_IN_PROJECT)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "P1", Type: core.ShareTreeNodeProject,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser},
								{Name: "alice", Type: core.ShareTreeNodeUser},
							},
						},
					},
				},
			}
			got := codes(core.ValidateShareTree(tree, nil))
			Expect(got).To(ContainElement(core.ShareCodeUserDuplicateInProject))
			Expect(got).To(ContainElement(core.ShareCodeDuplicatePath))
		})

		It("flags a user appearing twice outside any project (SHARE_USER_DUPLICATE_OUTSIDE_PROJECT)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "IT", Type: core.ShareTreeNodeUser,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser},
							},
						},
						{Name: "Ops", Type: core.ShareTreeNodeUser,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser},
							},
						},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeUserDuplicateOutside))
		})

		It("flags negative shares (SHARE_NEGATIVE_SHARES)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "X", Shares: -1},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeNegativeShares))
		})

		It("flags empty node names (SHARE_EMPTY_NAME)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: ""},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeEmptyName))
		})

		It("flags 'default' used as a project (SHARE_DEFAULT_RESERVED)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "default", Type: core.ShareTreeNodeProject},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeDefaultReserved))
		})

		It("flags 'default' used as an interior node (SHARE_DEFAULT_RESERVED)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "default", Type: core.ShareTreeNodeUser,
							Children: []*core.StructuredShareTreeNode{
								{Name: "leaf"},
							},
						},
					},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeDefaultReserved))
		})

		It("flags nil children (SHARE_NIL_NODE)", func() {
			tree := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name:     "Root",
					Children: []*core.StructuredShareTreeNode{nil},
				},
			}
			Expect(codes(core.ValidateShareTree(tree, nil))).To(ContainElement(core.ShareCodeNilNode))
		})
	})

	Context("state-dependent rules (opts provided)", func() {
		baseTree := func() *core.StructuredShareTree {
			return &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root",
					Children: []*core.StructuredShareTreeNode{
						{Name: "P1", Type: core.ShareTreeNodeProject,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser},
							},
						},
					},
				},
			}
		}

		It("flags an unknown user leaf (SHARE_LEAF_UNKNOWN_NAME)", func() {
			opts := &core.ShareTreeValidationOptions{
				KnownUsers:    map[string]bool{},
				KnownProjects: map[string]bool{"P1": true},
			}
			Expect(codes(core.ValidateShareTree(baseTree(), opts))).To(ContainElement(core.ShareCodeLeafUnknownName))
		})

		It("flags an unknown project (SHARE_LEAF_UNKNOWN_NAME)", func() {
			opts := &core.ShareTreeValidationOptions{
				KnownUsers:    map[string]bool{"alice": true},
				KnownProjects: map[string]bool{},
			}
			Expect(codes(core.ValidateShareTree(baseTree(), opts))).To(ContainElement(core.ShareCodeLeafUnknownName))
		})

		It("flags a user without project access (SHARE_USER_NO_PROJECT_ACCESS)", func() {
			opts := &core.ShareTreeValidationOptions{
				KnownUsers:    map[string]bool{"alice": true},
				KnownProjects: map[string]bool{"P1": true},
				ProjectACLs:   map[string]map[string]bool{"P1": {"bob": true}},
			}
			Expect(codes(core.ValidateShareTree(baseTree(), opts))).To(ContainElement(core.ShareCodeUserNoProjectAccess))
		})

		It("passes when all state checks are satisfied", func() {
			opts := &core.ShareTreeValidationOptions{
				KnownUsers:    map[string]bool{"alice": true},
				KnownProjects: map[string]bool{"P1": true},
				ProjectACLs:   map[string]map[string]bool{"P1": {"alice": true}},
			}
			Expect(core.ValidateShareTree(baseTree(), opts)).To(BeEmpty())
		})
	})
})

var _ = Describe("ShareTreeValidationErrors", func() {
	It("implements error with all codes", func() {
		e := &core.ShareTreeValidationErrors{Errs: []core.ShareTreeValidationError{
			{Code: "A", Path: "/x", Message: "a"},
			{Code: "B", Path: "/y", Message: "b"},
		}}
		msg := e.Error()
		Expect(msg).To(ContainSubstring("A at /x"))
		Expect(msg).To(ContainSubstring("B at /y"))
	})

	It("returns a placeholder message when empty", func() {
		e := &core.ShareTreeValidationErrors{}
		Expect(e.Error()).To(ContainSubstring("no validation errors"))
	})
})
