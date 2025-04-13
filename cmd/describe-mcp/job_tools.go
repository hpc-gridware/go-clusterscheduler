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

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerJobTools registers all job submission and status related tools
func registerJobTools(s *SchedulerServer, config SchedulerServerConfig) error {
	// Add qsub_help tool
	if !config.ReadOnly {
		s.server.AddTool(mcp.NewTool(
			"qsub_help",
			mcp.WithDescription("Retrieves the complete help documentation for the qsub command, showing all available options, parameters, and usage examples. This provides comprehensive information about job submission syntax, resource requests, environment settings, and other relevant flags that can be used when submitting jobs to the Gridware Cluster Scheduler."),
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

	if !config.ReadOnly {
		// Add submit_job tool
		s.server.AddTool(mcp.NewTool(
			"submit_job",
			mcp.WithDescription("Submits a job to the Gridware Cluster Scheduler using SGE-compatible command line parameters. This tool allows for the direct submission of jobs with full control over resource requests, scheduling parameters, and execution environment. For a complete list of available command line parameters, use the qsub_help tool. For binary jobs use -b y. When submitting multiple same jobs use job array syntax (e.g., -t 1-10). Jobs scripts must exist on the server."),
			mcp.WithString("command",
				mcp.Description("The command or script to execute as a job."),
				mcp.Required(),
			),
			mcp.WithArray("arguments",
				mcp.Description("SGE-compatible command line arguments for job submission (e.g., -q queue_name, -l resource=value)."),
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

	// Add qstat tool
	s.server.AddTool(mcp.NewTool(
		"qstat",
		mcp.WithDescription("Retrieves information about jobs and queues in the Gridware Cluster Scheduler. This tool allows viewing job status, queue information, and other scheduling details. Use with various options like '-j job_id' to see specific job details or '-help' to see all available options."),
		mcp.WithArray("arguments",
			mcp.Description("Command line arguments for qstat (e.g., '-j 123' for job information, '-f' for full output, '-help' for help documentation)."),
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

// getJobStatus executes qstat with the given arguments
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
