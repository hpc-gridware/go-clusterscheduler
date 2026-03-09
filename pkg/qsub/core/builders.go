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

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Submitter is the minimal interface required by JobBuilder. Both v9.0
// and v9.1 Qsub implementations satisfy this interface.
type Submitter interface {
	SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error)
}

// JobBuilder provides a fluent API for constructing and submitting qsub
// jobs. It builds raw CLI arguments and submits via
// SubmitWithNativeSpecification, making it version-agnostic. Use Flag()
// for any options not covered by the named methods.
type JobBuilder struct {
	submitter Submitter
	args      []string
	command   string
	cmdArgs   []string
	sync      bool
}

// NewJobBuilder creates a new JobBuilder for the given command and
// optional arguments. The submitter can be any Qsub client (v9.0 or
// v9.1).
func NewJobBuilder(submitter Submitter, command string, args ...string) *JobBuilder {
	return &JobBuilder{
		submitter: submitter,
		command:   command,
		cmdArgs:   args,
	}
}

// --- Execution ---

// Binary marks the command as a binary executable (-b y).
func (b *JobBuilder) Binary() *JobBuilder {
	b.args = append(b.args, "-b", "y")
	return b
}

// Script marks the command as a script (-b n).
func (b *JobBuilder) Script() *JobBuilder {
	b.args = append(b.args, "-b", "n")
	return b
}

// Name sets the job name (-N).
func (b *JobBuilder) Name(name string) *JobBuilder {
	b.args = append(b.args, "-N", name)
	return b
}

// Array defines a job array with the given task range, e.g. "1-10" or
// "1-100:2" (-t).
func (b *JobBuilder) Array(taskRange string) *JobBuilder {
	b.args = append(b.args, "-t", taskRange)
	return b
}

// MaxConcurrentTasks limits the number of concurrently running array
// tasks (-tc).
func (b *JobBuilder) MaxConcurrentTasks(n int) *JobBuilder {
	b.args = append(b.args, "-tc", fmt.Sprintf("%d", n))
	return b
}

// Shell controls whether the command is wrapped in a shell (-shell).
func (b *JobBuilder) Shell(yes bool) *JobBuilder {
	if yes {
		b.args = append(b.args, "-shell", "y")
	} else {
		b.args = append(b.args, "-shell", "n")
	}
	return b
}

// Interpreter sets the command interpreter path (-S).
func (b *JobBuilder) Interpreter(path string) *JobBuilder {
	b.args = append(b.args, "-S", path)
	return b
}

// CommandPrefix sets the command prefix for job script directives (-C).
func (b *JobBuilder) CommandPrefix(prefix string) *JobBuilder {
	b.args = append(b.args, "-C", prefix)
	return b
}

// CurrentDir requests execution in the current working directory (-cwd).
func (b *JobBuilder) CurrentDir() *JobBuilder {
	b.args = append(b.args, "-cwd")
	return b
}

// WorkDir sets the working directory for the job (-wd).
func (b *JobBuilder) WorkDir(path string) *JobBuilder {
	b.args = append(b.args, "-wd", path)
	return b
}

// --- Resources ---

// Account sets the account string (-A).
func (b *JobBuilder) Account(account string) *JobBuilder {
	b.args = append(b.args, "-A", account)
	return b
}

// Project sets the project name (-P).
func (b *JobBuilder) Project(project string) *JobBuilder {
	b.args = append(b.args, "-P", project)
	return b
}

// Priority sets the job priority (-p). Valid range: -1023 to 1024.
func (b *JobBuilder) Priority(p int) *JobBuilder {
	b.args = append(b.args, "-p", fmt.Sprintf("%d", p))
	return b
}

// Queue adds one or more destination queues (-q).
func (b *JobBuilder) Queue(queues ...string) *JobBuilder {
	b.args = append(b.args, "-q", strings.Join(queues, ","))
	return b
}

// MasterQueue sets the master queue for parallel jobs (-masterq).
func (b *JobBuilder) MasterQueue(queues ...string) *JobBuilder {
	b.args = append(b.args, "-masterq", strings.Join(queues, ","))
	return b
}

// PE requests a parallel environment with the given specification,
// e.g. "smp 1-8" (-pe).
func (b *JobBuilder) PE(spec string) *JobBuilder {
	b.args = append(b.args, "-pe", spec)
	return b
}

// Resource adds a simple global hard resource request (-l key=value).
func (b *JobBuilder) Resource(key, value string) *JobBuilder {
	b.args = append(b.args, "-l", fmt.Sprintf("%s=%s", key, value))
	return b
}

