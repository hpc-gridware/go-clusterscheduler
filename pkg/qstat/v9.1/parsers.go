/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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
	"strconv"
	"strings"
	"time"
	"unicode"

	v90 "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

var ParseClusterQueueSummary = v90.ParseClusterQueueSummary

const QstatDateFormat = "2006-01-02 15:04:05"

// QstatDateFormatUS is the alternate qstat date layout emitted by some
// 9.x builds (US-style MM/DD/YYYY).
const QstatDateFormatUS = "01/02/2006 15:04:05"

// clusterDateLayouts lists the layouts parseClusterDate accepts. Hoisted
// to a package var so the slice is allocated once rather than per call.
var clusterDateLayouts = []string{
	QstatDateFormat,
	QstatDateFormat + ".000",
	QstatDateFormatUS,
	QstatDateFormatUS + ".000",
}

// parseClusterDate parses a qstat date/time field accepting either the
// ISO-style "YYYY-MM-DD HH:MM:SS" layout or the US-style
// "MM/DD/YYYY HH:MM:SS" layout, with an optional ".NNN" sub-second part.
func parseClusterDate(s string) (time.Time, error) {
	for _, l := range clusterDateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised qstat date layout: %q", s)
}

// ParseSchedulerJobInfo parses the v9.1 qstat -j output into
// SchedulerJobInfo instances with per-task details.
func ParseSchedulerJobInfo(input string) ([]SchedulerJobInfo, error) {
	var jobs []SchedulerJobInfo
	blocks := strings.Split(input,
		"\n==============================================================\n")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		info, err := parseJob(block)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, info)
	}

	return jobs, nil
}

func parseJob(block string) (SchedulerJobInfo, error) {
	var info SchedulerJobInfo
	lines := strings.Split(block, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "==============================================================" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		rawKey := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Per-task fields have the task ID embedded in the key,
		// e.g. "job_state                  49" or "usage                      49".
		taskID, baseKey := extractTaskID(rawKey)
		if taskID > 0 {
			task := findOrCreateTask(&info, taskID)
			switch baseKey {
			case "job_state":
				task.State = value
			case "usage":
				task.Usage = parseTaskUsage(value)
			case "exec_binding_list":
				task.BindingList = value
			case "exec_queue_list":
				task.QueueList = value
			case "exec_host_list":
				task.ExecHostList = parseExecHostList(value)
			case "start_time":
				task.StartTime = value
			case "resource_map":
				task.ResourceMap = value
			case "granted_request":
				task.GrantedRequests = append(task.GrantedRequests,
					GrantedRequest{
						PTGID:      len(task.GrantedRequests),
						GrantedReq: value,
					})
			}
			continue
		}

		switch baseKey {
		case "job_number":
			info.JobNumber, _ = strconv.Atoi(value)
		case "category_id":
			info.CategoryID, _ = strconv.Atoi(value)
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
		case "groups":
			info.Groups = value
		case "sge_o_home":
			info.SgeOHome = value
		case "sge_o_log_name":
			info.SgeOLogName = value
		case "sge_o_path":
			info.SgeOPath = value
		case "sge_o_shell":
			info.SgeOShell = value
		case "sge_o_workdir":
			info.SgeOWorkDir = value
		case "sge_o_host":
			info.SgeOHost = value
		case "account":
			info.Account = value
		case "hard_resource_list":
			info.HardResourceList = value
		case "hard_queue_list":
			info.HardQueueList = value
		case "soft_resource_list":
			info.SoftResourceList = value
		case "soft_queue_list":
			info.SoftQueueList = value
		case "mail_list":
			info.MailList = value
		case "notify":
			info.Notify = strings.EqualFold(value, "true")
		case "job_name":
			info.JobName = value
		case "priority":
			info.Priority, _ = strconv.Atoi(value)
		case "jobshare":
			info.JobShare, _ = strconv.Atoi(value)
		case "env_list":
			info.EnvList = value
		case "job_args":
			info.JobArgs = value
		case "script_file":
			info.ScriptFile = value
		case "department":
			info.Department = value
		case "sync_options":
			info.SyncOptions = value
		case "parallel_environment":
			info.ParallelEnvironment = value
		case "job-array tasks":
			info.JobArrayTasks = strings.TrimSpace(value)
		case "task_concurrency":
			info.TaskConcurrency = value
		case "pending_tasks":
			info.PendingTasks, _ = strconv.Atoi(value)
		case "binding":
			info.Binding = value
		case "scheduling info":
			info.SchedulingInfo = value
		}
	}
	return info, nil
}

