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

package qacct

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

// QAcct defines the methods for interacting with the Open Cluster Scheduler
// to retrieve accounting information for finished jobs using the qacct command.
type QAcct = core.QAcct

// SummaryBuilder provides a fluent interface for building summary usage queries
type SummaryBuilder = core.SummaryBuilder

// JobsBuilder provides a fluent interface for building job detail queries with filtering
type JobsBuilder = core.JobsBuilder
