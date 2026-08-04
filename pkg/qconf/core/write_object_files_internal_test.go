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

package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// White-box specs pinning the Add-path ExtraFields contract: the files
// handed to qconf -Ap / -Acal must carry unknown keys verbatim, the
// same way the Modify* paths do. Template instantiation depends on it.
var _ = Describe("Add-path object file writers", func() {

	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	openTemp := func(name string) *os.File {
		f, err := os.Create(filepath.Join(dir, name))
		Expect(err).NotTo(HaveOccurred())
		return f
	}

	Describe("writePE", func() {
		It("emits ExtraFields after the typed fields", func() {
			f := openTemp("pe")
			pe := ParallelEnvironmentConfig{
				Name:           "test",
				AllocationRule: "$pe_slots",
				ExtraFields: map[string]string{
					"future_parameter": "some_value",
				},
			}
			Expect(writePE(f, pe)).To(Succeed())
			content, err := os.ReadFile(f.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring(
				"future_parameter some_value\n"))
		})

		It("leaves the file handle open for the caller", func() {
			// ModifyParallelEnvironment writes through the same handle
			// after writePE; a Close inside writePE broke every Modify
			// carrying ExtraFields (write on closed file).
			f := openTemp("pe")
			Expect(writePE(f, ParallelEnvironmentConfig{Name: "t"})).To(Succeed())
			_, err := f.WriteString("after_write ok\n")
			Expect(err).NotTo(HaveOccurred(),
				"writePE must not close the caller's file")
			Expect(f.Close()).To(Succeed())
		})

		It("drops ExtraFields entries colliding with typed keys", func() {
			f := openTemp("pe")
			pe := ParallelEnvironmentConfig{
				Name:  "test",
				Slots: 4,
				ExtraFields: map[string]string{
					"slots": "999",
				},
			}
			Expect(writePE(f, pe)).To(Succeed())
			content, err := os.ReadFile(f.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("slots             4\n"))
			Expect(string(content)).NotTo(ContainSubstring("999"))
		})
	})

	Describe("writeCalendar", func() {
		It("emits ExtraFields after the typed fields", func() {
			f := openTemp("cal")
			cal := CalendarConfig{
				Name: "test",
				Year: "NONE",
				Week: "NONE",
				ExtraFields: map[string]string{
					"future_key": "future_value",
				},
			}
			Expect(writeCalendar(f, cal)).To(Succeed())
			f.Close()
			content, err := os.ReadFile(f.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring(
				"future_key future_value\n"))
		})
	})
})

// The reflection-based writers derive qconf attribute names from json
// tags. Since ExtraFields became serialisable (json:"extra_fields") the
// tag no longer excludes it, so the loops skip it by Go field name
// instead. These specs fail if that skip is lost -- without them the
// regression surfaces only as a corrupted file sent to the qmaster.
var _ = Describe("Reflection-based config writers", func() {

	Describe("writeGlobalConfig", func() {
		It("emits typed fields followed by ExtraFields", func() {
			var buf bytes.Buffer
			cfg := GlobalConfig{
				ExecdSpoolDir: "/opt/spool",
				ExtraFields:   map[string]string{"port_range": "2000-2100"},
			}
			Expect(writeGlobalConfig(&buf, cfg)).To(Succeed())
			Expect(buf.String()).To(ContainSubstring("execd_spool_dir /opt/spool\n"))
			Expect(buf.String()).To(ContainSubstring("port_range 2000-2100\n"))
		})

		It("never emits an extra_fields attribute line", func() {
			var buf bytes.Buffer
			cfg := GlobalConfig{
				ExecdSpoolDir: "/opt/spool",
				ExtraFields:   map[string]string{"port_range": "2000-2100"},
			}
			Expect(writeGlobalConfig(&buf, cfg)).To(Succeed())
			for _, line := range strings.Split(buf.String(), "\n") {
				Expect(strings.HasPrefix(line, "extra_fields")).To(BeFalse(),
					"qconf would reject the attribute line %q", line)
			}
		})

		It("rejects extras that would corrupt the file", func() {
			var buf bytes.Buffer
			cfg := GlobalConfig{
				ExtraFields: map[string]string{"a_key": "1\nprolog /tmp/pwn.sh"},
			}
			Expect(writeGlobalConfig(&buf, cfg)).NotTo(Succeed())
		})
	})

	Describe("writeSchedulerConfig", func() {
		It("emits typed fields followed by ExtraFields", func() {
			var buf bytes.Buffer
			cfg := SchedulerConfig{
				Algorithm:   "default",
				ExtraFields: map[string]string{"future_param": "42"},
			}
			Expect(writeSchedulerConfig(&buf, cfg)).To(Succeed())
			Expect(buf.String()).To(ContainSubstring("algorithm default\n"))
			Expect(buf.String()).To(ContainSubstring("future_param 42\n"))
		})

		It("never emits an extra_fields attribute line", func() {
			var buf bytes.Buffer
			cfg := SchedulerConfig{
				Algorithm:   "default",
				ExtraFields: map[string]string{"future_param": "42"},
			}
			Expect(writeSchedulerConfig(&buf, cfg)).To(Succeed())
			for _, line := range strings.Split(buf.String(), "\n") {
				Expect(strings.HasPrefix(line, "extra_fields")).To(BeFalse(),
					"qconf would reject the attribute line %q", line)
			}
		})
	})
})