// extractTaskID checks if rawKey contains a trailing integer task ID
// separated by whitespace, e.g. "job_state                  49".
// Returns (taskID, baseKey) or (0, rawKey) if no task ID is found.
func extractTaskID(rawKey string) (int, string) {
	fields := strings.Fields(rawKey)
	if len(fields) == 2 {
		if id, err := strconv.Atoi(fields[1]); err == nil {
			return id, fields[0]
		}
	}
	return 0, rawKey
}

// findOrCreateTask looks up the TaskDetail for taskID in info.Tasks,
// creating a new entry with empty slices if not found.
func findOrCreateTask(info *SchedulerJobInfo, taskID int) *TaskDetail {
	for i := range info.Tasks {
		if info.Tasks[i].TaskID == taskID {
			return &info.Tasks[i]
		}
	}
	info.Tasks = append(info.Tasks, TaskDetail{
		TaskID:          taskID,
		ExecHostList:    []ExecHostEntry{},
		GrantedRequests: []GrantedRequest{},
		GrantedLicenses: []interface{}{},
		GPUUsage:        []interface{}{},
		CgroupsUsage:    []interface{}{},
	})
	return &info.Tasks[len(info.Tasks)-1]
}

// parseExecHostList parses a comma-separated "hostname=slots" string
// into a slice of ExecHostEntry values. "NONE" and empty entries are ignored.
//
// Example input: "compute01=8,compute02=4"
func parseExecHostList(s string) []ExecHostEntry {
	var entries []ExecHostEntry
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" || strings.EqualFold(part, "NONE") {
			continue
		}
		// Find the last '=' to split hostname from slot count.
		eqIdx := strings.LastIndex(part, "=")
		if eqIdx < 0 {
			continue
		}
		hostname := part[:eqIdx]
		slots, _ := strconv.Atoi(part[eqIdx+1:])
		entries = append(entries, ExecHostEntry{
			Hostname: hostname,
			Slots:    slots,
		})
	}
	return entries
}

// parseTaskUsage parses a comma-separated "key=value" usage string into a
// TaskUsageDetail. Values may contain spaces (e.g. "117504.22 GB") and are
// extracted by scanning for the next ",lowercaseletter" delimiter.
//
// Example input:
//
//	"wallclock=00:36:41,cpu=03:16:20,mem=117504.22 GB,io=12.87 GB,..."
func parseTaskUsage(s string) TaskUsageDetail {
	get := func(key string) string { return extractUsageField(s, key) }
	return TaskUsageDetail{
		WallClock: get("wallclock"),
		CPU:       get("cpu"),
		Mem:       get("mem"),
		IO:        get("io"),
		IOW:       get("iow"),
		IOOps:     get("ioops"),
		VMem:      get("vmem"),
		MaxVMem:   get("maxvmem"),
		RSS:       get("rss"),
		MaxRSS:    get("maxrss"),
		PSS:       get("pss"),
		SMem:      get("smem"),
		PMem:      get("pmem"),
		MaxPSS:    get("maxpss"),
	}
}

// extractUsageField extracts the value for a named key from a
// comma-separated "key=value" string where values may contain spaces.
// The value ends at the next ",<lowercase-letter>" sequence or end-of-string.
func extractUsageField(s, key string) string {
	prefix := key + "="
	idx := strings.Index(s, prefix)
	if idx == -1 {
		return ""
	}
	rest := s[idx+len(prefix):]
	for i := 0; i < len(rest)-1; i++ {
		if rest[i] == ',' {
			next := rest[i+1]
			if next >= 'a' && next <= 'z' {
				return rest[:i]
			}
		}
	}
	return rest
}

