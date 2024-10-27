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
	"strconv"
	"strings"
)

// parseJobs parses the input text into a slice of SchedulerJobInfo instances.
func parseJobs(input string) ([]SchedulerJobInfo, error) {
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
		case "binding":
			info.Binding = value
		case "scheduling info":
			info.SchedulingInfo = value
		}
	}
	return info, nil
}
