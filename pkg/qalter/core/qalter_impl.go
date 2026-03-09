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
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const dateTimeFormat = "200601021504.05"

// WhenMode controls when job modifications are applied (GCS 9.1+).
type WhenMode string

const (
	// WhenNone leaves the default behavior (no -when flag).
	WhenNone WhenMode = ""
	// WhenNow applies modifications immediately.
	WhenNow WhenMode = "now"
	// WhenOnReschedule defers modifications until the job is rescheduled.
	WhenOnReschedule WhenMode = "on_reschedule"
)

// CommandLineQAlter is a concrete implementation of the QAlter interface
// using the qalter command line tool.
type CommandLineQAlter struct {
	config CommandLineQAlterConfig
}

// CommandLineQAlterConfig holds configuration for the CommandLineQAlter.
type CommandLineQAlterConfig struct {
	Executable string
	DryRun     bool
	// DelayAfter is the time to wait after executing a command.
	// This is useful for not overloading qmaster when 1000s of
	// configuration objects (like queues) are defined.
	DelayAfter time.Duration
	// When controls when modifications are applied.
	// Use WhenNow or WhenOnReschedule. Leave as WhenNone (default)
	// for default behavior. This option is only supported in GCS 9.1+.
	When WhenMode
}

// NewCommandLineQAlter creates a new instance of CommandLineQAlter.
func NewCommandLineQAlter(config CommandLineQAlterConfig) (*CommandLineQAlter, error) {
	if config.Executable == "" {
		config.Executable = "qalter"
	}
	return &CommandLineQAlter{config: config}, nil
}

// RunCommand executes the qalter command with the specified arguments.
func (c *CommandLineQAlter) RunCommand(args ...string) (string, error) {
	if c.config.DryRun {
		fmt.Printf("Executing: %s %v\n", c.config.Executable, args)
		return "", nil
	}
	cmd := exec.Command(c.config.Executable, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Env = append(cmd.Environ(), "SGE_SINGLE_LINE=true")
	err := cmd.Run()
	if c.config.DelayAfter != 0 {
		<-time.After(c.config.DelayAfter)
	}
	if err != nil {
		return out.String(), fmt.Errorf("failed to run command (%s): %v",
			out.String(), err)
	}
	return out.String(), nil
}

// GlobalArgs returns the global arguments that should be prepended to
// all qalter operations (e.g. -when). This is exported so that
// version-specific packages can use it in their own methods.
func (c *CommandLineQAlter) GlobalArgs() []string {
	if c.config.When != WhenNone {
		return []string{"-when", string(c.config.When)}
	}
	return nil
}

func (c *CommandLineQAlter) runOption(jobTaskList, flag, value string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, flag, value, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) runFlag(jobTaskList, flag string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, flag, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) runBoolOption(jobTaskList, flag string, value bool) (string, error) {
	v := "n"
	if value {
		v = "y"
	}
	return c.runOption(jobTaskList, flag, v)
}

func (c *CommandLineQAlter) runListOption(jobTaskList, flag string, list []string) (string, error) {
	return c.runOption(jobTaskList, flag, strings.Join(list, ","))
}

func (c *CommandLineQAlter) runIntOption(jobTaskList, flag string, value int) (string, error) {
	return c.runOption(jobTaskList, flag, strconv.Itoa(value))
}

func (c *CommandLineQAlter) runTimeOption(jobTaskList, flag string, t time.Time) (string, error) {
	return c.runOption(jobTaskList, flag, t.Format(dateTimeFormat))
}

func (c *CommandLineQAlter) runCompound(jobTaskList string, flagsAndValues ...string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, flagsAndValues...)
	args = append(args, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) runCompoundList(jobTaskList string, list []string, flagsBeforeList ...string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, flagsBeforeList...)
	args = append(args, strings.Join(list, ","), jobTaskList)
	return c.RunCommand(args...)
}

// --- Time & Scheduling ---

