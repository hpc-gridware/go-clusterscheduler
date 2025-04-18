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
	"fmt"
	"strings"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

// Helper functions for cluster configuration

// isEmpty checks if the cluster configuration is empty
func isEmpty(config *qconf.ClusterConfig) bool {
	if config == nil {
		return true
	}

	return false
}

// generateConfigSummary creates a human-readable summary of the cluster configuration
func generateConfigSummary(config *qconf.ClusterConfig) string {
	if config == nil {
		return "Empty configuration"
	}

	var summary strings.Builder

	// Cluster environment
	if config.ClusterEnvironment != nil {
		summary.WriteString(fmt.Sprintf("Cluster Name: %s\n", config.ClusterEnvironment.Name))
		summary.WriteString(fmt.Sprintf("Cell: %s\n", config.ClusterEnvironment.Cell))
		summary.WriteString(fmt.Sprintf("Version: %s\n", config.ClusterEnvironment.Version))
	}

	// Count of various entities
	hostGroupCount := 0
	if config.HostGroups != nil {
		hostGroupCount = len(config.HostGroups)
	}

	queueCount := 0
	if config.ClusterQueues != nil {
		queueCount = len(config.ClusterQueues)
	}

	execHostCount := 0
	if config.ExecHosts != nil {
		execHostCount = len(config.ExecHosts)
	}

	projectCount := 0
	if config.Projects != nil {
		projectCount = len(config.Projects)
	}

	userCount := 0
	if config.Users != nil {
		userCount = len(config.Users)
	}

	peCount := 0
	if config.ParallelEnvironments != nil {
		peCount = len(config.ParallelEnvironments)
	}

	summary.WriteString("\nConfiguration contains:\n")
	summary.WriteString(fmt.Sprintf("- %d host groups\n", hostGroupCount))
	summary.WriteString(fmt.Sprintf("- %d execution hosts\n", execHostCount))
	summary.WriteString(fmt.Sprintf("- %d cluster queues\n", queueCount))
	summary.WriteString(fmt.Sprintf("- %d projects\n", projectCount))
	summary.WriteString(fmt.Sprintf("- %d users\n", userCount))
	summary.WriteString(fmt.Sprintf("- %d parallel environments\n", peCount))

	if len(config.AdminHosts) > 0 {
		summary.WriteString(fmt.Sprintf("- %d admin hosts\n", len(config.AdminHosts)))
	}

	if len(config.SubmitHosts) > 0 {
		summary.WriteString(fmt.Sprintf("- %d submit hosts\n", len(config.SubmitHosts)))
	}

	return summary.String()
}

// validateConfiguration performs basic validation of the cluster configuration
func validateConfiguration(config *qconf.ClusterConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Additional validation can be added as needed

	return nil
}
