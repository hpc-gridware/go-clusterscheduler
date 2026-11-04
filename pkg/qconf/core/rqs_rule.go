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
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// RQSRule is one parsed limit rule of a resource quota set, i.e. one
// "limit" line of sge_resource_quota(5):
//
//	limit [name <rule_name>] [users ...] [projects ...] [pes ...] [queues ...] [hosts ...] to <resource>=<value>[,...]
//
// The rule is kept as written; matching it against jobs or queue instances
// is up to the caller.
type RQSRule struct {
	// Name is the explicit rule name or "" (the scheduler numbers unnamed
	// rules by position, 1-based).
	Name     string
	Users    RQSScope
	Projects RQSScope
	PEs      RQSScope
	Queues   RQSScope
	Hosts    RQSScope
	// Limits are the resource=value pairs after "to", in order.
	Limits []RQSLimit
}

// RQSScope is one filter of a rule.
type RQSScope struct {
	// Names are the included names without "!"; "*" when "all"; empty means
	// "not filtered".
	Names []string
	// Excluded are the names given with a leading "!".
	Excluded []string
	// Expand is true when the names were given in {...}: the limit then
	// applies to each member separately instead of to all of them together.
	Expand bool
}

// Filtered reports whether the scope restricts the rule, i.e. whether any
// included or excluded name was given.
func (r RQSScope) Filtered() bool {
	return len(r.Names) > 0 || len(r.Excluded) > 0
}

// RQSLimit is one resource limit of a rule.
type RQSLimit struct {
	Resource string
	// Value is the limit as written: a literal ("24", "4G") or a dynamic
	// expression ("$num_proc*2").
	Value string
}

// ParseRQSRule parses one limit line of a resource quota set, as found in
// ResourceQuotaSetConfig.Limits. A leading "limit" keyword is accepted and
// ignored, so a raw line of "qconf -srqs" parses too.
func ParseRQSRule(limit string) (RQSRule, error) {
	fields := strings.Fields(limit)
	if len(fields) > 0 && fields[0] == "limit" {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return RQSRule{}, errors.New("empty resource quota rule")
	}

	var rule RQSRule
	seen := map[string]bool{}
	for i := 0; i < len(fields); i++ {
		keyword := fields[i]
		if keyword == "to" {
			limits, err := parseRQSLimits(fields[i+1:])
			if err != nil {
				return RQSRule{}, fmt.Errorf("rule %q: %w", limit, err)
			}
			rule.Limits = limits
			return rule, nil
		}
		var target *RQSScope
		switch keyword {
		case "name":
		case "users":
			target = &rule.Users
		case "projects":
			target = &rule.Projects
		case "pes":
			target = &rule.PEs
		case "queues":
			target = &rule.Queues
		case "hosts":
			target = &rule.Hosts
		default:
			return RQSRule{}, fmt.Errorf("rule %q: unknown keyword %q", limit, keyword)
		}
		if seen[keyword] {
			return RQSRule{}, fmt.Errorf("rule %q: keyword %q given twice", limit, keyword)
		}
		seen[keyword] = true
		if i+1 >= len(fields) {
			return RQSRule{}, fmt.Errorf("rule %q: keyword %q without value", limit, keyword)
		}
		value := fields[i+1]
		i++
		if target == nil {
			rule.Name = value
			continue
		}
		scope, err := parseRQSScope(value)
		if err != nil {
			return RQSRule{}, fmt.Errorf("rule %q: %s: %w", limit, keyword, err)
		}
		*target = scope
	}
	return RQSRule{}, fmt.Errorf("rule %q: missing \"to\"", limit)
}

// ParseRQSRules parses every limit line of a resource quota set. A rule
// without an explicit name gets its 1-based position as name, matching
// the "<set>/<position>" naming qquota uses for unnamed rules.
func ParseRQSRules(limits []string) ([]RQSRule, error) {
	rules := make([]RQSRule, 0, len(limits))
	for pos, limit := range limits {
		rule, err := ParseRQSRule(limit)
		if err != nil {
			return nil, err
		}
		if rule.Name == "" {
			rule.Name = strconv.Itoa(pos + 1)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// parseRQSScope reads one filter value such as "alice,bob", "{*}",
// "!short.q", "!{a,b}" or "{!a,b}". Braces mark an expanded scope and a
// leading "!" marks an excluded name, in either order.
func parseRQSScope(value string) (RQSScope, error) {
	var scope RQSScope
	excludeAll := strings.HasPrefix(value, "!{")
	if excludeAll {
		value = value[1:]
	}
	if strings.HasPrefix(value, "{") {
		if !strings.HasSuffix(value, "}") {
			return RQSScope{}, fmt.Errorf("unterminated {...} in %q", value)
		}
		scope.Expand = true
		value = value[1 : len(value)-1]
	}
	if value == "" {
		return RQSScope{}, errors.New("empty scope")
	}
	for _, name := range strings.Split(value, ",") {
		excluded := excludeAll
		if strings.HasPrefix(name, "!") {
			excluded = true
			name = name[1:]
		}
		if name == "" || strings.ContainsAny(name, "{}!") {
			return RQSScope{}, fmt.Errorf("invalid name %q in scope", name)
		}
		if excluded {
			scope.Excluded = append(scope.Excluded, name)
		} else {
			scope.Names = append(scope.Names, name)
		}
	}
	return scope, nil
}

// parseRQSLimits reads the resource=value pairs after "to". Whitespace
// between the pairs is tolerated, so "slots=24, h_vmem=4G" parses as two
// limits.
func parseRQSLimits(fields []string) ([]RQSLimit, error) {
	joined := strings.Join(fields, "")
	if joined == "" {
		return nil, errors.New("no limits after \"to\"")
	}
	var limits []RQSLimit
	for _, pair := range strings.Split(joined, ",") {
		resource, value, found := strings.Cut(pair, "=")
		if !found || resource == "" || value == "" || strings.Contains(value, "=") {
			return nil, fmt.Errorf("invalid limit %q, want resource=value", pair)
		}
		limits = append(limits, RQSLimit{Resource: resource, Value: value})
	}
	return limits, nil
}
