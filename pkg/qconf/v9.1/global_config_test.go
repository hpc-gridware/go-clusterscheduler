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

package qconf

import (
	"bytes"
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// GlobalConfig embeds core.GlobalConfig anonymously, so core keys and
// v9.1 keys share one flat JSON object and one TypedKeysOf set. Neither
// property is visible from the core package's own tests.
var _ = Describe("v9.1 GlobalConfig", func() {

	populated := func() GlobalConfig {
		cfg := GlobalConfig{
			JsvParams:     "jsv_value",
			TopologyFile:  "/opt/topo",
			BindingParams: map[string]string{"mode": "default"},
		}
		cfg.ExecdSpoolDir = "/opt/spool"
		cfg.ExtraFields = map[string]string{"port_range": "2000-2100"}
		return cfg
	}

	Describe("JSON representation", func() {

		It("keeps embedded core keys flat alongside v9.1 keys", func() {
			data, err := json.Marshal(populated())
			Expect(err).NotTo(HaveOccurred())

			var obj map[string]any
			Expect(json.Unmarshal(data, &obj)).To(Succeed())
			// Flat: core keys are promoted to the top level rather than
			// nested under a "GlobalConfig" object. Tagging or naming
			// the embedded field would break every REST/MCP client.
			Expect(obj).To(HaveKey("execd_spool_dir"))
			Expect(obj).To(HaveKey("extra_fields"))
			Expect(obj).To(HaveKey("jsv_params"))
			Expect(obj).NotTo(HaveKey("GlobalConfig"))
		})

		It("round-trips core and v9.1 fields together", func() {
			data, err := json.Marshal(populated())
			Expect(err).NotTo(HaveOccurred())

			var back GlobalConfig
			Expect(json.Unmarshal(data, &back)).To(Succeed())
			Expect(back.ExecdSpoolDir).To(Equal("/opt/spool"))
			Expect(back.JsvParams).To(Equal("jsv_value"))
			Expect(back.ExtraFields).To(HaveKeyWithValue("port_range", "2000-2100"))
			Expect(back.BindingParams).To(HaveKeyWithValue("mode", "default"))
		})
	})

	Describe("TypedKeysOf across the embedding", func() {
		It("returns the union of core and v9.1 attribute names", func() {
			keys := core.TypedKeysOf(GlobalConfig{})
			Expect(keys).To(HaveKey("execd_spool_dir")) // core
			Expect(keys).To(HaveKey("jsv_params"))      // v9.1
			Expect(keys).To(HaveKey("extra_fields"))    // carrier key, never an attribute
		})
	})

	Describe("writeGlobalConfig", func() {

		It("emits core fields, v9.1 fields, then extras", func() {
			var buf bytes.Buffer
			Expect(writeGlobalConfig(&buf, populated())).To(Succeed())
			out := buf.String()
			Expect(out).To(ContainSubstring("execd_spool_dir /opt/spool\n"))
			Expect(out).To(ContainSubstring("jsv_params jsv_value\n"))
			Expect(out).To(ContainSubstring("binding_params mode=default\n"))
			Expect(out).To(ContainSubstring("port_range 2000-2100\n"))
		})

		It("never emits an extra_fields attribute line", func() {
			var buf bytes.Buffer
			Expect(writeGlobalConfig(&buf, populated())).To(Succeed())
			for _, line := range strings.Split(buf.String(), "\n") {
				Expect(strings.HasPrefix(line, "extra_fields")).To(BeFalse(),
					"qconf would reject the attribute line %q", line)
			}
		})

		It("does not write a promoted key twice", func() {
			// A stale extras entry for a key that v9.1 promotes to a
			// typed field must lose to the typed value, otherwise the
			// file carries two jsv_params lines and the last one wins.
			cfg := populated()
			cfg.ExtraFields["jsv_params"] = "stale_value"

			var buf bytes.Buffer
			Expect(writeGlobalConfig(&buf, cfg)).To(Succeed())
			Expect(strings.Count(buf.String(), "jsv_params ")).To(Equal(1))
			Expect(buf.String()).NotTo(ContainSubstring("stale_value"))
		})

		It("writes NONE for empty v9.1 fields", func() {
			var buf bytes.Buffer
			Expect(writeGlobalConfig(&buf, GlobalConfig{})).To(Succeed())
			Expect(buf.String()).To(ContainSubstring("mail_tag NONE\n"))
			Expect(buf.String()).To(ContainSubstring("binding_params NONE\n"))
		})

		It("rejects extras that would corrupt the file", func() {
			cfg := GlobalConfig{}
			cfg.ExtraFields = map[string]string{"a_key": "1\nprolog /tmp/pwn.sh"}
			var buf bytes.Buffer
			Expect(writeGlobalConfig(&buf, cfg)).NotTo(Succeed())
		})
	})
})
