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

package sharetree

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSharetree(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sharetree Suite")
}

var _ = Describe("Sharetree Parser", func() {
	Context("when parsing SGE content", func() {
		It("should correctly parse valid SGE content", func() {
			sgeContent := `id=0
name=Root
type=0
shares=1
childnodes=1,2,3
id=1
name=P1
type=1
shares=50
childnodes=NONE
id=2
name=P2
type=1
shares=50
childnodes=NONE
id=3
name=default
type=0
shares=10
childnodes=NONE`

			nodes, err := ParseSgeContent(sgeContent)

			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(HaveLen(4))

			// Check root node
			Expect(nodes[0].ID).To(Equal(0))
			Expect(nodes[0].Name).To(Equal("Root"))
			Expect(nodes[0].Type).To(Equal(0))
			Expect(nodes[0].Shares).To(Equal(1))
			Expect(nodes[0].ChildNodes).To(Equal("1,2,3"))

			// Check P1 node
			Expect(nodes[1].ID).To(Equal(1))
			Expect(nodes[1].Name).To(Equal("P1"))
			Expect(nodes[1].Type).To(Equal(1))
			Expect(nodes[1].Shares).To(Equal(50))
			Expect(nodes[1].ChildNodes).To(Equal("NONE"))

			// Check P2 node
			Expect(nodes[2].ID).To(Equal(2))
			Expect(nodes[2].Name).To(Equal("P2"))

			// Check default node
			Expect(nodes[3].ID).To(Equal(3))
			Expect(nodes[3].Name).To(Equal("default"))
			Expect(nodes[3].Type).To(Equal(0))
			Expect(nodes[3].Shares).To(Equal(10))
			Expect(nodes[3].ChildNodes).To(Equal("NONE"))
		})

		It("should handle comment lines and whitespace", func() {
			sgeContent := `# This is a comment
id=0
name=Root
type=0
shares=1
childnodes=1,2,3

# Another comment
id=1
name=P1
type=1
shares=50
childnodes=NONE`

			nodes, err := ParseSgeContent(sgeContent)

			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(HaveLen(2))
			Expect(nodes[0].ID).To(Equal(0))
			Expect(nodes[1].ID).To(Equal(1))
		})

		It("should return error for empty content", func() {
			sgeContent := ``

			_, err := ParseSgeContent(sgeContent)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid nodes"))
		})

		It("should ignore invalid lines", func() {
			sgeContent := `id=0
name=Root
type=0
invalid line
shares=1
childnodes=NONE`

			nodes, err := ParseSgeContent(sgeContent)

			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(HaveLen(1))
			Expect(nodes[0].ID).To(Equal(0))
			Expect(nodes[0].Shares).To(Equal(1))
		})
	})
})
