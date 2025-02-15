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

import "context"

// QStat defines the methods for interacting with the Open Cluster Scheduler
// to retrieve job and queue status information using the qstat command.
//
// Most of the methods are wrappers around the qstat command line tool arguments
// for testing and replicating the qstat command line tool.
type QStat interface {
	// WatchJob returns a channel that will receive the current status of the
	// job with the given job ID. It waits up to 3 seconds for the job status
	// to appear in the system. When the job left the system (due to job end),
	// the channel will be closed.
	WatchJobs(ctx context.Context, jobIds []int64) (chan SchedulerJobInfo, error)
	// NativeSpecification calls qstat with the native specification of args
	// and returns the raw output
	NativeSpecification(args []string) (string, error)
	// qstat -ext
	ShowJobsWithAdditionalAttributes() ([]ExtendedJobInfo, error)
	// qstat -explain <reason> a|c|A|E
	ShowQueueExplanation(reason string) ([]QueueExplanation, error)
	// qstat -f
	ShowFullOutput() ([]FullQueueInfo, error)
	// qstat -F <resource_attributes>
	ShowFullOutputWithResources(resourceAttributes string) ([]JobInfo, error)
	// qstat -g c
	DisplayClusterQueueSummary() ([]ClusterQueueSummary, error)
	// qstat -g d shows all job array tasks individually
	DisplayAllJobArrayTasks() ([]JobArrayTask, error)
	// qstat -g p shows all parallel job tasks individually
	DisplayAllParallelJobTasks() ([]ParallelJobTask, error)
	// qstat -help
	ShowHelp() (string, error)
	ShowSchedulerJobInformation(jobIdentifierList []string) ([]SchedulerJobInfo, error)
	RequestResources(resourceList string) ([]JobInfo, error)
	HideEmptyQueues() ([]QueueInfo, error)
	SuppressAdditionalBindingParams() ([]JobInfo, error)
	SelectQueuesWithPE(peList []string) ([]QueueInfo, error)
	PrintQueueInformation(queueNameList []string) ([]QueueInfo, error)
	SelectQueuesInState(states string) ([]QueueInfo, error)
	// qstat -r
	ShowRequestedResourcesOfJobs() ([]JobInfo, error)
	// qstat -s <state>
	// [-s {p|r|s|z|hu|ho|hs|hd|hj|ha|h|a}] show pending, running, suspended, zombie jobs,
	// jobs with a user/operator/system/array-dependency hold,
	// jobs with a start time in future or any combination only
	ShowJobStatus(filter string) ([]JobStatus, error)
	// qstat -t
	ShowTaskInformation() ([]TaskInfo, error)
	// qstat -u <user_list>
	ViewJobsOfUser(userList []string) ([]JobInfo, error)
	// qstat -U <user_list>
	SelectQueuesWithUserAccess(userList []string) ([]QueueInfo, error)
	// qstat -urg
	DisplayJobUrgencyInformation() ([]JobUrgency, error)
	// qstat -pri
	DisplayJobPriorityInformation() ([]JobPriority, error)
	// qstat -xml
	DisplayInformationInXMLFormat() (string, error)
}
