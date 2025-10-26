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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

// FatalOnError prints the error and exits with status 1 if error is not nil
func FatalOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// PrintOnError prints the error if it is not nil
func PrintOnError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// prettyPrint formats and prints a value as indented JSON
func prettyPrint(v interface{}) {
	js, err := json.MarshalIndent(v, "", "  ")
	FatalOnError(err)
	fmt.Println(string(js))
}

// prepareSimulationConfig prepares the cluster configuration for simulation
// by adding necessary components like load_report_host complex and simulation params
func prepareSimulationConfig(cs *qconf.CommandLineQConf, config *qconf.ClusterConfig) error {
	// Add load_report_host complex to current configuration
	if err := addLoadReportHostComplex(cs); err != nil {
		return fmt.Errorf("failed to add complex: %w", err)
	}
	fmt.Println("Complex added to current configuration")

	// Add load_report_host complex to new configuration
	addLoadReportHostComplexToConfig(config)

	// Add simulation parameters to current configuration
	if err := addSimulationParams(cs); err != nil {
		return fmt.Errorf("failed to add global config: %w", err)
	}
	fmt.Println("Global configuration added to current configuration")

	// Add simulation parameters to new configuration
	addSimulationParamsToConfig(config)

	// Prepare host groups and add load_report_host to all hosts
	prepareHostConfiguration(config)

	return nil
}

// prepareHostConfiguration sets up host groups and configures load reporting
func prepareHostConfiguration(config *qconf.ClusterConfig) {
	allhosts := config.HostGroups["@allhosts"]
	if allhosts.Name == "" {
		allhosts.Name = "@allhosts"
	}

	// Configure each host with load_report_host
	for k, v := range config.ExecHosts {
		if v.ComplexValues == nil {
			v.ComplexValues = make(map[string]string)
		}
		v.ComplexValues["load_report_host"] = "master"
		config.ExecHosts[k] = v
		// Add to @allhosts
		allhosts.Hosts = append(allhosts.Hosts, v.Name)
	}

	config.HostGroups["@allhosts"] = allhosts
}

// ensureDefaultEntities ensures that master host and root user exist in the config
func ensureDefaultEntities(config *qconf.ClusterConfig) {
	// Ensure master host exists
	if config.ExecHosts == nil {
		config.ExecHosts = make(map[string]qconf.HostExecConfig)
	}
	if _, exists := config.ExecHosts["master"]; !exists {
		config.ExecHosts["master"] = qconf.HostExecConfig{
			Name: "master",
		}
	}

	// Ensure root user exists
	if config.Users == nil {
		config.Users = make(map[string]qconf.UserConfig)
	}
	if _, exists := config.Users["root"]; !exists {
		config.Users["root"] = qconf.UserConfig{
			Name: "root",
		}
	}

	// Ensure root is in operators list
	rootInOperators := false
	for _, op := range config.Operators {
		if op == "root" {
			rootInOperators = true
			break
		}
	}
	if !rootInOperators {
		config.Operators = append(config.Operators, "root")
	}

	// Ensure root is in managers list
	rootInManagers := false
	for _, mgr := range config.Managers {
		if mgr == "root" {
			rootInManagers = true
			break
		}
	}
	if !rootInManagers {
		config.Managers = append(config.Managers, "root")
	}
}

// addHostsToEtcHosts adds simulated hosts to /etc/hosts for name resolution
func addHostsToEtcHosts(config qconf.ClusterConfig) error {
	hostsFile, err := os.OpenFile("/etc/hosts", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer hostsFile.Close()

	// Start at private address 10.0.0.0
	baseIP := [4]int{10, 0, 0, 0}

	// Add each simulated host to /etc/hosts
	i := 0
	for _, host := range config.ExecHosts {
		thirdOctet := i / 256
		fourthOctet := i % 256
		ip := fmt.Sprintf("10.%d.%d.%d", baseIP[1], baseIP[2]+thirdOctet, fourthOctet)
		line := fmt.Sprintf("%s %s\n", ip, host.Name)
		_, err := hostsFile.WriteString(line)
		if err != nil {
			return err
		}
		if baseIP[2]+thirdOctet == 255 && fourthOctet == 255 {
			// Increment the second octet and reset the third and fourth octet
			baseIP[1]++
			baseIP[2] = 0
		}
		i++
	}
	return nil
}

// addLoadReportHostComplex adds the load_report_host complex to the cluster
func addLoadReportHostComplex(cs *qconf.CommandLineQConf) error {
	complexes, err := cs.ShowAllComplexes()
	if err != nil {
		return err
	}

	for _, complex := range complexes {
		if complex.Name == "load_report_host" {
			fmt.Println("Complex already exists")
			return nil
		}
	}

	return cs.AddComplexEntry(qconf.ComplexEntryConfig{
		Name:        "load_report_host",
		Shortcut:    "lrh",
		Type:        "STRING",
		Relop:       "==",
		Requestable: "YES",
		Consumable:  "NO",
		Default:     "NONE",
		Urgency:     0,
	})
}

// addLoadReportHostComplexToConfig adds the load_report_host complex to the config
func addLoadReportHostComplexToConfig(config *qconf.ClusterConfig) {
	for _, complex := range config.ComplexEntries {
		if complex.Name == "load_report_host" {
			return
		}
	}
	config.ComplexEntries["load_report_host"] = qconf.ComplexEntryConfig{
		Name:        "load_report_host",
		Shortcut:    "lrh",
		Type:        "STRING",
		Relop:       "==",
		Requestable: "YES",
		Consumable:  "NO",
		Default:     "NONE",
		Urgency:     0,
	}
}

// addSimulationParams adds simulation parameters to the global configuration
func addSimulationParams(cs *qconf.CommandLineQConf) error {
	global, err := cs.ShowGlobalConfiguration()
	if err != nil {
		return err
	}
	global.QmasterParams = append(global.QmasterParams, "SIMULATE_EXECDS=TRUE")
	global.ExecdParams = append(global.ExecdParams, "SIMULATE_JOBS=TRUE")
	return cs.ModifyGlobalConfig(*global)
}

// addSimulationParamsToConfig adds simulation parameters to the config
func addSimulationParamsToConfig(config *qconf.ClusterConfig) {
	config.GlobalConfig.QmasterParams = append(config.GlobalConfig.QmasterParams,
		"SIMULATE_EXECDS=TRUE")
	config.GlobalConfig.ExecdParams = append(config.GlobalConfig.ExecdParams,
		"SIMULATE_JOBS=TRUE")
}

// restartQmaster restarts the qmaster daemon
func restartQmaster(config qconf.ClusterConfig, cs *qconf.CommandLineQConf) error {
	fmt.Println("Restarting qmaster")
	err := cs.ShutdownMasterDaemon()
	if err != nil {
		return fmt.Errorf("failed to shut down qmaster: %w", err)
	}
	<-time.After(5 * time.Second)
	cs.ShutdownMasterDaemon()
	fmt.Println("waiting for qmaster to shut down...")
	<-time.After(30 * time.Second)
	sgemaster := filepath.Join(config.ClusterEnvironment.Root,
		config.ClusterEnvironment.Cell, "common", "sgemaster")
	cmd := exec.Command(sgemaster, []string{"start"}...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to restart qmaster: %s", out.String())
	}
	fmt.Print(out.String())
	fmt.Println("qmaster restarted")
	return nil
}
