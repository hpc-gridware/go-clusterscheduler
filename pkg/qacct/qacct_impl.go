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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// CommandLineQAcct is an implementation of the QAcct interface using command-line.
type CommandLineQAcct struct {
	executable                string
	alterNativeAccountingFile string
	runCommand                func(args ...string) (string, error)
}

func NewDefaultRunCommand(executable, alternativeAccountingFile string) func(args ...string) (string, error) {
	return func(args ...string) (string, error) {
		if alternativeAccountingFile != "" {
			args = append([]string{"-f", alternativeAccountingFile}, args...)
		}
		cmd := exec.Command(executable, args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		// ensure that the column size is wide enough to avoid truncation
		//cmd.Env = append(cmd.Env, "LC_ALL=C")
		err := cmd.Run()
		if err != nil {
			return out.String(),
				fmt.Errorf("failed to run command (%s): %v", out.String(), err)
		}
		return out.String(), nil
	}
}

// NewCommandLineQAcct creates a new instance of CommandLineQAcct.
func NewCommandLineQAcct(executable string) (*CommandLineQAcct, error) {
	if executable == "" {
		executable = "qacct"
	}
	return &CommandLineQAcct{
		executable: executable,
		runCommand: NewDefaultRunCommand(executable, ""),
	}, nil
}

// WithAlternativeAccountingFile sets the alternative accounting file to be used by qacct.
// qacct -f <accountingFile> ...
func (c *CommandLineQAcct) WithAlternativeAccountingFile(accountingFile string) error {
	if _, err := os.Stat(accountingFile); os.IsNotExist(err) {
		return fmt.Errorf("accounting file does not exist: %s", accountingFile)
	}
	c.runCommand = NewDefaultRunCommand(c.executable, accountingFile)
	return nil
}

// WithDefaultAccountingFile sets the default accounting file to be used by qacct.
func (c *CommandLineQAcct) WithDefaultAccountingFile() {
	c.runCommand = NewDefaultRunCommand(c.executable, "")
}

// WithRunCommand sets the function which overrides the default qacct command execution.
// This is useful for testing.
func (c *CommandLineQAcct) WithRunCommand(fn func(args ...string) (string, error)) error {
	c.runCommand = fn
	return nil
}

// RunCommand executes the qacct command with the specified arguments.
func (c *CommandLineQAcct) RunCommand(args ...string) (string, error) {
	return c.runCommand(args...)
}

func parseUsage(output string) Usage {
	lines := strings.Split(output, "\n")
	// if first line has prefix "Total System Usage", remove
	// the first line
	if strings.HasPrefix(lines[0], "Total System Usage") {
		lines = lines[1:]
	}
	if len(lines) < 3 {
		return Usage{}
	}
	fields := strings.Fields(lines[2])
	if len(fields) < 7 {
		return Usage{}
	}
	return Usage{
		WallClock:  parseStringToFloat(fields[0]),
		UserTime:   parseStringToFloat(fields[1]),
		SystemTime: parseStringToFloat(fields[2]),
		CPU:        parseStringToFloat(fields[3]),
		Memory:     parseStringToFloat(fields[4]),
		IO:         parseStringToFloat(fields[5]),
		IOWait:     parseStringToFloat(fields[6]),
	}
}

func parseUsageAtIndex(line string, startIndex int) Usage {
	fields := strings.Fields(line)
	if len(fields) < 7+startIndex {
		return Usage{}
	}
	return Usage{
		WallClock:  parseStringToFloat(fields[startIndex+0]),
		UserTime:   parseStringToFloat(fields[startIndex+1]),
		SystemTime: parseStringToFloat(fields[startIndex+2]),
		CPU:        parseStringToFloat(fields[startIndex+3]),
		Memory:     parseStringToFloat(fields[startIndex+4]),
		IO:         parseStringToFloat(fields[startIndex+5]),
		IOWait:     parseStringToFloat(fields[startIndex+6]),
	}
}

func parseSingleFieldUsage(output, prefix string, startAtIndex int) []Usage {
	lines := strings.Split(output, "\n")
	var usages []Usage
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			continue
		}
		if len(line) > 0 && !strings.HasPrefix(line, "=") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			var usage Usage
			usage.WallClock = parseStringToFloat(fields[startAtIndex])
			usage.UserTime = parseStringToFloat(fields[startAtIndex+1])
			usage.SystemTime = parseStringToFloat(fields[startAtIndex+2])
			usage.CPU = parseStringToFloat(fields[startAtIndex+3])
			usage.Memory = parseStringToFloat(fields[startAtIndex+4])
			usage.IO = parseStringToFloat(fields[startAtIndex+5])
			usage.IOWait = parseStringToFloat(fields[startAtIndex+6])
			usages = append(usages, usage)
		}
	}
	return usages
}

