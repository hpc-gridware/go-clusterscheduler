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

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// qsubClient is a concrete implementation of the Qsub interface.
type qsubClient struct {
	config CommandLineQSubConfig
}

type CommandLineQSubConfig struct {
	QsubPath string
	DryRun   bool
}

// NewCommandLineQSub creates a new Qsub client.
// If qsubPath is empty, it defaults to "qsub".
func NewCommandLineQSub(config CommandLineQSubConfig) (Qsub, error) {
	if config.QsubPath == "" {
		config.QsubPath = "qsub"
	}
	if config.DryRun == false {
		_, err := exec.LookPath(config.QsubPath)
		if err != nil {
			return nil, fmt.Errorf("executable not found: %w", err)
		}
	}
	return &qsubClient{config: config}, nil
}

// SubmitWithNativeSpecification submits a job with the given options and
// returns the job submission output with the job ID or an error.
func (c *qsubClient) SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no arguments provided")
	}
	if c.config.DryRun {
		return fmt.Sprintf("Dry run: qsub %s", strings.Join(args, " ")), nil
	}
	// execute qsub with the given options
	cmd := exec.CommandContext(ctx, c.config.QsubPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qsub error: %v, output: %s", err, string(output))
	}
	return string(output), nil
}

// Submit constructs and executes the qsub command based on the provided
// JobOptions. It returns the jobID, the raw output of the job submission
// command, and an error if the submission command failed.
func (c *qsubClient) Submit(ctx context.Context, opts JobOptions) (int64, string, error) {
	// default to terse output if not specified, to parse the job ID
	// (if terse is not specified, qsub does not return the job ID, it
	// returns the full output of the qsub command)
	if opts.Terse == nil {
		opts.Terse = new(bool)
		*opts.Terse = true
	}

	cmdArgs, err := buildQsubArgs(opts)
	if err != nil {
		return 0, "", err
	}

	output, err := c.SubmitWithNativeSpecification(ctx, cmdArgs)
	if err != nil {
		return 0, output, err
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return 0, "", errors.New("qsub did not return a job ID")
	}

	// if terse is specified, qsub returns just the job ID, so we need to
	// strip any trailing newlines
	if *opts.Terse {
		outputStr = strings.TrimRight(outputStr, "\n")
		// jobid could be a number or like 7.1-100:2
		jobidstr := strings.Split(outputStr, ".")[0]
		// parse the job ID as an int64
		jobIDInt, err := strconv.ParseInt(jobidstr, 10, 64)
		if err != nil {
			return 0, outputStr, fmt.Errorf("invalid job ID: %s", jobidstr)
		}
		return jobIDInt, outputStr, nil
	}

	return 0, outputStr, nil
}

// SubmitSimple submits a job with just the command (expected to be a
// script in the path of the execution host) and arguments.
func (c *qsubClient) SubmitSimple(ctx context.Context, command string, args ...string) (int64, string, error) {
	opts := JobOptions{
		Command:     command,
		CommandArgs: args,
	}
	return c.Submit(ctx, opts)
}

// SubmitSimpleBinary submits a simple executable with minimal options.
func (c *qsubClient) SubmitSimpleBinary(ctx context.Context, command string, args ...string) (int64, string, error) {
	binary := true

	opts := JobOptions{
		Command:     command,
		CommandArgs: args,
		Binary:      &binary,
	}
	return c.Submit(ctx, opts)
}

// SubmitWithQueue submits a job to a specific queue.
func (c *qsubClient) SubmitWithQueue(ctx context.Context, queue string, opts JobOptions) (int64, string, error) {
	opts.Queue = append(opts.Queue, queue)
	return c.Submit(ctx, opts)
}

