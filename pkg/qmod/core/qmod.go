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

// QMod defines the methods for interacting with the Open Cluster Scheduler
// qmod command. qmod enables users to modify the state of queues and jobs.
type QMod interface {
	// ClearErrorState clears the error state of the specified jobs
	// and/or queues.
	// Deprecated: Use ClearJobErrorState or ClearQueueErrorState instead.
	ClearErrorState(jobOrQueueList []string) (string, error)

	// ClearJobErrorState clears the error state of the specified jobs.
	ClearJobErrorState(jobList []string) (string, error)

	// ClearQueueErrorState clears the error state of the specified queues.
	ClearQueueErrorState(queueList []string) (string, error)

	// Disable disables the specified queues. No further jobs are dispatched
	// to disabled queues while jobs already executing are allowed to finish.
	Disable(queueList []string) (string, error)

	// Enable enables the specified queues.
	Enable(queueList []string) (string, error)

	// RescheduleJobs reschedules the specified running jobs.
	RescheduleJobs(jobList []string) (string, error)

	// RescheduleQueues reschedules all jobs currently running in the
	// specified queues.
	RescheduleQueues(queueList []string) (string, error)

	// Suspend suspends the specified jobs and/or queues.
	// Deprecated: Use SuspendJobs or SuspendQueues instead.
	Suspend(jobOrQueueList []string) (string, error)

	// SuspendJobs suspends the specified running jobs.
	SuspendJobs(jobList []string) (string, error)

	// SuspendQueues suspends the specified queues and any active jobs.
	SuspendQueues(queueList []string) (string, error)

	// Unsuspend unsuspends the specified jobs and/or queues.
	// Deprecated: Use UnsuspendJobs or UnsuspendQueues instead.
	Unsuspend(jobOrQueueList []string) (string, error)

	// UnsuspendJobs unsuspends the specified jobs.
	UnsuspendJobs(jobList []string) (string, error)

	// UnsuspendQueues unsuspends the specified queues and any active jobs.
	UnsuspendQueues(queueList []string) (string, error)

	// NativeSpecification runs qmod with the given raw arguments.
	NativeSpecification(args []string) (string, error)
}
