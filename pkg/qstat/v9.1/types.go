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

// TaskDetail holds per-task runtime information from qstat -j output.
type TaskDetail struct {
	TaskID        int    `json:"task_id"`
	State         string `json:"job_state"`
	Usage         string `json:"usage"`
	BindingList   string `json:"exec_binding_list"`
	QueueList     string `json:"exec_queue_list"`
	HostList      string `json:"exec_host_list"`
	StartTime     string `json:"start_time"`
	ResourceMap   string `json:"resource_map"`
}

// SchedulerJobInfo extends the v9.0 SchedulerJobInfo with v9.1 fields.
type SchedulerJobInfo struct {
	v90.SchedulerJobInfo

	CategoryID    int          `json:"category_id"`
	Groups        string       `json:"groups"`
	SgeOLogName   string       `json:"sge_o_log_name"`
	SgeOShell     string       `json:"sge_o_shell"`
	Priority      int          `json:"priority"`
	Department    string       `json:"department"`
	SyncOptions   string       `json:"sync_options"`
	JobArrayTasks string       `json:"job_array_tasks"`
	Tasks         []TaskDetail `json:"tasks"`
}
