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

package qalter

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
)

// QAlter defines the methods for interacting with the OCS 9.0 qalter
// command. It extends the core QAlter interface with v9.0-specific
// binding support.
type QAlter interface {
	core.QAlter

	// SetBinding binds the job to processor cores
	// (-binding [env|pe|set] exp|lin|str).
	// instance must be one of: env, pe, set.
	// spec is the binding specification (e.g. "linear:4",
	// "striding:2:4", "explicit:0,0:1,0").
	SetBinding(jobTaskList, instance, spec string) (string, error)
}
