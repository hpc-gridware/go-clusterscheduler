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
	"strings"
)

// usagePatternsKey is the reporting parameter which selects the custom usage
// values written to the accounting file.
const usagePatternsKey = "usage_patterns="

// MergeUsagePatterns returns the reporting parameters (the ReportingParams of
// a GlobalConfig) with the usage_patterns entry updated so that the given
// group contains every requested pattern. The input slice is not modified.
//
// The value has the form "<name>:<expression>[;<name>:<expression>]" and the
// expression is matched with fnmatch, so alternatives are joined with "|"
// (for example "gpu:nvidia*|amd*"). Groups of other tools are preserved and
// the order of existing patterns is kept, so that applying the function twice
// changes nothing: installers are expected to run repeatedly. If there is
// more than one usage_patterns entry the last one is updated, because that
// is the one the scheduler uses.
//
// An empty group or an empty pattern list changes nothing.
func MergeUsagePatterns(reportingParams []string, group string,
	patterns ...string) []string {

	merged := make([]string, len(reportingParams))
	copy(merged, reportingParams)
	if group == "" || len(patterns) == 0 {
		return merged
	}

	for i := len(merged) - 1; i >= 0; i-- {
		if !strings.HasPrefix(merged[i], usagePatternsKey) {
			continue
		}
		value := strings.TrimPrefix(merged[i], usagePatternsKey)
		merged[i] = usagePatternsKey + mergeIntoUsageGroups(value, group, patterns)
		return merged
	}

	return append(merged, usagePatternsKey+group+":"+strings.Join(patterns, "|"))
}

// mergeIntoUsageGroups adds the patterns to the named group of a
// usage_patterns value and leaves every other group untouched.
func mergeIntoUsageGroups(value, group string, patterns []string) string {
	var groups []string
	if strings.TrimSpace(value) != "" {
		groups = strings.Split(value, ";")
	}

	for i, entry := range groups {
		name, expression, found := strings.Cut(entry, ":")
		if !found || strings.TrimSpace(name) != group {
			continue
		}
		groups[i] = name + ":" + addUsagePatterns(expression, patterns)
		return strings.Join(groups, ";")
	}

	return strings.Join(append(groups, group+":"+strings.Join(patterns, "|")), ";")
}

// addUsagePatterns appends the patterns which are not part of the expression
// yet.
func addUsagePatterns(expression string, patterns []string) string {
	var existing []string
	if strings.TrimSpace(expression) != "" {
		existing = strings.Split(expression, "|")
	}
	for _, pattern := range patterns {
		found := false
		for _, e := range existing {
			if strings.TrimSpace(e) == pattern {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, pattern)
		}
	}
	return strings.Join(existing, "|")
}
