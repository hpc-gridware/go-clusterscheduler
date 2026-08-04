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

// Regression tests for CLI argument-injection defense: when a caller value is
// tainted (leading '-', embedded list separator, control character, or a path
// that escapes the temp dir), the wrapper must reject it BEFORE spawning qconf.
// Each spec asserts both that an error is returned and that the fake qconf was
// never invoked. Uses the shared newFakeQConf/newQConfWith helpers defined in
// share_tree_impl_offline_test.go.

package core_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

var _ = Describe("CLI argument validation", func() {

	Describe("ModifyAttribute / AddAttribute / DeleteAttribute", func() {
		It("rejects a forged leading-dash object-instance and never spawns qconf", func() {
			// objIDList "goodq,-ke somehost" would let qconf parse "-ke" as the
			// kill-event subcommand once it splits the list.
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.ModifyAttribute("queue", "slots", "8", "goodq,-ke somehost")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty(),
				"qconf must not be invoked when the object list is tainted")
		})

		It("rejects a leading-dash object name and never spawns qconf", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.AddAttribute("-help", "slots", "8", "all.q")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("rejects a control character in the value and never spawns qconf", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.DeleteAttribute("queue", "slots", "8\nrm -rf", "all.q")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("accepts a legitimate multi-object list and does spawn qconf", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.ModifyAttribute("queue", "slots", "8", "all.q,gpu.q")
			Expect(err).ToNot(HaveOccurred())
			Expect(f.AllArgvLines()).To(HaveLen(1))
			Expect(f.Argv()).To(Equal([]string{"-mattr", "queue", "slots", "8", "all.q,gpu.q"}))
		})

		It("accepts a complex value that itself contains a comma list", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			// complex_values legitimately carries a comma-separated name=value
			// list in the value token; this must not be rejected.
			err := qc.ModifyAttribute("exechost", "complex_values",
				"slots=8,mem_free=4G", "node01")
			Expect(err).ToNot(HaveOccurred())
			Expect(f.AllArgvLines()).To(HaveLen(1))
		})
	})

	Describe("comma-joined list operations", func() {
		It("rejects a comma-injected admin host and never spawns qconf", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.AddAdminHost([]string{"host1,evil"})
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("rejects a leading-dash user in the manager list", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.AddUserToManagerList([]string{"-x"})
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("accepts a clean multi-host add and does spawn qconf", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.AddAdminHost([]string{"host1", "host2.cluster.local"})
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Argv()).To(Equal([]string{"-ah", "host1,host2.cluster.local"}))
		})
	})

	Describe("ModifyResourceQuotaSet (trailing-operand injection)", func() {
		It("rejects a leading-dash rqsName that qconf would re-parse as a subcommand", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			// "qconf -Mrqs <tmpfile> -km" would execute `qconf -km` (shut down
			// the qmaster). The wrapper must reject it before any temp file or
			// process is created.
			err := qc.ModifyResourceQuotaSet("-km", core.ResourceQuotaSetConfig{Name: "x"})
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})
	})

	Describe("user set list operations", func() {
		It("rejects a leading-dash listname in AddUserToUserSetList", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.AddUserToUserSetList("root", "-x")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("accepts a legitimate multi-user comma list", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			// The comma is the intended list separator; two valid users pass.
			err := qc.AddUserToUserSetList("root,admin", "admins")
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Argv()).To(Equal([]string{"-au", "root,admin", "admins"}))
		})

		It("rejects a space-injected element in DeleteUserFromUserSetList", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.DeleteUserFromUserSetList("root -x", "admins")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})
	})

	Describe("strict mode (GCS_VALIDATION=strict)", func() {
		BeforeEach(func() { GinkgoT().Setenv("GCS_VALIDATION", "strict") })

		It("accepts a host group name containing '@' through a runNamed path", func() {
			f := newFakeQConf("master sim1 sim2\n", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			hosts, err := qc.ShowHostGroupResolved("@allhosts")
			Expect(err).ToNot(HaveOccurred())
			Expect(hosts).To(Equal([]string{"master", "sim1", "sim2"}))
		})

		It("rejects an object name containing a space (KEY_TABLE) before spawning", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			err := qc.DeleteClusterQueue("bad name")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})

		It("rejects a comma-list element failing the charset via a runNamedList path", func() {
			f := newFakeQConf("", 0)
			defer f.Cleanup()
			qc := newQConfWith(f)

			// -de list; "bad name" element has a space -> rejected in strict mode.
			err := qc.DeleteExecHost("ok,bad name")
			Expect(err).To(HaveOccurred())
			Expect(f.AllArgvLines()).To(BeEmpty())
		})
	})

	Describe("CreateTempDirWithFileName", func() {
		It("rejects a traversal object name", func() {
			_, err := core.CreateTempDirWithFileName("..")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid object name"))
		})

		It("rejects an object name containing a path separator", func() {
			_, err := core.CreateTempDirWithFileName("a/b")
			Expect(err).To(HaveOccurred())
		})

		It("accepts a plain object name", func() {
			file, err := core.CreateTempDirWithFileName("all.q")
			Expect(err).ToNot(HaveOccurred())
			Expect(file).ToNot(BeNil())
			DeferCleanup(func() { os.RemoveAll(filepath.Dir(file.Name())) })
		})
	})
})
