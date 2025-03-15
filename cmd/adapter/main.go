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
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/adapter"
	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
	"github.com/spf13/cobra"
)

var (
	port int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "adapter",
		Short: "REST adapter for Open Cluster Scheduler",
		Long:  "REST adapter for Open Cluster Scheduler providing an HTTP API to interact with the scheduler.",
		Run:   runAdapter,
	}

	// Add port flag with default value
	rootCmd.Flags().IntVarP(&port, "port", "p", 8282, "Port to run the adapter on")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runAdapter(cmd *cobra.Command, args []string) {
	qc, err := qconf.NewCommandLineQConf(
		qconf.CommandLineQConfConfig{Executable: "qconf", DryRun: false})
	if err != nil {
		log.Fatalf("Error creating qconf: %v", err)
	}

	router := mux.NewRouter()
	router.Handle("/api/v0/command", adapter.NewAdapter(qc)).Methods("POST")

	// Use the port from command line flag
	serverAddress := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on port %d", port)
	if err := http.ListenAndServe(serverAddress, router); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
