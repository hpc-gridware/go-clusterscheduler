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

package app_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/app"
	st "github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}

var _ = Describe("Sharetree App", func() {
	var testApp *app.App

	BeforeEach(func() {
		var err error
		testApp, err = app.NewApp()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up temp file if it exists
		if testApp != nil && testApp.TempFilePath != "" {
			os.Remove(testApp.TempFilePath)
		}
	})

	Context("initialization", func() {
		It("should create a temp file", func() {
			Expect(testApp.TempFilePath).NotTo(BeEmpty())

			// Check file exists
			_, err := os.Stat(testApp.TempFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should initialize with a root node", func() {
			Expect(testApp.CurrentSharetree).To(HaveLen(1))
			Expect(testApp.CurrentSharetree[0].ID).To(Equal(0))
			Expect(testApp.CurrentSharetree[0].Name).To(Equal("Root"))
		})
	})

	Context("validating root node", func() {
		It("should validate a valid root node", func() {
			nodes := []st.SharetreeNode{
				{ID: 0, Name: "Root", Type: 0, Shares: 1, ChildNodes: "1"},
				{ID: 1, Name: "Child", Type: 0, Shares: 10, ChildNodes: "NONE"},
			}

			err := testApp.ValidateRootNode(nodes)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject missing root node", func() {
			nodes := []st.SharetreeNode{
				{ID: 1, Name: "Child", Type: 0, Shares: 10, ChildNodes: "NONE"},
			}

			err := testApp.ValidateRootNode(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing Root node"))
		})

		It("should reject multiple root nodes", func() {
			nodes := []st.SharetreeNode{
				{ID: 0, Name: "Root", Type: 0, Shares: 1, ChildNodes: "NONE"},
				{ID: 0, Name: "Root", Type: 0, Shares: 1, ChildNodes: "NONE"},
			}

			err := testApp.ValidateRootNode(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Multiple root nodes"))
		})

		It("should reject incorrect root name", func() {
			nodes := []st.SharetreeNode{
				{ID: 0, Name: "NotRoot", Type: 0, Shares: 1, ChildNodes: "NONE"},
			}

			err := testApp.ValidateRootNode(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be named 'Root'"))
		})

		It("should reject non-user root", func() {
			nodes := []st.SharetreeNode{
				{ID: 0, Name: "Root", Type: 1, Shares: 1, ChildNodes: "NONE"},
			}

			err := testApp.ValidateRootNode(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be User type"))
		})

		It("should reject root siblings", func() {
			nodes := []st.SharetreeNode{
				{ID: 0, Name: "Root", Type: 0, Shares: 1, ChildNodes: "NONE"},
				{ID: 1, Name: "Sibling", Type: 0, Shares: 10, ChildNodes: "NONE"},
			}

			err := testApp.ValidateRootNode(nodes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot have siblings"))
		})
	})

	Context("working with SGE files", func() {
		It("should save and load sharetree", func() {
			// Create test sharetree
			testTree := []st.SharetreeNode{
				{ID: 0, Name: "Root", Type: 0, Shares: 1, ChildNodes: "1"},
				{ID: 1, Name: "TestNode", Type: 0, Shares: 100, ChildNodes: "NONE"},
			}

			// Save to temp file
			tmpFile := testApp.TempFilePath + ".test"
			err := testApp.SaveSharetreeAsSGE(testTree, tmpFile)
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile)

			// Read content
			content, err := os.ReadFile(tmpFile)
			Expect(err).NotTo(HaveOccurred())

			// Verify content format
			contentStr := string(content)
			Expect(contentStr).To(ContainSubstring("id=0"))
			Expect(contentStr).To(ContainSubstring("name=Root"))
			Expect(contentStr).To(ContainSubstring("id=1"))
			Expect(contentStr).To(ContainSubstring("name=TestNode"))

			// Load back
			loadedTree, err := testApp.LoadSharetreeFromSGE(contentStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(loadedTree).To(HaveLen(2))
			Expect(loadedTree[0].ID).To(Equal(0))
			Expect(loadedTree[0].Name).To(Equal("Root"))
			Expect(loadedTree[1].ID).To(Equal(1))
			Expect(loadedTree[1].Name).To(Equal("TestNode"))
		})
	})
})
