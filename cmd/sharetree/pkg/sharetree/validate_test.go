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

package sharetree_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

var _ = Describe("Sharetree Validation", func() {

	Context("ValidateSharetree function", func() {
		It("should validate a proper sharetree structure", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,2"},
				{ID: 1, Name: "Child1", ChildNodes: "3"},
				{ID: 2, Name: "Child2", ChildNodes: "NONE"},
				{ID: 3, Name: "Grandchild", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject an empty sharetree", func() {
			nodes := []sharetree.SharetreeNode{}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Sharetree is empty"))
		})

		It("should detect duplicate node IDs", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,2"},
				{ID: 1, Name: "Child1", ChildNodes: "NONE"},
				{ID: 1, Name: "DuplicateID", ChildNodes: "NONE"}, // Duplicate ID
				{ID: 2, Name: "Child2", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Duplicate node ID found: 1"))
		})

		It("should detect nodes with missing names", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,2"},
				{ID: 1, Name: "", ChildNodes: "NONE"}, // Missing name
				{ID: 2, Name: "Child2", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Node with ID 1 is missing a name"))
		})

		It("should detect invalid child ID formats", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,abc"}, // Invalid child ID format
				{ID: 1, Name: "Child1", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid child ID format: abc"))
		})

		It("should detect references to non-existent child nodes", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,99"}, // 99 doesn't exist
				{ID: 1, Name: "Child1", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Node 0 references child 99 which doesn't exist"))
		})

		It("should detect orphaned nodes (except root)", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1"}, // Only references node 1
				{ID: 1, Name: "Child1", ChildNodes: "NONE"},
				{ID: 2, Name: "Orphaned", ChildNodes: "NONE"}, // Orphaned, not referenced anywhere
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Orphaned node found: 2"))
		})

		It("should handle empty and NONE childNodes values properly", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,2"},
				{ID: 1, Name: "Child1", ChildNodes: "NONE"},
				{ID: 2, Name: "Child2", ChildNodes: ""},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle spaces in childNodes list", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1, 2, 3"},
				{ID: 1, Name: "Child1", ChildNodes: "NONE"},
				{ID: 2, Name: "Child2", ChildNodes: "NONE"},
				{ID: 3, Name: "Child3", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle a complex valid sharetree", func() {
			nodes := []sharetree.SharetreeNode{
				{ID: 0, Name: "Root", ChildNodes: "1,2,3"},
				{ID: 1, Name: "Branch1", ChildNodes: "4,5"},
				{ID: 2, Name: "Branch2", ChildNodes: "6"},
				{ID: 3, Name: "Branch3", ChildNodes: "7,8"},
				{ID: 4, Name: "Leaf1", ChildNodes: "NONE"},
				{ID: 5, Name: "Leaf2", ChildNodes: "NONE"},
				{ID: 6, Name: "Leaf3", ChildNodes: "NONE"},
				{ID: 7, Name: "Leaf4", ChildNodes: "NONE"},
				{ID: 8, Name: "Leaf5", ChildNodes: "NONE"},
			}

			err := sharetree.ValidateSharetree(nodes)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
