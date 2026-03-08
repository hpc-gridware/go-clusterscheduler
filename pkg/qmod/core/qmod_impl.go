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

package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandLineQMod is a concrete implementation of the QMod interface
// using the qmod command line tool.
type CommandLineQMod struct {
	config CommandLineQModConfig
}

// CommandLineQModConfig holds configuration for the CommandLineQMod.
type CommandLineQModConfig struct {
	Executable string
	DryRun     bool
	// Force enables the -f flag for all modification actions.
	Force bool
	// DelayAfter is the time to wait after executing a command.
	DelayAfter time.Duration
}

// NewCommandLineQMod creates a new instance of CommandLineQMod.
func NewCommandLineQMod(config CommandLineQModConfig) (*CommandLineQMod, error) {
	if config.Executable == "" {
		config.Executable = "qmod"
	}
	return &CommandLineQMod{config: config}, nil
}

// RunCommand executes the qmod command with the specified arguments.
func (c *CommandLineQMod) RunCommand(args ...string) (string, error) {
	if c.config.DryRun {
		fmt.Printf("Executing: %s %v\n", c.config.Executable, args)
		return "", nil
	}
	cmd := exec.Command(c.config.Executable, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
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

func (c *CommandLineQMod) runAction(flag string, targets []string) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("no targets specified")
	}
	args := []string{}
	if c.config.Force {
		args = append(args, "-f")
	}
	args = append(args, flag, strings.Join(targets, ","))
	return c.RunCommand(args...)
}

func (c *CommandLineQMod) ClearErrorState(jobOrQueueList []string) (string, error) {
	return c.runAction("-c", jobOrQueueList)
}

func (c *CommandLineQMod) ClearJobErrorState(jobList []string) (string, error) {
	return c.runAction("-cj", jobList)
}

func (c *CommandLineQMod) ClearQueueErrorState(queueList []string) (string, error) {
	return c.runAction("-cq", queueList)
}

func (c *CommandLineQMod) Disable(queueList []string) (string, error) {
	return c.runAction("-d", queueList)
}

func (c *CommandLineQMod) Enable(queueList []string) (string, error) {
	return c.runAction("-e", queueList)
}

func (c *CommandLineQMod) RescheduleJobs(jobList []string) (string, error) {
	return c.runAction("-rj", jobList)
}

func (c *CommandLineQMod) RescheduleQueues(queueList []string) (string, error) {
	return c.runAction("-rq", queueList)
}

func (c *CommandLineQMod) Suspend(jobOrQueueList []string) (string, error) {
	return c.runAction("-s", jobOrQueueList)
}

func (c *CommandLineQMod) SuspendJobs(jobList []string) (string, error) {
	return c.runAction("-sj", jobList)
}

func (c *CommandLineQMod) SuspendQueues(queueList []string) (string, error) {
	return c.runAction("-sq", queueList)
}

func (c *CommandLineQMod) Unsuspend(jobOrQueueList []string) (string, error) {
	return c.runAction("-us", jobOrQueueList)
}

func (c *CommandLineQMod) UnsuspendJobs(jobList []string) (string, error) {
	return c.runAction("-usj", jobList)
}

func (c *CommandLineQMod) UnsuspendQueues(queueList []string) (string, error) {
	return c.runAction("-usq", queueList)
}

func (c *CommandLineQMod) NativeSpecification(args []string) (string, error) {
	return c.RunCommand(args...)
}
