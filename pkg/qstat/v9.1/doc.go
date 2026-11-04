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

// Package qstat is the qstat client for Gridware and Open Cluster Scheduler
// 9.1. It re-exports the 9.0 client where the command line and the output
// are unchanged and adds the parsers whose answer differs in 9.1.
//
// # Breaking changes (unreleased)
//
// QStatImpl was a type alias of v9.0.QStatImpl and is now a struct that
// embeds *v9.0.QStatImpl. Callers notice this in three ways:
//
//   - A *v9.0.QStatImpl no longer assigns to a *QStatImpl variable or
//     field, nor the other way round. Use NewCommandLineQstat of this
//     package, or the embedded field QStatImpl.QStatImpl for the 9.0 value.
//   - QStatImpl.SchedulerJobInfoContext now parses with the 9.1 parser and
//     returns []SchedulerJobInfo of this package (it returned the 9.0 type).
//   - The zero value &QStatImpl{} panics on every promoted method because
//     the embedded client is nil; construct with NewCommandLineQstat.
//
// The queue instance detail listing (StreamQueueInstanceDetailsXML) is not
// a breaking change: an empty QueueDetailOptions.Queues still returns
// ErrNoQueues, and the whole cluster needs the new AllQueues option.
//
// ErrNoSuchJob of this package, of v9.0 and of qalter core are now the same
// value, helper.ErrNoSuchJob, with the unchanged text "job does not exist".
package qstat
