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
	"fmt"
	"strings"
)

// usagePatternsKey is the reporting parameter which selects the custom usage
// values written to the accounting file. qmaster matches the name without
// regard to case.
const usagePatternsKey = "usage_patterns="

// MergeUsagePatterns returns the reporting parameters (the ReportingParams of
// a GlobalConfig, one parameter per element) with the usage_patterns entry
// updated so that the given group contains every requested pattern. The
// input slice is not modified.
//
// The value has the form "<name>:<expression>[;<name>:<expression>]" and the
// expression is matched with fnmatch, so alternatives are joined with "|"
// (for example "gpu:nvidia*|amd*"). A pattern argument may itself be such an
// alternative list. Groups of other tools are preserved and the order of
// existing patterns is kept, so that applying the function twice changes
// nothing: installers are expected to run repeatedly. If there is more than
// one usage_patterns entry the last one is updated, because that is the one
// qmaster uses.
//
// Without patterns nothing changes. An error is returned, and nothing is
// changed, for a group or pattern qmaster would read differently than it is
// given here (one with a blank or one of ",;:="), and for the group names
// "rusage", "eusage" and "usage": qmaster accepts them but writes accounting
// records with that key twice, which no reader of the file gets right.
func MergeUsagePatterns(reportingParams []string, group string,
	patterns ...string) ([]string, error) {

	merged := make([]string, len(reportingParams))
	copy(merged, reportingParams)

	var wanted []string
	for _, argument := range patterns {
		for _, pattern := range strings.Split(argument, "|") {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if strings.ContainsAny(pattern, usagePatternsSeparators) {
				return merged, fmt.Errorf("usage pattern %q contains a blank or one of \",;:=\"", pattern)
			}
			wanted = appendMissing(wanted, pattern)
		}
	}
	if len(wanted) == 0 {
		return merged, nil
	}

	group = strings.TrimSpace(group)
	switch {
	case group == "":
		return merged, fmt.Errorf("usage patterns need a group name")
	case strings.ContainsAny(group, usagePatternsSeparators+"|"):
		return merged, fmt.Errorf("usage patterns group %q contains a blank or one of \",;:=|\"", group)
	case group == "rusage" || group == "eusage" || group == "usage":
		return merged, fmt.Errorf("usage patterns group %q is the name of a standard usage group of the accounting record", group)
	}

	for i := len(merged) - 1; i >= 0; i-- {
		entry := merged[i]
		if len(entry) < len(usagePatternsKey) ||
			!strings.EqualFold(entry[:len(usagePatternsKey)], usagePatternsKey) {
			continue
		}
		key, value := entry[:len(usagePatternsKey)], entry[len(usagePatternsKey):]
		merged[i] = key + mergeIntoUsageGroups(value, group, wanted)
		return merged, nil
	}

	return append(merged, usagePatternsKey+group+":"+strings.Join(wanted, "|")), nil
}

// usagePatternsSeparators are the characters which end a reporting
// parameter (blank and comma) or structure the usage_patterns value.
const usagePatternsSeparators = " \t\n,;:="

// mergeIntoUsageGroups adds the patterns to the named group of a
// usage_patterns value and leaves every other group untouched.
func mergeIntoUsageGroups(value, group string, patterns []string) string {
	var groups []string
	for _, entry := range strings.Split(value, ";") {
		if strings.TrimSpace(entry) != "" {
			groups = append(groups, entry)
		}
	}

	for i, entry := range groups {
		name, expression, found := strings.Cut(entry, ":")
		if !found || strings.TrimSpace(name) != group {
			continue
		}
		var existing []string
		for _, pattern := range strings.Split(expression, "|") {
			if strings.TrimSpace(pattern) != "" {
				existing = append(existing, pattern)
			}
		}
		for _, pattern := range patterns {
			existing = appendMissing(existing, pattern)
		}
		groups[i] = name + ":" + strings.Join(existing, "|")
		return strings.Join(groups, ";")
	}

	return strings.Join(append(groups, group+":"+strings.Join(patterns, "|")), ";")
}

// appendMissing appends the pattern unless the list has it already.
func appendMissing(list []string, pattern string) []string {
	for _, existing := range list {
		if strings.TrimSpace(existing) == pattern {
			return list
		}
	}
	return append(list, pattern)
}
