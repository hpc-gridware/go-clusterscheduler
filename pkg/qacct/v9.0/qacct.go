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

package qacct

// QAcct defines the methods for interacting with the Open Cluster Scheduler
// to retrieve accounting information for finished jobs using the qacct command.
type QAcct interface {
	WithAlternativeAccountingFile(accountingFile string) error
	WithDefaultAccountingFile()
	// NativeSpecification calls qacct with the given command and args
	// and returns the raw unparsed output.
	NativeSpecification(args []string) (string, error)
	ShowHelp() (string, error)
	ShowJobDetails(jobID []int64) ([]JobDetail, error)
	// Summary returns a builder for summary usage queries
	Summary() *SummaryBuilder
	// Jobs returns a builder for job detail queries with filtering
	Jobs() *JobsBuilder
}

// SummaryBuilder provides a fluent interface for building summary usage queries
type SummaryBuilder struct {
	qacct QAcct
	args  []string
}

// JobsBuilder provides a fluent interface for building job detail queries with filtering
type JobsBuilder struct {
	qacct QAcct
	args  []string
}
