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

package core

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

func ParseQacctJobOutputWithScanner(scanner *bufio.Scanner) ([]JobDetail, error) {
	var jobs []JobDetail
	var job JobDetail
	// qacct prints the custom usage values (usage_patterns) of a record
	// after its standard usage, which ends with maxrss and, when execd
	// reports memory details, maxpss. From there on a line is custom usage
	// even if it has the name of a standard field: an expression such as
	// "io*" prints "ioops 102.000" a second time, and that line must not
	// replace the standard value.
	var standardUsageDone, maxPSSSeen bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "==============================================================") {
			if job.JobNumber != 0 {
				jobs = append(jobs, job)
			}
			job = JobDetail{}
			standardUsageDone, maxPSSSeen = false, false
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if standardUsageDone && key != "arid" && (key != "maxpss" || maxPSSSeen) {
			if v, ok := parseCustomUsageValue(value); ok && key != "" {
				job.JobUsage.addCustom(CustomUsageNoGroup, map[string]float64{key: v})
			}
			continue
		}

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
			job.SubmitTime = parseTime(value)
		case "submit_cmd_line":
			job.SubmitCommandLine = value
		case "start_time":
			job.StartTime = parseTime(value)
		case "end_time":
			job.EndTime = parseTime(value)
		case "granted_pe":
			job.GrantedPE = value
		case "slots":
			job.Slots = parseInt64(value)
		case "failed":
			// qacct prints the failed code with a trailing description, e.g.
			// "100 : assumedly after job"; keep only the leading numeric code.
			job.Failed = parseInt64(firstField(value))
		case "exit_status":
			job.ExitStatus = parseInt64(value)
		case "ru_wallclock":
			job.JobUsage.RUsage.RuWallclock = parseInt64(value)
		case "ru_utime":
			job.JobUsage.RUsage.RuUtime = parseFloat(value)
		case "ru_stime":
			job.JobUsage.RUsage.RuStime = parseFloat(value)
		case "ru_maxrss":
			job.JobUsage.RUsage.RuMaxrss = parseInt64(value)
		case "ru_ixrss":
			job.JobUsage.RUsage.RuIxrss = parseInt64(value)
		case "ru_ismrss":
			job.JobUsage.RUsage.RuIsmrss = parseInt64(value)
		case "ru_idrss":
			job.JobUsage.RUsage.RuIdrss = parseInt64(value)
		case "ru_isrss":
			job.JobUsage.RUsage.RuIsrss = parseInt64(value)
		case "ru_minflt":
			job.JobUsage.RUsage.RuMinflt = parseInt64(value)
		case "ru_majflt":
			job.JobUsage.RUsage.RuMajflt = parseInt64(value)
		case "ru_nswap":
			job.JobUsage.RUsage.RuNswap = parseInt64(value)
		case "ru_inblock":
			job.JobUsage.RUsage.RuInblock = parseInt64(value)
		case "ru_oublock":
			job.JobUsage.RUsage.RuOublock = parseInt64(value)
		case "ru_msgsnd":
			job.JobUsage.RUsage.RuMsgsnd = parseInt64(value)
		case "ru_msgrcv":
			job.JobUsage.RUsage.RuMsgrcv = parseInt64(value)
		case "ru_nsignals":
			job.JobUsage.RUsage.RuNsignals = parseInt64(value)
		case "ru_nvcsw":
			job.JobUsage.RUsage.RuNvcsw = parseInt64(value)
		case "ru_nivcsw":
			job.JobUsage.RUsage.RuNivcsw = parseInt64(value)
		case "wallclock":
			job.JobUsage.Usage.WallClock = parseFloat(value)
		case "cpu":
			job.JobUsage.Usage.CPU = parseFloat(value)
		case "mem":
			job.JobUsage.Usage.Memory = parseFloat(value)
		case "io":
			job.JobUsage.Usage.IO = parseFloat(value)
		case "ioops":
			job.JobUsage.Usage.IOOps = parseInt64(value)
		case "iow":
			job.JobUsage.Usage.IOWait = parseFloat(value)
		case "maxvmem":
			job.JobUsage.Usage.MaxVMem = parseFloat(value)
		case "maxrss":
			job.JobUsage.Usage.MaxRSS = parseFloat(value)
			standardUsageDone = true
		case "maxpss":
			job.JobUsage.Usage.MaxPSS = parseFloat(value)
			maxPSSSeen = true
		case "arid":
			job.ArID = value
		}
	}

	if job.JobNumber != 0 {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ParseQAcctJobOutput parses the output of the qacct command and returns
// a slice of JobDetail.
func ParseQAcctJobOutput(output string) ([]JobDetail, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	jobs, err := ParseQacctJobOutputWithScanner(scanner)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func parseTime(value string) int64 {
	layout := "2006-01-02 15:04:05.999999"
	t, err := time.Parse(layout, value)
	if err != nil {
		return 0
	}
	return t.UnixNano() / 1000
}

func parseInt(value string) int {
	i, _ := strconv.Atoi(value)
	return i
}

func parseInt64(value string) int64 {
	i, _ := strconv.ParseInt(value, 10, 64)
	return i
}

// firstField returns the first whitespace-delimited token of value, or "" when
// value is blank. Used for fields whose value carries a trailing description
// (e.g. the qacct "failed" code "100 : assumedly after job").
func firstField(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func parseFloat(value string) float64 {
	f, _ := strconv.ParseFloat(value, 64)
	return f
}

// parseCustomUsageValue reads a custom usage value of the qacct -j output.
// qacct prints these with %f, so only plain decimal numbers are accepted:
// "nan" and "inf" (which could not be marshalled to JSON afterwards) and
// other spellings strconv.ParseFloat knows are not usage values.
func parseCustomUsageValue(value string) (float64, bool) {
	if value == "" || strings.Trim(value, "0123456789+-.") != "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(value, 64)
	return v, err == nil
}

// ParseAccountingJSONLine parses one line (record) of the accounting file.
func ParseAccountingJSONLine(line string) (JobDetail, error) {
	var job JobDetail
	err := json.Unmarshal([]byte(line), &job)
	if err != nil {
		return JobDetail{}, err
	}
	return job, nil
}

// ParseSummaryOutput parses the summary output from qacct command
// and returns aggregated usage statistics.
func ParseSummaryOutput(output string) (Usage, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Total System Usage") {
			if !scanner.Scan() {
				continue
			}
			if !scanner.Scan() {
				continue
			}
			if scanner.Scan() {
				valuesLine := strings.TrimSpace(scanner.Text())
				return parseSummaryValues(valuesLine, []string{"WALLCLOCK", "UTIME", "STIME", "CPU", "MEMORY", "IO", "IOW"})
			}
		}

		if strings.Contains(line, "WALLCLOCK") && strings.Contains(line, "CPU") {
			headers := strings.Fields(line)

			if !scanner.Scan() {
				continue
			}

			if scanner.Scan() {
				valuesLine := strings.TrimSpace(scanner.Text())
				if valuesLine != "" {
					return parseSummaryValues(valuesLine, headers)
				}
			}
			return Usage{}, nil
		}
	}

	return Usage{}, nil
}

func parseSummaryValues(line string, headers []string) (Usage, error) {
	fields := strings.Fields(line)

	usage := Usage{}
	var err error

	usageStart := len(fields) - 7
	if usageStart < 0 {
		return Usage{}, fmt.Errorf("insufficient fields for usage data: got %d, need at least 7", len(fields))
	}

	if usage.WallClock, err = strconv.ParseFloat(fields[usageStart], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing wallclock: %w", err)
	}
	if usage.UserTime, err = strconv.ParseFloat(fields[usageStart+1], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing utime: %w", err)
	}
	if usage.SystemTime, err = strconv.ParseFloat(fields[usageStart+2], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing stime: %w", err)
	}
	if usage.CPU, err = strconv.ParseFloat(fields[usageStart+3], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing cpu: %w", err)
	}
	if usage.Memory, err = strconv.ParseFloat(fields[usageStart+4], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing memory: %w", err)
	}
	if usage.IO, err = strconv.ParseFloat(fields[usageStart+5], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing io: %w", err)
	}
	if usage.IOWait, err = strconv.ParseFloat(fields[usageStart+6], 64); err != nil {
		return Usage{}, fmt.Errorf("error parsing iow: %w", err)
	}

	return usage, nil
}