// buildQsubArgs constructs the qsub command-line arguments from JobOptions.
func buildQsubArgs(opts JobOptions) ([]string, error) {
	var args []string

	// Helper function to add flags
	addFlag := func(flag string, value string) {
		if value != "" {
			args = append(args, flag, value)
		} else {
			args = append(args, flag)
		}
	}

	// Time options
	if opts.StartTime != nil {
		addFlag("-a", *opts.StartTime)
	}
	if opts.Deadline != nil {
		addFlag("-dl", *opts.Deadline)
	}
	if opts.AdvanceReservationID != nil {
		addFlag("-ar", *opts.AdvanceReservationID)
	}

	// Resource options
	if opts.Account != nil {
		addFlag("-A", *opts.Account)
	}
	if opts.Project != nil {
		addFlag("-P", *opts.Project)
	}
	if opts.Priority != nil {
		addFlag("-p", fmt.Sprintf("%d", *opts.Priority))
	}
	if len(opts.ResourcesHardRequest) > 0 {
		var res []string
		for k, v := range opts.ResourcesHardRequest {
			res = append(res, fmt.Sprintf("%s=%s", k, v))
		}
		addFlag("-hard", "")
		addFlag("-l", strings.Join(res, ","))
	}
	if len(opts.ResourcesSoftRequest) > 0 {
		var res []string
		for k, v := range opts.ResourcesSoftRequest {
			res = append(res, fmt.Sprintf("%s=%s", k, v))
		}
		addFlag("-soft", "")
		addFlag("-l", strings.Join(res, ","))
	}
	if len(opts.Queue) > 0 {
		addFlag("-q", strings.Join(opts.Queue, ","))
	}
	if opts.Slots != nil {
		addFlag("-pe", *opts.Slots)
	}

	// Output/Input options
	if len(opts.StdErr) > 0 {
		addFlag("-e", strings.Join(opts.StdErr, ","))
	}
	if len(opts.StdOut) > 0 {
		addFlag("-o", strings.Join(opts.StdOut, ","))
	}
	if len(opts.StdIn) > 0 {
		addFlag("-i", strings.Join(opts.StdIn, ","))
	}

	// Execution options
	if opts.Binary != nil {
		if *opts.Binary {
			addFlag("-b", "y")
		} else {
			addFlag("-b", "n")
		}
	}
	if opts.WorkingDir != nil {
		addFlag("-cwd", *opts.WorkingDir)
	}
	if opts.CommandPrefix != nil {
		addFlag("-C", *opts.CommandPrefix)
	}
	if opts.Shell != nil {
		if *opts.Shell {
			addFlag("-shell", "y")
		} else {
			addFlag("-shell", "n")
		}
	}
	if opts.CommandInterpreter != nil {
		addFlag("-S", *opts.CommandInterpreter)
	}
	if opts.JobName != nil {
		addFlag("-N", *opts.JobName)
	}
	if opts.JobArray != nil {
		addFlag("-t", *opts.JobArray)
	}
	if opts.MaxRunningTasks != nil {
		addFlag("-tc", fmt.Sprintf("%d", *opts.MaxRunningTasks))
	}

	// Notification options
	if opts.MailOptions != nil {
		addFlag("-m", *opts.MailOptions)
	}
	if len(opts.MailList) > 0 {
		addFlag("-M", strings.Join(opts.MailList, ","))
	}
	if opts.Notify != nil && *opts.Notify {
		addFlag("-notify", "")
	}

	// Dependency options
	if len(opts.HoldJobIDs) > 0 {
		addFlag("-hold_jid", strings.Join(opts.HoldJobIDs, ","))
	}
	if len(opts.HoldArrayJobIDs) > 0 {
		addFlag("-hold_jid_ad", strings.Join(opts.HoldArrayJobIDs, ","))
	}

	// Other options
	if opts.Checkpoint != nil {
		addFlag("-ckpt", *opts.Checkpoint)
	}
	if opts.CheckpointSelector != nil {
		addFlag("-c", *opts.CheckpointSelector)
	}
	if opts.MergeStdOutErr != nil {
		if *opts.MergeStdOutErr {
			addFlag("-j", "y")
		} else {
			addFlag("-j", "n")
		}
	}
	if opts.Verify != nil && *opts.Verify {
		addFlag("-verify", "")
	}
	if opts.ExportAllEnv {
		addFlag("-V", "")
	}
	if len(opts.ExportVariables) > 0 {
		var envVars []string
		for k, v := range opts.ExportVariables {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		addFlag("-v", strings.Join(envVars, ","))
	}
	if opts.Hold != nil && *opts.Hold {
		addFlag("-h", "y")
	}

	if opts.Synchronize != nil && *opts.Synchronize {
		addFlag("-sync", "y")
	}
	if opts.ReservationDesired != nil {
		if *opts.ReservationDesired {
			addFlag("-R", "y")
		} else {
			addFlag("-R", "n")
		}
	}
	if opts.Restartable != nil {
		if *opts.Restartable {
			addFlag("-r", "y")
		} else {
			addFlag("-r", "n")
		}
	}
	if opts.Clear != nil && *opts.Clear {
		addFlag("-clear", "")
	}
	if opts.Terse != nil && *opts.Terse {
		addFlag("-terse", "")
	}
	if opts.PTTY != nil {
		if *opts.PTTY {
			addFlag("-pty", "y")
		} else {
			addFlag("-pty", "n")
		}
	}

	// Command and arguments
	if opts.Command != "" {
		args = append(args, opts.Command)
		if len(opts.CommandArgs) > 0 {
			args = append(args, opts.CommandArgs...)
		}
	}

	return args, nil
}
