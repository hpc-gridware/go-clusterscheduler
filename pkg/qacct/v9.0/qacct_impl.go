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

import (
	"fmt"
	"os/exec"
)

type QAcctImpl struct {
	config CommandLineQAcctConfig
}

type CommandLineQAcctConfig struct {
	Executable     string
	AccountingFile string
	DryRun         bool
}

func NewCommandLineQAcct(config CommandLineQAcctConfig) (*QAcctImpl, error) {
	if config.Executable == "" {
		config.Executable = "qacct"
	}
	if !config.DryRun {
		_, err := exec.LookPath(config.Executable)
		if err != nil {
			return nil, fmt.Errorf("%s not found in PATH", config.Executable)
		}
	}
	return &QAcctImpl{config: config}, nil
}

func (q *QAcctImpl) WithAlternativeAccountingFile(accountingFile string) error {
	q.config.AccountingFile = accountingFile
	return nil
}

func (q *QAcctImpl) WithDefaultAccountingFile() {
	q.config.AccountingFile = ""
}

// NativeSpecification runs the qacct command with the given arguments
// and returns the raw output.
func (q *QAcctImpl) NativeSpecification(args []string) (string, error) {
	if q.config.AccountingFile != "" {
		args = append(args, "-f", q.config.AccountingFile)
	}

	if q.config.DryRun {
		return fmt.Sprintf("Dry run: %s %v", q.config.Executable, args), nil
	}

	command := exec.Command(q.config.Executable, args...)
	out, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get output of qacct: %w", err)
	}
	return string(out), nil
}

func (q *QAcctImpl) ListAdvanceReservations(arID string) ([]ReservationUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) JobsAccountedTo(accountString string) (Usage, error) {
	return Usage{}, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) JobsStartedAfter(beginTime string) (Usage, error) {
	return Usage{}, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) JobsStartedBefore(endTime string) (Usage, error) {
	return Usage{}, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) JobsStartedLastDays(days int) (Usage, error) {
	return Usage{}, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListDepartment(department string) ([]DepartmentUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListGroup(groupIDOrName string) ([]GroupUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListHost(host string) ([]HostUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListJobs(jobIDOrNameOrPattern string) ([]JobDetail, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) RequestComplexAttributes(attributes string) ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListOwner(owner string) ([]OwnerUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListParallelEnvironment(peName string) ([]PeUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListProject(project string) ([]ProjectUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListQueue(queue string) ([]QueueUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListJobUsageBySlots(usedSlots int) ([]SlotsUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ListTasks(jobID, taskIDRange string) ([]TaskUsage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ShowHelp() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (q *QAcctImpl) ShowTotalSystemUsage() (Usage, error) {
	return Usage{}, fmt.Errorf("not implemented")
}

// ShowJobDetails returns the job details for the given job IDs. If the jobIDs
// is nil, all jobs are returned.
func (q *QAcctImpl) ShowJobDetails(jobIDs []int64) ([]JobDetail, error) {

	allJobDetails := []JobDetail{}

	// if jobIDs is nil, return all jobs
	if len(jobIDs) == 0 {
		out, err := q.NativeSpecification([]string{"-j", "*"})
		if err != nil {
			return nil, fmt.Errorf("error getting job details: %w", err)
		}
		jobDetails, err := ParseQAcctJobOutput(out)
		if err != nil {
			return nil, fmt.Errorf("error parsing job details: %w", err)
		}
		allJobDetails = append(allJobDetails, jobDetails...)
		return allJobDetails, nil
	}

	for _, jobID := range jobIDs {
		args := []string{"-j"}
		args = append(args, fmt.Sprintf("%d", jobID))

		out, err := q.NativeSpecification(args)
		if err != nil {
			return nil, fmt.Errorf("error getting job details: %w", err)
		}

		jobDetails, err := ParseQAcctJobOutput(out)
		if err != nil {
			return nil, fmt.Errorf("error parsing job details: %w", err)
		}
		allJobDetails = append(allJobDetails, jobDetails...)
	}

	return allJobDetails, nil
}
