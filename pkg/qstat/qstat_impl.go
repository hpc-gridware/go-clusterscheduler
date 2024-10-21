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
	"fmt"
	"os/exec"
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

func (q *QStatImpl) NativeSpecification(args []string) (string, error) {
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qstat %v", args), nil
	}

	command := exec.Command(q.config.Executable, args...)
	out, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get output of qstat: %w", err)
	}
	return string(out), nil
}

func (q *QStatImpl) ShowAdditionalAttributes() ([]ExtendedJobInfo, error) {
	return nil, fmt.Errorf("not implemented")
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

func (q *QStatImpl) DisplayClusterQueueSummary() ([]ClusterQueueSummary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (q *QStatImpl) DisplayAllJobArrayTasks() ([]JobArrayTask, error) {
	return nil, fmt.Errorf("not implemented")
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
