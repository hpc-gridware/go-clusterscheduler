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
	"strconv"
	"strings"
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
// Example: "15.6G" -> 15.6 * 1024 * 1024 * 1024
func ParseMemoryFromString(m string) (int64, error) {
	if len(m) == 0 {
		return 0, errors.New("empty string")
	}

	if m == "0" || m == "0.0" || m == "0.00" || m == "0.000" {
		return 0, nil
	}

	// last character must be a multiplier
	if !strings.HasSuffix(m, "k") && !strings.HasSuffix(m, "K") &&
		!strings.HasSuffix(m, "m") && !strings.HasSuffix(m, "M") &&
		!strings.HasSuffix(m, "g") && !strings.HasSuffix(m, "G") {
		// no unit, return the number as is
		return strconv.ParseInt(m, 10, 64)
	}

	unit := m[len(m)-1]
	numberStr := m[:len(m)-1]

	// Parse the number part
	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s: %v", numberStr, err)
	}

	// Determine the multiplier
	multiplier := int64(1) // Default is bytes if no unit
	switch strings.ToUpper(string(unit)) {
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

	return int64(number * float64(multiplier)), nil
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
