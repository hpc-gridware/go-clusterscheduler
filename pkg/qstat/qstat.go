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

// QStat defines the methods for interacting with the Open Cluster Scheduler
// to retrieve job and queue status information using the qstat command.
type QStat interface {
	// NativeSpecification calls qstat with the native specification of args
	// and returns the raw output
	NativeSpecification(args []string) (string, error)
	// qstat -ext
	ShowAdditionalAttributes() ([]ExtendedJobInfo, error)
	// qstat -explain <reason> a|c|A|E
	ShowQueueExplanation(reason string) ([]QueueExplanation, error)
	// qstat -f
	ShowFullOutput() ([]JobInfo, error)
	// qstat -F <resource_attributes>
	ShowFullOutputWithResources(resourceAttributes string) ([]JobInfo, error)
	DisplayClusterQueueSummary() ([]ClusterQueueSummary, error)
	DisplayAllJobArrayTasks() ([]JobArrayTask, error)
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
