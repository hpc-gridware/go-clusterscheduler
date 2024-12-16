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
	"time"
)

const QstatDateFormat = "2006-01-02 03:04:05"

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
		// ignore lines with only dashes
		if strings.HasPrefix(line, "---") {
			continue
		}
		// ignore lines with description
		if strings.HasPrefix(line, "job-ID") {
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
				// could be a numer or something like "7-99:2"
				task.JobInfo.JaTaskIDs = parseJaTaskIDs(fields[8])
				//task.JobInfo.TaskID = fields[8]
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

// "7-99:2" or "1" to 7, 9, 11, 13, ... 99 or 1
func parseJaTaskIDs(s string) []int64 {
	if s == "" {
		return []int64{}
	}

	ids := []int64{}

	parts := strings.Split(s, ":")
	// has a step
	if len(parts) == 2 {
		step, err := strconv.Atoi(parts[1])
		if err != nil {
			return []int64{}
		}
		start := 0
		end := 0
		rangeParts := strings.Split(parts[0], "-")
		if len(rangeParts) == 2 {
			start, err = strconv.Atoi(rangeParts[0])
			if err != nil {
				return []int64{}
			}
			end, err = strconv.Atoi(rangeParts[1])
			if err != nil {
				return []int64{}
			}
		} else {
			return []int64{}
		}
		for i := start; i <= end; i += step {
			ids = append(ids, int64(i))
		}
		return ids
	}

	// no step, either number or range

	split := strings.Split(parts[0], "-")
	// range
	if len(split) == 2 {
		start, err := strconv.Atoi(split[0])
		if err != nil {
			return []int64{}
		}
		end, err := strconv.Atoi(split[1])
		if err != nil {
			return []int64{}
		}
		for i := start; i <= end; i++ {
			ids = append(ids, int64(i))
		}
	} else {
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return []int64{}
		}
		ids = append(ids, int64(id))
	}

	return ids
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
	submitTime, err := time.Parse(QstatDateFormat, fields[5])
	if err != nil {
		return nil, fmt.Errorf("invalid submit time: %v", err)
	}

	jobInfo := &JobInfo{
		JobID:    jobID,
		Priority: priority,
		Name:     fields[2],
		User:     fields[3],
		State:    fields[4],
		Queue:    fields[6],
		Slots:    1,
	}
	if strings.Contains(jobInfo.State, "r") {
		jobInfo.StartTime = submitTime
	}
	if strings.Contains(jobInfo.State, "q") {
		jobInfo.SubmitTime = submitTime
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

/*
 qstat -ext
job-ID  prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
     31 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1
     32 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 1
     32 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 3
     32 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 5
     32 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 7
     32 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 9
*/

func ParseExtendedJobInfo(output string) ([]ExtendedJobInfo, error) {
	ext := []ExtendedJobInfo{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "job-ID") {
			continue
		}
		if strings.HasPrefix(line, "-------------------") {
			continue
		}
		info, err := parseExtendedJobInfoLine(line)
		if err != nil {
			return nil, err
		}
		ext = append(ext, info)
	}

	return ext, nil
}

/*
qstat -ext
job-ID  prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 1
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 2
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 3
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 4
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 5
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 6
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 7
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 8
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 9
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 10
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 11
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 12
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 13
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 14
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 15
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 16
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 17
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 18
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 19
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 20
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 21
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 22
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 23
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 24
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 25
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 26
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 27
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 28
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 29
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 30
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 31
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 32
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 33
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 34
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 35
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 36
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 37
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 38
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 39
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 40
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 41
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 42
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 43
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 44
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 45
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 46
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 47
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 48
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 49
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 50
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 51
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 52
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 53
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 54
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 55
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 56
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 57
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 58
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 59
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 60
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 61
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 62
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 63
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 64
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 65
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 66
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 67
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 68
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 69
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 70
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 71
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 72
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 73
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 74
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 75
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 76
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 77
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 78
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 79
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 80
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 81
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 82
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 83
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 84
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 85
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 86
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 87
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 88
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 89
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 90
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 91
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 92
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 93
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 94
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 95
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 96
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 97
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 98
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 99
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 100
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 1
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 2
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 3
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 4
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 5
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 6
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 7
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 8
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 9
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 10
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 11
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 12
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 13
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 14
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 15
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 16
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 17
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 18
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 19
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 20
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 21
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 22
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 23
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 24
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 25
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 26
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 27
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 28
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 29
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 30
     34 0.55500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 31-100:1
     35 0.55500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1
*/

func parseExtendedJobInfoLine(line string) (ExtendedJobInfo, error) {
	fields := strings.Fields(line)

	// Initialize variables
	var jobID int
	var prior float64
	var ntckts float64
	var name, user, project, department, state, cpu, mem, io string
	var tckts, ovrts, otckt, ftckt, stckt int
	var share float64
	var queue string
	var slots int
	var jaTaskID string

	fmt.Println(len(fields))

	if len(fields) == 19 || len(fields) == 20 {
		// Expected number of fields when job is in 'r' state
		var err error
		jobID, err = strconv.Atoi(fields[0])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse jobID: %v", err)
		}
		prior, err = strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse prior: %v", err)
		}
		ntckts, err = strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse ntckts: %v", err)
		}
		name = fields[3]
		user = fields[4]
		project = fields[5]
		department = fields[6]
		state = fields[7]
		cpu = fields[8]
		mem = fields[9]
		io = fields[10]
		tckts, err = strconv.Atoi(fields[11])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse tckts: %v", err)
		}
		ovrts, err = strconv.Atoi(fields[12])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse ovrts: %v", err)
		}
		otckt, err = strconv.Atoi(fields[13])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse otckt: %v", err)
		}
		ftckt, err = strconv.Atoi(fields[14])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse ftckt: %v", err)
		}
		stckt, err = strconv.Atoi(fields[15])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse stckt: %v", err)
		}
		share, err = strconv.ParseFloat(fields[16], 64)
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse share: %v", err)
		}
		queue = fields[17]
		slots, err = strconv.Atoi(fields[18])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse slots: %v", err)
		}
		if len(fields) == 20 {
			jaTaskID = fields[19]
		}
	} else if len(fields) == 16 || len(fields) == 15 {
		// Expected number of fields when job is in 'qw' state
		var err error
		jobID, err = strconv.Atoi(fields[0])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse jobID: %v", err)
		}
		prior, err = strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse prior: %v", err)
		}
		ntckts, err = strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse ntckts: %v", err)
		}
		name = fields[3]
		user = fields[4]
		project = fields[5]
		department = fields[6]
		state = fields[7]
		// cpu, mem, io are missing; set to default values
		cpu = ""
		mem = ""
		io = ""
		tckts, err = strconv.Atoi(fields[8])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse tckts: %v", err)
		}
		ovrts, err = strconv.Atoi(fields[9])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse ovrts: %v", err)
		}
		otckt, err = strconv.Atoi(fields[10])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse otckt: %v", err)
		}
		ftckt, err = strconv.Atoi(fields[11])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse ftckt: %v", err)
		}
		stckt, err = strconv.Atoi(fields[12])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse stckt: %v", err)
		}
		share, err = strconv.ParseFloat(fields[13], 64)
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse share: %v", err)
		}
		// 'queue' is missing in 'qw' state; set to default value
		queue = ""
		slots, err = strconv.Atoi(fields[14])
		if err != nil {
			return ExtendedJobInfo{}, fmt.Errorf("failed to parse slots: %v", err)
		}
		if len(fields) == 16 {
			jaTaskID = fields[15]
		}
	} else {
		return ExtendedJobInfo{}, fmt.Errorf("unexpected number of fields: %d (%s)",
			len(fields), line)
	}

	// TODO convert them correctly
	mem = io
	io = mem

	return ExtendedJobInfo{
		JobID:      jobID,
		Priority:   prior,
		Name:       name,
		User:       user,
		Project:    project,
		Department: department,
		State:      state,
		CPU:        cpu,
		//Memory:     mem,
		//IO:         io,
		Tckts:    tckts,
		Ovrts:    ovrts,
		Otckt:    otckt,
		Ftckt:    ftckt,
		Stckt:    stckt,
		Ntckts:   ntckts,
		Share:    share,
		Queue:    queue,
		Slots:    slots,
		JATaskID: jaTaskID,
	}, nil
}

