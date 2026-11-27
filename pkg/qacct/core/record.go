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
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/goccy/go-json"
)

// ErrNotAccountingRecord is returned for a line which is JSON but not an
// accounting record: it is not an object or has no job_number.
var ErrNotAccountingRecord = errors.New("not an accounting record")

// UnmarshalJSON reads an accounting record, and what JobDetail marshals to.
//
// It is tolerant on purpose. One accounting file holds records of every
// release the cluster ran, and a release may add a member or write one with
// another type (qmaster writes "arid" as a number, older versions of this
// package marshalled it as a string). A member which has no field, or whose
// value does not fit its field, is kept in Extra; it does not fail the record.
// Only a standard usage group which cannot be read does.
func (j *JobDetail) UnmarshalJSON(data []byte) error {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(data, &members); err != nil {
		return err
	}
	*j = JobDetail{}
	for key, raw := range members {
		if key == "usage" {
			// The one member which fails the record: a standard usage
			// group which cannot be read (see JobUsage.UnmarshalJSON).
			if err := json.Unmarshal(raw, &j.JobUsage); err != nil {
				return fmt.Errorf("usage: %w", err)
			}
			continue
		}
		if j.setMember(key, raw) {
			continue
		}
		if j.Extra == nil {
			j.Extra = map[string]json.RawMessage{}
		}
		j.Extra[key] = append(json.RawMessage(nil), raw...)
	}
	return nil
}

// setMember stores one member of a record in its field. It returns false
// when the member has no field or its value does not fit; the field is then
// left as it was. The value is decoded into a variable of its own first: the
// decoder writes what it has read so far before it fails, so "1e3" decoded
// straight into an integer field left 1 there.
func (j *JobDetail) setMember(key string, raw json.RawMessage) bool {
	text := func(field *string) bool {
		var v string
		if json.Unmarshal(raw, &v) != nil {
			return false
		}
		*field = v
		return true
	}
	number := func(field *int64) bool {
		var v int64
		if json.Unmarshal(raw, &v) != nil {
			return false
		}
		*field = v
		return true
	}
	switch key {
	case "qname":
		return text(&j.QName)
	case "hostname":
		return text(&j.HostName)
	case "group":
		return text(&j.Group)
	case "owner":
		return text(&j.Owner)
	case "project":
		return text(&j.Project)
	case "department":
		return text(&j.Department)
	case "job_name":
		return text(&j.JobName)
	case "job_number":
		return number(&j.JobNumber)
	case "task_number":
		return number(&j.TaskID)
	case "pe_taskid":
		return text(&j.PETaskID)
	case "account":
		return text(&j.Account)
	case "priority":
		return number(&j.Priority)
	case "submission_time":
		return number(&j.SubmitTime)
	case "submit_cmd_line":
		return text(&j.SubmitCommandLine)
	case "start_time":
		return number(&j.StartTime)
	case "end_time":
		return number(&j.EndTime)
	case "granted_pe":
		return text(&j.GrantedPE)
	case "slots":
		return number(&j.Slots)
	case "failed":
		return number(&j.Failed)
	case "exit_status":
		return number(&j.ExitStatus)
	case "ar_submission_time":
		return number(&j.ARSubmissionTime)
	case "category":
		return text(&j.Category)
	case "deleted_by":
		return text(&j.DeletedBy)
	case "arid":
		id, ok := numberOrString(raw)
		if ok {
			j.ArID = id
		}
		return ok
	}
	return false
}

// numberOrString reads a value which is an unsigned number in the
// accounting file and a string in what this package marshals.
func numberOrString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", false
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return "", false
		}
		return s, true
	}
	if _, err := strconv.ParseUint(string(raw), 10, 64); err != nil {
		return "", false
	}
	return string(raw), true
}