// --- I/O ---

// StdOut sets the standard output path(s) (-o).
func (b *JobBuilder) StdOut(paths ...string) *JobBuilder {
	b.args = append(b.args, "-o", strings.Join(paths, ","))
	return b
}

// StdErr sets the standard error path(s) (-e).
func (b *JobBuilder) StdErr(paths ...string) *JobBuilder {
	b.args = append(b.args, "-e", strings.Join(paths, ","))
	return b
}

// StdIn sets the standard input file(s) (-i).
func (b *JobBuilder) StdIn(files ...string) *JobBuilder {
	b.args = append(b.args, "-i", strings.Join(files, ","))
	return b
}

// MergeOutput merges stderr into stdout (-j y).
func (b *JobBuilder) MergeOutput() *JobBuilder {
	b.args = append(b.args, "-j", "y")
	return b
}

// --- Notification ---

// MailOptions sets when mail is sent (-m). Values: b, e, a, s, n.
func (b *JobBuilder) MailOptions(options string) *JobBuilder {
	b.args = append(b.args, "-m", options)
	return b
}

// MailTo sets the mail recipient(s) (-M).
func (b *JobBuilder) MailTo(addresses ...string) *JobBuilder {
	b.args = append(b.args, "-M", strings.Join(addresses, ","))
	return b
}

// Notify enables the SIGUSR1/SIGUSR2 notification mechanism (-notify).
func (b *JobBuilder) Notify() *JobBuilder {
	b.args = append(b.args, "-notify")
	return b
}

// --- Dependencies ---

// HoldJobs sets job dependencies by job ID or name (-hold_jid).
func (b *JobBuilder) HoldJobs(ids ...string) *JobBuilder {
	b.args = append(b.args, "-hold_jid", strings.Join(ids, ","))
	return b
}

// HoldArrayJobs sets array job dependencies (-hold_jid_ad).
func (b *JobBuilder) HoldArrayJobs(ids ...string) *JobBuilder {
	b.args = append(b.args, "-hold_jid_ad", strings.Join(ids, ","))
	return b
}

// Hold puts the job on hold at submission (-h).
func (b *JobBuilder) Hold() *JobBuilder {
	b.args = append(b.args, "-h", "y")
	return b
}

// --- Environment ---

// ExportAllEnv exports the current environment to the job (-V).
func (b *JobBuilder) ExportAllEnv() *JobBuilder {
	b.args = append(b.args, "-V")
	return b
}

// Env adds an environment variable to the job (-v key=value). If value
// is empty, the variable is exported from the submitting environment.
func (b *JobBuilder) Env(key, value string) *JobBuilder {
	if value != "" {
		b.args = append(b.args, "-v", fmt.Sprintf("%s=%s", key, value))
	} else {
		b.args = append(b.args, "-v", key)
	}
	return b
}

// --- Checkpointing ---

// Checkpoint selects a checkpointing environment (-ckpt).
func (b *JobBuilder) Checkpoint(name string) *JobBuilder {
	b.args = append(b.args, "-ckpt", name)
	return b
}

// CheckpointSelector sets when checkpointing occurs (-c).
func (b *JobBuilder) CheckpointSelector(selector string) *JobBuilder {
	b.args = append(b.args, "-c", selector)
	return b
}

// --- Timing ---

// StartTime sets the earliest start time for the job (-a).
func (b *JobBuilder) StartTime(t time.Time) *JobBuilder {
	b.args = append(b.args, "-a", ConvertTimeToQsubDateTime(t))
	return b
}

// Deadline sets the job deadline (-dl).
func (b *JobBuilder) Deadline(t time.Time) *JobBuilder {
	b.args = append(b.args, "-dl", ConvertTimeToQsubDateTime(t))
	return b
}

// --- Scheduling ---

// Sync waits for the job to complete before returning (-sync y).
func (b *JobBuilder) Sync() *JobBuilder {
	b.args = append(b.args, "-sync", "y")
	b.sync = true
	return b
}

// Reservation requests a resource reservation (-R y).
func (b *JobBuilder) Reservation() *JobBuilder {
	b.args = append(b.args, "-R", "y")
	return b
}

// Restartable controls whether the job can be restarted (-r).
func (b *JobBuilder) Restartable(yes bool) *JobBuilder {
	if yes {
		b.args = append(b.args, "-r", "y")
	} else {
		b.args = append(b.args, "-r", "n")
	}
	return b
}

