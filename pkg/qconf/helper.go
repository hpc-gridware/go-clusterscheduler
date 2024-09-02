/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
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
	"fmt"
	"strings"
)

// JoinList joins a list of elements with a separator.
// If the list is empty, it returns "NONE" as it is the default value for
// empty lists in qconf.
func JoinList(elements []string, sep string) string {
	if len(elements) == 0 {
		return "NONE"
	}
	return strings.Join(elements, sep)
}

func JoinStringFloatMap(m map[string]float64, sep string) string {
	if len(m) == 0 {
		return "NONE"
	}
	elems := make([]string, 0, len(m))
	for k, v := range m {
		elems = append(elems, fmt.Sprintf("%s=%f", k, v))
	}
	return strings.Join(elems, sep)
}

func JoinStringStringMap(m map[string]string, sep string) string {
	if len(m) == 0 {
		return "NONE"
	}
	elems := make([]string, 0, len(m))
	for k, v := range m {
		elems = append(elems, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(elems, sep)
}

func ParseCommaSeparatedMultiLineValues(lines []string, i int) []string {
	vals, _ := ParseMultiLineValue(lines, i)
	entries := strings.Split(vals, ",")
	if len(entries) == 1 && strings.ToUpper(entries[0]) == "NONE" {
		return nil
	}
	return entries
}

func ParseSpaceSeparatedMultiLineValues(lines []string, i int) []string {
	vals, _ := ParseMultiLineValue(lines, i)
	entries := strings.Fields(vals)
	if len(entries) == 1 && strings.ToUpper(entries[0]) == "NONE" {
		return nil
	}
	return entries
}

// ParseSpaceAndCommaSeparatedMultiLineValues splits on spaces and commas.
func ParseSpaceAndCommaSeparatedMultiLineValues(lines []string, i int) []string {
	vals, _ := ParseMultiLineValue(lines, i)

	// Split on spaces and commas
	entries := strings.FieldsFunc(vals, func(r rune) bool {
		return r == ' ' || r == ','
	})
	if len(entries) == 1 && strings.ToUpper(entries[0]) == "NONE" {
		return nil
	}
	return entries
}

// parseMultiLineValue parses a multi-line value from the output.
// This is tricky because the output is not structured and the values can be
// split over multiple lines.
// The input is an array of all lines, the current index, the current line,
// and the fields of the current line. fields[0] is the detected key (like "reporting_params").
// The function returns the value and a boolean indicating if the value is multi-line.
//
// Example:
// ...
// qmaster_params               none
// execd_params                 none
//
//	reporting_params             accounting=true reporting=false finished_jobs=0 \
//		  test=blub test=bla
//
// ...
// lines is the array of all lines
// i is the line number with "reporting_params"
// The rule is that each non-multi-line output does not have a "  " prefix.
func ParseMultiLineValue(lines []string, i int) (string, bool) {
	line := lines[i]
	fields := strings.Fields(line)
	value := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	if strings.HasSuffix(value, "\\") {
		// multi-line value
		value = strings.TrimSuffix(value, "\\")
		for i, line := range lines {
			fds := strings.Fields(line)
			if len(fds) == 0 {
				continue
			}
			// find key like "reporting_params"
			if fds[0] == fields[0] {
				// multi-line values are indented by spaces, find all remaining lines
				for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], "  "); j++ {
					// Now the question is if we do at " " or "," or other
					// separators? We expect that the line ends with a separator.
					value += "" + strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(lines[j]), "\\"))
				}
			}
		}
		return value, true
	}
	return value, false
}
