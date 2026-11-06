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

// QConf defines the methods for interacting with the Open Cluster Scheduler
// configuration. This is a type alias to the core QConf interface.
type QConf = core.QConf

// ConfigContextReader reads configuration under the caller's context; see
// core.ConfigContextReader. CommandLineQConf implements it.
type ConfigContextReader = core.ConfigContextReader

// ThreadInfo is one row of "qconf -stl"; see core.ThreadInfo.
type ThreadInfo = core.ThreadInfo

// ThreadAnswer is qmaster's answer to "qconf -at" / "qconf -kt"; see
// core.ThreadAnswer.
type ThreadAnswer = core.ThreadAnswer

const (
	ThreadStarted        = core.ThreadStarted
	ThreadAlreadyRunning = core.ThreadAlreadyRunning
	ThreadTerminated     = core.ThreadTerminated
	ThreadNotRunning     = core.ThreadNotRunning
	SchedulerThread      = core.SchedulerThread
)

// ErrNotSupported is returned when qconf does not know an option; see
// core.ErrNotSupported.
var ErrNotSupported = core.ErrNotSupported

// ParseResourceQuotaSets reads "qconf -srqs" output holding any number of sets.
var ParseResourceQuotaSets = core.ParseResourceQuotaSets

// ParseClusterQueues reads "qconf -sq" output holding any number of cluster queues.
var ParseClusterQueues = core.ParseClusterQueues

// RQSRule is one parsed limit rule of a resource quota set; see core.RQSRule.
type RQSRule = core.RQSRule

// RQSScope is one filter of an RQSRule; see core.RQSScope.
type RQSScope = core.RQSScope

// RQSLimit is one resource limit of an RQSRule; see core.RQSLimit.
type RQSLimit = core.RQSLimit

// ParseRQSRule parses one limit line of a resource quota set; see core.ParseRQSRule.
var ParseRQSRule = core.ParseRQSRule

// ParseRQSRules parses every limit line of a resource quota set; see core.ParseRQSRules.
var ParseRQSRules = core.ParseRQSRules
