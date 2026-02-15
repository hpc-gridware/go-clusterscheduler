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

package qconf

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// CommandLineQConf is a type alias to the core implementation.
// All methods are inherited from core.CommandLineQConf.
type CommandLineQConf = core.CommandLineQConf

// CommandLineQConfConfig is a type alias to the core configuration.
type CommandLineQConfConfig = core.CommandLineQConfConfig

// NewCommandLineQConf creates a new instance of CommandLineQConf.
func NewCommandLineQConf(config CommandLineQConfConfig) (*CommandLineQConf, error) {
	return core.NewCommandLineQConf(config)
}
