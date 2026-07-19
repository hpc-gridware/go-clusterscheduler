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
	"os"
	"path/filepath"

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
