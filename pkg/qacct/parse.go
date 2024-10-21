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
	"bufio"
	"strconv"
	"strings"
)

func ParseQacctJobOutputWithScanner(scanner *bufio.Scanner) ([]JobDetail, error) {
	var jobs []JobDetail
	var job JobDetail

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "==============================================================") {
			if job.JobNumber != 0 {
				jobs = append(jobs, job)
			}
			job = JobDetail{}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "qname":
			job.QName = value
		case "hostname":
			job.HostName = value
		case "group":
			job.Group = value
		case "owner":
			job.Owner = value
		case "project":
			job.Project = value
		case "department":
			job.Department = value
		case "jobname":
			job.JobName = value
		case "jobnumber":
			job.JobNumber = parseInt64(value)
		case "taskid":
			job.TaskID = parseInt64(value)
		case "pe_taskid":
			job.PETaskID = value
		case "account":
			job.Account = value
		case "priority":
			job.Priority = parseInt64(value)
		case "qsub_time":
			job.QSubTime = value
		case "submit_cmd_line":
			job.SubmitCommandLine = value
		case "start_time":
			job.StartTime = value
		case "end_time":
			job.EndTime = value
		case "granted_pe":
			job.GrantedPE = value
		case "slots":
			job.Slots = parseInt64(value)
		case "failed":
			job.Failed = parseInt64(value)
		case "exit_status":
			job.ExitStatus = parseInt64(value)
		case "ru_wallclock":
			job.RuWallClock = parseFloat(value)
		case "ru_utime":
			job.RuUTime = parseFloat(value)
		case "ru_stime":
			job.RuSTime = parseFloat(value)
		case "ru_maxrss":
			job.RuMaxRSS = parseInt64(value)
		case "ru_ixrss":
			job.RuIXRSS = parseInt64(value)
		case "ru_ismrss":
			job.RuISMRSS = parseInt64(value)
		case "ru_idrss":
			job.RuIDRSS = parseInt64(value)
		case "ru_isrss":
			job.RuISRss = parseInt64(value)
		case "ru_minflt":
			job.RuMinFlt = parseInt64(value)
		case "ru_majflt":
			job.RuMajFlt = parseInt64(value)
		case "ru_nswap":
			job.RuNSwap = parseInt64(value)
		case "ru_inblock":
			job.RuInBlock = parseInt64(value)
		case "ru_oublock":
			job.RuOuBlock = parseInt64(value)
		case "ru_msgsnd":
			job.RuMsgSend = parseInt64(value)
		case "ru_msgrcv":
			job.RuMsgRcv = parseInt64(value)
		case "ru_nsignals":
			job.RuNSignals = parseInt64(value)
		case "ru_nvcsw":
			job.RuNVCSw = parseInt64(value)
		case "ru_nivcsw":
			job.RuNiVCSw = parseInt64(value)
		case "wallclock":
			job.WallClock = parseFloat(value)
		case "cpu":
			job.CPU = parseFloat(value)
		case "mem":
			job.Memory = parseFloat(value)
		case "io":
			job.IO = parseFloat(value)
		case "iow":
			job.IOWait = parseFloat(value)
		case "maxvmem":
			job.MaxVMem = parseInt64(value)
		case "maxrss":
			job.MaxRSS = parseInt64(value)
		case "arid":
			job.ArID = value
		}
	}

	if job.JobNumber != 0 {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ParseQacctOutput parses the output of the qacct command and returns
// a slice of JobDetail.
func ParseQAcctJobOutput(output string) ([]JobDetail, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	jobs, err := ParseQacctJobOutputWithScanner(scanner)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func parseInt(value string) int {
	i, _ := strconv.Atoi(value)
	return i
}

func parseInt64(value string) int64 {
	i, _ := strconv.ParseInt(value, 10, 64)
	return i
}

func parseFloat(value string) float64 {
	f, _ := strconv.ParseFloat(value, 64)
	return f
}
