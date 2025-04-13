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
	"log"
	"os"
	"strconv"

	//"os"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

// This is a simple MCP server that can be used to answer questions
// about the Gridware Cluster Scheduler. It is mainly for research and
// educational purposes.

// Define a static ClusterConfig for demonstration purposes
var clusterConfig = &qconf.ClusterConfig{}

func main() {

	// if READ_ONLY is set, we run in read-only mode
	readOnly := os.Getenv("READ_ONLY")

	// is true value (use parsebool)
	if readOnly == "" {
		readOnly = "false"
	}

	readOnlyBool, err := strconv.ParseBool(readOnly)
	if err != nil {
		log.Fatalf("Error parsing READ_ONLY environment variable: %v", err)
	}

	server, err := NewSchedulerServer(SchedulerServerConfig{
		ReadOnly: readOnlyBool,
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if err := server.Serve(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
