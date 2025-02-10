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

package qstat

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type QStatImpl struct {
	config CommandLineQStatConfig
}

type CommandLineQStatConfig struct {
	Executable string
	DryRun     bool
}

func NewCommandLineQstat(config CommandLineQStatConfig) (*QStatImpl, error) {
	if config.Executable == "" {
		config.Executable = "qstat"
	}
	if config.DryRun == false {
		// check if executable is reachable
		_, err := exec.LookPath(config.Executable)
		if err != nil {
			return nil, fmt.Errorf("executable not found: %w", err)
		}
	}
	return &QStatImpl{config: config}, nil
}

// WatchJobs returns a channel that emits SchedulerJobInfo objects for
// the given job ids. The channel is closed when all jobs left the system,
// or when the context is cancelled.
func (q *QStatImpl) WatchJobs(ctx context.Context, jobIds []int64) (chan SchedulerJobInfo, error) {
	// Convert jobIds from []int64 to []string
	jobIdStrings := make([]string, len(jobIds))
	for i, id := range jobIds {
		jobIdStrings[i] = fmt.Sprintf("%d", id)
	}

	jobId := strings.Join(jobIdStrings, ",")

	var jobs []SchedulerJobInfo

	// wait until the job is in the system
	for i := 0; i < 3; i++ {
		out, err := q.NativeSpecification([]string{"-j", jobId})
		if err != nil {
			if i >= 2 {
				fmt.Printf("error getting qstat output: %v\n", err)
				return nil, fmt.Errorf("error getting qstat output: %w", err)
			}
			// wait 1 second and try again
			<-time.After(1 * time.Second)
			continue
		}
		// found job
		jobs, err = ParseSchedulerJobInfo(out)
		if err != nil {
			return nil, fmt.Errorf("error parsing jobs: %w", err)
		}
		break
	}

	ch := make(chan SchedulerJobInfo)
	go func() {
		defer close(ch)

		for {
			for _, job := range jobs {
				// check if the context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
					// Send job to channel only if context is not cancelled
					ch <- job
				}
			}
			// it does not make sense to check more often than
			// the load report time interval
			<-time.After(15 * time.Second)
			out, err := q.NativeSpecification([]string{"-j", jobId})
			if err != nil {
				// all jobs left the system
				break
			}
			// found jobs
			jobs, err = ParseSchedulerJobInfo(out)
			if err != nil || len(jobs) == 0 {
				// all jobs left the system
				break
			}
		}
	}()
	return ch, nil
}

// NativeSpecification returns the output of the qstat command for the given
// arguments. The arguments are passed to the qstat command as is.
// The output is returned as a string.
func (q *QStatImpl) NativeSpecification(args []string) (string, error) {
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qstat %v", args), nil
	}
	command := exec.Command(q.config.Executable, args...)
	out, err := command.Output()
	if err != nil {
		// convert error in exit error
		ee, ok := err.(*exec.ExitError)
		if ok {
			if !ee.Success() {
				return "", fmt.Errorf("qstat command failed with exit code %d", ee.ExitCode())
			}
			return "", nil
		}
		return "", fmt.Errorf("failed to get output of qstat: %w", err)
	}
	return string(out), nil
}

func (q *QStatImpl) ShowJobs() ([]JobInfo, error) {
	out, err := q.NativeSpecification(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get output of qstat: %w", err)
	}
	return ParseJobInfo(out)
}

func (q *QStatImpl) ShowJobsWithAdditionalAttributes() ([]ExtendedJobInfo, error) {
	out, err := q.NativeSpecification([]string{"-ext"})
	if err != nil {
		return nil, fmt.Errorf("failed to get output of qstat: %w", err)
	}
	return ParseExtendedJobInfo(out)
}

func (q *QStatImpl) ShowQueueExplanation(reason string) ([]QueueExplanation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowFullOutput() ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowFullOutputWithResources(resourceAttributes string) ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

// DisplayClusterQueueSummary is equivalent to "qstat -g c"
func (q *QStatImpl) DisplayClusterQueueSummary() ([]ClusterQueueSummary, error) {
	out, err := q.NativeSpecification([]string{"-g", "c"})
	if err != nil {
		return nil, fmt.Errorf("failed to get output of qstat: %w", err)
	}
	return ParseClusterQueueSummary(out)
}

// DisplayAllJobArrayTasks is equivalent to "qstat -g d"
func (q *QStatImpl) DisplayAllJobArrayTasks() ([]JobArrayTask, error) {
	out, err := q.NativeSpecification([]string{"-g", "d"})
	if err != nil {
		return nil, fmt.Errorf("failed to get output of qstat: %w", err)
	}
	jobArrayTasks, err := ParseJobArrayTask(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job array tasks: %w", err)
	}

	return jobArrayTasks, nil
}

func (q *QStatImpl) DisplayAllParallelJobTasks() ([]ParallelJobTask, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowHelp() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowSchedulerJobInformation(jobIdentifierList []string) ([]SchedulerJobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) RequestResources(resourceList string) ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) HideEmptyQueues() ([]QueueInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) SuppressAdditionalBindingParams() ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) SelectQueuesWithPE(peList []string) ([]QueueInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) PrintQueueInformation(queueNameList []string) ([]QueueInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) SelectQueuesInState(states string) ([]QueueInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowRequestedResourcesOfJobs() ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowJobStatus(filter string) ([]JobStatus, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ShowTaskInformation() ([]TaskInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) ViewJobsOfUser(userList []string) ([]JobInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) SelectQueuesWithUserAccess(userList []string) ([]QueueInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) DisplayJobUrgencyInformation() ([]JobUrgency, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) DisplayJobPriorityInformation() ([]JobPriority, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) DisplayInformationInXMLFormat() (string, error) {
	return "", fmt.Errorf("not implemented")
}
