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
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// QSubClient is a concrete implementation of the Qsub interface.
type QSubClient struct {
	config CommandLineQSubConfig
}

type CommandLineQSubConfig struct {
	QsubPath string
	DryRun   bool
}

// NewCommandLineQSub creates a new Qsub client.
// If QsubPath is empty, it defaults to "qsub".
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
	return &QSubClient{config: config}, nil
}

// SubmitWithNativeSpecification submits a job with the given options and
// returns the job submission output with the job ID or an error.
func (c *QSubClient) SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no arguments provided")
	}
	if c.config.DryRun {
		return fmt.Sprintf("Dry run: qsub %s", strings.Join(args, " ")), nil
	}
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
func (c *QSubClient) Submit(ctx context.Context, opts JobOptions) (int64, string, error) {
	if opts.Terse == nil {
		opts.Terse = &True
	}

	cmdArgs, err := BuildQsubArgs(opts)
	if err != nil {
		return 0, "", err
	}

	output, err := c.SubmitWithNativeSpecification(ctx, cmdArgs)
	if err != nil {
		return 0, output, err
	}

	return ParseSubmitOutput(output, opts.Terse != nil && *opts.Terse,
		opts.Synchronize != nil && *opts.Synchronize, c.config.DryRun)
}

// SubmitSimple submits a job with just the command (expected to be a
// script in the path of the execution host) and arguments.
func (c *QSubClient) SubmitSimple(ctx context.Context, additionalOptions *JobOptions, command string, args ...string) (int64, string, error) {
	if additionalOptions == nil {
		additionalOptions = &JobOptions{}
	}
	additionalOptions.Command = command
	additionalOptions.CommandArgs = args
	return c.Submit(ctx, *additionalOptions)
}

// SubmitSimpleBinary submits a simple executable with minimal options.
func (c *QSubClient) SubmitSimpleBinary(ctx context.Context, command string, args ...string) (int64, string, error) {
	binary := true
	opts := JobOptions{
		Command:     command,
		CommandArgs: args,
		Binary:      &binary,
	}
	return c.Submit(ctx, opts)
}

// IsDryRun returns whether this client is in dry-run mode.
func (c *QSubClient) IsDryRun() bool {
	return c.config.DryRun
}

// ParseSubmitOutput parses the output of a qsub submission and extracts
// the job ID. This is exported so version-specific packages can reuse it.
func ParseSubmitOutput(output string, terse, synchronize, dryRun bool) (int64, string, error) {
	outputStr := strings.TrimSpace(output)
	if outputStr == "" {
		return 0, "", errors.New("qsub did not return a job ID")
	}

	if terse && !dryRun {
		outputStr = strings.TrimRight(outputStr, "\n")
		if synchronize {
			outputStr = strings.Split(outputStr, "\n")[0]
		}
		jobidstr := strings.Split(outputStr, ".")[0]
		jobIDInt, err := strconv.ParseInt(jobidstr, 10, 64)
		if err != nil {
			return 0, outputStr, fmt.Errorf("invalid job ID: %s", jobidstr)
		}
		return jobIDInt, outputStr, nil
	}

	return 0, outputStr, nil
}