// ParseExtendedJobInfo parses the v9.1 qstat -ext output.
// v9.1 uses a wider job-ID column (10 chars) and cpu may be a duration
// like "0:00:00:00" instead of "NA".
func ParseExtendedJobInfo(output string) ([]ExtendedJobInfo, error) {
	var ext []ExtendedJobInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "job-ID") {
			continue
		}
		if strings.HasPrefix(line, "---") {
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

func parseExtendedJobInfoLine(line string) (ExtendedJobInfo, error) {
	fields := strings.Fields(line)

	if len(fields) >= 19 && !isNumber(fields[8]) {
		return parseExtRunning(fields)
	}
	if len(fields) >= 15 && isNumber(fields[8]) {
		return parseExtWaiting(fields)
	}
	return ExtendedJobInfo{}, fmt.Errorf("unexpected number of fields: %d (%s)",
		len(fields), line)
}

func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func parseExtRunning(fields []string) (ExtendedJobInfo, error) {
	info := ExtendedJobInfo{}
	var err error

	info.JobID, err = strconv.Atoi(fields[0])
	if err != nil {
		return info, fmt.Errorf("failed to parse jobID: %w", err)
	}
	info.Priority, err = strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return info, fmt.Errorf("failed to parse prior: %w", err)
	}
	info.Ntckts, err = strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return info, fmt.Errorf("failed to parse ntckts: %w", err)
	}
	info.Name = fields[3]
	info.User = fields[4]
	info.Project = fields[5]
	info.Department = fields[6]
	info.State = fields[7]
	info.CPU = fields[8]
	// mem and io may be "NA" or numeric
	info.Memory = parseFloat64OrZero(fields[9])
	info.IO = parseFloat64OrZero(fields[10])

	info.Tckts, _ = strconv.Atoi(fields[11])
	info.Ovrts, _ = strconv.Atoi(fields[12])
	info.Otckt, _ = strconv.Atoi(fields[13])
	info.Ftckt, _ = strconv.Atoi(fields[14])
	info.Stckt, _ = strconv.Atoi(fields[15])
	info.Share, _ = strconv.ParseFloat(fields[16], 64)
	info.Queue = fields[17]
	info.Slots, _ = strconv.Atoi(fields[18])

	if len(fields) >= 20 {
		info.JATaskID = fields[19]
	}
	return info, nil
}

func parseExtWaiting(fields []string) (ExtendedJobInfo, error) {
	info := ExtendedJobInfo{}
	var err error

	info.JobID, err = strconv.Atoi(fields[0])
	if err != nil {
		return info, fmt.Errorf("failed to parse jobID: %w", err)
	}
	info.Priority, err = strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return info, fmt.Errorf("failed to parse prior: %w", err)
	}
	info.Ntckts, err = strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return info, fmt.Errorf("failed to parse ntckts: %w", err)
	}
	info.Name = fields[3]
	info.User = fields[4]
	info.Project = fields[5]
	info.Department = fields[6]
	info.State = fields[7]
	info.Tckts, _ = strconv.Atoi(fields[8])
	info.Ovrts, _ = strconv.Atoi(fields[9])
	info.Otckt, _ = strconv.Atoi(fields[10])
	info.Ftckt, _ = strconv.Atoi(fields[11])
	info.Stckt, _ = strconv.Atoi(fields[12])
	info.Share, _ = strconv.ParseFloat(fields[13], 64)
	info.Slots, _ = strconv.Atoi(fields[14])

	if len(fields) >= 16 {
		info.JATaskID = fields[15]
	}
	return info, nil
}

