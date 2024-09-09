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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) {
	configFile := args[0]

	var config qconf.ClusterConfig

	// Read ClusterConfig from specified JSON file
	file, err := os.Open(configFile)
	FatalOnError(err)
	defer file.Close()

	// Decode JSON file into ClusterConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	FatalOnError(err)

	cs, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
		Executable: "qconf",
	})
	FatalOnError(err)

	// Add simulated hosts to /etc/hosts so that they are resolvable
	err = AddHostsToEtcHosts(config)
	FatalOnError(err)
	fmt.Printf("Hosts added to /etc/hosts\n")

	// Add load_report_host complex in current configuration...
	err = addComplex(cs)
	FatalOnError(err)
	fmt.Printf("Complex added to current configuration\n")

	// ...and in new configuration
	addComplexToConfig(&config)

	// Add simulation related qmaster_params and execd_params
	err = addGlobalConfig(cs)
	FatalOnError(err)
	fmt.Printf("Global configuration added to current configuration\n")

	// ...and in new configuration
	addGlobalConfigToConfig(&config)

	var allhosts qconf.HostGroupConfig
	for _, hg := range config.HostGroups {
		if hg.Name == "@allhosts" {
			allhosts = hg
			break
		}
	}
	if allhosts.Name == "" {
		allhosts.Name = "@allhosts"
	}

	// Go through each host and add it to the cluster
	//
	for key := range config.ExecHosts {
		// add "master" host (here in the container) to be load
		// report host for the simulated host
		if config.ExecHosts[key].ComplexValues == nil {
			//
		}
		config.ExecHosts[key].ComplexValues["load_report_host"] = "master"
		// add to @allhosts
		allhosts.Hosts = append(allhosts.Hosts, config.ExecHosts[key].Name)
	}

	// add @allhosts to the list of host groups (TODO should be map)
	if _, ok := config.HostGroups["@allhosts"]; !ok {
		config.HostGroups["@allhosts"] = allhosts
	}

	// append "master" host to the list of hosts as it severs
	// the fake load for all simulated hosts and is not in the
	// list of hosts from the JSON file
	if config.ExecHosts == nil {
		config.ExecHosts = make(map[string]qconf.HostExecConfig)
	}
	config.ExecHosts["master"] = qconf.HostExecConfig{
		Name: "master",
	}
	// add root as operator
	if config.Users == nil {
		config.Users = make(map[string]qconf.UserConfig)
	}
	config.Users["root"] = qconf.UserConfig{
		Name: "root",
	}
	config.Operators = append(config.Operators, "root")
	config.Managers = append(config.Managers, "root")

	// Get the curernt cluster configuration (of the container here)
	currentConfig, err := cs.GetClusterConfiguration()
	FatalOnError(err)

	// Compare the current configuration with the simulated configuration
	comparison, err := currentConfig.CompareTo(config)
	FatalOnError(err)

	// add everything to the cluster which is not already there
	_, err = qconf.AddAllEntries(cs, *comparison.DiffAdded)
	FatalOnError(err)

	// change everything which is different
	_, err = qconf.ModifyAllEntries(cs, *comparison.DiffModified)
	FatalOnError(err)

	// remove everything which is not in the simulated configuration
	_, err = qconf.DeleteAllEnries(cs, *comparison.DiffRemoved, true)
	PrintOnError(err)

	fmt.Printf("Simulated cluster configuration applied\n")

	// Restart the qmaster so that simulated hosts get the load
	// from the "real" host "master".
	RestartQmaster(currentConfig, cs)
}

func RestartQmaster(config qconf.ClusterConfig, cs *qconf.CommandLineQConf) error {
	fmt.Printf("Restarting qmaster\n")
	err := cs.ShutdownMasterDaemon()
	if err != nil {
		return fmt.Errorf("Error shutting down qmaster: %s", err)
	}
	<-time.After(5 * time.Second)
	cs.ShutdownMasterDaemon()
	fmt.Printf("waiting for qmaster to shut down...\n")
	<-time.After(30 * time.Second)
	sgemaster := filepath.Join(config.ClusterEnvironment.Root,
		config.ClusterEnvironment.Cell, "common", "sgemaster")
	cmd := exec.Command(sgemaster, []string{"start"}...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Error restarting qmaster: %s", out.String())
	}
	fmt.Printf(out.String())
	fmt.Printf("qmaster restarted\n")
	return nil
}

func AddHostsToEtcHosts(config qconf.ClusterConfig) error {
	hostsFile, err := os.OpenFile("/etc/hosts", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer hostsFile.Close()

	// Presuming starting at private address 10.0.0.1
	baseIP := [4]int{10, 0, 0, 0}

	// Add simulated hosts to /etc/hosts so that they are resolvable
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
			// Increment the second octet and reset the third and fourth octet if we
			// reach the limit.
			baseIP[1]++
			baseIP[2] = 0
		}
		i++
	}
	return nil
}

func addComplex(cs *qconf.CommandLineQConf) error {
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

func addComplexToConfig(config *qconf.ClusterConfig) {
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

func addGlobalConfig(cs *qconf.CommandLineQConf) error {
	global, err := cs.ShowGlobalConfiguration()
	if err != nil {
		return err
	}
	global.QmasterParams = append(global.QmasterParams, "SIMULATE_EXECDS=TRUE")
	global.ExecdParams = append(global.ExecdParams, "SIMULATE_JOBS=TRUE")
	return cs.ModifyGlobalConfig(*global)
}

func addGlobalConfigToConfig(config *qconf.ClusterConfig) {
	config.GlobalConfig.QmasterParams = append(config.GlobalConfig.QmasterParams,
		"SIMULATE_EXECDS=TRUE")
	config.GlobalConfig.ExecdParams = append(config.GlobalConfig.ExecdParams,
		"SIMULATE_JOBS=TRUE")
}

func addExecHost(cs *qconf.CommandLineQConf, host qconf.HostExecConfig) error {
	if host.ComplexValues == nil {
		host.ComplexValues = make(map[string]string)
	}
	host.ComplexValues["load_report_host"] = "master"
	return cs.AddExecHost(host)
}
