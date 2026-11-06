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

// Package nostdout_test holds a tripwire: library code under pkg/ must not
// write to stdout. A caller may use its stdout as a protocol channel -- an
// MCP server on the stdio transport speaks JSON-RPC there -- and one stray
// print corrupts every message after it. Errors are returned; warnings go
// to the standard logger (stderr).
//
// Two kinds of print are allowed because printing is their documented job:
// the command echo of a DryRun configuration (the line right after an
// "if ... DryRun" check), and Apply, which prints its plan.
package nostdout_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var stdoutWrite = regexp.MustCompile(`\bfmt\.Print(f|ln)?\(|\bos\.Stdout\b`)

// allowedFiles print by contract (see the package comment).
var allowedFiles = map[string]bool{
	"qconf/core/apply.go": true,
}

var _ = Describe("library code under pkg/", func() {
	It("never writes to stdout outside a DryRun echo", func() {
		root, err := filepath.Abs("../..")
		Expect(err).NotTo(HaveOccurred())
		var offenders []string
		err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if allowedFiles[filepath.ToSlash(rel)] {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				code := strings.TrimSpace(line)
				if strings.HasPrefix(code, "//") || !stdoutWrite.MatchString(code) {
					continue
				}
				if i > 0 && strings.Contains(lines[i-1], "DryRun") {
					continue
				}
				offenders = append(offenders, rel+":"+strconv.Itoa(i+1)+": "+code)
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(offenders).To(BeEmpty(), "library code writes to stdout:\n%s", strings.Join(offenders, "\n"))
	})
})
