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
	"strconv"
	"strings"
)

// ResourceMapInstance is one instance of an RSMAP complex_values entry.
type ResourceMapInstance struct {
	// ID names the instance, e.g. "gpu0" or "1".
	ID string
	// Characteristics is the instance's characteristics block, e.g.
	// {"devices": "/dev/nvidia0:rw;/dev/nvidiactl:r"}, or nil for a bare id.
	// Characteristics are a Gridware Cluster Scheduler feature.
	Characteristics map[string]string
}

// ParseResourceMap parses the value of an RSMAP entry in complex_values --
// the part after "<name>=", such as "2(gpu0[devices=/dev/nvidia0:rw] gpu1)"
// -- into its amount and instances, following sge_complex(5):
//
//   - an id-spec is a bare id ("gpu0"), an integer range ("1-3", expanded to
//     1 2 3), or an id followed by a characteristics block
//     ("gpu0[device=/dev/nvidia0,memory=80G]")
//   - without an id list ("4") the instances are named 0..amount-1
//   - whitespace separates id-specs, and is ignored inside a block
//   - a repeated id models a shared device and is kept as listed
func ParseResourceMap(value string) (int, []ResourceMapInstance, error) {
	v := strings.TrimSpace(value)
	open := strings.IndexByte(v, '(')
	amountText := v
	if open >= 0 {
		amountText = v[:open]
	}
	amount, err := strconv.Atoi(strings.TrimSpace(amountText))
	if err != nil || amount < 0 {
		return 0, nil, fmt.Errorf("RSMAP %q: amount %q is not a non-negative integer", value, amountText)
	}
	if open < 0 {
		instances := make([]ResourceMapInstance, amount)
		for i := range instances {
			instances[i].ID = strconv.Itoa(i)
		}
		return amount, instances, nil
	}
	if !strings.HasSuffix(v, ")") {
		return 0, nil, fmt.Errorf("RSMAP %q: id list is missing its closing ')'", value)
	}

	body := v[open+1 : len(v)-1]
	var instances []ResourceMapInstance
	for i := 0; i < len(body); {
		if isBlank(body[i]) {
			i++
			continue
		}
		start := i
		for i < len(body) && !isBlank(body[i]) && body[i] != '[' {
			i++
		}
		id := body[start:i]
		if id == "" {
			return 0, nil, fmt.Errorf("RSMAP %q: characteristics block without an id", value)
		}

		if i < len(body) && body[i] == '[' {
			end := strings.IndexByte(body[i:], ']')
			if end < 0 {
				return 0, nil, fmt.Errorf("RSMAP %q: characteristics of %q are missing their closing ']'", value, id)
			}
			if _, _, isRange := integerRange(id); isRange {
				return 0, nil, fmt.Errorf("RSMAP %q: characteristics cannot be attached to the range %q", value, id)
			}
			chars, err := parseCharacteristics(body[i+1 : i+end])
			if err != nil {
				return 0, nil, fmt.Errorf("RSMAP %q: instance %q: %w", value, id, err)
			}
			instances = append(instances, ResourceMapInstance{ID: id, Characteristics: chars})
			i += end + 1
			continue
		}

		if lo, hi, isRange := integerRange(id); isRange {
			for n := lo; n <= hi; n++ {
				instances = append(instances, ResourceMapInstance{ID: strconv.Itoa(n)})
			}
			continue
		}
		instances = append(instances, ResourceMapInstance{ID: id})
	}
	return amount, instances, nil
}

// parseCharacteristics parses the inside of a characteristics block. All
// whitespace is dropped first: a value may not contain any, and a wrapped
// definition leaves some behind after its continuation is folded.
func parseCharacteristics(block string) (map[string]string, error) {
	chars := map[string]string{}
	for _, token := range strings.Split(strings.Join(strings.Fields(block), ""), ",") {
		if token == "" {
			continue
		}
		name, val, found := strings.Cut(token, "=")
		if !found || name == "" {
			return nil, fmt.Errorf("characteristic %q is not name=value", token)
		}
		chars[name] = val
	}
	return chars, nil
}

// integerRange reports whether s is an integer range such as "1-3".
func integerRange(s string) (int, int, bool) {
	loText, hiText, found := strings.Cut(s, "-")
	if !found {
		return 0, 0, false
	}
	lo, errLo := strconv.Atoi(loText)
	hi, errHi := strconv.Atoi(hiText)
	if errLo != nil || errHi != nil || lo < 0 || hi < lo {
		return 0, 0, false
	}
	return lo, hi, true
}

// splitTopLevel splits s on sep, but never inside (...) or [...].
func splitTopLevel(s, sep string) []string {
	if sep == "" {
		return []string{s}
	}
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && strings.HasPrefix(s[i:], sep) {
				parts = append(parts, s[start:i])
				i += len(sep) - 1
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

func isBlank(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
