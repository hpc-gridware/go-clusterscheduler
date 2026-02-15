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

package core

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

// JoinListWithOverrides needs to create strings like this:
// pe1,p2,[host=p2],[master=pe1] or
// slots=10,mem_free=1G,[sim1=slots=20 mem_free=2G],[sim2=slots=30,mem_free=3G]
// but always the non override elements first and then the overrides.
func JoinListWithOverrides(elements []string, sep string) string {
	if len(elements) == 0 {
		return "NONE"
	}

	// Separate regular elements from overrides
	overrides := make([]string, 0)
	mainelems := make([]string, 0)
	for _, elem := range elements {
		if strings.Contains(elem, "[") {
			overrides = append(overrides, elem)
			continue
		}
		mainelems = append(mainelems, elem)
	}

	// If there are no main elements, just join the overrides
	if len(mainelems) == 0 {
		return strings.Join(overrides, sep)
	}

	// Join main elements first
	result := strings.Join(mainelems, sep)

	// Add overrides at the end with proper separator - it always "
	if len(overrides) > 0 {
		// Only add separator if we have main elements
		result += "," + strings.Join(overrides, ",")
	}

	return result
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

// ParseSpaceSeparatedValuesWithOverrides. Like PE list in queue config.
// It can look like this: pe1 p2,[host=p2]
func ParseSpaceSeparatedValuesWithOverrides(lines []string, i int) []string {
	// Assuming single line output! First part is the name of the element.
	// pe_list       make test test2,[master=make test2],[global=test]
	// remove the "pe_list" (which can have different names)
	values := strings.SplitAfterN(lines[i], " ", 2)
	if len(values) == 1 {
		return nil
	}

	value := strings.TrimSpace(values[1])

	if value == "NONE" {
		return nil
	}

	// now split on ","
	fields := strings.Split(value, ",")

	// fields 0 has only space separated values
	pelist := strings.Split(fields[0], " ")

	// in fields[1:] we have the overrides ([master=make test2])
	return append(pelist, fields[1:]...)
}

// ParseCommaSeparatedValuesWithOverrides. Like complex_values in queue config.
// It can look like this: slots=10,mem_free=1G,[sim1=slots=20,mem_free=2G]
func ParseCommaSeparatedValuesWithOverrides(lines []string, i int) []string {
	// Example:
	// slots=10,mem_free=1G,[sim1=slots=20,mem_free=2G],[sim2=slots=30,mem_free=3G]
	// results in:
	// []string{"slots=10", "mem_free=1G", "[sim1=slots=20,mem_free=2G]", "[sim2=slots=30,mem_free=3G]"}

	// Assuming single line output! First part is the name of the element.
	// complex_values    slots=10,mem_free=1G,[sim1=slots=20,mem_free=2G],[sim2=slots=30,mem_free=3G]
	// remove the field name (which can have different names)
	values := strings.SplitAfterN(lines[i], " ", 2)
	if len(values) == 1 {
		return nil
	}

	value := strings.TrimSpace(values[1])
	if value == "NONE" {
		return nil
	}

	// Process the value string to handle the special case of overrides
	var result []string
	var currentValue string
	var inOverride bool
	var overrideDepth int

	for _, char := range value {
		switch char {
		case '[':
			if inOverride {
				// Nested override - just add to current value
				overrideDepth++
				currentValue += string(char)
			} else {
				// Start of an override
				inOverride = true
				// If we have a current value, add it to results
				if currentValue != "" {
					result = append(result, strings.TrimSpace(currentValue))
					currentValue = ""
				}
				currentValue += string(char)
			}
		case ']':
			if overrideDepth > 0 {
				// Closing a nested override
				overrideDepth--
				currentValue += string(char)
			} else {
				// End of an override
				inOverride = false
				currentValue += string(char)
				result = append(result, strings.TrimSpace(currentValue))
				currentValue = ""
			}
		case ',':
			if inOverride {
				// Comma inside an override - just add to current value
				currentValue += string(char)
			} else {
				// Comma outside override - end of a value
				if currentValue != "" {
					result = append(result, strings.TrimSpace(currentValue))
					currentValue = ""
				}
			}
		default:
			currentValue += string(char)
		}
	}

	// Add any remaining value
	if currentValue != "" {
		result = append(result, strings.TrimSpace(currentValue))
	}

	return result
}

// pe1,p2 vs p1,[host=p2]
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
