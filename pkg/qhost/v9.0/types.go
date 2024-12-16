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

// Host is a struct that contains all values displayed by qhost output.
type Host struct {
	Name   string  `json:"name"`
	Arch   string  `json:"arch"`
	NCPU   int     `json:"ncpu"`
	NSOC   int     `json:"nsoc"`
	NCOR   int     `json:"ncor"`
	NTHR   int     `json:"nth"`
	LOAD   float64 `json:"load"`
	MEMTOT int64   `json:"mem_total"`
	MEMUSE int64   `json:"mem_used"`
	SWAPTO int64   `json:"swap_total"`
	SWAPUS int64   `json:"swap_used"`
}

// HostFullMetrics is a struct that contains all values displayed by
// qhost -F output.
type HostFullMetrics struct {
	Name          string  `json:"name"`
	Arch          string  `json:"arch"`
	NumProc       float64 `json:"num_proc"`
	MemTotal      int64   `json:"mem_total"`
	SwapTotal     int64   `json:"swap_total"`
	VirtualTotal  int64   `json:"virtual_total"`
	LoadAvg       float64 `json:"load_avg"`
	LoadShort     float64 `json:"load_short"`
	LoadMedium    float64 `json:"load_medium"`
	LoadLong      float64 `json:"load_long"`
	MemFree       int64   `json:"mem_free"`
	SwapFree      int64   `json:"swap_free"`
	VirtualFree   int64   `json:"virtual_free"`
	MemUsed       int64   `json:"mem_used"`
	SwapUsed      int64   `json:"swap_used"`
	VirtualUsed   int64   `json:"virtual_used"`
	CPU           float64 `json:"cpu"`
	Topology      string  `json:"topology"`
	TopologyInuse string  `json:"topology_inuse"`
	Socket        int64   `json:"socket"`
	Core          int64   `json:"core"`
	Thread        int64   `json:"thread"`
	NPLoadAvg     float64 `json:"np_load_avg"`
	NPLoadShort   float64 `json:"np_load_short"`
	NPLoadMedium  float64 `json:"np_load_medium"`
	NPLoadLong    float64 `json:"np_load_long"`
	// Cluster defined metrics
	Resources map[string]ResourceAvailability `json:"resources"`
}

// ResourceAvailability is a struct that contains the availability of a resource
// on a host.
type ResourceAvailability struct {
	Name        string  `json:"name"`
	StringValue string  `json:"value"`
	FloatValue  float64 `json:"float_value"`
	// ResourceAvailabilityLimitedBy indices whether the resource availability
	// is dominated by host "g" (global) or "l" (local).
	ResourceAvailabilityLimitedBy string `json:"resource_availability_limited_by"`
	// Source of the resource availability value:
	// - "l" load value for a resource
	// - "L" load value of a resource after an admin defined load scaling
	// - "c" availabililty derived from the consumable calculation
	// - "F" Non-consumable resource; Fixed value
	Source string `json:"source"`
	// The full output string "hl:np_load_medium=0.127500"
	FullString string `json:"full_string"`
}
