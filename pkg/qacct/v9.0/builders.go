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
	"strconv"
)

// NewSummaryBuilder creates a new SummaryBuilder instance
func NewSummaryBuilder(qacct QAcct) *SummaryBuilder {
	return &SummaryBuilder{
		qacct: qacct,
		args:  []string{},
	}
}

// NewJobsBuilder creates a new JobsBuilder instance
func NewJobsBuilder(qacct QAcct) *JobsBuilder {
	return &JobsBuilder{
		qacct: qacct,
		args:  []string{"-j"},
	}
}

// SummaryBuilder methods for fluent interface

// Account filters by account string
func (s *SummaryBuilder) Account(accountString string) *SummaryBuilder {
	s.args = append(s.args, "-A", accountString)
	return s
}

// BeginTime filters jobs started after the given time
func (s *SummaryBuilder) BeginTime(beginTime string) *SummaryBuilder {
	s.args = append(s.args, "-b", beginTime)
	return s
}

// EndTime filters jobs started before the given time
func (s *SummaryBuilder) EndTime(endTime string) *SummaryBuilder {
	s.args = append(s.args, "-e", endTime)
	return s
}

// LastDays filters jobs started in the last N days
func (s *SummaryBuilder) LastDays(days int) *SummaryBuilder {
	s.args = append(s.args, "-d", strconv.Itoa(days))
	return s
}

// Department filters by department
func (s *SummaryBuilder) Department(department string) *SummaryBuilder {
	s.args = append(s.args, "-D", department)
	return s
}

// Group filters by group ID or name
func (s *SummaryBuilder) Group(groupIDOrName string) *SummaryBuilder {
	s.args = append(s.args, "-g", groupIDOrName)
	return s
}

// Host filters by hostname
func (s *SummaryBuilder) Host(host string) *SummaryBuilder {
	s.args = append(s.args, "-h", host)
	return s
}

// Owner filters by owner
func (s *SummaryBuilder) Owner(owner string) *SummaryBuilder {
	s.args = append(s.args, "-o", owner)
	return s
}

// ParallelEnvironment filters by parallel environment
func (s *SummaryBuilder) ParallelEnvironment(peName string) *SummaryBuilder {
	s.args = append(s.args, "-pe", peName)
	return s
}

// Project filters by project
func (s *SummaryBuilder) Project(project string) *SummaryBuilder {
	s.args = append(s.args, "-P", project)
	return s
}

// Queue filters by queue
func (s *SummaryBuilder) Queue(queue string) *SummaryBuilder {
	s.args = append(s.args, "-q", queue)
	return s
}

// Slots filters by number of slots used
func (s *SummaryBuilder) Slots(usedSlots int) *SummaryBuilder {
	s.args = append(s.args, "-slots", strconv.Itoa(usedSlots))
	return s
}

// Execute runs the summary query and returns aggregated usage
func (s *SummaryBuilder) Execute() (Usage, error) {
	output, err := s.qacct.NativeSpecification(s.args)
	if err != nil {
		return Usage{}, fmt.Errorf("error executing summary query: %w", err)
	}
	return ParseSummaryOutput(output)
}

// JobsBuilder methods for fluent interface

// Account filters by account string
func (j *JobsBuilder) Account(accountString string) *JobsBuilder {
	j.args = append(j.args, "-A", accountString)
	return j
}

// BeginTime filters jobs started after the given time
func (j *JobsBuilder) BeginTime(beginTime string) *JobsBuilder {
	j.args = append(j.args, "-b", beginTime)
	return j
}

// EndTime filters jobs started before the given time
func (j *JobsBuilder) EndTime(endTime string) *JobsBuilder {
	j.args = append(j.args, "-e", endTime)
	return j
}

// LastDays filters jobs started in the last N days
func (j *JobsBuilder) LastDays(days int) *JobsBuilder {
	j.args = append(j.args, "-d", strconv.Itoa(days))
	return j
}

// Department filters by department
func (j *JobsBuilder) Department(department string) *JobsBuilder {
	j.args = append(j.args, "-D", department)
	return j
}

// Group filters by group ID or name
func (j *JobsBuilder) Group(groupIDOrName string) *JobsBuilder {
	j.args = append(j.args, "-g", groupIDOrName)
	return j
}

// Host filters by hostname
func (j *JobsBuilder) Host(host string) *JobsBuilder {
	j.args = append(j.args, "-h", host)
	return j
}

// JobPattern filters by job ID, name, or pattern
func (j *JobsBuilder) JobPattern(jobIDOrNameOrPattern string) *JobsBuilder {
	j.args = append(j.args, jobIDOrNameOrPattern)
	return j
}

// Owner filters by owner
func (j *JobsBuilder) Owner(owner string) *JobsBuilder {
	j.args = append(j.args, "-o", owner)
	return j
}

// ParallelEnvironment filters by parallel environment
func (j *JobsBuilder) ParallelEnvironment(peName string) *JobsBuilder {
	j.args = append(j.args, "-pe", peName)
	return j
}

// Project filters by project
func (j *JobsBuilder) Project(project string) *JobsBuilder {
	j.args = append(j.args, "-P", project)
	return j
}

// Queue filters by queue
func (j *JobsBuilder) Queue(queue string) *JobsBuilder {
	j.args = append(j.args, "-q", queue)
	return j
}

// Slots filters by number of slots used
func (j *JobsBuilder) Slots(usedSlots int) *JobsBuilder {
	j.args = append(j.args, "-slots", strconv.Itoa(usedSlots))
	return j
}

// Tasks filters by task ID range
func (j *JobsBuilder) Tasks(jobID, taskIDRange string) *JobsBuilder {
	j.args = append(j.args, "-t", jobID+"."+taskIDRange)
	return j
}

// Execute runs the job detail query and returns job details
func (j *JobsBuilder) Execute() ([]JobDetail, error) {
	output, err := j.qacct.NativeSpecification(j.args)
	if err != nil {
		return nil, fmt.Errorf("error executing job query: %w", err)
	}
	return ParseQAcctJobOutput(output)
}