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
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

type QAcctImpl struct {
	config CommandLineQAcctConfig
}

type CommandLineQAcctConfig struct {
	Executable     string
	AccountingFile string
	DryRun         bool
}

func NewCommandLineQAcct(config CommandLineQAcctConfig) (*QAcctImpl, error) {
	if config.Executable == "" {
		config.Executable = "qacct"
	}
	if !config.DryRun {
		_, err := exec.LookPath(config.Executable)
		if err != nil {
			return nil, fmt.Errorf("%s not found in PATH", config.Executable)
		}
	}
	return &QAcctImpl{config: config}, nil
}

func (q *QAcctImpl) WithAlternativeAccountingFile(accountingFile string) error {
	q.config.AccountingFile = accountingFile
	return nil
}

func (q *QAcctImpl) WithDefaultAccountingFile() {
	q.config.AccountingFile = ""
}

// NativeSpecification runs the qacct command with the given arguments
// and returns the raw output.
func (q *QAcctImpl) NativeSpecification(args []string) (string, error) {
	if q.config.AccountingFile != "" {
		args = append(args, "-f", q.config.AccountingFile)
	}

	// Layer 1 guard: reject control characters and invalid UTF-8 in any argv
	// token (including the appended accounting file path) before spawning
	// qacct. Native passthrough, so only the universal check applies.
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", err
	}

	if q.config.DryRun {
		return fmt.Sprintf("Dry run: %s %v", q.config.Executable, args), nil
	}

	command := exec.Command(q.config.Executable, args...)
	out, err := command.Output()
	if err != nil {
		return "", commandError(err)
	}
	return string(out), nil
}

// maxStderrLineBytes bounds the part of qacct's stderr kept in an error.
const maxStderrLineBytes = 1024

// commandError is the error of a failed qacct run, in the shape the other
// command wrappers of this module use: "failed to run command (<stderr>):
// <cause>". Without the stderr part a refusal read "exit status 1" only,
// although qacct had named the cause (an unknown job, a date it cannot
// read).
//
// Only qacct's message is kept: after a usage error qacct prints its
// version banner and whole help text, below the message if it has one.
func commandError(cause error) error {
	var exitErr *exec.ExitError
	stderr := ""
	if errors.As(cause, &exitErr) {
		stderr = stderrMessage(string(exitErr.Stderr))
	}
	return fmt.Errorf("failed to run command (%s): %w", stderr, cause)
}

// usagePrefix starts the help text qacct prints after a usage error.
const usagePrefix = "usage:"

// stderrMessage returns the first non-empty line of qacct's stderr, trimmed
// and cut at maxStderrLineBytes.
//
// It returns "" when that line is only the version banner. For an unknown
// option or a missing argument qacct names no cause: it prints the banner
// ("OCS 9.1.5 (250826-0734)") directly followed by the usage text, and the
// banner would read like a cause without being one.
func stderrMessage(text string) string {
	lines := nonEmptyLines(text)
	if len(lines) == 0 || strings.HasPrefix(lines[0], usagePrefix) {
		return ""
	}
	if len(lines) > 1 && strings.HasPrefix(lines[1], usagePrefix) {
		return ""
	}
	message := lines[0]
	if len(message) > maxStderrLineBytes {
		message = message[:maxStderrLineBytes]
	}
	return message
}

// nonEmptyLines returns the trimmed lines of text that are not empty.
func nonEmptyLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func (q *QAcctImpl) ShowHelp() (string, error) {
	return q.NativeSpecification([]string{"-help"})
}

func (q *QAcctImpl) Summary() *SummaryBuilder {
	return NewSummaryBuilder(q)
}

func (q *QAcctImpl) Jobs() *JobsBuilder {
	return NewJobsBuilder(q)
}

// ShowJobDetails returns the job details for the given job IDs. If the jobIDs
// is nil, all jobs are returned.
func (q *QAcctImpl) ShowJobDetails(jobIDs []int64) ([]JobDetail, error) {

	allJobDetails := []JobDetail{}

	// if jobIDs is nil, return all jobs
	if len(jobIDs) == 0 {
		out, err := q.NativeSpecification([]string{"-j", "*"})
		if err != nil {
			return nil, fmt.Errorf("error getting job details: %w", err)
		}
		jobDetails, err := ParseQAcctJobOutput(out)
		if err != nil {
			return nil, fmt.Errorf("error parsing job details: %w", err)
		}
		allJobDetails = append(allJobDetails, jobDetails...)
		return allJobDetails, nil
	}

	for _, jobID := range jobIDs {
		args := []string{"-j"}
		args = append(args, fmt.Sprintf("%d", jobID))

		out, err := q.NativeSpecification(args)
		if err != nil {
			return nil, fmt.Errorf("error getting job details: %w", err)
		}

		jobDetails, err := ParseQAcctJobOutput(out)
		if err != nil {
			return nil, fmt.Errorf("error parsing job details: %w", err)
		}
		allJobDetails = append(allJobDetails, jobDetails...)
	}

	return allJobDetails, nil
}