func parseFloat64OrZero(s string) float64 {
	if s == "NA" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// ParseGroupByTask parses the v9.1 qstat -g t output. v9.1 uses a wider
// job-ID column (10 chars) but we parse by whitespace splitting for
// robustness across versions.
func ParseGroupByTask(input string) ([]ParallelJobTask, error) {
	var tasks []ParallelJobTask

	input = strings.TrimSpace(input)
	if input == "" {
		return tasks, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(input))
	headerSkipped := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "job-ID") {
			headerSkipped = true
			continue
		}
		if strings.HasPrefix(trimmed, "---") {
			headerSkipped = true
			continue
		}
		if !headerSkipped {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		// Continuation lines have a non-numeric first field (e.g. a queue name).
		// Regular job lines always start with a numeric job ID, even if the
		// line has leading whitespace from wider column formatting.
		jobID, err := strconv.Atoi(fields[0])
		if err != nil {
			if len(tasks) > 0 && len(fields) >= 2 {
				last := &tasks[len(tasks)-1]
				last.Queue = fields[0]
				last.Master = fields[1]
				last.Slots++
			}
			continue
		}

		if len(fields) < 6 {
			continue
		}
		priority, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}

		name := fields[2]
		user := fields[3]
		state := fields[4]
		timeStr := fields[5] + " " + fields[6]
		jobTime, err := parseClusterDate(timeStr)
		if err != nil {
			continue
		}

		ji := JobInfo{
			JobID:    jobID,
			Priority: priority,
			Name:     name,
			User:     user,
			State:    state,
			Slots:    1,
		}
		if strings.Contains(state, "r") {
			ji.StartTime = jobTime
		} else {
			ji.SubmitTime = jobTime
		}

		task := ParallelJobTask{JobInfo: ji}

		idx := 7
		// queue (optional, present for running jobs)
		if idx < len(fields) && !isValidMaster(fields[idx]) && !looksLikeTaskRange(fields[idx]) {
			task.Queue = fields[idx]
			ji.Queue = fields[idx]
			task.JobInfo = ji
			idx++
		}
		// master (optional)
		if idx < len(fields) && isValidMaster(fields[idx]) {
			task.Master = fields[idx]
			idx++
		}
		// ja-task-ID (optional)
		if idx < len(fields) {
			task.JobInfo.JaTaskIDs = parseJaTaskIDs(fields[idx])
		}

		tasks = append(tasks, task)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return tasks, nil
}

func isValidMaster(s string) bool {
	return s == "MASTER" || s == "SLAVE"
}

func looksLikeTaskRange(s string) bool {
	if s == "" {
		return false
	}
	// Task ranges always start with a digit (e.g. "116-1000:1", "42").
	return s[0] >= '0' && s[0] <= '9'
}

// parseJaTaskIDs parses task ID specifications into expanded task IDs.
// Supported formats:
//   - single ID: "42"
//   - range with step: "7-99:2"
//   - comma-separated IDs: "1,51,101"
func parseJaTaskIDs(s string) []int64 {
	if s == "" {
		return nil
	}

	if strings.Contains(s, ",") {
		var ids []int64
		for _, part := range strings.Split(s, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				return nil
			}
			ids = append(ids, int64(id))
		}
		return ids
	}

	parts := strings.SplitN(s, ":", 2)
	rangePart := parts[0]
	step := 1
	if len(parts) == 2 {
		step, _ = strconv.Atoi(parts[1])
		if step <= 0 {
			step = 1
		}
	}

	bounds := strings.SplitN(rangePart, "-", 2)
	start, err := strconv.Atoi(bounds[0])
	if err != nil {
		return nil
	}
	end := start
	if len(bounds) == 2 {
		end, err = strconv.Atoi(bounds[1])
		if err != nil {
			return nil
		}
	}

	var ids []int64
	for i := start; i <= end; i += step {
		ids = append(ids, int64(i))
	}
	return ids
}

// ParseQstatFullExtendedOutput parses qstat -f -ext output.
// The queue section headers are identical to qstat -f, but each job line
// carries the extended column set (ntckts, project, dept, cpu, mem, io,
// ticket columns, share, slots) without a queue column, since the queue is
// already known from the section header.
func ParseQstatFullExtendedOutput(out string) ([]FullQueueExtendedInfo, error) {
	lines := strings.Split(out, "\n")
	var results []FullQueueExtendedInfo
	var currentQueue *FullQueueExtendedInfo

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "####") {
			break
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "queuename") {
			continue
		}

		if isSeparatorLine(trimmed) {
			if currentQueue != nil {
				results = append(results, *currentQueue)
				currentQueue = nil
			}
			continue
		}

		if !startsWithWhitespace(line) {
			if currentQueue != nil {
				results = append(results, *currentQueue)
			}
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return nil, fmt.Errorf("invalid queue header format: %q", line)
			}
			parts := strings.Split(fields[2], "/")
			if len(parts) != 3 {
				return nil, fmt.Errorf("invalid resv/used/tot format: %q", line)
			}
			reserved, _ := strconv.Atoi(parts[0])
			used, _ := strconv.Atoi(parts[1])
			total, _ := strconv.Atoi(parts[2])
			loadAvg, _ := strconv.ParseFloat(fields[3], 64)
			currentQueue = &FullQueueExtendedInfo{
				QueueName: fields[0],
				QueueType: fields[1],
				Reserved:  reserved,
				Used:      used,
				Total:     total,
				LoadAvg:   loadAvg,
				Arch:      fields[4],
				Jobs:      []ExtendedJobInfo{},
			}
			if len(fields) > 5 {
				currentQueue.States = fields[5]
			}
		} else {
			if currentQueue == nil {
				return nil, fmt.Errorf("job info found without preceding queue header: %q", line)
			}
			job, err := parseFullExtendedJobLine(line, currentQueue.QueueName)
			if err != nil {
				return nil, err
			}
			currentQueue.Jobs = append(currentQueue.Jobs, job)
		}
	}

	if currentQueue != nil {
		results = append(results, *currentQueue)
	}
	return results, nil
}