func (c *CommandLineQAlter) SetStartTime(jobTaskList string, dateTime time.Time) (string, error) {
	return c.runTimeOption(jobTaskList, "-a", dateTime)
}

func (c *CommandLineQAlter) SetDeadline(jobTaskList string, dateTime time.Time) (string, error) {
	return c.runTimeOption(jobTaskList, "-dl", dateTime)
}

// --- Job Identity & Metadata ---

func (c *CommandLineQAlter) SetJobName(jobTaskList, name string) (string, error) {
	return c.runOption(jobTaskList, "-N", name)
}

func (c *CommandLineQAlter) SetAccountString(jobTaskList, account string) (string, error) {
	return c.runOption(jobTaskList, "-A", account)
}

func (c *CommandLineQAlter) SetProject(jobTaskList, project string) (string, error) {
	return c.runOption(jobTaskList, "-P", project)
}

func (c *CommandLineQAlter) SetDepartment(jobTaskList, department string) (string, error) {
	return c.runOption(jobTaskList, "-dept", department)
}

// --- Context Variables ---

func (c *CommandLineQAlter) AddContext(jobTaskList string, contextList []string) (string, error) {
	return c.runListOption(jobTaskList, "-ac", contextList)
}

func (c *CommandLineQAlter) DeleteContext(jobTaskList string, contextList []string) (string, error) {
	return c.runListOption(jobTaskList, "-dc", contextList)
}

func (c *CommandLineQAlter) SetContext(jobTaskList string, contextList []string) (string, error) {
	return c.runListOption(jobTaskList, "-sc", contextList)
}

// --- Resource Requests ---

func (c *CommandLineQAlter) SetHardResourceList(jobTaskList string, resourceList []string) (string, error) {
	return c.runCompoundList(jobTaskList, resourceList, "-hard", "-l")
}

func (c *CommandLineQAlter) SetSoftResourceList(jobTaskList string, resourceList []string) (string, error) {
	return c.runCompoundList(jobTaskList, resourceList, "-soft", "-l")
}

// --- Queue Binding ---

func (c *CommandLineQAlter) SetHardQueue(jobTaskList string, queueList []string) (string, error) {
	return c.runCompoundList(jobTaskList, queueList, "-hard", "-q")
}

func (c *CommandLineQAlter) SetSoftQueue(jobTaskList string, queueList []string) (string, error) {
	return c.runCompoundList(jobTaskList, queueList, "-soft", "-q")
}

func (c *CommandLineQAlter) SetHardMasterQueue(jobTaskList string, queueList []string) (string, error) {
	return c.runCompoundList(jobTaskList, queueList, "-hard", "-masterq")
}

func (c *CommandLineQAlter) SetSoftMasterQueue(jobTaskList string, queueList []string) (string, error) {
	return c.runCompoundList(jobTaskList, queueList, "-soft", "-masterq")
}

// --- Parallel Environment ---

func (c *CommandLineQAlter) SetParallelEnvironment(jobTaskList, peName, slotRange string) (string, error) {
	return c.runCompound(jobTaskList, "-pe", peName, slotRange)
}

// --- I/O & Paths ---

func (c *CommandLineQAlter) SetErrorPath(jobTaskList string, pathList []string) (string, error) {
	return c.runListOption(jobTaskList, "-e", pathList)
}

func (c *CommandLineQAlter) SetOutputPath(jobTaskList string, pathList []string) (string, error) {
	return c.runListOption(jobTaskList, "-o", pathList)
}

func (c *CommandLineQAlter) SetInputFile(jobTaskList string, fileList []string) (string, error) {
	return c.runListOption(jobTaskList, "-i", fileList)
}

func (c *CommandLineQAlter) SetShellPath(jobTaskList string, pathList []string) (string, error) {
	return c.runListOption(jobTaskList, "-S", pathList)
}

func (c *CommandLineQAlter) SetMergeOutput(jobTaskList string, merge bool) (string, error) {
	return c.runBoolOption(jobTaskList, "-j", merge)
}

