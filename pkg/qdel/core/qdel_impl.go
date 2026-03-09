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
	"strings"
	"time"
)

// CommandLineQDel is a concrete implementation of the QDel interface
// using the qdel command line tool.
type CommandLineQDel struct {
	config CommandLineQDelConfig
}

// CommandLineQDelConfig holds configuration for the CommandLineQDel.
type CommandLineQDelConfig struct {
	Executable string
	DryRun     bool
	// Force enables the -f flag for all delete actions.
	Force bool
	// DelayAfter is the time to wait after executing a command.
	DelayAfter time.Duration
}

// NewCommandLineQDel creates a new instance of CommandLineQDel.
func NewCommandLineQDel(config CommandLineQDelConfig) (*CommandLineQDel, error) {
	if config.Executable == "" {
		config.Executable = "qdel"
	}
	return &CommandLineQDel{config: config}, nil
}

func (c *CommandLineQDel) runCommand(args ...string) (string, error) {
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

func (c *CommandLineQDel) forceArgs() []string {
	if c.config.Force {
		return []string{"-f"}
	}
	return nil
}

func (c *CommandLineQDel) DeleteJobs(jobTaskList []string) (string, error) {
	if len(jobTaskList) == 0 {
		return "", fmt.Errorf("no jobs specified")
	}
	args := c.forceArgs()
	args = append(args, strings.Join(jobTaskList, ","))
	return c.runCommand(args...)
}

func (c *CommandLineQDel) DeleteUserJobs(userList []string) (string, error) {
	if len(userList) == 0 {
		return "", fmt.Errorf("no users specified")
	}
	args := c.forceArgs()
	args = append(args, "-u", strings.Join(userList, ","))
	return c.runCommand(args...)
}

func (c *CommandLineQDel) NativeSpecification(args []string) (string, error) {
	return c.runCommand(args...)
}