// Helper functions to parse string to int and float
func parseStringToInt(s string) int64 {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseStringToFloat(s string) float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return 0.0
}

func (c *CommandLineQAcct) ListAdvanceReservations(arID string) ([]ReservationUsage, error) {
	args := []string{"-ar"}
	if arID != "" {
		args = append(args, arID)
	}
	out, err := c.RunCommand(args...)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "AR", 1)
	var reservations []ReservationUsage
	for _, usage := range usages {
		reservations = append(reservations,
			ReservationUsage{ArID: arID, Usage: usage})
	}
	return reservations, nil
}

func (c *CommandLineQAcct) JobsAccountedTo(accountString string) (Usage, error) {
	out, err := c.RunCommand("-A", accountString)
	if err != nil {
		return Usage{}, err
	}
	return parseUsage(out), nil
}

func (c *CommandLineQAcct) JobsStartedAfter(beginTime string) (Usage, error) {
	out, err := c.RunCommand("-b", beginTime)
	if err != nil {
		return Usage{}, err
	}
	return parseUsage(out), nil
}

func (c *CommandLineQAcct) JobsStartedBefore(endTime string) (Usage, error) {
	out, err := c.RunCommand("-e", endTime)
	if err != nil {
		return Usage{}, err
	}
	return parseUsage(out), nil
}

func (c *CommandLineQAcct) JobsStartedLastDays(days int) (Usage, error) {
	out, err := c.RunCommand("-d", strconv.Itoa(days))
	if err != nil {
		return Usage{}, err
	}
	usage := parseUsage(out)
	return usage, nil
}

func (c *CommandLineQAcct) ListDepartment(department string) ([]DepartmentUsage, error) {
	out, err := c.RunCommand("-D", department)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "DEPARTMENT", 1)
	var departments []DepartmentUsage
	for _, usage := range usages {
		departments = append(departments, DepartmentUsage{Department: department,
			Usage: usage})
	}
	return departments, nil
}

func (c *CommandLineQAcct) ListGroup(groupIDOrName string) ([]GroupUsage, error) {
	out, err := c.RunCommand("-g", groupIDOrName)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "GROUP", 1)
	var groups []GroupUsage
	for _, usage := range usages {
		groups = append(groups, GroupUsage{Group: groupIDOrName, Usage: usage})
	}
	return groups, nil
}

func (c *CommandLineQAcct) ListHost(host string) ([]HostUsage, error) {
	out, err := c.RunCommand("-h", host)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "HOST", 1)
	var hosts []HostUsage
	for _, usage := range usages {
		hosts = append(hosts, HostUsage{HostName: host, Usage: usage})
	}
	return hosts, nil
}

func (c *CommandLineQAcct) ListJobs(jobIDOrNameOrPattern string) ([]JobDetail, error) {
	out, err := c.RunCommand("-j", jobIDOrNameOrPattern)
	if err != nil {
		return nil, err
	}

	var jobDetails []JobDetail
	jobDetail, err := c.parseJobDetail(out)
	if err != nil {
		return nil, err
	}
	jobDetails = append(jobDetails, jobDetail)

	return jobDetails, nil
}

func (c *CommandLineQAcct) RequestComplexAttributes(attributes string) ([]JobInfo, error) {
	out, err := c.RunCommand("-l", attributes)
	if err != nil {
		return nil, err
	}

	return c.parseJobListOutput(out)
}

func (c *CommandLineQAcct) ListOwner(owner string) ([]OwnerUsage, error) {
	out, err := c.RunCommand("-o", owner)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "OWNER", 1)
	var owners []OwnerUsage
	for _, usage := range usages {
		owners = append(owners, OwnerUsage{OwnerName: owner, Usage: usage})
	}
	return owners, nil
}

func (c *CommandLineQAcct) ListParallelEnvironment(peName string) ([]PeUsage, error) {
	out, err := c.RunCommand("-pe", peName)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "PE", 1)
	var peUsages []PeUsage
	for _, usage := range usages {
		peUsages = append(peUsages, PeUsage{Pename: peName, Usage: usage})
	}
	return peUsages, nil
}

