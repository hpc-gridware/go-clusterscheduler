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

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

// cleanupInvalidPEReferences removes references to parallel environments
// from queues when those PEs are not defined in the configuration.
// This prevents errors when comparing configurations.
func cleanupInvalidPEReferences(config *qconf.ClusterConfig) {
	// Build a set of valid PE names
	validPEs := make(map[string]bool)
	for peName := range config.ParallelEnvironments {
		validPEs[peName] = true
	}

	// Check each queue and remove invalid PE references
	for queueName, queue := range config.ClusterQueues {
		if len(queue.PeList) == 0 {
			continue
		}

		// Filter out invalid PE names
		validPEList := make([]string, 0)
		for _, peRef := range queue.PeList {
			// PE references can be in format "pename" or "[host=pename]"
			// Extract the PE name
			peName := peRef
			if len(peRef) > 0 && peRef[0] == '[' {
				// Skip host-specific PE references as they're harder to parse
				// and we're being conservative here
				continue
			}

			// Only keep if PE is defined
			if validPEs[peName] {
				validPEList = append(validPEList, peRef)
			} else {
				fmt.Printf("Warning: Removing reference to undefined PE '%s' from queue '%s'\n",
					peName, queueName)
			}
		}

		// Update the queue's PE list
		if len(validPEList) == 0 {
			queue.PeList = nil
		} else {
			queue.PeList = validPEList
		}
		config.ClusterQueues[queueName] = queue
	}
}

// removeQueuePEReferences modifies queues in the cluster to remove references
// to parallel environments that are about to be deleted.
func removeQueuePEReferences(cs *qconf.CommandLineQConf, currentConfig qconf.ClusterConfig,
	pesToDelete map[string]qconf.ParallelEnvironmentConfig) error {

	// Build a set of PE names to delete
	peNamesToDelete := make(map[string]bool)
	for peName := range pesToDelete {
		peNamesToDelete[peName] = true
	}

	// Check each queue in the current configuration
	for queueName, queue := range currentConfig.ClusterQueues {
		if len(queue.PeList) == 0 {
			continue
		}

		modified := false
		newPEList := make([]string, 0)

		for _, peRef := range queue.PeList {
			// PE references can be in format "pename" or "[host=pename]"
			peName := peRef
			if len(peRef) > 0 && peRef[0] == '[' {
				// For host-specific references, skip for simplicity
				// Most configs won't use this
				newPEList = append(newPEList, peRef)
				continue
			}

			// Keep PE reference only if it's not being deleted
			if !peNamesToDelete[peName] {
				newPEList = append(newPEList, peRef)
			} else {
				fmt.Printf("Removing PE reference '%s' from queue '%s' before deletion\n",
					peName, queueName)
				modified = true
			}
		}

		// If we modified the queue, update it in the cluster
		if modified {
			queue.PeList = newPEList
			if len(newPEList) == 0 {
				queue.PeList = nil
			}
			err := cs.ModifyClusterQueue(queueName, queue)
			if err != nil {
				return fmt.Errorf("failed to modify queue %s to remove PE references: %w",
					queueName, err)
			}
		}
	}

	return nil
}
