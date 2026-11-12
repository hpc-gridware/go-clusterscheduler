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
	"encoding/json"
	"fmt"
)

type ReservationUsage struct {
	ArID string `json:"ar"`
	Usage
}

type JobInfo struct {
	JobID      int64   `json:"job_number"`
	JobName    string  `json:"job_name"`
	Priority   float64 `json:"priority"`
	State      string  `json:"state"`
	User       string  `json:"user"`
	Queue      string  `json:"queue"`
	SubmitTime string  `json:"submission_time"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
	Slots      int64   `json:"slots"`
	CPU        float64 `json:"cpu"`
	Memory     int64   `json:"memory"`
	WallClock  int64   `json:"wallclock"`
}

type DepartmentUsage struct {
	Department string `json:"department"`
	Usage
}

type GroupUsage struct {
	Group string `json:"group"`
	Usage
}

type HostUsage struct {
	HostName string `json:"host"`
	Usage
}

type OwnerUsage struct {
	OwnerName string `json:"owner"`
	Usage
}

type ProjectUsage struct {
	ProjectName string `json:"project"`
	Usage
}

type SlotsUsage struct {
	Slots int64 `json:"slots"`
	Usage
}

type QueueUsage struct {
	HostName  string `json:"host"`
	QueueName string `json:"queue"`
	Usage
}

type TaskUsage struct {
	JobID     int64 `json:"job_id"`
	TaskID    int64 `json:"task_id"`
	JobDetail JobDetail
}

type JobDetail struct {
	QName             string   `json:"qname"`
	HostName          string   `json:"hostname"`
	Group             string   `json:"group"`
	Owner             string   `json:"owner"`
	Project           string   `json:"project"`
	Department        string   `json:"department"`
	JobName           string   `json:"job_name"`
	JobNumber         int64    `json:"job_number"`
	TaskID            int64    `json:"task_number"`
	PETaskID          string   `json:"pe_taskid"`
	Account           string   `json:"account"`
	Priority          int64    `json:"priority"`
	SubmitTime        int64    `json:"submission_time"`
	SubmitCommandLine string   `json:"submit_cmd_line"`
	StartTime         int64    `json:"start_time"`
	EndTime           int64    `json:"end_time"`
	GrantedPE         string   `json:"granted_pe"`
	Slots             int64    `json:"slots"`
	Failed            int64    `json:"failed"`
	ExitStatus        int64    `json:"exit_status"`
	ArID              string   `json:"arid"`
	JobUsage          JobUsage `json:"usage"`
}

// JobUsage is the usage part of an accounting record.
type JobUsage struct {
	Usage  Usage  `json:"usage"`
	RUsage RUsage `json:"rusage"`
	// Custom holds the usage values which have no field in Usage or RUsage:
	// the values selected by usage_patterns in the reporting_params of the
	// global configuration (GPU or agent metrics, for example).
	//
	// The accounting file stores them in one group per usage_patterns name
	// ("usage":{"gpu":{"nvidia_energy_consumed":12.5}}), which is the key of
	// the outer map. The qacct -j output does not show the group, so values
	// parsed from it are in CustomUsageNoGroup.
	Custom map[string]map[string]float64 `json:"custom,omitempty"`
}

// CustomUsageNoGroup is the key of JobUsage.Custom for values whose
// usage_patterns group is not known (qacct -j output).
const CustomUsageNoGroup = ""

// CustomValue returns the custom usage value with the given name from
// whatever group it is in.
func (u JobUsage) CustomValue(name string) (float64, bool) {
	for _, group := range u.Custom {
		if v, ok := group[name]; ok {
			return v, true
		}
	}
	return 0, false
}

// UnmarshalJSON reads the usage object of an accounting record. The extended
// usage is named "usage" by 9.0 and "eusage" by 9.1; every other group besides
// "rusage" is custom usage. It also reads what JobUsage marshals to.
func (u *JobUsage) UnmarshalJSON(data []byte) error {
	var groups map[string]json.RawMessage
	if err := json.Unmarshal(data, &groups); err != nil {
		return err
	}
	*u = JobUsage{}
	for name, raw := range groups {
		switch name {
		case "rusage":
			if err := json.Unmarshal(raw, &u.RUsage); err != nil {
				return fmt.Errorf("rusage: %w", err)
			}
		case "usage", "eusage":
			// qmaster writes all extended usage values as floating point
			// numbers ("ioops":334.0), which do not fit an integer field.
			var values map[string]float64
			if err := json.Unmarshal(raw, &values); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			u.Usage = usageFromValues(values)
		default:
			var values map[string]float64
			if err := json.Unmarshal(raw, &values); err == nil {
				u.addCustom(name, values)
				continue
			}
			// The "custom" object JobUsage itself marshals to.
			var nested map[string]map[string]float64
			if name == "custom" && json.Unmarshal(raw, &nested) == nil {
				for group, values := range nested {
					u.addCustom(group, values)
				}
			}
			// Anything else is not usage this version knows about.
		}
	}
	return nil
}

func (u *JobUsage) addCustom(group string, values map[string]float64) {
	if len(values) == 0 {
		return
	}
	if u.Custom == nil {
		u.Custom = map[string]map[string]float64{}
	}
	if u.Custom[group] == nil {
		u.Custom[group] = map[string]float64{}
	}
	for name, value := range values {
		u.Custom[group][name] = value
	}
}

func usageFromValues(values map[string]float64) Usage {
	return Usage{
		WallClock:  values["wallclock"],
		UserTime:   values["utime"],
		SystemTime: values["stime"],
		CPU:        values["cpu"],
		Memory:     values["mem"],
		IO:         values["io"],
		IOOps:      int64(values["ioops"]),
		IOWait:     values["iow"],
		MaxVMem:    values["maxvmem"],
		MaxRSS:     values["maxrss"],
	}
}

// RUsage represents the resource usage data structure.
type RUsage struct {
	RuWallclock int64   `json:"ru_wallclock"`
	RuUtime     float64 `json:"ru_utime"`
	RuStime     float64 `json:"ru_stime"`
	RuMaxrss    int64   `json:"ru_maxrss"`
	RuIxrss     int64   `json:"ru_ixrss"`
	RuIsmrss    int64   `json:"ru_ismrss"`
	RuIdrss     int64   `json:"ru_idrss"`
	RuIsrss     int64   `json:"ru_isrss"`
	RuMinflt    int64   `json:"ru_minflt"`
	RuMajflt    int64   `json:"ru_majflt"`
	RuNswap     int64   `json:"ru_nswap"`
	RuInblock   int64   `json:"ru_inblock"`
	RuOublock   int64   `json:"ru_oublock"`
	RuMsgsnd    int64   `json:"ru_msgsnd"`
	RuMsgrcv    int64   `json:"ru_msgrcv"`
	RuNsignals  int64   `json:"ru_nsignals"`
	RuNvcsw     int64   `json:"ru_nvcsw"`
	RuNivcsw    int64   `json:"ru_nivcsw"`
}

type PeUsage struct {
	Pename string `json:"pe"`
	Usage
}

type Usage struct {
	WallClock  float64 `json:"wallclock"`
	UserTime   float64 `json:"utime"`
	SystemTime float64 `json:"stime"`
	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"mem"`
	IO         float64 `json:"io"`
	IOOps      int64   `json:"ioops"`
	IOWait     float64 `json:"iow"`
	MaxVMem    float64 `json:"maxvmem"`
	MaxRSS     float64 `json:"maxrss"`
}
