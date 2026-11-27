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
	"io"

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

// WatchFileFrom follows the accounting file with options; see
// core.WatchFileFrom.
func WatchFileFrom(ctx context.Context, path string, o WatchOptions) (<-chan JobDetail, error) {
	return core.WatchFileFrom(ctx, path, o)
}

// ParseAccountingRecord parses one line of the accounting file and refuses
// a line which is no record.
func ParseAccountingRecord(line []byte) (JobDetail, error) {
	return core.ParseAccountingRecord(line)
}

// AccountingUsageValues returns every value of the usage object of one line
// of the accounting file.
func AccountingUsageValues(line []byte) ([]UsageValue, error) {
	return core.AccountingUsageValues(line)
}

// ReadTail reads the last records of the accounting file; see core.ReadTail.
func ReadTail(ctx context.Context, r io.ReaderAt, size int64, o ReadOptions) (ReadResult, error) {
	return core.ReadTail(ctx, r, size, o)
}

// ReadForward reads the records from offset on; see core.ReadForward.
func ReadForward(ctx context.Context, r io.ReaderAt, offset, size int64, o ReadOptions) (ReadResult, error) {
	return core.ReadForward(ctx, r, offset, size, o)
}

// FindJob reads the records of one job; see core.FindJob.
func FindJob(ctx context.Context, r io.ReaderAt, size int64, jobID int64, o ReadOptions) (ReadResult, error) {
	return core.FindJob(ctx, r, size, jobID, o)
}

// FailedText returns the scheduler's text for a failed code.
func FailedText(code int64) string {
	return core.FailedText(code)
}
