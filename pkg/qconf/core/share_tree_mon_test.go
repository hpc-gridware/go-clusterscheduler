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

// Offline tests for the sge_share_mon parser. The fixtures under
// testdata/ are verbatim captures from a live OCS 9.0.x container, so
// these specs pin the parser to ground-truth output format:
//
//   - one record per line
//   - TAB-separated key=value fields within a record
//   - node_name in path form ("/", "/P1", "/default/alice")
//   - "No share tree" (literal) when qmaster has no tree configured

package core_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("ParseShareMonOutput", func() {

	Context("with the real sge_share_mon -c 1 -n capture", func() {
		var mon *core.ShareTreeMonitoring

		BeforeEach(func() {
			raw, err := os.ReadFile("testdata/sge_share_mon_sample.txt")
			Expect(err).NotTo(HaveOccurred())
			mon, err = core.ParseShareMonOutput(strings.NewReader(string(raw)))
			Expect(err).NotTo(HaveOccurred())
		})

		It("keys nodes by path and includes all five tree + auto-injected entries", func() {
			Expect(mon.Nodes).To(HaveKey("/"))
			Expect(mon.Nodes).To(HaveKey("/default"))
			Expect(mon.Nodes).To(HaveKey("/default/root"))
			Expect(mon.Nodes).To(HaveKey("/P1"))
			Expect(mon.Nodes).To(HaveKey("/P2"))
		})

		It("parses root shares and percentages", func() {
			Expect(mon.Nodes["/"].Shares).To(Equal(1))
			Expect(mon.Nodes["/"].LevelPercent).To(Equal(1.0))
			Expect(mon.Nodes["/"].TotalPercent).To(Equal(1.0))
		})

		It("parses /default/root as a user leaf with UserName set", func() {
			n := mon.Nodes["/default/root"]
			Expect(n.UserName).To(Equal("root"))
			Expect(n.ProjectName).To(Equal(""))
			Expect(n.Shares).To(Equal(10))
		})

		It("parses /P1 as a project leaf with ProjectName set", func() {
			n := mon.Nodes["/P1"]
			Expect(n.UserName).To(Equal(""))
			Expect(n.ProjectName).To(Equal("P1"))
			Expect(n.Shares).To(Equal(100))
			Expect(n.LevelPercent).To(Equal(0.476190))
			Expect(n.TotalPercent).To(Equal(0.476190))
		})

		It("parses resource breakdown columns as floats (including zero)", func() {
			n := mon.Nodes["/P1"]
			Expect(n.CPU).To(Equal(0.0))
			Expect(n.Mem).To(Equal(0.0))
			Expect(n.IO).To(Equal(0.0))
			Expect(n.LtCPU).To(Equal(0.0))
			Expect(n.LtMem).To(Equal(0.0))
			Expect(n.LtIO).To(Equal(0.0))
		})

		It("stamps CollectedAt on the envelope", func() {
			Expect(mon.CollectedAt.IsZero()).To(BeFalse())
		})

		It("does not emit curr_time or usage_time as node fields", func() {
			// Those columns are intentionally ignored: timestamps belong
			// to the envelope, not the per-node stats.
			_, hasCurr := map[string]any{}["curr_time"]
			Expect(hasCurr).To(BeFalse())
		})
	})

	Context("with the 'No share tree' fixture", func() {
		It("parses to a successful zero-node snapshot", func() {
			raw, err := os.ReadFile("testdata/sge_share_mon_no_tree.txt")
			Expect(err).NotTo(HaveOccurred())
			mon, err := core.ParseShareMonOutput(strings.NewReader(string(raw)))
			Expect(err).NotTo(HaveOccurred())
			Expect(mon.Nodes).To(BeEmpty())
		})
	})

	Context("with a -h header row prefix", func() {
		It("ignores header tokens (no '=' sign) and keeps records", func() {
			// The header line has no '=' signs in its tokens, so every
			// field falls through parseShareMonRecord's eq < 0 branch.
			headerLine := "curr_time\tusage_time\tnode_name\tuser_name\tproject_name\tshares"
			recordLine := "curr_time=1\tusage_time=0\tnode_name=/X\tuser_name=\tproject_name=\tshares=42"
			in := strings.Join([]string{headerLine, "", recordLine}, "\n")
			mon, err := core.ParseShareMonOutput(strings.NewReader(in))
			Expect(err).NotTo(HaveOccurred())
			Expect(mon.Nodes).To(HaveLen(1))
			Expect(mon.Nodes["/X"].Shares).To(Equal(42))
		})
	})

	Context("with empty input", func() {
		It("returns an empty map without error", func() {
			mon, err := core.ParseShareMonOutput(strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())
			Expect(mon.Nodes).To(BeEmpty())
		})
	})

	Context("with records containing unknown future fields", func() {
		It("ignores fields the parser does not recognise", func() {
			line := "node_name=/n1\tshares=1\texotic_future=42\tanother_new_column=hello"
			mon, err := core.ParseShareMonOutput(strings.NewReader(line))
			Expect(err).NotTo(HaveOccurred())
			Expect(mon.Nodes).To(HaveKey("/n1"))
			Expect(mon.Nodes["/n1"].Shares).To(Equal(1))
		})
	})

	Context("with a record missing node_name", func() {
		It("drops the record rather than keying on empty string", func() {
			line := "shares=10\tlevel%=25.0"
			mon, err := core.ParseShareMonOutput(strings.NewReader(line))
			Expect(err).NotTo(HaveOccurred())
			Expect(mon.Nodes).To(BeEmpty())
		})
	})

	Context("with malformed numeric values", func() {
		It("silently treats them as zero (atoiOrZero/atofOrZero)", func() {
			line := "node_name=/n2\tshares=not_a_number\tlevel%=not_a_float"
			mon, err := core.ParseShareMonOutput(strings.NewReader(line))
			Expect(err).NotTo(HaveOccurred())
			Expect(mon.Nodes["/n2"].Shares).To(Equal(0))
			Expect(mon.Nodes["/n2"].LevelPercent).To(Equal(0.0))
		})
	})
})