// ParseAccountingRecord parses one line of the accounting file. A line which
// is not a record (no JSON object, or no job_number) is an error; a record
// with members this version does not know is not (see JobDetail.Extra).
func ParseAccountingRecord(line []byte) (JobDetail, error) {
	var job JobDetail
	if err := json.Unmarshal(line, &job); err != nil {
		return JobDetail{}, err
	}
	// 0 is what a record without a job_number decodes to (or with one this
	// version cannot read); no job has that number.
	if job.JobNumber <= 0 {
		return JobDetail{}, ErrNotAccountingRecord
	}
	return job, nil
}

// leanRecord is what ReadOptions.Lean decodes: the members a list shows.
type leanRecord struct {
	JobNumber  int64  `json:"job_number"`
	TaskID     int64  `json:"task_number"`
	PETaskID   string `json:"pe_taskid"`
	JobName    string `json:"job_name"`
	Owner      string `json:"owner"`
	Group      string `json:"group"`
	Project    string `json:"project"`
	Department string `json:"department"`
	QName      string `json:"qname"`
	HostName   string `json:"hostname"`
	GrantedPE  string `json:"granted_pe"`
	Slots      int64  `json:"slots"`
	Priority   int64  `json:"priority"`
	SubmitTime int64  `json:"submission_time"`
	StartTime  int64  `json:"start_time"`
	EndTime    int64  `json:"end_time"`
	Failed     int64  `json:"failed"`
	ExitStatus int64  `json:"exit_status"`
	DeletedBy  string `json:"deleted_by"`
	Usage      struct {
		// The extended usage is "eusage", and "usage" in the records of
		// 9.0.0 to 9.0.5.
		EUsage *struct {
			WallClock float64 `json:"wallclock"`
		} `json:"eusage"`
		Usage *struct {
			WallClock float64 `json:"wallclock"`
		} `json:"usage"`
		RUsage struct {
			RuWallclock int64 `json:"ru_wallclock"`
		} `json:"rusage"`
	} `json:"usage"`
}

// parseLean decodes the members of leanRecord into job. It returns false
// for anything it is not sure of -- a line which does not decode, or has no
// job number -- and the caller then uses the full decoder, which decides.
func parseLean(line []byte, job *JobDetail) bool {
	var l leanRecord
	if err := json.Unmarshal(line, &l); err != nil || l.JobNumber <= 0 {
		return false
	}
	*job = JobDetail{
		JobNumber: l.JobNumber, TaskID: l.TaskID, PETaskID: l.PETaskID, JobName: l.JobName,
		Owner: l.Owner, Group: l.Group, Project: l.Project, Department: l.Department,
		QName: l.QName, HostName: l.HostName, GrantedPE: l.GrantedPE, Slots: l.Slots,
		Priority: l.Priority, SubmitTime: l.SubmitTime, StartTime: l.StartTime, EndTime: l.EndTime,
		Failed: l.Failed, ExitStatus: l.ExitStatus, DeletedBy: l.DeletedBy,
	}
	switch {
	case l.Usage.EUsage != nil:
		job.JobUsage.Usage.WallClock = l.Usage.EUsage.WallClock
	case l.Usage.Usage != nil:
		job.JobUsage.Usage.WallClock = l.Usage.Usage.WallClock
	}
	job.JobUsage.RUsage.RuWallclock = l.Usage.RUsage.RuWallclock
	return true
}

// UsageValue is one value of the usage object of an accounting record.
type UsageValue struct {
	// Group is "eusage", "rusage" or the name of a usage_patterns group.
	// The extended usage is "eusage" also for the records of 9.0.0 to
	// 9.0.5, which name it "usage".
	Group string `json:"group"`
	Name  string `json:"name"`
	// Number is the value when Numeric is true.
	Number  float64 `json:"number"`
	Numeric bool    `json:"numeric"`
	// Text is the value when it is not a number: a string as it is, true
	// and false as these words, an object or array as its JSON text.
	// qmaster writes numbers only today; this is for a release which
	// writes more.
	Text string `json:"text,omitempty"`
}

// extendedGroup is the name UsageValues reports the extended usage under.
const extendedGroup = "eusage"

