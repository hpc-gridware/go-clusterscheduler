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

package helper

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// ParseMemoryFromString takes a string like "4.078G" and returns bytes.
// Example: "15.6G" -> 15.6 * 1024 * 1024 * 1024
//
// man sge_types:
//
// "Memory specifiers are positive decimal, hexadecimal or octal integer
// constants which may be followed by a multiplier letter. Valid multiplier
// letters are k, K, m, M, g and G, where k means multiply the value by
// 1000, K multiply by 1024, m multiply by 1000*1000, M multiply by 1024*1024,
// g multiply by 1000*1000*1000 and G multiply by 1024*1024*1024. If no
// multiplier is present, the value is just counted in bytes."
//
// Following that convention t/T and p/P are accepted too (1000^4 and
// 1024^4, 1000^5 and 1024^5). The number may carry a fraction with or
// without a multiplier; the result is truncated to whole bytes.
//
// Negative values, NaN, infinities and values that do not fit into an
// int64 are rejected with an error.
func ParseMemoryFromString(m string) (int64, error) {
	if len(m) == 0 {
		return 0, errors.New("empty string")
	}

	// special value for unknown memory
	if m == "-" {
		return 0, nil
	}

	numberStr := m
	multiplier := int64(1)
	switch m[len(m)-1] {
	case 'K':
		multiplier = 1 << 10
	case 'M':
		multiplier = 1 << 20
	case 'G':
		multiplier = 1 << 30
	case 'T':
		multiplier = 1 << 40
	case 'P':
		multiplier = 1 << 50
	case 'k':
		multiplier = 1000
	case 'm':
		multiplier = 1000 * 1000
	case 'g':
		multiplier = 1000 * 1000 * 1000
	case 't':
		multiplier = 1000 * 1000 * 1000 * 1000
	case 'p':
		multiplier = 1000 * 1000 * 1000 * 1000 * 1000
	default:
		// no unit: integers are returned as is, decimals are truncated
		if value, err := strconv.ParseInt(m, 10, 64); err == nil {
			if value < 0 {
				return 0, fmt.Errorf("negative memory value: %s", m)
			}
			return value, nil
		}
	}
	if multiplier > 1 {
		numberStr = m[:len(m)-1]
	}

	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s: %v", numberStr, err)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("invalid number: %s: not finite", numberStr)
	}
	if number < 0 {
		return 0, fmt.Errorf("negative memory value: %s", m)
	}
	// float64(math.MaxInt64) rounds up to 2^63, which is already out of
	// range, so the comparison must include equality. Converting an out of
	// range float to int64 is implementation-defined in Go.
	value := number * float64(multiplier)
	if value >= float64(math.MaxInt64) {
		return 0, fmt.Errorf("memory value out of range: %s", m)
	}
	return int64(value), nil
}

// MemoryToString formats bytes in the largest of K, M and G the value
// reaches, truncating the remainder (1536 -> "1K", 1 TiB -> "1024G").
// Values below 1024, negative ones included, are printed as plain numbers.
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