func (c *CommandLineQAcct) ListProject(project string) ([]ProjectUsage, error) {
	out, err := c.RunCommand("-P", project)
	if err != nil {
		return nil, err
	}

	usages := parseSingleFieldUsage(out, "PROJECT", 1)
	var projects []ProjectUsage
	for _, usage := range usages {
		projects = append(projects, ProjectUsage{ProjectName: project, Usage: usage})
	}
	return projects, nil
}

func (c *CommandLineQAcct) ListQueue(queue string) ([]QueueUsage, error) {
	out, err := c.RunCommand("-q", queue)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	if len(lines) <= 2 {
		return nil, fmt.Errorf("no usage information found for queues %s", queue)
	}
	var queueUsages []QueueUsage
	// host / queue and usage
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		host := fields[0]
		queue := fields[1]
		usage := parseUsageAtIndex(line, 2)
		queueUsages = append(queueUsages, QueueUsage{
			QueueName: queue, HostName: host, Usage: usage})
	}
	return queueUsages, nil
}

// ListJobUsageBySlots returns the Usage of all jobs that used
// the specified number of slots.
func (c *CommandLineQAcct) ListJobUsageBySlots(slots int) ([]SlotsUsage, error) {
	out, err := c.RunCommand("-slots", strconv.Itoa(slots))
	if err != nil {
		return nil, err
	}
	usage := parseSingleFieldUsage(out, "SLOTS", 1)
	if len(usage) == 0 {
		return nil, fmt.Errorf("no usage information found for slots %d", slots)
	}
	return []SlotsUsage{{Slots: int64(slots), Usage: usage[0]}}, nil
}

func (c *CommandLineQAcct) ListTasks(jobID, taskIDRange string) ([]TaskUsage, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID is required")
	}
	if taskIDRange == "" {
		return nil, fmt.Errorf("task ID range is required")
	}
	out, err := c.RunCommand("-j", jobID, "-t", taskIDRange)
	if err != nil {
		return nil, err
	}
	return c.parseTaskInfoOutput(out)
}

func (c *CommandLineQAcct) ShowHelp() (string, error) {
	out, err := c.RunCommand("-help")
	if err != nil {
		return "", err
	}
	return out, nil
}

// ShowTotalSystemUsage returns the total system usage (qacct).
func (c *CommandLineQAcct) ShowTotalSystemUsage() (Usage, error) {
	out, err := c.RunCommand()
	if err != nil {
		return Usage{}, err
	}
	usage := parseUsage(out)
	return usage, nil
}

func (c *CommandLineQAcct) ShowJobDetails(jobID int) (JobDetail, error) {
	out, err := c.RunCommand("-j", strconv.Itoa(jobID))
	if err != nil {
		return JobDetail{}, err
	}

	return c.parseJobDetail(out)
}

