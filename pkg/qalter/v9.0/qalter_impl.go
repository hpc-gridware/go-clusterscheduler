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

package qalter

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

type CommandLineQAlter struct {
	config CommandLineQAlterConfig
}

type CommandLineQAlterConfig struct {
	Executable string
	DryRun     bool
	// DelayAfter is the time to wait after executing a command.
	// This is useful for not overloading qmaster when 1000s of
	// configuration objects (like queues) are defined.
	DelayAfter time.Duration
}

// NewCommandLineQAlter creates a new instance of CommandLineQAlter.
func NewCommandLineQAlter(config CommandLineQAlterConfig) (*CommandLineQAlter, error) {
	if config.Executable == "" {
		config.Executable = "qalter"
	}
	return &CommandLineQAlter{config: config}, nil
}

func (q *CommandLineQAlter) NativeSpecification(args []string) (string, error) {
	return q.RunCommand(args...)
}

// RunCommand executes the qalter command with the specified arguments.
func (c *CommandLineQAlter) RunCommand(args ...string) (string, error) {
	if c.config.DryRun {
		fmt.Printf("Executing: %s, %v", c.config.Executable, args)
		return "", nil
	}
	cmd := exec.Command(c.config.Executable, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	// ensure that qalter returns a single line of output for each entry
	cmd.Env = append(cmd.Environ(), "SGE_SINGLE_LINE=true")
	err := cmd.Run()
	if c.config.DelayAfter != 0 {
		<-time.After(c.config.DelayAfter)
	}
	if err != nil {
		return out.String(), fmt.Errorf("failed to run command (%s): %v",
			out.String(), err)
	}
	return out.String(), err
}
