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
	"bufio"
	"context"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

type QAcctImpl = core.QAcctImpl
type CommandLineQAcctConfig = core.CommandLineQAcctConfig

func NewCommandLineQAcct(config CommandLineQAcctConfig) (*QAcctImpl, error) {
	return core.NewCommandLineQAcct(config)
}

func NewSummaryBuilder(qacct QAcct) *SummaryBuilder {
	return core.NewSummaryBuilder(qacct)
}

func NewJobsBuilder(qacct QAcct) *JobsBuilder {
	return core.NewJobsBuilder(qacct)
}

func ParseQacctJobOutputWithScanner(scanner *bufio.Scanner) ([]JobDetail, error) {
	return core.ParseQacctJobOutputWithScanner(scanner)
}

func ParseQAcctJobOutput(output string) ([]JobDetail, error) {
	return core.ParseQAcctJobOutput(output)
}

func ParseAccountingJSONLine(line string) (JobDetail, error) {
	return core.ParseAccountingJSONLine(line)
}

func ParseSummaryOutput(output string) (Usage, error) {
	return core.ParseSummaryOutput(output)
}

func GetDefaultQacctFile() string {
	return core.GetDefaultQacctFile()
}

func WatchFile(ctx context.Context, path string, bufferSize int) (<-chan JobDetail, error) {
	return core.WatchFile(ctx, path, bufferSize)
}
