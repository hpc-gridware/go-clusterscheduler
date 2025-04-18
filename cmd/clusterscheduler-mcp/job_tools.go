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

package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	qalter "github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/v9.0"
	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
	"github.com/mark3labs/mcp-go/mcp"
)

// QSubHelpDescription is the description for the qsub_help tool
const QSubHelpDescription = `Retrieves the complete help documentation for the qsub command, showing all available options, 
parameters, and usage examples. This provides comprehensive information about job submission syntax, 
resource requests, environment settings, and other relevant flags that can be used when submitting jobs 
to the Gridware Cluster Scheduler.`

// SubmitJobDescription is the description for the submit_job tool
const SubmitJobDescription = `Submits a job to the Gridware Cluster Scheduler using SGE-compatible command line parameters. 
This tool allows for the direct submission of jobs with full control over resource requests, scheduling parameters, 
and execution environment. For a complete list of available command line parameters, use the qsub_help tool. 
For binary jobs use -b y. When submitting multiple same jobs use job array syntax (e.g., -t 1-10). 
Jobs scripts must exist on the server.`

// SubmitJobCommandDescription is the description for the submit_job tool command parameter
const SubmitJobCommandDescription = `The command or script to execute as a job.`

// SubmitJobArgumentsDescription is the description for the submit_job tool arguments parameter
const SubmitJobArgumentsDescription = `SGE-compatible command line arguments for job submission 
(e.g., -q queue_name, -l resource=value).`

// QStatDescription is the description for the qstat tool
const QStatDescription = `Retrieves information about jobs and queues in the Gridware Cluster Scheduler. 
This tool allows viewing job status, queue information, and other scheduling details. 
Use with various options like '-j job_id' to see specific job details or '-help' to see all available options.`

// QStatArgumentsDescription is the description for the qstat tool arguments parameter
const QStatArgumentsDescription = `Command line arguments for qstat (e.g., '-j 123' for job information, 
'-f' for full output, '-help' for help documentation).`

// registerJobTools registers all job submission and status related tools
func registerJobTools(s *SchedulerServer, config SchedulerServerConfig) error {
	// Add qsub_help tool
	if config.WithJobSubmissionAccess {
		s.server.AddTool(mcp.NewTool(
			"qsub_help",
			mcp.WithDescription(QSubHelpDescription),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			log.Printf("Getting qsub help information")

			output, err := getQSubHelp(ctx)
			if err != nil {
				log.Printf("Failed to get qsub help: %v", err)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to retrieve qsub help information: %v", err),
						},
					},
					IsError: true,
				}, nil
			}

			log.Printf("Successfully retrieved qsub help")

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: output,
					},
				},
			}, nil
		})
	}

	if config.WithJobSubmissionAccess {
		// Add submit_job tool
		s.server.AddTool(mcp.NewTool(
			"submit_job",
			mcp.WithDescription(SubmitJobDescription),
			mcp.WithString("command",
				mcp.Description(SubmitJobCommandDescription),
				mcp.Required(),
			),
			mcp.WithArray("arguments",
				mcp.Description(SubmitJobArgumentsDescription),
			),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			log.Printf("Submitting job")

			// Get command from the request
			command, ok := req.Params.Arguments["command"].(string)
			if !ok || len(strings.TrimSpace(command)) == 0 {
				log.Printf("Invalid or missing command")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: Job command is required and must be a non-empty string.",
						},
					},
					IsError: true,
				}, nil
			}

			// Get optional arguments
			var arguments []string
			if args, ok := req.Params.Arguments["arguments"].([]interface{}); ok {
				for _, arg := range args {
					if strArg, ok := arg.(string); ok {
						arguments = append(arguments, strArg)
					}
				}
			}

			// Prepare submission arguments
			submitArgs := append(arguments, command)

			// Submit the job
			output, err := submitJob(ctx, submitArgs)
			if err != nil {
				log.Printf("Failed to submit job: %v", err)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to submit job: %v", err),
						},
					},
					IsError: true,
				}, nil
			}

			log.Printf("Successfully submitted job: %s", output)

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Job submitted successfully.\n\nSubmission response:\n%s", output),
					},
				},
			}, nil
		})
	}

	// Add raw qstat tool
	s.server.AddTool(mcp.NewTool(
		"qstat",
		mcp.WithDescription(QStatDescription),
		mcp.WithArray("arguments",
			mcp.Description(QStatArgumentsDescription),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Executing qstat command")

		// Get optional arguments
		var arguments []string
		if args, ok := req.Params.Arguments["arguments"].([]interface{}); ok {
			for _, arg := range args {
				if strArg, ok := arg.(string); ok {
					arguments = append(arguments, strArg)
				}
			}
		}

		// Execute qstat command
		output, err := getJobStatus(ctx, arguments)
		if err != nil {
			log.Printf("Failed to execute qstat: %v", err)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to execute qstat command: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		log.Printf("Successfully executed qstat command")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: output,
				},
			},
		}, nil
	})

	// Add diagnose_pending_job tool: "Why is my job waiting?"
	s.server.AddTool(mcp.NewTool(
		"diagnose_pending_job",
		mcp.WithDescription("Diagnoses why a job is pending. For this, it will execute qstat and qalter -w p commands."),
		mcp.WithString("job_id",
			mcp.Description("Job ID"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		jobID, ok := req.Params.Arguments["job_id"].(string)
		if !ok {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: Job ID is required and must be a non-empty string.",
					},
				},
				IsError: true,
			}, nil
		}

		if jobID == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: Job ID is required and must be a non-empty string.",
					},
				},
				IsError: true,
			}, nil
		}

		output, err := diagnosePendingJob(ctx, jobID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to diagnose pending job: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: output,
				},
			},
		}, nil
	})

	return nil
}

// Helper functions for job submission and status

// getQSubHelp retrieves the help output from qsub
func getQSubHelp(ctx context.Context) (string, error) {
	qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qsub command line tool: %v", err)
	}

	output, err := qs.SubmitWithNativeSpecification(ctx,
		[]string{"-help"})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to execute qsub help: %v", err)
	}

	return output, nil
}

// getJobStatus retrieves the status of a job using qstat
func getJobStatus(ctx context.Context, args []string) (string, error) {
	q, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qstat command line tool: %v", err)
	}

	output, err := q.NativeSpecification(args)
	if err != nil {
		return "", fmt.Errorf("internal error: failed to execute qstat command: %v", err)
	}

	return output, nil
}

// submitJob submits a job using qsub
func submitJob(ctx context.Context, args []string) (string, error) {
	qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qsub command line tool: %v", err)
	}

	output, err := qs.SubmitWithNativeSpecification(ctx, args)
	if err != nil {
		return "", fmt.Errorf("internal error: failed to submit job: %v", err)
	}

	return output, nil
}

// diagnosePendingJob collects all information about a pending job which
// might help to diagnose why it is pending.
func diagnosePendingJob(ctx context.Context, jobID string) (string, error) {
	q, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qstat command line tool: %v", err)
	}

	// get detailed job status
	jobStatus, err := q.NativeSpecification([]string{"-j", jobID})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to execute qstat command: %v", err)
	}

	qalterCmd, err := qalter.NewCommandLineQAlter(
		qalter.CommandLineQAlterConfig{})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qalter command line tool: %v", err)
	}

	// -w p
	withPeak, err := qalterCmd.NativeSpecification([]string{"-w", "p", jobID})
	if err != nil {
		jobStatus += fmt.Sprintf("Failed to execute qalter command: %v", err)
	}

	return jobStatus + "\n\nqalter -w p " + jobID + "\n\n" + withPeak, nil
}
