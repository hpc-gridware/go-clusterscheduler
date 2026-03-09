/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
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

package qsub

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
)

// NewCommandLineQSub creates a new Qsub client.
func NewCommandLineQSub(config CommandLineQSubConfig) (Qsub, error) {
	return core.NewCommandLineQSub(config)
}

type JobBuilder = core.JobBuilder

// NewJobBuilder creates a new fluent JobBuilder for the given command.
func NewJobBuilder(q Qsub, command string, args ...string) *JobBuilder {
	return core.NewJobBuilder(q, command, args...)
}