// More parsing helpers as needed to parse various formats
func (c *CommandLineQAcct) parseJobListOutput(output string) ([]JobInfo, error) {
	lines := strings.Split(output, "\n")
	var jobs []JobInfo
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 9 { // adjust based on expected number of fields
			continue
		}
		job := JobInfo{
			JobID:     parseStringToInt(fields[0]),
			Priority:  parseStringToFloat(fields[1]),
			JobName:   fields[2],
			User:      fields[3],
			State:     fields[4],
			StartTime: fields[5] + " " + fields[6], // Combining date and time field
			Queue:     fields[7],
			Slots:     parseStringToInt(fields[8]),
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

/*
==============================================================
qname                              all.q
hostname                           master
group                              root
owner                              root
project                            NONE
department                         defaultdepartment
jobname                            memhog
jobnumber                          22
taskid                             undefined
pe_taskid                          NONE
account                            accountingstring
priority                           0
qsub_time                          2024-08-18 08:13:15.547136
submit_cmd_line                    qsub -b y -A accountingstring memhog 1g
start_time                         2024-08-18 08:13:15.995298
end_time                           2024-08-18 08:13:16.551426
granted_pe                         NONE
slots                              1
failed                             0
exit_status                        0
ru_wallclock                       0
ru_utime                           0.336
ru_stime                           0.202
ru_maxrss                          1050068
ru_ixrss                           0
ru_ismrss                          0
ru_idrss                           0
ru_isrss                           0
ru_minflt                          1051
ru_majflt                          0
ru_nswap                           0
ru_inblock                         0
ru_oublock                         24
ru_msgsnd                          0
ru_msgrcv                          0
ru_nsignals                        0
ru_nvcsw                           200
ru_nivcsw                          0
wallclock                          1.004
cpu                                0.539
mem                                0.000
io                                 0.000
iow                                0.000
maxvmem                            0
maxrss                             0
arid                               undefined
*/

// Parse job details from output string
func (c *CommandLineQAcct) parseJobDetail(output string) (JobDetail, error) {
	lines := strings.Split(output, "\n")
	jobDetail := JobDetail{}

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "qname":
			jobDetail.QName = value
		case "hostname":
			jobDetail.HostName = value
		case "group":
			jobDetail.Group = value
		case "owner":
			jobDetail.Owner = value
		case "project":
			jobDetail.Project = value
		case "department":
			jobDetail.Department = value
		case "jobname":
			jobDetail.JobName = value
		case "jobnumber":
			jobDetail.JobNumber = parseStringToInt(value)
		case "taskid":
			jobDetail.TaskID = parseStringToInt(value)
		case "pe_taskid":
			jobDetail.PETaskID = value
		case "account":
			jobDetail.Account = value
		case "priority":
			jobDetail.Priority = parseStringToInt(value)
		case "qsub_time":
			jobDetail.QSubTime = value
		case "submit_cmd_line":
			jobDetail.SubmitCommandLine = value
		case "start_time":
			jobDetail.StartTime = value
		case "end_time":
			jobDetail.EndTime = value
		case "granted_pe":
			jobDetail.GrantedPE = value
		case "slots":
			jobDetail.Slots = parseStringToInt(value)
		case "failed":
			jobDetail.Failed = parseStringToInt(value)
		case "exit_status":
			jobDetail.ExitStatus = parseStringToInt(value)
		case "ru_wallclock":
			jobDetail.RuWallClock = parseStringToFloat(value)
		case "ru_utime":
			jobDetail.RuUTime = parseStringToFloat(value)
		case "ru_stime":
			jobDetail.RuSTime = parseStringToFloat(value)
		case "ru_maxrss":
			jobDetail.RuMaxRSS = parseStringToInt(value)
		case "ru_ixrss":
			jobDetail.RuIXRSS = parseStringToInt(value)
		case "ru_ismrss":
			jobDetail.RuISMRSS = parseStringToInt(value)
		case "ru_idrss":
			jobDetail.RuIDRSS = parseStringToInt(value)
		case "ru_isrss":
			jobDetail.RuISRss = parseStringToInt(value)
		case "ru_minflt":
			jobDetail.RuMinFlt = parseStringToInt(value)
		case "ru_majflt":
			jobDetail.RuMajFlt = parseStringToInt(value)
		case "ru_nswap":
			jobDetail.RuNSwap = parseStringToInt(value)
		case "ru_inblock":
			jobDetail.RuInBlock = parseStringToInt(value)
		case "ru_oublock":
			jobDetail.RuOuBlock = parseStringToInt(value)
		case "ru_msgsnd":
			jobDetail.RuMsgSend = parseStringToInt(value)
		case "ru_msgrcv":
			jobDetail.RuMsgRcv = parseStringToInt(value)
		case "ru_nsignals":
			jobDetail.RuNSignals = parseStringToInt(value)
		case "ru_nvcsw":
			jobDetail.RuNVCSw = parseStringToInt(value)
		case "ru_nivcsw":
			jobDetail.RuNiVCSw = parseStringToInt(value)
		case "wallclock":
			jobDetail.WallClock = parseStringToFloat(value)
		case "cpu":
			jobDetail.CPU = parseStringToFloat(value)
		case "mem":
			jobDetail.Memory = parseStringToInt(value)
		case "io":
			jobDetail.IO = parseStringToFloat(value)
		case "iow":
			jobDetail.IOWait = parseStringToFloat(value)
		case "maxvmem":
			jobDetail.MaxVMem = parseStringToInt(value)
		case "maxrss":
			jobDetail.MaxRSS = parseStringToInt(value)
		case "arid":
			jobDetail.ArID = value
		}
	}
	return jobDetail, nil
}

func (c *CommandLineQAcct) parseTaskInfoOutput(output string) ([]TaskUsage, error) {
	// tasks are separated by "==========..." lines
	separator := "=============================================================="
	outTasks := strings.Split(output, separator)
	var tasks []TaskUsage
	// remove the first element which is the header
	if len(outTasks) < 2 {
		return tasks, fmt.Errorf("no tasks found in output")
	}
	outTasks = outTasks[1:]

	for _, oTask := range outTasks {
		task, err := c.parseJobDetail(oTask)
		if err != nil {
			return tasks, err
		}
		tasks = append(tasks, TaskUsage{
			JobID:     task.JobNumber,
			TaskID:    task.TaskID,
			JobDetail: task,
		})
	}
	return tasks, nil
}