// Now requests immediate scheduling (-now y).
func (b *JobBuilder) Now() *JobBuilder {
	b.args = append(b.args, "-now", "y")
	return b
}

// AdvanceReservation assigns the job to an advance reservation (-ar).
func (b *JobBuilder) AdvanceReservation(id string) *JobBuilder {
	b.args = append(b.args, "-ar", id)
	return b
}

// Binding sets the processor binding specification (-binding). This is
// the v9.0 legacy binding format. For v9.1, use the granular Flag()
// calls: -bamount, -bstrategy, etc.
func (b *JobBuilder) Binding(spec string) *JobBuilder {
	b.args = append(b.args, "-binding", spec)
	return b
}

// PTY requests a pseudo-terminal (-pty y).
func (b *JobBuilder) PTY() *JobBuilder {
	b.args = append(b.args, "-pty", "y")
	return b
}

// JobShare sets the job share value (-js).
func (b *JobBuilder) JobShare(share int) *JobBuilder {
	b.args = append(b.args, "-js", fmt.Sprintf("%d", share))
	return b
}

// JSV sets a job submission verification script (-jsv).
func (b *JobBuilder) JSV(url string) *JobBuilder {
	b.args = append(b.args, "-jsv", url)
	return b
}

// Scope sets the scope for resource requests (-scope).
func (b *JobBuilder) Scope(name string) *JobBuilder {
	b.args = append(b.args, "-scope", name)
	return b
}

// VerifyMode sets the verification mode (-w).
func (b *JobBuilder) VerifyMode(mode string) *JobBuilder {
	b.args = append(b.args, "-w", mode)
	return b
}

// CommandFile reads additional options from a file (-@).
func (b *JobBuilder) CommandFile(path string) *JobBuilder {
	b.args = append(b.args, "-@", path)
	return b
}

// Department sets the department for the job (-dept).
func (b *JobBuilder) Department(name string) *JobBuilder {
	b.args = append(b.args, "-dept", name)
	return b
}

// Verify enables job verification without submission (-verify).
func (b *JobBuilder) Verify() *JobBuilder {
	b.args = append(b.args, "-verify")
	return b
}

// --- Context ---

// AddContext adds a context variable to the job (-ac key=value).
func (b *JobBuilder) AddContext(key, value string) *JobBuilder {
	b.args = append(b.args, "-ac", fmt.Sprintf("%s=%s", key, value))
	return b
}

// DeleteContext removes context variable(s) from the job (-dc).
func (b *JobBuilder) DeleteContext(vars ...string) *JobBuilder {
	b.args = append(b.args, "-dc", strings.Join(vars, ","))
	return b
}

// SetContext sets a context variable on the job (-sc key=value).
func (b *JobBuilder) SetContext(key, value string) *JobBuilder {
	b.args = append(b.args, "-sc", fmt.Sprintf("%s=%s", key, value))
	return b
}

// Clear resets all job options to defaults (-clear).
func (b *JobBuilder) Clear() *JobBuilder {
	b.args = append(b.args, "-clear")
	return b
}

// --- Escape hatch ---

// Flag appends a raw flag and optional value. Use this for any options
// not covered by the named methods, including v9.1-specific binding
// options like -bamount, -bstrategy, etc.
func (b *JobBuilder) Flag(name string, value ...string) *JobBuilder {
	b.args = append(b.args, name)
	if len(value) > 0 && value[0] != "" {
		b.args = append(b.args, value[0])
	}
	return b
}

// Args returns the accumulated arguments without the command. Useful
// for inspection or testing.
func (b *JobBuilder) Args() []string {
	return b.args
}

// --- Submit ---

// Submit executes the job submission and returns the job ID, raw output,
// and any error. The builder automatically adds -terse for reliable job
// ID parsing.
func (b *JobBuilder) Submit(ctx context.Context) (int64, string, error) {
	allArgs := make([]string, 0, len(b.args)+2+len(b.cmdArgs))
	allArgs = append(allArgs, "-terse")
	allArgs = append(allArgs, b.args...)
	allArgs = append(allArgs, b.command)
	allArgs = append(allArgs, b.cmdArgs...)

	output, err := b.submitter.SubmitWithNativeSpecification(ctx, allArgs)
	if err != nil {
		return 0, output, err
	}

	outputStr := strings.TrimSpace(output)
	if strings.HasPrefix(outputStr, "Dry run:") {
		return 0, outputStr, nil
	}

	return ParseSubmitOutput(output, true, b.sync, false)
}
