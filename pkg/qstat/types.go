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

package qstat

type JobInfo struct {
	JobID         int     `json:"job_id"`
	Priority      float64 `json:"prior"`
	Name          string  `json:"name"`
	User          string  `json:"user"`
	State         string  `json:"state"`
	SubmitStartAt string  `json:"submit_start_at"`
	Queue         string  `json:"queue"`
	Slots         int     `json:"slots"`
	TaskID        string  `json:"ja_task_id"`
}

type ExtendedJobInfo struct {
	JobID      int     `json:"job_id"`
	Priority   float64 `json:"prior"`
	Ntckts     float64 `json:"ntckts"`
	Name       string  `json:"name"`
	User       string  `json:"user"`
	Project    string  `json:"project"`
	Department string  `json:"department"`
	State      string  `json:"state"`
	CPU        string  `json:"cpu"`
	Memory     float64 `json:"mem"`
	IO         float64 `json:"io"`
	Tckts      int     `json:"tckts"`
	Ovrts      int     `json:"ovrts"`
	Otckt      int     `json:"otckt"`
	Ftckt      int     `json:"ftckt"`
	Stckt      int     `json:"stckt"`
	Share      float64 `json:"share"`
	Queue      string  `json:"queue"`
	Slots      int     `json:"slots"`
	JATaskID   string  `json:"ja_task_id"`
}

type QueueExplanation struct {
	QueueName   string    `json:"queuename"`
	QueueType   string    `json:"qtype"`
	ResvUsedTot string    `json:"resv_used_tot"`
	LoadAvg     string    `json:"load_avg"`
	Arch        string    `json:"arch"`
	States      string    `json:"states"`
	Jobs        []JobInfo `json:"jobs"`
}

type FullQueueInfo struct {
	QueueName   string    `json:"queuename"`
	QueueType   string    `json:"qtype"`
	ResvUsedTot string    `json:"resv_used_tot"`
	LoadAvg     string    `json:"load_avg"`
	Arch        string    `json:"arch"`
	States      string    `json:"states"`
	Jobs        []JobInfo `json:"jobs"`
}

type FullQueueInfoWithResources struct {
	FullQueueInfo
	Resources map[string]interface{} `json:"resources"` // Key-value pairs representing resources
}

type ClusterQueueSummary struct {
	ClusterQueue string  `json:"cluster_queue"`
	CQLoad       float64 `json:"cqload"`
	Used         int     `json:"used"`
	Reserved     int     `json:"res"`
	Available    int     `json:"avail"`
	Total        int     `json:"total"`
	AoACDS       int     `json:"aoacds"`
	CdsuE        int     `json:"cdsue"`
}

type JobArrayTask struct {
	JobInfo
	Master string `json:"master"`
}

type ParallelJobTask struct {
	JobInfo
	Master string `json:"master"`
}

type QueueInfo struct {
	QueueName string    `json:"queuename"`
	Jobs      []JobInfo `json:"jobs"`
}

type JobStatus struct {
	JobInfo
	Filter string `json:"filter"` // Filter criteria
}

type TaskInfo struct {
	JobInfo
	TaskDetail string `json:"task_detail"`
}

type JobUrgency struct {
	JobID    int     `json:"job_id"`
	Priority float64 `json:"prior"`
	Nurg     float64 `json:"nurg"`
	Urg      int     `json:"urg"`
	RrContr  int     `json:"rrcontr"`
	WtContr  int     `json:"wtcontr"`
	DlContr  int     `json:"dlcontr"`
	Name     string  `json:"name"`
	User     string  `json:"user"`
	State    string  `json:"state"`
	StartAt  string  `json:"submit_start_at"`
	Deadline string  `json:"deadline"`
	Queue    string  `json:"queue"`
	Slots    int     `json:"slots"`
}

type JobPriority struct {
	JobID    int     `json:"job_id"`
	Priority float64 `json:"prior"`
	Nurg     float64 `json:"nurg"`
	NpPrior  float64 `json:"npprior"`
	Ntckts   float64 `json:"ntckts"`
	Ppri     int     `json:"ppri"`
	Name     string  `json:"name"`
	User     string  `json:"user"`
	State    string  `json:"state"`
	StartAt  string  `json:"submit_start_at"`
	Queue    string  `json:"queue"`
	Slots    int     `json:"slots"`
}

// SchedulerJobInfo represents detailed information about a scheduled job
// retrieved with the qstat -j <job_id> command.
type SchedulerJobInfo struct {
	JobNumber      int           `json:"job_number"`
	ExecFile       string        `json:"exec_file"`
	SubmissionTime string        `json:"submission_time"`
	Owner          string        `json:"owner"`
	UID            int           `json:"uid"`
	Group          string        `json:"group"`
	GID            int           `json:"gid"`
	SgeOHome       string        `json:"sge_o_home"`
	SgeOPath       string        `json:"sge_o_path"`
	SgeOWorkDir    string        `json:"sge_o_workdir"`
	SgeOHost       string        `json:"sge_o_host"`
	Account        string        `json:"account"`
	MailList       string        `json:"mail_list"`
	Notify         bool          `json:"notify"`
	JobName        string        `json:"job_name"`
	JobShare       int           `json:"jobshare"`
	EnvList        string        `json:"env_list"`
	JobArgs        string        `json:"job_args"`
	ScriptFile     string        `json:"script_file"`
	Binding        string        `json:"binding"`
	Usage          []UsageDetail `json:"usage"`
	SchedulingInfo string        `json:"scheduling_info"`
}

type UsageDetail struct {
	WallClock string `json:"wallclock"`
	CPU       string `json:"cpu"`
	Mem       string `json:"mem"`
	IO        string `json:"io"`
	VMem      string `json:"vmem"`
	MaxVMem   string `json:"maxvmem"`
	RSS       string `json:"rss"`
	MaxRSS    string `json:"maxrss"`
	Binding   string `json:"binding"`
	Map       string `json:"map"`
}