// AccountingUsageValues returns EVERY value of the usage object of one line
// of the accounting file: the members JobUsage has fields for, the ones it
// has none for (a new member of eusage, for example), and the ones which
// are not numbers. Nothing is left out but null.
//
// The order is fixed: eusage, rusage, then the usage_patterns groups by
// name; the values of a group by name.
func AccountingUsageValues(line []byte) ([]UsageValue, error) {
	var record struct {
		Usage map[string]json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(line, &record); err != nil {
		return nil, err
	}
	_, hasExtended := record.Usage[extendedGroup]
	names := make([]string, 0, len(record.Usage))
	for name := range record.Usage {
		names = append(names, name)
	}
	rank := func(name string) int {
		switch {
		case name == extendedGroup, name == "usage" && !hasExtended:
			return 0
		case name == "rusage":
			return 1
		}
		return 2
	}
	sort.Slice(names, func(a, b int) bool {
		if ra, rb := rank(names[a]), rank(names[b]); ra != rb {
			return ra < rb
		}
		return names[a] < names[b]
	})

	var values []UsageValue
	for _, name := range names {
		group := name
		if rank(name) == 0 {
			group = extendedGroup
		}
		var members map[string]json.RawMessage
		if json.Unmarshal(record.Usage[name], &members) != nil {
			// Not an object: one value under the group's own name.
			if v, ok := usageValue(group, "", record.Usage[name]); ok {
				values = append(values, v)
			}
			continue
		}
		keys := make([]string, 0, len(members))
		for key := range members {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if v, ok := usageValue(group, key, members[key]); ok {
				values = append(values, v)
			}
		}
	}
	return values, nil
}

func usageValue(group, name string, raw json.RawMessage) (UsageValue, bool) {
	raw = bytes.TrimSpace(raw)
	v := UsageValue{Group: group, Name: name}
	switch {
	case len(raw) == 0, string(raw) == "null":
		return v, false
	case raw[0] == '"':
		if json.Unmarshal(raw, &v.Text) != nil {
			v.Text = string(raw)
		}
	case raw[0] == '{', raw[0] == '[', raw[0] == 't', raw[0] == 'f':
		v.Text = string(raw)
	default:
		number, err := strconv.ParseFloat(string(raw), 64)
		if err != nil {
			v.Text = string(raw)
			break
		}
		v.Number, v.Numeric = number, true
	}
	return v, true
}

// failedTexts are the scheduler's texts for the failed code of a record
// (source/common/execution_states.cc).
var failedTexts = map[int64]string{
	1:   "assumedly before job",
	2:   "on executing shepherd",
	3:   "before writing config",
	4:   "before writing pid",
	5:   "on reading config file",
	6:   "setting processor set",
	7:   "before prolog",
	8:   "in prolog",
	9:   "before pestart",
	10:  "in pestart",
	11:  "before job",
	12:  "before pestop",
	13:  "in pestop",
	14:  "before epilog",
	15:  "in epilog",
	16:  "releasing processor set",
	17:  "through signal",
	18:  "shepherd returned error",
	19:  "before writing exit_status",
	20:  "found unexpected error file",
	21:  "in recognizing job",
	22:  "removed manually",
	23:  "on reading the job's pids",
	24:  "migrating",
	25:  "rescheduling",
	26:  "opening input/output file",
	27:  "searching requested shell",
	28:  "changing into working directory",
	29:  "setting up AFS credentials",
	30:  "rescheduling on application error",
	36:  "checking configured daemons",
	37:  "qmaster enforced h_rt limit",
	38:  "setting the additional group id",
	39:  "in a systemd call",
	100: "assumedly after job",
}

// FailedText returns the scheduler's text for the failed code of an
// accounting record: where in the life of the job the failure happened
// ("in prolog", "assumedly after job"). It is "" for 0, a job which ran
// and ended by itself, and "invalid execution state" for a code the
// scheduler has no text for.
func FailedText(code int64) string {
	if code == 0 {
		return ""
	}
	if text, ok := failedTexts[code]; ok {
		return text
	}
	return "invalid execution state"
}