// --- Working Directory ---

func (c *CommandLineQAlter) SetCwd(jobTaskList string) (string, error) {
	return c.runFlag(jobTaskList, "-cwd")
}

func (c *CommandLineQAlter) SetWorkingDirectory(jobTaskList, path string) (string, error) {
	return c.runOption(jobTaskList, "-wd", path)
}

// --- Checkpointing ---

func (c *CommandLineQAlter) SetCheckpointSelector(jobTaskList, selector string) (string, error) {
	return c.runOption(jobTaskList, "-c", selector)
}

func (c *CommandLineQAlter) SetCheckpointMethod(jobTaskList, name string) (string, error) {
	return c.runOption(jobTaskList, "-ckpt", name)
}

// --- Holds & Dependencies ---

func (c *CommandLineQAlter) SetHold(jobTaskList, holdList string) (string, error) {
	return c.runOption(jobTaskList, "-h", holdList)
}

func (c *CommandLineQAlter) SetHoldJobDependency(jobTaskList string, jobIDList []string) (string, error) {
	return c.runListOption(jobTaskList, "-hold_jid", jobIDList)
}

func (c *CommandLineQAlter) SetHoldArrayDependency(jobTaskList string, jobIDList []string) (string, error) {
	return c.runListOption(jobTaskList, "-hold_jid_ad", jobIDList)
}

// --- Priority & Tickets ---

func (c *CommandLineQAlter) SetPriority(jobTaskList string, priority int) (string, error) {
	return c.runIntOption(jobTaskList, "-p", priority)
}

func (c *CommandLineQAlter) SetJobShare(jobTaskList string, share int) (string, error) {
	return c.runIntOption(jobTaskList, "-js", share)
}

func (c *CommandLineQAlter) SetOverrideTickets(jobTaskList string, tickets int) (string, error) {
	return c.runIntOption(jobTaskList, "-ot", tickets)
}

// --- Notification ---

func (c *CommandLineQAlter) SetMailOptions(jobTaskList, options string) (string, error) {
	return c.runOption(jobTaskList, "-m", options)
}

func (c *CommandLineQAlter) SetMailRecipients(jobTaskList string, mailList []string) (string, error) {
	return c.runListOption(jobTaskList, "-M", mailList)
}

func (c *CommandLineQAlter) SetNotify(jobTaskList string) (string, error) {
	return c.runFlag(jobTaskList, "-notify")
}

// --- Environment Variables ---

func (c *CommandLineQAlter) SetEnvironmentVariables(jobTaskList string, variableList []string) (string, error) {
	return c.runListOption(jobTaskList, "-v", variableList)
}

func (c *CommandLineQAlter) ExportAllEnvironmentVariables(jobTaskList string) (string, error) {
	return c.runFlag(jobTaskList, "-V")
}

// --- Reservation & Restart ---

func (c *CommandLineQAlter) SetReservation(jobTaskList string, reservation bool) (string, error) {
	return c.runBoolOption(jobTaskList, "-R", reservation)
}

func (c *CommandLineQAlter) SetRestartable(jobTaskList string, restartable bool) (string, error) {
	return c.runBoolOption(jobTaskList, "-r", restartable)
}

// --- Advance Reservation ---

func (c *CommandLineQAlter) SetAdvanceReservation(jobTaskList, arID string) (string, error) {
	return c.runOption(jobTaskList, "-ar", arID)
}

// --- Task Control ---

func (c *CommandLineQAlter) SetMaxRunningTasks(jobTaskList string, maxTasks int) (string, error) {
	return c.runIntOption(jobTaskList, "-tc", maxTasks)
}

// --- Verification ---

func (c *CommandLineQAlter) SetVerifyMode(jobTaskList, mode string) (string, error) {
	return c.runOption(jobTaskList, "-w", mode)
}

// --- Raw ---

func (c *CommandLineQAlter) NativeSpecification(args []string) (string, error) {
	return c.RunCommand(args...)
}
