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
	"strings"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetClusterConfigurationDescription is the description for the get_cluster_configuration tool
const GetClusterConfigurationDescription = `Fetch the complete cluster configuration of the Gridware Cluster Scheduler. 
This tool retrieves all configuration data including hosts, queues, users, projects, and resource settings, 
formatted as JSON. Use this method for efficient retrieval of the entire configuration state.`

// SetClusterConfigurationDescription is the description for the set_cluster_configuration tool
const SetClusterConfigurationDescription = `Apply a complete cluster configuration to the Gridware Cluster Scheduler. 
This tool accepts a JSON representation of the entire cluster configuration and applies it to the system. 
Use this for bulk updates, migrations, or restoring configuration from a backup. 
The configuration must be properly formatted JSON matching the ClusterConfig structure.`

// SetClusterConfigurationParamDescription is the description for the set_cluster_configuration tool parameter
const SetClusterConfigurationParamDescription = `Complete JSON representation of the cluster configuration to set. 
Must be a valid JSON string matching the ClusterConfig structure. 
A valid JSON string can be generated using the get_cluster_configuration tool.`

// registerClusterTools registers all cluster configuration related tools
func registerClusterTools(s *SchedulerServer, config SchedulerServerConfig) error {
	// Add get_cluster_configuration tool
	s.server.AddTool(mcp.NewTool(
		"get_cluster_configuration",
		mcp.WithDescription(GetClusterConfigurationDescription),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Getting cluster configuration")
		clusterConfig, err := s.conn.GetClusterConfiguration()
		if err != nil {
			log.Printf("Failed to get cluster configuration: %v", err)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to retrieve cluster configuration: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		s.clusterConfig = &clusterConfig
		log.Printf("Successfully retrieved cluster configuration")

		// If configuration is empty, return informative message
		if isEmpty(s.clusterConfig) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Cluster configuration was retrieved successfully but appears to be empty. This may indicate a new or unconfigured cluster.",
					},
				},
			}, nil
		}

		// Format and return the configuration
		data, err := json.MarshalIndent(s.clusterConfig, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal cluster configuration: %v", err)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Retrieved configuration but failed to format it: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Add a summary of the configuration
		summary := generateConfigSummary(s.clusterConfig)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Configuration Summary:\n%s\n\nFull Configuration:\n%s", summary, string(data)),
				},
			},
		}, nil
	})

	if !config.ReadOnly {

		// Add set_cluster_configuration tool
		s.server.AddTool(mcp.NewTool(
			"set_cluster_configuration",
			mcp.WithDescription(SetClusterConfigurationDescription),
			mcp.WithString("cluster_configuration",
				mcp.Description(SetClusterConfigurationParamDescription),
				mcp.Required(),
			),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			log.Printf("Setting cluster configuration")

			// Get the configuration string from the request
			clusterConfigStr := req.GetString("cluster_configuration", "")
			if clusterConfigStr == "" {
				log.Printf("Invalid input: configuration must be provided as a string")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Invalid input: Cluster configuration must be provided as a valid JSON string.",
						},
					},
					IsError: true,
				}, nil
			}

			// Validate the input is not empty
			if len(strings.TrimSpace(clusterConfigStr)) == 0 {
				log.Printf("Empty configuration provided")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: Empty configuration provided. Please provide a valid JSON cluster configuration.",
						},
					},
					IsError: true,
				}, nil
			}

			// Parse the configuration
			var config qconf.ClusterConfig
			err := json.Unmarshal([]byte(clusterConfigStr), &config)
			if err != nil {
				log.Printf("Failed to parse configuration JSON: %v", err)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid configuration format: %v\n\nPlease provide a valid JSON string that matches the ClusterConfig structure.", err),
						},
					},
					IsError: true,
				}, nil
			}
			log.Printf("Parsed configuration: %v", config)

			// Validate configuration (basic structural validation)
			if validationErr := validateConfiguration(&config); validationErr != nil {
				log.Printf("Configuration validation failed: %v", validationErr)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Configuration validation failed: %v", validationErr),
						},
					},
					IsError: true,
				}, nil
			}

			// Apply the configuration
			log.Printf("Applying cluster configuration")
			err = s.conn.ApplyClusterConfiguration(config)
			if err != nil {
				log.Printf("Failed to apply configuration: %v", err)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to apply the configuration: %v", err),
						},
					},
					IsError: true,
				}, nil
			}
			log.Printf("Successfully applied configuration")

			// Update local cache
			s.clusterConfig = &config

			// Generate a summary of changes applied
			summary := generateConfigSummary(&config)

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Configuration successfully applied.\n\nSummary of applied configuration:\n%s", summary),
					},
				},
			}, nil
		})
	}

	return nil
}
