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
	"bytes"
	"encoding/json"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// ExtraFields must survive JSON serialization so that a consumer on
// the other side of a REST/MCP boundary can show a config, edit a
// typed field, and send it back without silently deleting parameters
// this struct version does not know about.
var _ = Describe("ExtraFields JSON round-trip", func() {

	It("serializes GlobalConfig.ExtraFields as extra_fields", func() {
		cfg := core.GlobalConfig{
			ExecdSpoolDir: "/opt/spool",
			ExtraFields:   map[string]string{"port_range": "2000-2100"},
		}
		data, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring(`"extra_fields":{"port_range":"2000-2100"}`))

		var back core.GlobalConfig
		Expect(json.Unmarshal(data, &back)).To(Succeed())
		Expect(back.ExtraFields).To(HaveKeyWithValue("port_range", "2000-2100"))
	})

	It("omits extra_fields when the map is empty", func() {
		data, err := json.Marshal(core.GlobalConfig{})
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).NotTo(ContainSubstring("extra_fields"))
	})

	It("round-trips HostExecConfig.ExtraFields", func() {
		cfg := core.HostExecConfig{
			Name:        "exec1",
			ExtraFields: map[string]string{"future_param": "42"},
		}
		data, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		var back core.HostExecConfig
		Expect(json.Unmarshal(data, &back)).To(Succeed())
		Expect(back.ExtraFields).To(HaveKeyWithValue("future_param", "42"))
	})

	It("round-trips SchedulerConfig.ExtraFields", func() {
		cfg := core.SchedulerConfig{
			Algorithm:   "default",
			ExtraFields: map[string]string{"new_sched_param": "on"},
		}
		data, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		var back core.SchedulerConfig
		Expect(json.Unmarshal(data, &back)).To(Succeed())
		Expect(back.ExtraFields).To(HaveKeyWithValue("new_sched_param", "on"))
	})

	It("never emits the extra_fields carrier key as a qconf attribute", func() {
		var buf bytes.Buffer
		err := core.WriteExtraFields(&buf,
			map[string]string{"extra_fields": "bogus", "port_range": "2000-2100"},
			core.TypedKeysOf(core.GlobalConfig{}))
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("port_range 2000-2100\n"))
	})

	// ExtraFields used to be unreachable from JSON (json:"-"), so the
	// key/value shape was enforced only in CaptureExtraField on the
	// parse path. Now that a map can arrive straight from a request
	// body, the emit path must reject what the parser would have
	// dropped -- otherwise a line feed in a value injects an arbitrary
	// attribute line into the qconf -M* file.
	Describe("emit-side validation of JSON-supplied extras", func() {

		DescribeTable("rejects values that would corrupt the config file",
			func(value string) {
				var buf bytes.Buffer
				err := core.WriteExtraFields(&buf,
					map[string]string{"a_key": value}, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("a_key"))
				Expect(buf.String()).NotTo(ContainSubstring("prolog"))
			},
			Entry("line feed injecting a second directive", "1\nprolog /tmp/pwn.sh"),
			Entry("carriage return", "1\rprolog /tmp/pwn.sh"),
			Entry("NUL truncating the line", "1\x00rest"),
			Entry("tab breaking field tokenisation", "a\tb"),
		)

		DescribeTable("rejects key shapes qconf cannot express",
			func(key string) {
				var buf bytes.Buffer
				err := core.WriteExtraFields(&buf,
					map[string]string{key: "value"}, nil)
				Expect(err).To(HaveOccurred())
				Expect(buf.String()).To(BeEmpty())
			},
			Entry("comment prefix", "#comment"),
			Entry("embedded space", "bad key"),
			Entry("uppercase", "UPPER"),
			Entry("equals sign", "a=b"),
			Entry("empty key", ""),
		)

		It("accepts values that round-trip from the parser unchanged", func() {
			var buf bytes.Buffer
			err := core.WriteExtraFields(&buf,
				map[string]string{"port_range": "2000-2100", "some_key": "a b c"}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("port_range 2000-2100\nsome_key a b c\n"))
		})
	})

	// Guards the uniformity of the retag: a struct whose ExtraFields
	// regains json:"-" (or a typo'd tag) would silently stop
	// preserving unknown parameters across a serialization boundary.
	It("tags ExtraFields identically on every config struct", func() {
		structs := []any{
			core.CalendarConfig{}, core.ComplexEntryConfig{},
			core.CkptInterfaceConfig{}, core.GlobalConfig{},
			core.HostConfiguration{}, core.HostGroupConfig{},
			core.HostExecConfig{}, core.ResourceQuotaSetConfig{},
			core.ParallelEnvironmentConfig{}, core.ProjectConfig{},
			core.ClusterQueueConfig{}, core.UserSetListConfig{},
			core.UserConfig{}, core.SchedulerConfig{},
		}
		for _, s := range structs {
			t := reflect.TypeOf(s)
			f, ok := t.FieldByName("ExtraFields")
			Expect(ok).To(BeTrue(), "%s has no ExtraFields field", t.Name())
			Expect(f.Tag.Get("json")).To(Equal("extra_fields,omitempty"),
				"%s carries the wrong ExtraFields json tag", t.Name())
		}
	})

	It("accepts legacy payloads that predate the extra_fields key", func() {
		var cfg core.GlobalConfig
		Expect(json.Unmarshal([]byte(`{"execd_spool_dir":"/opt/spool"}`), &cfg)).To(Succeed())
		Expect(cfg.ExtraFields).To(BeEmpty())

		data, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).NotTo(ContainSubstring("extra_fields"))
	})
})
