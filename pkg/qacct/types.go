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

// Types representing the outputs

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
	// Other relevant fields
}

type DepartmentInfo struct {
	Department string `json:"department"`
	Usage
}

type GroupInfo struct {
	Group string `json:"group"`
	Usage
}

type HostInfo struct {
	HostName string `json:"host"`
	Usage
}

type OwnerInfo struct {
	OwnerName string `json:"owner"`
	Usage
}

type ProjectInfo struct {
	ProjectName string `json:"project"`
	Usage
}

type SlotsInfo struct {
	Slots int64 `json:"slots"`
	Usage
}

type QueueUsageDetail struct {
	HostName  string `json:"host"`
	QueueName string `json:"queue"`
	Usage
}

type TaskInfo struct {
	JobID     int64 `json:"job_id"`
	TaskID    int64 `json:"task_id"`
	JobDetail JobDetail
}

type JobDetail struct {
	QName             string  `json:"qname"`
	HostName          string  `json:"hostname"`
	Group             string  `json:"group"`
	Owner             string  `json:"owner"`
	Project           string  `json:"project"`
	Department        string  `json:"department"`
	JobName           string  `json:"jobname"`
	JobNumber         int64   `json:"jobnumber"`
	TaskID            int64   `json:"taskid"`
	PETaskID          string  `json:"pe_taskid"`
	Account           string  `json:"account"`
	Priority          int64   `json:"priority"`
	QSubTime          string  `json:"qsub_time"`
	SubmitCommandLine string  `json:"submit_command_line"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
	GrantedPE         string  `json:"granted_pe"`
	Slots             int64   `json:"slots"`
	Failed            int64   `json:"failed"`
	ExitStatus        int64   `json:"exit_status"`
	RuWallClock       float64 `json:"ru_wallclock"`
	RuUTime           float64 `json:"ru_utime"`
	RuSTime           float64 `json:"ru_stime"`
	RuMaxRSS          int64   `json:"ru_maxrss"`
	RuIXRSS           int64   `json:"ru_ixrss"`
	RuISMRSS          int64   `json:"ru_ismrss"`
	RuIDRSS           int64   `json:"ru_idrss"`
	RuISRss           int64   `json:"ru_isrss"`
	RuMinFlt          int64   `json:"ru_minflt"`
	RuMajFlt          int64   `json:"ru_majflt"`
	RuNSwap           int64   `json:"ru_nswap"`
	RuInBlock         int64   `json:"ru_inblock"`
	RuOuBlock         int64   `json:"ru_oublock"`
	RuMsgSend         int64   `json:"ru_msgsnd"`
	RuMsgRcv          int64   `json:"ru_msgrcv"`
	RuNSignals        int64   `json:"ru_nsignals"`
	RuNVCSw           int64   `json:"ru_nvcsw"`
	RuNiVCSw          int64   `json:"ru_nivcsw"`
	WallClock         float64 `json:"wallclock"`
	CPU               float64 `json:"cpu"`
	Memory            int64   `json:"mem"`
	IO                float64 `json:"io"`
	IOWait            float64 `json:"iow"`
	MaxVMem           int64   `json:"maxvmem"`
	MaxRSS            int64   `json:"maxrss"`
	ArID              string  `json:"arid"`
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
	Memory     float64 `json:"memory"`
	IO         float64 `json:"io"`
	IOWait     float64 `json:"iow"`
}
