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
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
	"github.com/mark3labs/mcp-go/mcp"
)

// QAcctToolDescription is the description for the qacct tool
const QAcctToolDescription = `Retrieves accounting information about finished jobs in the Gridware Cluster Scheduler. 
This tool allows querying job history, resource usage, and execution details for completed jobs. 
Use with various options or specify job IDs to get detailed information about specific jobs.
Assuming that the timestamps is in microseconds (1/1,000,000 second).`

// QAcctArgumentsDescription is the description for the qacct tool arguments
const QAcctArgumentsDescription = `Command line arguments for qacct (e.g., '-j 123' for job information, 
'-u username' for user jobs, '-help' for help documentation).`

// JobDetailsToolDescription is the description for the job_details tool
const JobDetailsToolDescription = `Retrieves detailed accounting information about finished jobs in a structured format. 
This tool returns comprehensive data about job execution, including resource usage, submission parameters, 
and execution timelines. Specify job IDs to get information about specific jobs or leave empty to get 
details for all finished jobs. Assuming that the timestamps is in microseconds (1/1,000,000 second).`

// JobDetailsJobIdsDescription is the description for the job_details tool job_ids parameter
const JobDetailsJobIdsDescription = `List of job IDs to retrieve details for. If omitted, details for all 
finished jobs will be returned.`

// registerAccountingTools registers all job accounting related tools
func registerAccountingTools(s *SchedulerServer, config SchedulerServerConfig) error {
	// Add qacct tool
	s.server.AddTool(mcp.NewTool(
		"qacct",
		mcp.WithDescription(QAcctToolDescription),
		mcp.WithArray("arguments",
			mcp.Description(QAcctArgumentsDescription),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Executing qacct command")

		// Get optional arguments
		var arguments []string
		if args, ok := req.Params.Arguments["arguments"].([]interface{}); ok {
			for _, arg := range args {
				if strArg, ok := arg.(string); ok {
					arguments = append(arguments, strArg)
				}
			}
		}

		// Execute qacct command
		output, err := getAccountingInfo(ctx, arguments)
		if err != nil {
			log.Printf("Failed to execute qacct: %v", err)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to execute qacct command: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		log.Printf("Successfully executed qacct command")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: output,
				},
			},
		}, nil
	})

	// Add job_details tool
	s.server.AddTool(mcp.NewTool(
		"job_details",
		mcp.WithDescription(JobDetailsToolDescription),
		mcp.WithArray("job_ids",
			mcp.Description(JobDetailsJobIdsDescription),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Retrieving job details")

		// Parse job IDs if provided
		var jobIDs []int64
		if ids, ok := req.Params.Arguments["job_ids"].([]interface{}); ok && len(ids) > 0 {
			for _, id := range ids {
				// Handle different possible formats (string, number)
				switch v := id.(type) {
				case string:
					if jobID, err := strconv.ParseInt(v, 10, 64); err == nil {
						jobIDs = append(jobIDs, jobID)
					} else {
						log.Printf("Invalid job ID format: %s", v)
					}
				case float64:
					jobIDs = append(jobIDs, int64(v))
				}
			}
		}

		// Get job details
		output, err := getStructuredJobDetails(ctx, jobIDs)
		if err != nil {
			log.Printf("Failed to get job details: %v", err)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to retrieve job details: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		log.Printf("Successfully retrieved job details")

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

// Helper functions for job accounting

// getAccountingInfo executes qacct with the given arguments
func getAccountingInfo(ctx context.Context, args []string) (string, error) {
	qa, err := qacct.NewCommandLineQAcct(qacct.CommandLineQAcctConfig{
		Executable: "qacct",
	})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qacct command line tool: %v", err)
	}

	output, err := qa.NativeSpecification(args)
	if err != nil {
		return "", fmt.Errorf("internal error: failed to execute qacct command: %v", err)
	}

	return output, nil
}

// getStructuredJobDetails retrieves job details using qacct
func getStructuredJobDetails(ctx context.Context, jobIDs []int64) (string, error) {
	qa, err := qacct.NewCommandLineQAcct(qacct.CommandLineQAcctConfig{
		Executable: "qacct",
	})
	if err != nil {
		return "", fmt.Errorf("internal error: failed to initialize qacct command line tool: %v", err)
	}

	jobDetails, err := qa.ShowJobDetails(jobIDs)
	if err != nil {
		return "", fmt.Errorf("internal error: failed to show job details: %v", err)
	}

	// Convert job details to JSON for structured output
	data, err := json.MarshalIndent(jobDetails, "", "  ")
	if err != nil {
		return "", fmt.Errorf("internal error: failed to format job details: %v", err)
	}

	return string(data), nil
}