// parseFullExtendedJobLine parses one indented job line from qstat -f -ext
// output. The format is the same as the flat qstat -ext job line but without
// the queue column, because the queue is provided by the enclosing section
// header.
//
// Format:
//
//	job_id prior ntckts name user project dept state cpu mem io tckts ovrts otckt ftckt stckt share slots [ja-task-id]
func parseFullExtendedJobLine(line, queueName string) (ExtendedJobInfo, error) {
	fields := strings.Fields(line)
	if len(fields) < 18 {
		return ExtendedJobInfo{}, fmt.Errorf("extended job line too short (%d fields): %q", len(fields), line)
	}
	info := ExtendedJobInfo{}
	var err error

	info.JobID, err = strconv.Atoi(fields[0])
	if err != nil {
		return info, fmt.Errorf("failed to parse job_id in extended job line %q: %w", line, err)
	}
	info.Priority, err = strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return info, fmt.Errorf("failed to parse prior in extended job line %q: %w", line, err)
	}
	info.Ntckts, err = strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return info, fmt.Errorf("failed to parse ntckts in extended job line %q: %w", line, err)
	}
	info.Name = fields[3]
	info.User = fields[4]
	info.Project = fields[5]
	info.Department = fields[6]
	info.State = fields[7]
	info.CPU = fields[8]
	info.Memory = parseFloat64OrZero(fields[9])
	info.IO = parseFloat64OrZero(fields[10])
	info.Tckts, _ = strconv.Atoi(fields[11])
	info.Ovrts, _ = strconv.Atoi(fields[12])
	info.Otckt, _ = strconv.Atoi(fields[13])
	info.Ftckt, _ = strconv.Atoi(fields[14])
	info.Stckt, _ = strconv.Atoi(fields[15])
	info.Share, _ = strconv.ParseFloat(fields[16], 64)
	info.Queue = queueName
	info.Slots, _ = strconv.Atoi(fields[17])
	if len(fields) >= 19 {
		info.JATaskID = fields[18]
	}
	return info, nil
}

// ParseQstatFullOutput parses the v9.1 qstat -f output.
func ParseQstatFullOutput(out string) ([]FullQueueInfo, error) {
	lines := strings.Split(out, "\n")
	var results []FullQueueInfo
	var currentQueue *FullQueueInfo

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "####") {
			break
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "queuename") {
			continue
		}

		if isSeparatorLine(trimmed) {
			if currentQueue != nil {
				results = append(results, *currentQueue)
				currentQueue = nil
			}
			continue
		}

		if !startsWithWhitespace(line) {
			if currentQueue != nil {
				results = append(results, *currentQueue)
			}

			fields := strings.Fields(line)
			if len(fields) < 5 {
				return nil, fmt.Errorf("invalid queue header format: %q", line)
			}
			resvUsedTot := fields[2]
			parts := strings.Split(resvUsedTot, "/")
			if len(parts) != 3 {
				return nil, fmt.Errorf("invalid resv/used/tot format: %q", line)
			}
			reserved, _ := strconv.Atoi(parts[0])
			used, _ := strconv.Atoi(parts[1])
			total, _ := strconv.Atoi(parts[2])
			loadAvg, _ := strconv.ParseFloat(fields[3], 64)

			currentQueue = &FullQueueInfo{
				QueueName: fields[0],
				QueueType: fields[1],
				Reserved:  reserved,
				Used:      used,
				Total:     total,
				LoadAvg:   loadAvg,
				Arch:      fields[4],
				Jobs:      []JobInfo{},
			}
			if len(fields) > 5 {
				currentQueue.States = fields[5]
			}
		} else {
			if currentQueue == nil {
				return nil, fmt.Errorf("job info found without preceding queue header: %q", line)
			}
			job, err := parseFullOutputJobLine(line, currentQueue.QueueName)
			if err != nil {
				return nil, err
			}
			currentQueue.Jobs = append(currentQueue.Jobs, job)
		}
	}

	if currentQueue != nil {
		results = append(results, *currentQueue)
	}
	return results, nil
}

