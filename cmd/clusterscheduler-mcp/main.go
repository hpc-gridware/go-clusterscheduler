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
// about the Open Cluster Scheduler. It can be used for research and
// educational purposes. It allows to dive deeper into the cluster
// configuration and understanding the current state of the cluster.

// Define a static ClusterConfig for demonstration purposes
var clusterConfig = &qconf.ClusterConfig{}

func main() {

	// if WITH_WRITE_ACCESS is set, the cluster configuration can be modified
	// which is dangerous, but useful research purposes
	withWriteAccessBool, _ := strconv.ParseBool(
		os.Getenv("WITH_WRITE_ACCESS"))

	// if WITH_JOB_SUBMISSION_ACCESS is set, jobs can be submitted
	withJobSubmissionAccessBool, _ := strconv.ParseBool(
		os.Getenv("WITH_JOB_SUBMISSION_ACCESS"))

	server, err := NewSchedulerServer(SchedulerServerConfig{
		ReadOnly:                !withWriteAccessBool,
		WithJobSubmissionAccess: withJobSubmissionAccessBool,
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if err := server.Serve(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
