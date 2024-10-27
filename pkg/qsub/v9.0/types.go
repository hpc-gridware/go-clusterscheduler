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

package qsub

import "time"

// ToPtr is a generic function that converts a value to a pointer to
// simplify building JobOptions.
func ToPtr[T any](v T) *T {
	return &v
}

// ResourceRequest is a struct that represents a resource request, which
// is on the command line of qsub the -l flag.
type ResourceRequest struct {
	Resources map[string]string
}

// SimpleLRequest creates a simple resource request for the global scope with
// hard requests. This is the standard request for most jobs and corresponds
// to the -l flag of qsub without any scope.
func SimpleLRequest(resources map[string]string) map[string]map[string]ResourceRequest {
	return map[string]map[string]ResourceRequest{
		"global": {
			"hard": {Resources: resources},
		},
	}
}

type JobOptions struct {
	// Time options
	StartTime *time.Time `flag:"-a"`
	Deadline  *time.Time `flag:"-dl"`

	AdvanceReservationID *string `flag:"-ar"`

	// Resource options
	Account  *string `flag:"-A"`
	Project  *string `flag:"-P"`
	Priority *int    `flag:"-p"`

	// Map to handle resources with flexible scopes and request types.
	// On the command line of qsub the -l flag is used. The -l flag can
	// can have multiple scopes, each with multiple resource requests.
	// If there is no scope given and not -hard or -soft, the resource
	// request is implicitly globa and hard.
	//
	// Example:
	//
	// qsub -scope global -hard -l mem_free=4G -soft cpu=2 -scope master gpu=1
	// ScopedResources: {
	// 	global: {
	// 		hard: {Resources: ResourceRequest{"mem": "4G"}},
	// 		soft: {Resources: ResourceRequest{"cpu": "2"}},
	// 	},
	// 	master: {
	// 		hard: {Resources: ResourceRequest{"gpu": "1"}},
	// 	},
	// }
	//
	// qsub -l mem_free=4G,m_core=1
	// ScopedResources: {
	// 	global: {
	// 		hard: {Resources: ResourceRequest{"mem": "4G", "core": "1"}},
	// 	},
	// }
	ScopedResources map[string]map[string]ResourceRequest

	Queue []string `flag:"-q"`
	// pe-name slot_range: smp 1-10
	ParallelEnvironment *string `flag:"-pe"`

	// Output/Input options
	StdErr []string `flag:"-e"`
	StdOut []string `flag:"-o"`
	StdIn  []string `flag:"-i"`

	// Execution options
	Binary             *bool   `flag:"-b"`
	WorkingDir         *string `flag:"-cwd,-wd"`
	CommandPrefix      *string `flag:"-C"`
	Shell              *bool   `flag:"-shell"`
	CommandInterpreter *string `flag:"-S"`
	JobName            *string `flag:"-N"`
	JobArray           *string `flag:"-t"`
	MaxRunningTasks    *int    `flag:"-tc"`

	// Notification options
	MailOptions   *string  `flag:"-m"`
	MailList      []string `flag:"-M"`
	Notify        *bool    `flag:"-notify"`
	MailAddresses []string `flag:"-M"`
	EmailOnStart  *bool    // Custom field for email on start
	EmailOnEnd    *bool    // Custom field for email on end

	// Dependency options
	HoldJobIDs      []string `flag:"-hold_jid"`
	HoldArrayJobIDs []string `flag:"-hold_jid_ad"`

	// Other options
	Checkpoint         *string           `flag:"-ckpt"`
	CheckpointSelector *string           `flag:"-c"`
	MergeStdOutErr     *bool             `flag:"-j"`
	UseCurrentDir      *bool             `flag:"-cwd"`
	Verify             *bool             `flag:"-verify"`
	ExportAllEnv       *bool             `flag:"-V"`
	EnvVariables       map[string]string `flag:"-v"`
	Hold               *bool             `flag:"-h"`
	Synchronize        *bool             `flag:"-sync"`
	ReservationDesired *bool             `flag:"-R"`
	Restartable        *bool             `flag:"-r"`
	Clear              *bool             `flag:"-clear"`
	Terse              *bool             `flag:"-terse"`
	PTTY               *bool             `flag:"-pty"`

	// Context Options
	AddContextVariables    []string          `flag:"-ac"`
	DeleteContextVariables []string          `flag:"-dc"`
	SetJobContext          map[string]string `flag:"-sc"`

	// Processor Binding
	ProcessorBinding *string `flag:"-binding"`

	// Job Hold and Priority Options
	JobShare                        *int    `flag:"-js"`
	JobSubmissionVerificationScript *string `flag:"-jsv"`

	// Scope and Environment Verification
	ScopeName  *string `flag:"-scope"`
	VerifyMode *string `flag:"-w"`

	// Immediate and Reservation Options
	StartImmediately *bool `flag:"-now"`

	// Command Options
	CommandFile *string `flag:"-@"`

	// Queue Master
	MasterQueue []string `flag:"-masterq"`

	// Checkpointing Details
	CheckpointInterval *string `flag:"ckpt_selector"` // Options like 'n', 's', 'm', 'x'

	// Synchronization and Job Start
	NotifyBeforeSuspend *bool `flag:"-notify"` // Custom field for detailed handling of notifications

	// Add other fields as necessary
	Command     string   // The command to execute
	CommandArgs []string // Arguments for the command
}
