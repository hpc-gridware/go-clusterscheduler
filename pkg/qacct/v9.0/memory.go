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

package qacct

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

// ParseMemoryFromString takes a string like "4.078G".
//
// man sge_types:
//
// "Memory specifiers are positive decimal, hexadecimal or octal integer
// constants which may be followed by a multiplier letter. Valid multiplier
// letters are k, K, m, M, g and G, where k means multiply the value by
// 1000, K multiply by 1024, m multiply by 1000*1000, M multiply by 1024*1024,
// g multiply by 1000*1000*1000 and G multiply by 1024*1024*1024. If no
// multiplier is present, the value is just counted in bytes.""
func ParseMemoryFromString(m string) (int64, error) {
	if len(m) == 0 {
		return 0, errors.New("empty string")
	}

	// Find position of the first non-digit character
	var i int
	for i = len(m) - 1; i >= 0; i-- {
		if !unicode.IsDigit(rune(m[i])) && m[i] != '.' {
			break
		}
	}

	// Separate the number part and the multiplier part
	numberStr := m[:i+1]
	unit := m[i+1:]

	// Parse the number part
	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, err
	}

	// Determine the multiplier
	multiplier := int64(1) // Default is bytes if no unit
	switch strings.ToUpper(unit) {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "k":
		multiplier = 1000
	case "m":
		multiplier = 1000 * 1000
	case "g":
		multiplier = 1000 * 1000 * 1000
	case "":
		multiplier = 1
	default:
		return 0, errors.New("invalid unit")
	}

	// Calculate the result
	result := int64(number * float64(multiplier))

	return result, nil
}

func MemoryToString(m int64) string {
	if m < 1024 {
		return strconv.FormatInt(m, 10)
	} else if m < 1024*1024 {
		return strconv.FormatInt(m/1024, 10) + "K"
	} else if m < 1024*1024*1024 {
		return strconv.FormatInt(m/(1024*1024), 10) + "M"
	} else {
		return strconv.FormatInt(m/(1024*1024*1024), 10) + "G"
	}
}
