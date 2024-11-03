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
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// ParseGroupByTask parses the input text into a slice of
// SchedulerJobInfo instances (qstat -g t output).
func ParseGroupByTask(input string) ([]ParallelJobTask, error) {

	// These are examples of the output format which needs to be parsed:

	/* job-ID  prior   name       user         state submit/start at     queue                          master ja-task-ID
	   ------------------------------------------------------------------------------------------------------------------
	        14 0.50500 sleep      root         r     2024-10-28 07:21:41 all.q@master                   MASTER
	        17 0.60500 sleep      root         r     2024-10-28 07:29:24 all.q@master                   MASTER
	                                                                     all.q@master                   SLAVE
	                                                                     all.q@master                   SLAVE
	                                                                     all.q@master                   SLAVE
	        12 0.50500 sleep      root         qw    2024-10-28 07:17:34
	*/

	/*
	   job-ID  prior   name       user         state submit/start at     queue                          master ja-task-ID
	   ------------------------------------------------------------------------------------------------------------------
	        14 0.50500 sleep      root         r     2024-10-28 07:21:41 all.q@master                   MASTER
	        15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 1
	        15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 3
	        15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 5
	        17 0.60500 sleep      root         qw    2024-10-28 07:27:50
	        12 0.50500 sleep      root         qw    2024-10-28 07:17:34
	        15 0.50500 sleep      root         qw    2024-10-28 07:26:14                                       7-99:2
	*/
	return parseFixedWidthJobs(input)

}
func parseFixedWidthJobs(input string) ([]ParallelJobTask, error) {
	var tasks []ParallelJobTask

	input = strings.TrimSpace(input)
	if input == "" {
		return tasks, nil
	}

	// Correct column positions based on your description
	columnPositions := []struct {
		start int
		end   int
	}{
		{start: 0, end: 8},     // job-ID
		{start: 8, end: 16},    // prior
		{start: 16, end: 26},   // name
		{start: 26, end: 38},   // user
		{start: 38, end: 44},   // state
		{start: 44, end: 65},   // submit/start at
		{start: 65, end: 94},   // queue
		{start: 94, end: 104},  // master
		{start: 104, end: 112}, // ja-task-ID (if exists)
	}

	scanner := bufio.NewScanner(strings.NewReader(input))
	if !scanner.Scan() || !scanner.Scan() {
		return nil, fmt.Errorf("input doesn't contain header or dashed line")
	}

	var currentJob *ParallelJobTask
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := make([]string, len(columnPositions))
		for i, pos := range columnPositions {
			if pos.start < len(line) {
				end := pos.end
				if end > len(line) {
					end = len(line)
				}
				fields[i] = strings.TrimSpace(line[pos.start:end])
			} else {
				fields[i] = ""
			}
		}

		if isContinuationLine(fields) {
			if currentJob != nil && len(fields) > 6 {
				currentJob.Queue = fields[6]
				currentJob.Master = fields[7]
				currentJob.Slots++
			}
		} else {
			jobInfo, err := parseFixedWidthJobInfo(fields)
			if err != nil {
				log.Println("Skipping line due to parsing error:", err)
				continue
			}

			task := ParallelJobTask{JobInfo: *jobInfo}
			if fields[7] != "" {
				task.Master = fields[7]
			}
			if len(fields) > 8 && fields[8] != "" {
				task.JobInfo.TaskID = fields[8]
			}
			tasks = append(tasks, task)
			currentJob = &tasks[len(tasks)-1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return tasks, nil
}

func isContinuationLine(fields []string) bool {
	return len(fields[0]) == 0
}

func parseFixedWidthJobInfo(fields []string) (*JobInfo, error) {
	jobID, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %v", err)
	}

	priority, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid priority: %v", err)
	}

	jobInfo := &JobInfo{
		JobID:         jobID,
		Priority:      priority,
		Name:          fields[2],
		User:          fields[3],
		State:         fields[4],
		SubmitStartAt: fields[5],
		Queue:         fields[6],
		Slots:         1,
	}

	return jobInfo, nil
}

// ParseSchedulerJobInfo parses the input text into a slice of
// SchedulerJobInfo instances (qstat -j output).
func ParseSchedulerJobInfo(input string) ([]SchedulerJobInfo, error) {
	var jobs []SchedulerJobInfo
	blocks := strings.Split(input, "\n==============================================================\n")

	for _, block := range blocks {
		if info, err := parseJob(block); err == nil {
			jobs = append(jobs, info)
		} else {
			return nil, err
		}
	}

	return jobs, nil
}

// parseJob parses a single job information block into a SchedulerJobInfo instance.
func parseJob(block string) (SchedulerJobInfo, error) {
	var info SchedulerJobInfo
	lines := strings.Split(block, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "job_number":
			info.JobNumber, _ = strconv.Atoi(value)
		case "exec_file":
			info.ExecFile = value
		case "submission_time":
			info.SubmissionTime = value
		case "submit_cmd_line":
			info.SubmitCmdLine = value
		case "effective_submit_cmd_line":
			info.EffectiveSubmitCmdLine = value
		case "owner":
			info.Owner = value
		case "uid":
			info.UID, _ = strconv.Atoi(value)
		case "group":
			info.Group = value
		case "gid":
			info.GID, _ = strconv.Atoi(value)
		case "sge_o_home":
			info.SgeOHome = value
		case "sge_o_path":
			info.SgeOPath = value
		case "sge_o_workdir":
			info.SgeOWorkDir = value
		case "sge_o_host":
			info.SgeOHost = value
		case "account":
			info.Account = value
		case "mail_list":
			info.MailList = value
		case "notify":
			info.Notify = strings.ToLower(value) == "true"
		case "job_name":
			info.JobName = value
		case "jobshare":
			info.JobShare, _ = strconv.Atoi(value)
		case "env_list":
			info.EnvList = value
		case "job_args":
			info.JobArgs = value
		case "script_file":
			info.ScriptFile = value
		case "parallel_environment":
			info.ParallelEnvironment = value
		case "binding":
			info.Binding = value
		case "scheduling info":
			info.SchedulingInfo = value
		}
	}
	return info, nil
}