/*
job-ID  prior   name       user         state submit/start at     queue                          slots ja-task-ID
-----------------------------------------------------------------------------------------------------------------

	 8 0.55500 sleep      root         r     2024-12-14 17:21:43 all.q@master                       1
	 9 0.55500 sleep      root         r     2024-12-14 17:21:44 all.q@master                       1
	10 0.55500 sleep      root         r     2024-12-14 17:21:44 all.q@master                       1
	11 0.55500 sleep      root         r     2024-12-14 17:21:45 all.q@master                       1
	12 0.55500 sleep      root         qw    2024-12-14 17:21:45                                    1
	13 0.55500 sleep      root         qw    2024-12-14 17:21:46                                    1
	14 0.55500 sleep      root         qw    2024-12-14 17:21:47                                    1
	15 0.55500 sleep      root         qw    2024-12-14 17:22:00                                    1 1-99:2
*/
func ParseJobInfo(out string) ([]JobInfo, error) {
	lines := strings.Split(out, "\n")
	jobInfos := make([]JobInfo, 0, len(lines)-3)
	for _, line := range lines {
		if strings.HasPrefix(line, "job-ID") {
			continue
		}
		if strings.HasPrefix(line, "---------") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		jobID, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse jobID: %v", err)
		}
		priority, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse priority: %v", err)
		}
		name := fields[2]
		user := fields[3]
		state := fields[4]

		var submitTime time.Time
		var startTime time.Time

		var queue string
		var slots int
		var jaTaskIDs []int64
		// the state defines the format of the rest of the fields
		if state == "r" {
			// we have a running job in a queue intance
			submitTimeString := fields[5] + fields[6]
			submitTime, err = time.Parse("2024-12-14 17:21:43", submitTimeString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse submit time: %v", err)
			}
			queue = fields[7]
			slots, err = strconv.Atoi(fields[8])
			if err != nil {
				return nil, fmt.Errorf("failed to parse slots: %v", err)
			}
			// TODO parse jaTaskIDs
		} else if state == "qw" {
			// we have a queued job
			startTimeString := fields[5] + fields[6]
			startTime, err = time.Parse("2024-12-14 17:21:43", startTimeString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse run time: %v", err)
			}
			slots, err = strconv.Atoi(fields[7])
			if err != nil {
				return nil, fmt.Errorf("failed to parse slots: %v", err)
			}
			// TODO parse jaTaskIDs
		}

		jobInfo := JobInfo{
			JobID:      jobID,
			Priority:   priority,
			Name:       name,
			User:       user,
			State:      state,
			SubmitTime: submitTime,
			StartTime:  startTime,
			Queue:      queue,
			Slots:      slots,
			JaTaskIDs:  jaTaskIDs,
		}
		jobInfos = append(jobInfos, jobInfo)
	}
	return jobInfos, nil
}