func parseFullOutputJobLine(line, queueName string) (JobInfo, error) {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return JobInfo{}, fmt.Errorf("invalid job line format: %q", line)
	}
	jobID, err := strconv.Atoi(fields[0])
	if err != nil {
		return JobInfo{}, fmt.Errorf("invalid job id in job line %q: %w", line, err)
	}
	score, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return JobInfo{}, fmt.Errorf("invalid score in job line %q: %w", line, err)
	}
	name := fields[2]
	owner := fields[3]
	state := fields[4]
	datetimeStr := fields[5] + " " + fields[6]
	jobTime, err := parseClusterDate(datetimeStr)
	if err != nil {
		return JobInfo{}, fmt.Errorf("failed to parse datetime in job line %q: %w", line, err)
	}

	var submitTime, startTime time.Time
	if strings.Contains(state, "r") {
		startTime = jobTime
	} else {
		submitTime = jobTime
	}

	slots, err := strconv.Atoi(fields[7])
	if err != nil {
		return JobInfo{}, fmt.Errorf("invalid slots in job line %q: %w", line, err)
	}

	var taskIDs []int64
	if len(fields) > 8 {
		taskIDs = parseJaTaskIDs(fields[8])
	}

	return JobInfo{
		JobID:      jobID,
		Priority:   score,
		Name:       name,
		User:       owner,
		State:      state,
		StartTime:  startTime,
		SubmitTime: submitTime,
		Queue:      queueName,
		Slots:      slots,
		JaTaskIDs:  taskIDs,
	}, nil
}

// ParseJobArrayTask parses qstat -g d output.
func ParseJobArrayTask(out string) ([]JobArrayTask, error) {
	lines := strings.Split(out, "\n")
	var tasks []JobArrayTask

	if len(lines) < 2 {
		return tasks, nil
	}

	headerDone := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "job-ID") || strings.HasPrefix(trimmed, "---") {
			headerDone = true
			continue
		}
		if !headerDone {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		jobID, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		priority, _ := strconv.ParseFloat(fields[1], 64)
		name := fields[2]
		user := fields[3]
		state := fields[4]
		timeStr := fields[5] + " " + fields[6]
		jobTime, err := parseClusterDate(timeStr)
		if err != nil {
			continue
		}

		var submitTime, startTime time.Time
		if strings.Contains(state, "qw") {
			submitTime = jobTime
		} else {
			startTime = jobTime
		}

		var slots int
		var taskIDs []int64
		var queue string

		if len(fields) > 7 {
			// If fields[7] is not a number, it is a queue name (running jobs).
			if slotsInt, err := strconv.Atoi(fields[7]); err != nil {
				queue = fields[7]
				if len(fields) > 8 {
					slots, _ = strconv.Atoi(fields[8])
				}
				if len(fields) > 9 {
					taskIDs = parseJaTaskIDs(fields[9])
				}
			} else {
				slots = slotsInt
				if len(fields) > 8 {
					taskIDs = parseJaTaskIDs(fields[8])
				}
			}
		}

		// Note: taskIDs stays nil when no ja-task-ID column is present
		// (bare flat-listing case). Earlier versions of this parser
		// synthesised []int64{0} here, but that caused downstream
		// converters to undo it and risked masking a real task ID of 0.
		ji := JobInfo{
			JobID:      jobID,
			Priority:   priority,
			Name:       name,
			User:       user,
			State:      state,
			SubmitTime: submitTime,
			StartTime:  startTime,
			Queue:      queue,
			Slots:      slots,
			JaTaskIDs:  taskIDs,
		}
		tasks = append(tasks, JobArrayTask{JobInfo: ji})
	}
	return tasks, nil
}

func startsWithWhitespace(s string) bool {
	for _, r := range s {
		return unicode.IsSpace(r)
	}
	return false
}

func isSeparatorLine(s string) bool {
	for _, r := range s {
		if r != '-' {
			return false
		}
	}
	return true
}
