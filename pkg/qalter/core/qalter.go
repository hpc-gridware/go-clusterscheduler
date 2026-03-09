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

import "time"

// QAlter defines the common methods for interacting with the Open Cluster
// Scheduler qalter command. qalter enables users to modify attributes of
// pending jobs.
//
// Version-specific methods (binding, -when) are defined in the
// respective v9.0 and v9.1 packages.
type QAlter interface {

	// --- Time & Scheduling ---

	// SetStartTime requests a start time for the job (-a date_time).
	SetStartTime(jobTaskList string, dateTime time.Time) (string, error)

	// SetDeadline requests a deadline initiation time (-dl date_time).
	SetDeadline(jobTaskList string, dateTime time.Time) (string, error)

	// --- Job Identity & Metadata ---

	// SetJobName specifies the job name (-N name).
	SetJobName(jobTaskList, name string) (string, error)

	// SetAccountString sets the account string in the accounting
	// record (-A account_string).
	SetAccountString(jobTaskList, account string) (string, error)

	// SetProject sets the job's project (-P project_name).
	SetProject(jobTaskList, project string) (string, error)

	// SetDepartment sets the job's department (-dept department_name).
	SetDepartment(jobTaskList, department string) (string, error)

	// --- Context Variables ---

	// AddContext adds context variable(s) to the job (-ac context_list).
	// Each element should be in the form "variable" or "variable=value".
	AddContext(jobTaskList string, contextList []string) (string, error)

	// DeleteContext deletes context variable(s) from the job
	// (-dc simple_context_list).
	DeleteContext(jobTaskList string, contextList []string) (string, error)

	// SetContext sets job context, replacing any existing context
	// (-sc context_list).
	// Each element should be in the form "variable" or "variable=value".
	SetContext(jobTaskList string, contextList []string) (string, error)

	// --- Resource Requests ---

	// SetHardResourceList requests hard resources for the job
	// (-hard -l resource_list).
	// Each element should be in the form "resource=value".
	SetHardResourceList(jobTaskList string, resourceList []string) (string, error)

	// SetSoftResourceList requests soft resources for the job
	// (-soft -l resource_list).
	// Each element should be in the form "resource=value".
	SetSoftResourceList(jobTaskList string, resourceList []string) (string, error)

	// --- Queue Binding ---

	// SetHardQueue binds the job to queue(s) as a hard request
	// (-hard -q wc_queue_list).
	SetHardQueue(jobTaskList string, queueList []string) (string, error)

	// SetSoftQueue binds the job to queue(s) as a soft request
	// (-soft -q wc_queue_list).
	SetSoftQueue(jobTaskList string, queueList []string) (string, error)

	// SetHardMasterQueue binds the master task to queue(s) as a hard
	// request (-hard -masterq wc_queue_list).
	SetHardMasterQueue(jobTaskList string, queueList []string) (string, error)

	// SetSoftMasterQueue binds the master task to queue(s) as a soft
	// request (-soft -masterq wc_queue_list).
	SetSoftMasterQueue(jobTaskList string, queueList []string) (string, error)

	// --- Parallel Environment ---

	// SetParallelEnvironment requests a slot range for parallel jobs
	// (-pe pe-name slot_range).
	SetParallelEnvironment(jobTaskList, peName, slotRange string) (string, error)

	// --- I/O & Paths ---

	// SetErrorPath specifies the standard error stream path(s)
	// (-e path_list).
	// Each element should be in the form "path" or "host:path".
	SetErrorPath(jobTaskList string, pathList []string) (string, error)

	// SetOutputPath specifies the standard output stream path(s)
	// (-o path_list).
	// Each element should be in the form "path" or "host:path".
	SetOutputPath(jobTaskList string, pathList []string) (string, error)

	// SetInputFile specifies the standard input stream file(s)
	// (-i file_list).
	// Each element should be in the form "file" or "host:file".
	SetInputFile(jobTaskList string, fileList []string) (string, error)

	// SetShellPath sets the command interpreter to be used
	// (-S path_list).
	SetShellPath(jobTaskList string, pathList []string) (string, error)

	// SetMergeOutput merges stdout and stderr stream of the job
	// (-j y|n).
	SetMergeOutput(jobTaskList string, merge bool) (string, error)

	// --- Working Directory ---

	// SetCwd uses the current working directory for the job (-cwd).
	SetCwd(jobTaskList string) (string, error)

	// SetWorkingDirectory sets the working directory (-wd path).
	SetWorkingDirectory(jobTaskList, path string) (string, error)

	// --- Checkpointing ---

	// SetCheckpointSelector defines the type of checkpointing for the
	// job (-c ckpt_selector). Selector: 'n' 's' 'm' 'x' <interval>
	SetCheckpointSelector(jobTaskList, selector string) (string, error)

	// SetCheckpointMethod requests a checkpoint method
	// (-ckpt ckpt-name).
	SetCheckpointMethod(jobTaskList, name string) (string, error)

	// --- Holds & Dependencies ---

	// SetHold assigns holds for jobs or tasks (-h hold_list).
	// Hold list: 'n' 'u' 's' 'o' 'U' 'S' 'O'
	SetHold(jobTaskList, holdList string) (string, error)

	// SetHoldJobDependency defines jobnet interdependencies
	// (-hold_jid job_identifier_list).
	SetHoldJobDependency(jobTaskList string, jobIDList []string) (string, error)

	// SetHoldArrayDependency defines jobnet array interdependencies
	// (-hold_jid_ad job_identifier_list).
	SetHoldArrayDependency(jobTaskList string, jobIDList []string) (string, error)

	// --- Priority & Tickets ---

	// SetPriority defines the job's relative priority (-p priority).
	// Range: -1023 to 1024.
	SetPriority(jobTaskList string, priority int) (string, error)

	// SetJobShare sets the share tree or functional job share
	// (-js job_share).
	SetJobShare(jobTaskList string, share int) (string, error)

	// SetOverrideTickets sets the job's override tickets (-ot tickets).
	SetOverrideTickets(jobTaskList string, tickets int) (string, error)

	// --- Notification ---

	// SetMailOptions defines mail notification events
	// (-m mail_options). Options: 'e' 'b' 'a' 'n' 's'
	SetMailOptions(jobTaskList, options string) (string, error)

	// SetMailRecipients sets the e-mail addresses for notifications
	// (-M mail_list).
	SetMailRecipients(jobTaskList string, mailList []string) (string, error)

	// SetNotify enables notification before the job is killed or
	// suspended (-notify).
	SetNotify(jobTaskList string) (string, error)

	// --- Environment Variables ---

	// SetEnvironmentVariables exports the specified environment
	// variables (-v variable_list).
	// Each element should be in the form "variable" or "variable=value".
	SetEnvironmentVariables(jobTaskList string, variableList []string) (string, error)

	// ExportAllEnvironmentVariables exports all environment
	// variables (-V).
	ExportAllEnvironmentVariables(jobTaskList string) (string, error)

	// --- Reservation & Restart ---

	// SetReservation sets whether reservation is desired (-R y|n).
	SetReservation(jobTaskList string, reservation bool) (string, error)

	// SetRestartable defines whether the job is restartable (-r y|n).
	SetRestartable(jobTaskList string, restartable bool) (string, error)

	// --- Advance Reservation ---

	// SetAdvanceReservation binds the job to an advance reservation
	// (-ar ar_id).
	SetAdvanceReservation(jobTaskList, arID string) (string, error)

	// --- Task Control ---

	// SetMaxRunningTasks throttles the number of concurrent tasks
	// (-tc max_running_tasks).
	SetMaxRunningTasks(jobTaskList string, maxTasks int) (string, error)

	// --- Verification ---

	// SetVerifyMode sets the verify mode (-w e|w|n|v|p).
	// Modes: error, warning, none, just verify, poke.
	SetVerifyMode(jobTaskList, mode string) (string, error)

	// --- Raw ---

	// NativeSpecification runs qalter with the given raw arguments.
	NativeSpecification(args []string) (string, error)
}
