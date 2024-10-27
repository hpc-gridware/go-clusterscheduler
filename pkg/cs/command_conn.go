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
	"path/filepath"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

type CommandLineInterface struct {
	executablePath string
}

// NewCommandLineScheduler creates a new instance of CommandLineInterface.
func NewCommandLineInterface(executablePath string) (*CommandLineInterface, error) {
	return &CommandLineInterface{executablePath: executablePath}, nil
}

func (c *CommandLineInterface) QConf() (qconf.QConf, error) {
	return qconf.NewCommandLineQConf(
		qconf.CommandLineQConfConfig{
			Executable: filepath.Join(c.executablePath, "qconf"),
		},
	)
}

// Additional methods for other operations can be added here.
