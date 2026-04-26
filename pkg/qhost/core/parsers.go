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

	helper "github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
)

/*
Parses following output of qhost command:

HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
exec                    lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
...
*/
func ParseHosts(out string) ([]Host, error) {
	hosts := []Host{}

	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "HOSTNAME") {
			continue
		}
		if strings.HasPrefix(line, "---------------") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		// split line by whitespace
		fields := strings.Fields(line)
		if len(fields) < 11 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}

		var err error
		host := Host{}
		host.Name = fields[0]
		host.Arch = fields[1]
		host.NCPU, err = ParseIntOrUnknown(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid NCPU: %s", fields[2])
		}

		host.NSOC, err = ParseIntOrUnknown(fields[3])
		if err != nil {
			return nil, fmt.Errorf("invalid NSOC: %s", fields[3])
		}

		host.NCOR, err = ParseIntOrUnknown(fields[4])
		if err != nil {
			return nil, fmt.Errorf("invalid NCOR: %s", fields[4])
		}
		host.NTHR, err = ParseIntOrUnknown(fields[5])
		if err != nil {
			return nil, fmt.Errorf("invalid NTHR: %s", fields[5])
		}
		host.LOAD, err = ParseFloatOrUnknown(fields[6])
		if err != nil {
			return nil, fmt.Errorf("invalid LOAD: %s", fields[6])
		}
		host.MEMTOT, err = helper.ParseMemoryFromString(fields[7])
		if err != nil {
			return nil, fmt.Errorf("invalid MEMTOT: %s: %w", fields[7], err)
		}
		host.MEMUSE, err = helper.ParseMemoryFromString(fields[8])
		if err != nil {
			return nil, fmt.Errorf("invalid MEMUSE: %s: %w", fields[8], err)
		}
		host.SWAPTO, err = helper.ParseMemoryFromString(fields[9])
		if err != nil {
			return nil, fmt.Errorf("invalid SWAPTO: %s: %w", fields[9], err)
		}
		host.SWAPUS, err = helper.ParseMemoryFromString(fields[10])
		if err != nil {
			return nil, fmt.Errorf("invalid SWAPUS: %s: %w", fields[10], err)
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

// ParseIntOrUnknown parses an integer or "-" as 0.
func ParseIntOrUnknown(v string) (int, error) {
	if v == "-" {
		return 0, nil
	}
	ret, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("no int or -: %s", v)
	}
	return ret, nil
}

// ParseFloatOrUnknown parses a float or "-" as 0.
func ParseFloatOrUnknown(v string) (float64, error) {
	if v == "-" {
		return 0, nil
	}
	return strconv.ParseFloat(v, 64)
}

/*
HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.34   15.6G  420.0M    1.5G     0.0

	hl:arch=lx-amd64
	hl:num_proc=4.000000
	hl:mem_total=15.617G
	...
*/

// ParseHostFullMetrics parses the output of qhost -F.
func ParseHostFullMetrics(out string) ([]HostFullMetrics, error) {
	hosts := []HostFullMetrics{}
	lines := strings.Split(out, "\n")

	var currentHost *HostFullMetrics = nil

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "HOSTNAME") ||
			strings.HasPrefix(line, "----") {
			continue
		}

		if len(lines[i]) > 0 && (lines[i][0] == ' ' || lines[i][0] == '\t') {
			// Attribute line or resource availability line
			if currentHost == nil {
				return nil, fmt.Errorf("attribute line encountered before any host line")
			}
			attributeLine := strings.TrimSpace(line)
			// Skip lines that don't match prefix:name=value format (e.g. queue
			// or job lines from qhost -F -q or qhost -F -j).
			_ = parseAttributeLine(attributeLine, currentHost)
		} else {
			// New host line
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &HostFullMetrics{}

			fields := strings.Fields(line)
			if len(fields) < 11 {
				return nil, fmt.Errorf("invalid host line: %s", line)
			}
			currentHost.Name = fields[0]
			currentHost.Arch = fields[1]

			ncpu, err := strconv.Atoi(fields[2])
			if err == nil {
				currentHost.NumProc = float64(ncpu)
			}

			loadAvg, err := strconv.ParseFloat(fields[6], 64)
			if err == nil {
				currentHost.LoadAvg = loadAvg
			}

			memTotal, err := helper.ParseMemoryFromString(fields[7])
			if err == nil {
				currentHost.MemTotal = memTotal
			}

			memUsed, err := helper.ParseMemoryFromString(fields[8])
			if err == nil {
				currentHost.MemUsed = memUsed
			}

			swapTotal, err := helper.ParseMemoryFromString(fields[9])
			if err == nil {
				currentHost.SwapTotal = swapTotal
			}

			swapUsed, err := helper.ParseMemoryFromString(fields[10])
			if err == nil {
				currentHost.SwapUsed = swapUsed
			}

			currentHost.Resources = make(map[string]ResourceAvailability)
		}
	}

	if currentHost != nil {
		hosts = append(hosts, *currentHost)
	}

	return hosts, nil
}

