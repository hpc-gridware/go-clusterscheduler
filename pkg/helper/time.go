/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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
	"fmt"
	"strconv"
	"strings"
)

// ParseTimeResourceValueToSeconds parses a time resource value in the format
// HH:MM:SS and returns the total number of seconds.
func ParseTimeResourceValueToSeconds(value string) (int64, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format - expected HH:MM:SS")
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid time format - expected HH:MM:SS")
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid time format - expected HH:MM:SS")
	}
	seconds, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid time format - expected HH:MM:SS")
	}
	return int64(hours*3600 + minutes*60 + seconds), nil
}

// FormatSecondsToTimeResourceValue converts a total number of seconds into a time resource string in the format HH:MM:SS.
// It pads each component with leading zeros. Negative input values are clamped to 0.
func FormatSecondsToTimeResourceValue(totalSeconds int64) string {
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
