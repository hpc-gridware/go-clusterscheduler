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

package qstat

import (
	v90 "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

type JobInfo = v90.JobInfo
type ExtendedJobInfo = v90.ExtendedJobInfo
type QueueExplanation = v90.QueueExplanation
type FullQueueInfo = v90.FullQueueInfo
type FullQueueInfoWithResources = v90.FullQueueInfoWithResources
type ClusterQueueSummary = v90.ClusterQueueSummary
type JobArrayTask = v90.JobArrayTask
type ParallelJobTask = v90.ParallelJobTask
type QueueInfo = v90.QueueInfo
type JobStatus = v90.JobStatus
type TaskInfo = v90.TaskInfo
type JobUrgency = v90.JobUrgency
type JobPriority = v90.JobPriority
type UsageDetail = v90.UsageDetail

// ExecHostEntry is a single slot-assignment entry from the exec_host_list
// field of qstat -j output.
type ExecHostEntry struct {
	Hostname string `json:"JG_qhostname"`
	Slots    int    `json:"JG_slots"`
}

// GrantedRequest is a single parallel-task-group resource grant from
// the granted_request field of qstat -j output.
type GrantedRequest struct {
	PTGID      int    `json:"ptg_id"`
	GrantedReq string `json:"granted_req"`
}

// TaskUsageDetail holds per-task resource usage statistics parsed from
// the usage field of qstat -j output.
type TaskUsageDetail struct {
	WallClock string `json:"wallclock"`
	CPU       string `json:"cpu"`
	Mem       string `json:"mem"`
	IO        string `json:"io"`
	IOW       string `json:"iow"`
	IOOps     string `json:"ioops"`
	VMem      string `json:"vmem"`
	MaxVMem   string `json:"maxvmem"`
	RSS       string `json:"rss"`
	MaxRSS    string `json:"maxrss"`
	PSS       string `json:"pss"`
	SMem      string `json:"smem"`
	PMem      string `json:"pmem"`
	MaxPSS    string `json:"maxpss"`
}

// TaskDetail holds per-task runtime information from qstat -j output.
type TaskDetail struct {
	TaskID          int              `json:"task_id"`
	State           string           `json:"job_state"`
	ExecHostList    []ExecHostEntry  `json:"exec_host_list"`
	GrantedRequests []GrantedRequest `json:"granted_requests"`
	GrantedLicenses []interface{}    `json:"granted_licenses"`
	Usage           TaskUsageDetail  `json:"usage"`
	GPUUsage        []interface{}    `json:"gpu_usage"`
	CgroupsUsage    []interface{}    `json:"cgroups_usage"`
	BindingList     string           `json:"exec_binding_list,omitempty"`
	QueueList       string           `json:"exec_queue_list,omitempty"`
	StartTime       string           `json:"start_time,omitempty"`
	ResourceMap     string           `json:"resource_map,omitempty"`
}

// FullQueueExtendedInfo is a queue section from qstat -f -ext output.
// It is like FullQueueInfo but job entries carry the extended column set
// (ntckts, project, department, cpu, mem, io, tickets, share) instead of
// the plain job-ID / datetime columns.
type FullQueueExtendedInfo struct {
	QueueName string            `json:"queuename"`
	QueueType string            `json:"qtype"`
	Reserved  int               `json:"reserved"`
	Used      int               `json:"used"`
	Total     int               `json:"total"`
	LoadAvg   float64           `json:"load_avg"`
	Arch      string            `json:"arch"`
	States    string            `json:"states,omitempty"`
	Jobs      []ExtendedJobInfo `json:"jobs"`
}

// SchedulerJobInfo extends the v9.0 SchedulerJobInfo with v9.1 fields.
type SchedulerJobInfo struct {
	v90.SchedulerJobInfo

	CategoryID      int          `json:"category_id"`
	Groups          string       `json:"groups"`
	SgeOLogName     string       `json:"sge_o_log_name"`
	SgeOShell       string       `json:"sge_o_shell"`
	Priority        int          `json:"priority"`
	Department      string       `json:"department"`
	SyncOptions     string       `json:"sync_options"`
	JobArrayTasks   string       `json:"job_array_tasks"`
	TaskConcurrency string       `json:"task_concurrency,omitempty"`
	PendingTasks    int          `json:"pending_tasks,omitempty"`
	Tasks           []TaskDetail `json:"tasks"`
}
