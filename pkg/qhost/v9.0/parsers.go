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

package qhost

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
		if strings.HasPrefix(line, "global") {
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
		host.NCPU, err = strconv.Atoi(fields[2])
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
			return nil, fmt.Errorf("invalid MEMTOT: %s: %v", fields[7], err)
		}
		host.MEMUSE, err = helper.ParseMemoryFromString(fields[8])
		if err != nil {
			return nil, fmt.Errorf("invalid MEMUSE: %s: %v", fields[8], err)
		}
		host.SWAPTO, err = helper.ParseMemoryFromString(fields[9])
		if err != nil {
			return nil, fmt.Errorf("invalid SWAPTO: %s: %v", fields[9], err)
		}
		host.SWAPUS, err = helper.ParseMemoryFromString(fields[10])
		if err != nil {
			return nil, fmt.Errorf("invalid SWAPUS: %s: %v", fields[10], err)
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

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
	hl:swap_total=1.500G
	hl:virtual_total=17.117G
	hl:load_avg=0.340000
	hl:load_short=0.550000
	hl:load_medium=0.340000
	hl:load_long=0.320000
	hl:mem_free=15.207G
	hl:swap_free=1.500G
	hl:virtual_free=16.707G
	hl:mem_used=419.988M
	hl:swap_used=0.000
	hl:virtual_used=419.988M
	hl:cpu=0.000000
	hl:m_topology=SCCCC
	hl:m_topology_inuse=SCCCC
	hl:m_socket=1.000000
	hl:m_core=4.000000
	hl:m_thread=4.000000
	hl:np_load_avg=0.085000
	hl:np_load_short=0.137500
	hl:np_load_medium=0.085000
	hl:np_load_long=0.080000
	hc:NVIDIA_GPUS=2.000000
*/

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

		if strings.HasPrefix(lines[i], "  ") {
			// Attribute line or resource availability line
			if currentHost == nil {
				return nil, fmt.Errorf("attribute line encountered before any host line")
			}
			attributeLine := strings.TrimSpace(line)
			err := parseAttributeLine(attributeLine, currentHost)
			if err != nil {
				return nil, fmt.Errorf("failed to parse attribute line: %v", err)
			}
		} else {
			// New host line
			// If currentHost is not nil, append it to hosts
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}
			// Create new currentHost
			currentHost = &HostFullMetrics{}

			// Parse host line
			fields := strings.Fields(line)
			if len(fields) < 10 {
				return nil, fmt.Errorf("invalid host line: %s", line)
			}
			currentHost.Name = fields[0]
			currentHost.Arch = fields[1]

			// Parse NCPU
			ncpu, err := strconv.Atoi(fields[2])
			if err == nil {
				currentHost.NumProc = float64(ncpu)
			}

			// The LOAD field is at index 6
			loadAvg, err := strconv.ParseFloat(fields[6], 64)
			if err == nil {
				currentHost.LoadAvg = loadAvg
			}

			// MEMTOT is at index 7
			memTotal, err := helper.ParseMemoryFromString(fields[7])
			if err == nil {
				currentHost.MemTotal = memTotal
			}

			// MEMUSE is at index 8
			memUsed, err := helper.ParseMemoryFromString(fields[8])
			if err == nil {
				currentHost.MemUsed = memUsed
			}

			// SWAPTO is at index 9
			swapTotal, err := helper.ParseMemoryFromString(fields[9])
			if err == nil {
				currentHost.SwapTotal = swapTotal
			}

			// SWAPUS is at index 10
			swapUsed, err := helper.ParseMemoryFromString(fields[10])
			if err == nil {
				currentHost.SwapUsed = swapUsed
			}

			// Initialize Resources map
			currentHost.Resources = make(map[string]ResourceAvailability)
		}
	}

	// Append the last host
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

	// Extract availability and source
	if len(prefix) != 2 {
		return fmt.Errorf("invalid prefix length: %s", prefix)
	}
	availabilityLetter := prefix[0]
	sourceLetter := prefix[1]

	// Now split rest into resource_name and value
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

	// Now process the resource_name and value
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

		// Try to parse the value as float
		floatValue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			ra.FloatValue = floatValue
		}

		currentHost.Resources[resourceName] = ra
	}
	return nil
}
