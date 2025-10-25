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
	"encoding/json"
	"fmt"
	"os"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) {
	configFile := args[0]

	// Read cluster configuration from JSON file
	config, err := readClusterConfig(configFile)
	FatalOnError(err)

	// Initialize qconf client
	cs, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
		Executable: "qconf",
	})
	FatalOnError(err)

	// Add simulated hosts to /etc/hosts for name resolution
	err = addHostsToEtcHosts(config)
	FatalOnError(err)
	fmt.Println("Hosts added to /etc/hosts")

	// Prepare simulation configuration (add complex, params, etc.)
	err = prepareSimulationConfig(cs, &config)
	FatalOnError(err)

	// Get current cluster configuration
	currentConfig, err := cs.GetClusterConfiguration()
	FatalOnError(err)

	// Ensure default entities (master host, root user) exist
	ensureDefaultEntities(&config)

	// Clean up invalid PE references in the configuration
	cleanupInvalidPEReferences(&config)

	// Compare configurations to determine changes needed
	comparison, err := currentConfig.CompareTo(config)
	FatalOnError(err)

	// Apply changes: Add, Modify, then Delete
	applyConfigurationChanges(cs, currentConfig, comparison)

	fmt.Println("Simulated cluster configuration applied")

	// Restart qmaster to activate simulation
	restartQmaster(currentConfig, cs)
}

// readClusterConfig reads and parses a cluster configuration from a JSON file
func readClusterConfig(configFile string) (qconf.ClusterConfig, error) {
	var config qconf.ClusterConfig

	file, err := os.Open(configFile)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return config, err
}

// applyConfigurationChanges applies the configuration differences to the cluster
func applyConfigurationChanges(cs *qconf.CommandLineQConf,
	currentConfig qconf.ClusterConfig, comparison *qconf.ClusterConfigComparison) {

	// Add new entries
	if comparison.DiffAdded != nil {
		_, err := qconf.AddAllEntries(cs, *comparison.DiffAdded)
		FatalOnError(err)
	}

	// Modify existing entries
	if comparison.DiffModified != nil {
		_, err := qconf.ModifyAllEntries(cs, *comparison.DiffModified)
		FatalOnError(err)
	}

	// Handle deletions with dependency cleanup
	if comparison.DiffRemoved != nil {
		// Before deleting PEs, remove references from queues
		if len(comparison.DiffRemoved.ParallelEnvironments) > 0 {
			err := removeQueuePEReferences(cs, currentConfig,
				comparison.DiffRemoved.ParallelEnvironments)
			PrintOnError(err)
		}

		// Delete removed entries
		_, err := qconf.DeleteAllEnries(cs, *comparison.DiffRemoved, true)
		PrintOnError(err)
	}
}