// BuildQsubArgs constructs the qsub command-line arguments from JobOptions.
func BuildQsubArgs(opts JobOptions) ([]string, error) {
	var args []string

	addFlag := func(flag string, value string) {
		if value != "" {
			args = append(args, flag, value)
		} else {
			args = append(args, flag)
		}
	}

	if opts.StartTime != nil {
		addFlag("-a", ConvertTimeToQsubDateTime(*opts.StartTime))
	}
	if opts.Deadline != nil {
		addFlag("-dl", ConvertTimeToQsubDateTime(*opts.Deadline))
	}
	if opts.AdvanceReservationID != nil {
		addFlag("-ar", *opts.AdvanceReservationID)
	}

	if opts.Account != nil {
		addFlag("-A", *opts.Account)
	}
	if opts.Project != nil {
		addFlag("-P", *opts.Project)
	}
	if opts.Priority != nil {
		addFlag("-p", fmt.Sprintf("%d", *opts.Priority))
	}

	if len(opts.ScopedResources) > 0 {
		scopes := []string{ResourceRequestScopeGlobal, ResourceRequestScopeMaster, ResourceRequestScopeSlave}
		for _, scope := range scopes {
			if requests, ok := opts.ScopedResources[scope]; ok {
				addFlag("-scope", scope)
				reqTypes := []string{ResourceRequestTypeHard, ResourceRequestTypeSoft}
				for _, reqType := range reqTypes {
					if resourceRequest, ok := requests[reqType]; ok {
						if reqType == ResourceRequestTypeHard {
							addFlag("-hard", "")
						} else if reqType == ResourceRequestTypeSoft {
							addFlag("-soft", "")
						} else {
							return nil, fmt.Errorf(
								"invalid resource request type (expected hard or soft): %s",
								reqType)
						}
						var res []string
						for k, v := range resourceRequest.Resources {
							res = append(res, fmt.Sprintf("%s=%s", k, v))
						}
						addFlag("-l", strings.Join(res, ","))
					}
				}
			}
		}
	}

	if len(opts.Queue) > 0 {
		addFlag("-q", strings.Join(opts.Queue, ","))
	}
	if opts.ParallelEnvironment != nil {
		addFlag("-pe", *opts.ParallelEnvironment)
	}

	if len(opts.StdErr) > 0 {
		addFlag("-e", strings.Join(opts.StdErr, ","))
	}
	if len(opts.StdOut) > 0 {
		addFlag("-o", strings.Join(opts.StdOut, ","))
	}
	if len(opts.StdIn) > 0 {
		addFlag("-i", strings.Join(opts.StdIn, ","))
	}

	if opts.Binary != nil {
		if *opts.Binary {
			addFlag("-b", "y")
		} else {
			addFlag("-b", "n")
		}
	}
	if opts.WorkingDir != nil {
		addFlag("-cwd", "")
		addFlag("-wd", *opts.WorkingDir)
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

	if opts.MailOptions != nil {
		addFlag("-m", *opts.MailOptions)
	}
	if len(opts.MailList) > 0 {
		addFlag("-M", strings.Join(opts.MailList, ","))
	}
	if opts.Notify != nil && *opts.Notify {
		addFlag("-notify", "")
	}

	if len(opts.HoldJobIDs) > 0 {
		addFlag("-hold_jid", strings.Join(opts.HoldJobIDs, ","))
	}
	if len(opts.HoldArrayJobIDs) > 0 {
		addFlag("-hold_jid_ad", strings.Join(opts.HoldArrayJobIDs, ","))
	}

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
	if opts.ExportAllEnv != nil && *opts.ExportAllEnv {
		addFlag("-V", "")
	}
	if len(opts.EnvVariables) > 0 {
		var envVars []string
		for k, v := range opts.EnvVariables {
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

	if len(opts.AddContextVariables) > 0 {
		addFlag("-ac", strings.Join(opts.AddContextVariables, ","))
	}
	if len(opts.DeleteContextVariables) > 0 {
		addFlag("-dc", strings.Join(opts.DeleteContextVariables, ","))
	}
	if len(opts.SetJobContext) > 0 {
		var contextPairs []string
		for k, v := range opts.SetJobContext {
			contextPairs = append(contextPairs, fmt.Sprintf("%s=%s", k, v))
		}
		addFlag("-sc", strings.Join(contextPairs, ","))
	}

	if opts.ProcessorBinding != nil {
		addFlag("-binding", *opts.ProcessorBinding)
	}

	if opts.JobShare != nil {
		addFlag("-js", fmt.Sprintf("%d", *opts.JobShare))
	}
	if opts.JobSubmissionVerificationScript != nil {
		addFlag("-jsv", *opts.JobSubmissionVerificationScript)
	}

	if opts.ScopeName != nil {
		addFlag("-scope", *opts.ScopeName)
	}
	if opts.VerifyMode != nil {
		addFlag("-w", *opts.VerifyMode)
	}

	if opts.StartImmediately != nil {
		if *opts.StartImmediately {
			addFlag("-now", "y")
		} else {
			addFlag("-now", "n")
		}
	}

	if opts.CommandFile != nil {
		addFlag("-@", *opts.CommandFile)
	}

	if len(opts.MasterQueue) > 0 {
		addFlag("-masterq", strings.Join(opts.MasterQueue, ","))
	}

	if opts.CheckpointInterval != nil {
		addFlag("-ckpt_selector", *opts.CheckpointInterval)
	}

	if opts.NotifyBeforeSuspend != nil && *opts.NotifyBeforeSuspend {
		addFlag("-notify", "")
	}

	if opts.Department != nil {
		addFlag("-dept", *opts.Department)
	}

	if opts.Command != "" {
		args = append(args, opts.Command)
		if len(opts.CommandArgs) > 0 {
			args = append(args, opts.CommandArgs...)
		}
	}

	return args, nil
}

// ConvertTimeToQsubDateTime converts a Go time.Time to the qsub date_time
// format [[CC]YY]MMDDhhmm[.SS]
func ConvertTimeToQsubDateTime(t time.Time) string {
	dateTimeWithoutSec := t.Format("200601021504")
	if t.Second() != 0 {
		return t.Format("200601021504.05")
	}
	return dateTimeWithoutSec
}