func parseAttributeLine(line string, currentHost *HostFullMetrics) error {
	// The line is expected to be in the format:
	// [Availability][Source]:[resource_name]=[value]
	// e.g., "hl:load_avg=0.600000"

	prefixAndRest := strings.SplitN(line, ":", 2)
	if len(prefixAndRest) != 2 {
		return fmt.Errorf("invalid attribute line format: %s", line)
	}
	prefix := prefixAndRest[0]
	rest := prefixAndRest[1]

	if len(prefix) != 2 {
		return fmt.Errorf("invalid prefix length: %s", prefix)
	}
	availabilityLetter := prefix[0]
	sourceLetter := prefix[1]

	attrParts := strings.SplitN(rest, "=", 2)
	if len(attrParts) != 2 {
		return fmt.Errorf("invalid attribute line, missing '=': %s", line)
	}
	resourceName := attrParts[0]
	value := attrParts[1]

	availabilityMap := map[byte]string{
		'g': "g", // cluster global
		'h': "h", // host total
		'q': "q", // queue total
	}
	sourceMap := map[byte]string{
		'l': "l", // load value
		'L': "L", // load value after scaling
		'c': "c", // consumable resource
		'f': "F", // fixed availability
	}

	resourceAvailabilityLimitedBy, ok := availabilityMap[availabilityLetter]
	if !ok {
		return fmt.Errorf("unknown availability letter: %c", availabilityLetter)
	}
	source, ok := sourceMap[sourceLetter]
	if !ok {
		return fmt.Errorf("unknown source letter: %c", sourceLetter)
	}

	switch resourceName {
	case "arch":
		currentHost.Arch = value
	case "num_proc":
		numProc, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid num_proc value: %s", value)
		}
		currentHost.NumProc = numProc
	case "mem_total":
		memTotal, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid mem_total value: %s", value)
		}
		currentHost.MemTotal = memTotal
	case "swap_total":
		swapTotal, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid swap_total value: %s", value)
		}
		currentHost.SwapTotal = swapTotal
	case "virtual_total":
		virtualTotal, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid virtual_total value: %s", value)
		}
		currentHost.VirtualTotal = virtualTotal
	case "load_avg":
		loadAvg, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid load_avg value: %s", value)
		}
		currentHost.LoadAvg = loadAvg
	case "load_short":
		loadShort, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid load_short value: %s", value)
		}
		currentHost.LoadShort = loadShort
	case "load_medium":
		loadMedium, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid load_medium value: %s", value)
		}
		currentHost.LoadMedium = loadMedium
	case "load_long":
		loadLong, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid load_long value: %s", value)
		}
		currentHost.LoadLong = loadLong
	case "mem_free":
		memFree, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid mem_free value: %s", value)
		}
		currentHost.MemFree = memFree
	case "swap_free":
		swapFree, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid swap_free value: %s", value)
		}
		currentHost.SwapFree = swapFree
	case "virtual_free":
		virtualFree, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid virtual_free value: %s", value)
		}
		currentHost.VirtualFree = virtualFree
	case "mem_used":
		memUsed, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid mem_used value: %s", value)
		}
		currentHost.MemUsed = memUsed
	case "swap_used":
		swapUsed, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid swap_used value: %s", value)
		}
		currentHost.SwapUsed = swapUsed
	case "virtual_used":
		virtualUsed, err := helper.ParseMemoryFromString(value)
		if err != nil {
			return fmt.Errorf("invalid virtual_used value: %s", value)
		}
		currentHost.VirtualUsed = virtualUsed
	case "cpu":
		cpu, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid cpu value: %s", value)
		}
		currentHost.CPU = cpu
	case "m_topology":
		currentHost.Topology = value
	case "m_topology_inuse":
		currentHost.TopologyInuse = value
	case "m_socket":
		socket, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid m_socket value: %s", value)
		}
		currentHost.Socket = int64(socket)
	case "m_core":
		core, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid m_core value: %s", value)
		}
		currentHost.Core = int64(core)
	case "m_thread":
		thread, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid m_thread value: %s", value)
		}
		currentHost.Thread = int64(thread)
	case "np_load_avg":
		npLoadAvg, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid np_load_avg value: %s", value)
		}
		currentHost.NPLoadAvg = npLoadAvg
	case "np_load_short":
		npLoadShort, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid np_load_short value: %s", value)
		}
		currentHost.NPLoadShort = npLoadShort
	case "np_load_medium":
		npLoadMedium, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid np_load_medium value: %s", value)
		}
		currentHost.NPLoadMedium = npLoadMedium
	case "np_load_long":
		npLoadLong, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid np_load_long value: %s", value)
		}
		currentHost.NPLoadLong = npLoadLong
	default:
		// Handle as a self-defined resource
		if currentHost.Resources == nil {
			currentHost.Resources = make(map[string]ResourceAvailability)
		}
		ra := ResourceAvailability{
			Name:                          resourceName,
			StringValue:                   value,
			ResourceAvailabilityLimitedBy: resourceAvailabilityLimitedBy,
			Source:                        source,
			FullString:                    line,
		}

		floatValue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			ra.FloatValue = floatValue
		}

		currentHost.Resources[resourceName] = ra
	}
	return nil
}

// ParseHostsRaw parses the same input as ParseHosts but returns the raw
// column tokens for each host (e.g. "-", "61.6G") without conversion.
// Callers that need to render the qhost columns verbatim (such as the
// native GCS JSON wrapper) use this entry point. The "global" row is
// included; header lines, separators, blank lines, and indented
// attribute lines (from qhost -F output) are skipped — so the same
// helper handles both bare qhost and qhost -F input.
func ParseHostsRaw(out string) ([]HostRaw, error) {
	hosts := []HostRaw{}

	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "HOSTNAME") {
			continue
		}
		if strings.HasPrefix(line, "---------------") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Indented lines belong to the previous host's attribute or
		// resource section (qhost -F). They are not host headers.
		if line[0] == ' ' || line[0] == '\t' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		hosts = append(hosts, HostRaw{
			Name: fields[0],
			Cols: append([]string(nil), fields[1:11]...),
		})
	}
	return hosts, nil
}
