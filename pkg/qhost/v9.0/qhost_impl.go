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

package qhost

import (
	"fmt"
	"os/exec"
)

type QHostImpl struct {
	config CommandLineQHostConfig
}

type CommandLineQHostConfig struct {
	Executable string
	DryRun     bool
}

func NewCommandLineQhost(config CommandLineQHostConfig) (*QHostImpl, error) {
	if config.Executable == "" {
		config.Executable = "qhost"
	}
	if config.DryRun == false {
		// check if executable is reachable
		_, err := exec.LookPath(config.Executable)
		if err != nil {
			return nil, fmt.Errorf("executable not found: %w", err)
		}
	}
	return &QHostImpl{config: config}, nil
}

// NativeSpecification returns the output of the qhost command for the given
// arguments. The arguments are passed to the qhost command as is.
// The output is returned as a string.
func (q *QHostImpl) NativeSpecification(args []string) (string, error) {
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qhost %v", args), nil
	}
	command := exec.Command(q.config.Executable, args...)
	out, err := command.Output()
	if err != nil {
		// convert error in exit error
		ee, ok := err.(*exec.ExitError)
		if ok {
			if !ee.Success() {
				return "", fmt.Errorf("qhost command failed with exit code %d",
					ee.ExitCode())
			}
			return "", nil
		}
		return "", fmt.Errorf("failed to get output of qhost: %w", err)
	}
	return string(out), nil
}

// GetHosts returns the output of the qhost command.
func (q *QHostImpl) GetHosts() ([]Host, error) {
	out, err := q.NativeSpecification(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get output of qhost: %w", err)
	}
	hosts, err := ParseHosts(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output of qhost: %w", err)
	}
	return hosts, nil
}

// GetHostsFullMetrics returns the output of the qhost command with
// the -F option.
func (q *QHostImpl) GetHostsFullMetrics() ([]HostFullMetrics, error) {
	out, err := q.NativeSpecification([]string{"-F"})
	if err != nil {
		return nil, fmt.Errorf("failed to get output of qhost: %w", err)
	}
	hosts, err := ParseHostFullMetrics(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output of qhost: %w", err)
	}
	return hosts, nil
}
