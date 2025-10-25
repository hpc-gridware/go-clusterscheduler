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
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate cluster configuration",
	Long:  "Simulate cluster configuration using a JSON cluster configuration file.",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the cluster simulation",
	Long:  "Run the cluster simulation using the provided JSON file.",
	Args:  cobra.MinimumNArgs(1),
	Run:   run,
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump the cluster configuration",
	Long:  "Dump the cluster configuration into JSON format.",
	Args:  cobra.MinimumNArgs(0),
	Run:   dump,
}

func main() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(dumpCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
