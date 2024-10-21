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

type JobOptions struct {
	// Time options
	StartTime *string `flag:"-a"`
	Deadline  *string `flag:"-dl"`

	AdvanceReservationID *string `flag:"-ar"`

	// Resource options
	Account  *string `flag:"-A"`
	Project  *string `flag:"-P"`
	Priority *int    `flag:"-p"`

	ResourcesHardRequest map[string]string `flag:"-l_hard"`
	ResourcesSoftRequest map[string]string `flag:"-l_soft"`

	Queue []string `flag:"-q"`
	Slots *string  `flag:"-pe"`

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
	ExportAllEnv       bool              `flag:"-V"`
	ExportVariables    map[string]string `flag:"-v"`
	Hold               *bool             `flag:"-h"`
	Synchronize        *bool             `flag:"-sync"`
	ReservationDesired *bool             `flag:"-R"`
	Restartable        *bool             `flag:"-r"`
	Clear              *bool             `flag:"-clear"`
	Terse              *bool             `flag:"-terse"`
	PTTY               *bool             `flag:"-pty"`

	// Add other fields as necessary
	Command     string   // The command to execute
	CommandArgs []string // Arguments for the command
}
