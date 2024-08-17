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

package cs

import (
	"fmt"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
)

// ClusterScheduler defines the methods that a cluster scheduler connection should implement.
type ClusterScheduler interface {
	QConf() (qconf.QConf, error)
	// Additional methods can be added here as needed.
}

// ConnectionType defines the types of connections available.
type ConnectionType string

const (
	CommandLine ConnectionType = "command_line"
	Network     ConnectionType = "network"
)

// NewClusterScheduler creates a new instance of Scheduler based on the specified connection type.
// The options map should contain the necessary parameters for the connection type.
// For CommandLine connection type, the options map should contain the "executablePath" key.
// The value of the "executablePath" key should be the path to the directory containing
// the scheduler's executables (path to qconf, qacct, etc.).
func NewClusterScheduler(connType ConnectionType, options map[string]string) (ClusterScheduler, error) {
	switch connType {
	case CommandLine:
		executablePath, ok := options["executablePath"]
		if !ok {
			return nil, fmt.Errorf(
				"missing 'executablePath' option for command line connection")
		}
		return NewCommandLineInterface(executablePath)
	case Network:
		// Placeholder for network implementation
		// address, ok := options["address"]
		// if !ok {
		//     return nil, fmt.Errorf("missing 'address' option for network connection")
		// }
		// return NewNetworkScheduler(address), nil
		return nil, fmt.Errorf("network connection not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", connType)
	}
}
