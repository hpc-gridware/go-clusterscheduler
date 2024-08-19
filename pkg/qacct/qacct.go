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

// QAcct defines the methods for interacting with the Open Cluster Scheduler
// to retrieve accounting information for finished jobs using the qacct command.
//
// The interface replicates the qacct command line options and arguments 1:1 so
// that it can be used for automating and testing.
type QAcct interface {
	WithAlternativeAccountingFile(accountingFile string) error
	WithDefaultAccountingFile()
	ListAdvanceReservations(arID string) ([]ReservationUsage, error)
	JobsAccountedTo(accountString string) (Usage, error)
	JobsStartedAfter(beginTime string) (Usage, error)
	JobsStartedBefore(endTime string) (Usage, error)
	JobsStartedLastDays(days int) (Usage, error)
	ListDepartment(department string) ([]DepartmentInfo, error)
	ListGroup(groupIDOrName string) ([]GroupInfo, error)
	ListHost(host string) ([]HostInfo, error)
	ListJobs(jobIDOrNameOrPattern string) ([]JobDetail, error)
	RequestComplexAttributes(attributes string) ([]JobInfo, error)
	ListOwner(owner string) ([]OwnerInfo, error)
	ListParallelEnvironment(peName string) ([]PeUsage, error)
	ListProject(project string) ([]ProjectInfo, error)
	ListQueue(queue string) ([]QueueUsageDetail, error)
	ListJobUsageBySlots(usedSlots int) ([]SlotsInfo, error)
	ListTasks(jobID, taskIDRange string) ([]TaskInfo, error)
	ShowHelp() (string, error)
	ShowTotalSystemUsage() (Usage, error)
	ShowJobDetails(jobID int) (JobDetail, error)
}
