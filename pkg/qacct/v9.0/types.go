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

// sampleOutput := `{"job_number":10,"task_number":1,"start_time":1730532913429415,"end_time":1730532913979016,"owner":"root","group":"root","account":"sge","qname":"all.q","hostname":"master","department":"defaultdepartment","slots":1,"job_name":"echo","priority":0,"submission_time":1730532912874519,"submit_cmd_line":"qsub -b y -terse echo 'job 1'","category":"","failed":0,"exit_status":0,"usage":{"rusage":{"ru_wallclock":0,"ru_utime":0.355821,"ru_stime":0.161309,"ru_maxrss":10284,"ru_ixrss":0,"ru_ismrss":0,"ru_idrss":0,"ru_isrss":0,"ru_minflt":504,"ru_majflt":0,"ru_nswap":0,"ru_inblock":0,"ru_oublock":11,"ru_msgsnd":0,"ru_msgrcv":0,"ru_nsignals":0,"ru_nvcsw":248,"ru_nivcsw":14},"usage":{"wallclock":2.022342,"cpu":0.51713,"mem":0.0043125152587890625,"io":0.000008341856300830841,"iow":0.0,"maxvmem":21049344.0,"maxrss":10530816.0}}}`

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

type JobUsage struct {
	Usage  Usage  `json:"usage"`
	RUsage RUsage `json:"rusage"`
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
	IOWait     float64 `json:"iow"`
	MaxVMem    float64 `json:"maxvmem"`
	MaxRSS     float64 `json:"maxrss"`
}
