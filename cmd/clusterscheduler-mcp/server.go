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
	"log"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
	"github.com/mark3labs/mcp-go/server"
)

// SchedulerServer represents the MCP server for the Gridware Cluster Scheduler
type SchedulerServer struct {
	server        *server.MCPServer
	conn          *qconf.CommandLineQConf
	clusterConfig *qconf.ClusterConfig
}

type SchedulerServerConfig struct {
	// ReadOnly is true if the server cannot modify the cluster
	// configuration. All tools that modify the cluster configuration
	// will be disabled.
	ReadOnly bool

	// WithJobSubmissionAccess is true if the server can submit jobs
	WithJobSubmissionAccess bool
}

// NewSchedulerServer creates a new instance of the scheduler MCP server
func NewSchedulerServer(config SchedulerServerConfig) (*SchedulerServer, error) {
	s := &SchedulerServer{
		server: server.NewMCPServer(
			"gridware-scheduler-info",
			"0.0.1",
		),
	}

	// background connection to the cluster
	conn, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
		Executable: "qconf",
	})
	if err != nil {
		fmt.Printf("Error: %v", err)
		return nil, err
	}
	s.conn = conn

	// Register all tools
	if err := registerClusterTools(s, config); err != nil {
		return nil, err
	}

	if err := registerJobTools(s, config); err != nil {
		return nil, err
	}

	if err := registerAccountingTools(s, config); err != nil {
		return nil, err
	}

	if err := RegisterPrompts(s); err != nil {
		return nil, err
	}

	return s, nil
}

// Serve starts the MCP server and the SSE server
func (s *SchedulerServer) Serve() error {
	sseServer := server.NewSSEServer(s.server,
		server.WithBaseURL("http://localhost:8888"))
	log.Printf("SSE server listening on :8888")
	if err := sseServer.Start(":8888"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	return nil
}
