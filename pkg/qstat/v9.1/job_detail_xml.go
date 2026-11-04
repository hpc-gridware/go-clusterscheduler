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

package qstat

// The job DETAIL document: "qstat -j <id> -xml", the whole job object as
// qmaster holds it (captured from GCS 9.1.7, see testdata/qstat-j-xml-*).
//
//	<detailed_job_info>
//	  <djob_info><element> JB_* ... </element></djob_info>
//	  <messages> ... </messages>
//	</detailed_job_info>
//
// The decoder reads the token stream with an allowlist: every element it
// knows is typed, everything else is skipped, so a field added by a later
// release is never an error. List entries are wrapped in elements whose
// name varies (ulong_sublist, element, task_id_range, job_predecessors,
// qstat_l_requests, destin_ident_list, Events, ...); the decoder treats
// ANY child of a list element as one entry and never depends on the
// wrapper name. The subtrees a job's owner fills (environment, arguments,
// context) and the per-task usage can be large and are skipped.
//
// The <messages> block is not read at all: it may be large and, with
// schedd_job_info on, carries an element literally named "job ids" that no
// XML parser accepts. Decoding stops at its start tag; the rest of the
// input is drained unparsed so a qstat writing into a pipe can finish.
//
// An unknown job answers with a different root, <unknown_jobs>, whose body
// is not well-formed either (an element without a name). It is recognised
// from the root element alone, before anything else is read.

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	v90 "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

// JobDetailXML is the job object of "qstat -j <id> -xml". Time fields are
// the text as printed, "0" when unset; ParseXMLTime turns them into a
// time.Time. Numbers that qstat prints as something else read as 0.
type JobDetailXML struct {
	JobNumber uint64
	Name      string
	Owner     string
	Group     string
	// Groups are the supplementary groups of the owner (JB_grp_list).
	Groups     []string
	Project    string
	Department string
	Account    string
	// SubmissionTime, ExecutionTime (the "-a" start time) and Deadline as
	// printed: local ISO 8601 with microseconds, or "0" when unset.
	SubmissionTime string
	ExecutionTime  string
	Deadline       string
	// Priority is JB_priority: 1024 plus the "-p" value.
	Priority int64
	// Reserve is the "-R y" flag.
	Reserve bool
	// Restart is JB_restart: 0 not set (the queue's rerun decides),
	// 1 submitted with "-r y", 2 with "-r n".
	Restart int
	// Type is the JB_type bit set; 192 marks an array job, 64 a plain one
	// on 9.1.7.
	Type uint64
	// AR is the advance reservation the job was submitted into, 0 for none.
	AR         uint64
	CategoryID uint64
	// TaskConcurrency is the "-tc" limit, 0 for none.
	TaskConcurrency uint64
	// PE and PERange are the requested parallel environment and its slot
	// range; empty for a sequential job.
	PE         string
	PERange    []TaskRange
	Checkpoint string
	// Requests are the request sets in document order: one global set for
	// a sequential job, up to a global, a master and a slave set for a
	// parallel job (JB_request_set_list). A 9.0 document, which carries
	// the requests at job level, is mapped to one global set.
	Requests []RequestSet
	// Predecessors are the still-open "-hold_jid" dependencies by number.
	Predecessors []uint64
	// PredecessorRequests are the "-hold_jid" arguments as submitted (ids
	// or names, as text).
	PredecessorRequests []string
	// ArrayPredecessorRequests are the "-hold_jid_ad" arguments.
	ArrayPredecessorRequests []string
	// Successors are the jobs waiting on this one.
	Successors []uint64
	// Structure is the task id range of the job ("-t"), 1-1 for a plain
	// job (JB_ja_structure).
	Structure []TaskRange
	// PendingTasks are the not yet enrolled tasks without a hold
	// (JB_ja_n_h_ids). Enrolled tasks are in Tasks.
	PendingTasks []TaskRange
	// UserHold, OperatorHold, SystemHold and ArrayHold are the not yet
	// enrolled tasks under a hold of that kind (JB_ja_u/o/s/a_h_ids).
	UserHold     []TaskRange
	OperatorHold []TaskRange
	SystemHold   []TaskRange
	ArrayHold    []TaskRange
	// Tasks are the enrolled tasks (JB_ja_tasks): running, in error or
	// otherwise known to the execution side.
	Tasks []TaskDetailXML
}

// Scope is the scope of a RequestSet: which part of a parallel job it
// applies to.
type Scope int

// The request set scopes as JRS_scope prints them.
const (
	ScopeGlobal Scope = 0
	ScopeMaster Scope = 1
	ScopeSlave  Scope = 2
)

// RequestSet is one set of resource and queue requests of a job.
type RequestSet struct {
	Scope         Scope
	HardResources []ResourceRequest
	SoftResources []ResourceRequest
	HardQueues    []string
	SoftQueues    []string
	// AllocationRule overrides the PE's allocation rule for the slave
	// scope; empty when not set.
	AllocationRule string
	// IgnoreSlaveRequestsOnMasterHost is the slave-scope flag of the same
	// name.
	IgnoreSlaveRequestsOnMasterHost bool
}

// ResourceRequest is one "-l name=value" request. It is the {Name, Value}
// pair of the 9.0 stream, so a request reads the same from both documents.
type ResourceRequest = v90.ResourceRequestXML

// TaskRange is a range of array task ids: Min, Min+Step, ... up to Max, as
// qmaster stores it (RN_min, RN_max, RN_step).
type TaskRange struct {
	Min  uint64
	Max  uint64
	Step uint64
}

// TaskDetailXML is one enrolled task of a job (an entry of JB_ja_tasks).
type TaskDetailXML struct {
	Number uint64
	// State, Status and Hold are the JAT_state, JAT_status and JAT_hold
	// bit sets as printed; the bit values belong to the scheduler.
	State  uint64
	Status uint64
	Hold   uint64
	// StartTime is as printed, "0" when the task has not started.
	StartTime string
	// Restarted is JAT_job_restarted, non-zero after a reschedule.
	Restarted uint64
	// Messages are the execution side's messages for the task, such as
	// the reason it is in error (JAT_message_list, entities decoded).
	Messages []string
	// GrantedResources are the consumables granted to the task.
	GrantedResources []GrantedResourceXML
	// GrantedQueues are the queue instances and slots the task runs in.
	GrantedQueues []GrantedQueueXML
}

// GrantedResourceXML is one entry of JAT_granted_resources_list.
type GrantedResourceXML struct {
	Type   uint64
	Name   string
	Amount string
	Host   string
}

// GrantedQueueXML is one entry of JAT_granted_destin_identifier_list.
type GrantedQueueXML struct {
	Queue string
	Host  string
	Slots uint32
}

// DecodeJobDetailXML reads one "qstat -j <id> -xml" document from r. It
// returns ErrNoSuchJob for the <unknown_jobs> answer and
// ErrIncompleteDocument, together with what was decoded, when the input
// ends before the root close tag or the <messages> start tag. When the
// document holds several jobs (several ids were asked for) the first one is
// returned. On success the rest of r has been read.
func DecodeJobDetailXML(r io.Reader) (*JobDetailXML, error) {
	dec := xml.NewDecoder(r)
	// qstat writes the document in the process locale and declares no
	// encoding; let invalid sequences through as text rather than fail.
	dec.Strict = false
	root, err := nextStartElement(dec)
	if err != nil {
		return nil, jobDetailError(err)
	}
	switch root {
	case "unknown_jobs":
		return nil, ErrNoSuchJob
	case "detailed_job_info":
	default:
		return nil, fmt.Errorf("unexpected root element <%s> in qstat -j -xml output", root)
	}
	job := &JobDetailXML{}
	found := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return job, jobDetailError(err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "djob_info":
				err = forEachChild(dec, func(string) error {
					if found {
						return dec.Skip()
					}
					found = true
					return decodeJob(dec, job)
				})
			case "messages":
				if !found {
					return nil, ErrNoSuchJob
				}
				_, err = io.Copy(io.Discard, r)
				return job, err
			default:
				err = dec.Skip()
			}
			if err != nil {
				return job, jobDetailError(err)
			}
		case xml.EndElement:
			if !found {
				return nil, ErrNoSuchJob
			}
			return job, nil
		}
	}
}

// JobDetailXMLContext runs "qstat -j <jobID> -xml" under ctx and decodes the
// answer while qstat writes it. jobID is one numeric job id, optionally
// with a task range (accepted by qstat, but it does not narrow the answer);
// lists, wildcards and job names are rejected before qstat runs, whatever
// GCS_VALIDATION says, because each can select several jobs and only the
// first would be returned. ErrNoSuchJob
// is returned for an unknown job, whether qstat answers with the
// <unknown_jobs> document or with "Following jobs do not exist" on stderr.
// In dry-run mode an empty job is returned.
func (q *QStatImpl) JobDetailXMLContext(ctx context.Context, env []string, jobID string) (*JobDetailXML, error) {
	if err := validate.SingleJob(jobID); err != nil {
		return nil, err
	}
	job := &JobDetailXML{}
	_, err := q.StreamXML(ctx, env, []string{"-j", jobID, "-xml"}, func(r io.Reader) (Stats, error) {
		decoded, err := DecodeJobDetailXML(r)
		stats := Stats{Complete: err == nil}
		if decoded != nil {
			job = decoded
			stats.Decoded = 1
		}
		return stats, err
	})
	switch {
	case err == nil:
		return job, nil
	case errors.Is(err, ErrNoSuchJob), v90.IsNoSuchJobOutput(err.Error()):
		// The second form is the stderr answer, which StreamXML carries
		// in the message of its command error.
		return nil, fmt.Errorf("%w: %s", ErrNoSuchJob, jobID)
	}
	return nil, err
}

// jobDetailError maps a token-level error: the input ending early (plain
// EOF between elements, or the decoder's "unexpected EOF" inside one) is
// ErrIncompleteDocument; anything else is a malformed document.
func jobDetailError(err error) error {
	var syntax *xml.SyntaxError
	if errors.Is(err, io.EOF) || (errors.As(err, &syntax) && syntax.Msg == "unexpected EOF") {
		return ErrIncompleteDocument
	}
	return fmt.Errorf("malformed qstat -j -xml document: %w", err)
}

// nextStartElement returns the name of the next start element, skipping
// the declaration, comments and text before it.
func nextStartElement(dec *xml.Decoder) (string, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		if t, ok := tok.(xml.StartElement); ok {
			return t.Name.Local, nil
		}
	}
}

// forEachChild calls fn for every direct child element of the element whose
// start tag was just read, and consumes that element's end tag. fn must
// consume the child through its end tag (readText, dec.Skip or a nested
// forEachChild all do).
func forEachChild(dec *xml.Decoder, fn func(name string) error) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if err := fn(t.Name.Local); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// readText returns the character data of the element whose start tag was
// just read, as printed, and consumes its end tag. Nested elements are
// skipped.
func readText(dec *xml.Decoder) (string, error) {
	var sb strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.StartElement:
			if err := dec.Skip(); err != nil {
				return "", err
			}
		case xml.EndElement:
			return sb.String(), nil
		}
	}
}

// readUint reads an unsigned number; text that is not one reads as 0.
func readUint(dec *xml.Decoder) (uint64, error) {
	s, err := readText(dec)
	if err != nil {
		return 0, err
	}
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v, nil
}

// readInt reads a signed number; text that is not one reads as 0.
func readInt(dec *xml.Decoder) (int64, error) {
	s, err := readText(dec)
	if err != nil {
		return 0, err
	}
	v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return v, nil
}

// readBool reads a "true"/"false" element.
func readBool(dec *xml.Decoder) (bool, error) {
	s, err := readText(dec)
	return strings.TrimSpace(s) == "true", err
}

// Every list element below holds one wrapper element per entry
// (<JB_grp_list><element>...</element></JB_grp_list>); the readers walk the
// wrapper's children without looking at its name.

// readNames collects the text of the field element of every entry of a
// list ("JB_grp_list > wrapper > ST_name"); entries without it are left out.
func readNames(dec *xml.Decoder, field string) ([]string, error) {
	var out []string
	err := forEachChild(dec, func(string) error {
		return forEachChild(dec, func(name string) error {
			if name != field {
				return dec.Skip()
			}
			s, err := readText(dec)
			if err == nil && s != "" {
				out = append(out, s)
			}
			return err
		})
	})
	return out, err
}

// readNumbers collects the numeric field element of every entry of a list
// ("JB_jid_predecessor_list > wrapper > JRE_job_number").
func readNumbers(dec *xml.Decoder, field string) ([]uint64, error) {
	var out []uint64
	err := forEachChild(dec, func(string) error {
		return forEachChild(dec, func(name string) error {
			if name != field {
				return dec.Skip()
			}
			v, err := readUint(dec)
			if err == nil {
				out = append(out, v)
			}
			return err
		})
	})
	return out, err
}

// readRanges reads a list of RN_min/RN_max/RN_step entries.
func readRanges(dec *xml.Decoder) ([]TaskRange, error) {
	var out []TaskRange
	err := forEachChild(dec, func(string) error {
		var cur TaskRange
		err := forEachChild(dec, func(field string) error {
			var err error
			switch field {
			case "RN_min":
				cur.Min, err = readUint(dec)
			case "RN_max":
				cur.Max, err = readUint(dec)
			case "RN_step":
				cur.Step, err = readUint(dec)
			default:
				err = dec.Skip()
			}
			return err
		})
		out = append(out, cur)
		return err
	})
	return out, err
}

// readResourceRequests reads a list of CE_name/CE_stringval entries.
func readResourceRequests(dec *xml.Decoder) ([]ResourceRequest, error) {
	var out []ResourceRequest
	err := forEachChild(dec, func(string) error {
		var cur ResourceRequest
		err := forEachChild(dec, func(field string) error {
			var err error
			switch field {
			case "CE_name":
				cur.Name, err = readText(dec)
			case "CE_stringval":
				cur.Value, err = readText(dec)
			default:
				err = dec.Skip()
			}
			return err
		})
		out = append(out, cur)
		return err
	})
	return out, err
}

// decodeRequestSet reads one entry of JB_request_set_list.
func decodeRequestSet(dec *xml.Decoder) (RequestSet, error) {
	var set RequestSet
	err := forEachChild(dec, func(name string) error {
		var err error
		switch name {
		case "JRS_scope":
			var v uint64
			v, err = readUint(dec)
			set.Scope = Scope(v)
		case "JRS_hard_resource_list":
			set.HardResources, err = readResourceRequests(dec)
		case "JRS_soft_resource_list":
			set.SoftResources, err = readResourceRequests(dec)
		case "JRS_hard_queue_list":
			set.HardQueues, err = readNames(dec, "QR_name")
		case "JRS_soft_queue_list":
			set.SoftQueues, err = readNames(dec, "QR_name")
		case "JRS_allocation_rule":
			set.AllocationRule, err = readText(dec)
		case "JRS_ignore_slave_requests_on_master_host":
			set.IgnoreSlaveRequestsOnMasterHost, err = readBool(dec)
		default:
			err = dec.Skip()
		}
		return err
	})
	return set, err
}

// decodeTask reads one entry of JB_ja_tasks.
func decodeTask(dec *xml.Decoder) (TaskDetailXML, error) {
	var task TaskDetailXML
	err := forEachChild(dec, func(name string) error {
		var err error
		switch name {
		case "JAT_task_number":
			task.Number, err = readUint(dec)
		case "JAT_state":
			task.State, err = readUint(dec)
		case "JAT_status":
			task.Status, err = readUint(dec)
		case "JAT_hold":
			task.Hold, err = readUint(dec)
		case "JAT_start_time":
			task.StartTime, err = readText(dec)
		case "JAT_job_restarted":
			task.Restarted, err = readUint(dec)
		case "JAT_message_list":
			task.Messages, err = readNames(dec, "QIM_message")
		case "JAT_granted_resources_list":
			task.GrantedResources, err = readGrantedResources(dec)
		case "JAT_granted_destin_identifier_list":
			task.GrantedQueues, err = readGrantedQueues(dec)
		default:
			err = dec.Skip()
		}
		return err
	})
	return task, err
}

// readGrantedResources reads a list of GRU_* entries.
func readGrantedResources(dec *xml.Decoder) ([]GrantedResourceXML, error) {
	var out []GrantedResourceXML
	err := forEachChild(dec, func(string) error {
		var cur GrantedResourceXML
		err := forEachChild(dec, func(field string) error {
			var err error
			switch field {
			case "GRU_type":
				cur.Type, err = readUint(dec)
			case "GRU_name":
				cur.Name, err = readText(dec)
			case "GRU_amount":
				cur.Amount, err = readText(dec)
			case "GRU_host":
				cur.Host, err = readText(dec)
			default:
				err = dec.Skip()
			}
			return err
		})
		out = append(out, cur)
		return err
	})
	return out, err
}

// readGrantedQueues reads a list of JG_* entries.
func readGrantedQueues(dec *xml.Decoder) ([]GrantedQueueXML, error) {
	var out []GrantedQueueXML
	err := forEachChild(dec, func(string) error {
		var cur GrantedQueueXML
		err := forEachChild(dec, func(field string) error {
			var err error
			switch field {
			case "JG_qname":
				cur.Queue, err = readText(dec)
			case "JG_qhostname":
				cur.Host, err = readText(dec)
			case "JG_slots":
				var v uint64
				v, err = readUint(dec)
				cur.Slots = uint32(v)
			default:
				err = dec.Skip()
			}
			return err
		})
		out = append(out, cur)
		return err
	})
	return out, err
}

// decodeJob reads the job element (the entry of djob_info) into job.
func decodeJob(dec *xml.Decoder, job *JobDetailXML) error {
	// A 9.0 document carries the requests at job level; they become one
	// global request set.
	var legacy RequestSet
	hasLegacy := false
	err := forEachChild(dec, func(name string) error {
		var err error
		switch name {
		case "JB_job_number":
			job.JobNumber, err = readUint(dec)
		case "JB_job_name":
			job.Name, err = readText(dec)
		case "JB_owner":
			job.Owner, err = readText(dec)
		case "JB_group":
			job.Group, err = readText(dec)
		case "JB_grp_list":
			job.Groups, err = readNames(dec, "ST_name")
		case "JB_project":
			job.Project, err = readText(dec)
		case "JB_department":
			job.Department, err = readText(dec)
		case "JB_account":
			job.Account, err = readText(dec)
		case "JB_submission_time":
			job.SubmissionTime, err = readText(dec)
		case "JB_execution_time":
			job.ExecutionTime, err = readText(dec)
		case "JB_deadline":
			job.Deadline, err = readText(dec)
		case "JB_priority":
			job.Priority, err = readInt(dec)
		case "JB_reserve":
			job.Reserve, err = readBool(dec)
		case "JB_restart":
			var v int64
			v, err = readInt(dec)
			job.Restart = int(v)
		case "JB_type":
			job.Type, err = readUint(dec)
		case "JB_ar":
			job.AR, err = readUint(dec)
		case "JB_category_id":
			job.CategoryID, err = readUint(dec)
		case "JB_ja_task_concurrency":
			job.TaskConcurrency, err = readUint(dec)
		case "JB_pe":
			job.PE, err = readText(dec)
		case "JB_pe_range":
			job.PERange, err = readRanges(dec)
		case "JB_checkpoint_name":
			job.Checkpoint, err = readText(dec)
		case "JB_request_set_list":
			err = forEachChild(dec, func(string) error {
				set, err := decodeRequestSet(dec)
				if err != nil {
					return err
				}
				job.Requests = append(job.Requests, set)
				return nil
			})
		case "JB_hard_resource_list":
			hasLegacy = true
			legacy.HardResources, err = readResourceRequests(dec)
		case "JB_soft_resource_list":
			hasLegacy = true
			legacy.SoftResources, err = readResourceRequests(dec)
		case "JB_hard_queue_list":
			hasLegacy = true
			legacy.HardQueues, err = readNames(dec, "QR_name")
		case "JB_soft_queue_list":
			hasLegacy = true
			legacy.SoftQueues, err = readNames(dec, "QR_name")
		case "JB_jid_predecessor_list":
			job.Predecessors, err = readNumbers(dec, "JRE_job_number")
		case "JB_jid_request_list":
			job.PredecessorRequests, err = readNames(dec, "JRE_job_name")
		case "JB_ja_ad_request_list":
			job.ArrayPredecessorRequests, err = readNames(dec, "JRE_job_name")
		case "JB_jid_successor_list":
			job.Successors, err = readNumbers(dec, "JRE_job_number")
		case "JB_ja_structure":
			job.Structure, err = readRanges(dec)
		case "JB_ja_n_h_ids":
			job.PendingTasks, err = readRanges(dec)
		case "JB_ja_u_h_ids":
			job.UserHold, err = readRanges(dec)
		case "JB_ja_o_h_ids":
			job.OperatorHold, err = readRanges(dec)
		case "JB_ja_s_h_ids":
			job.SystemHold, err = readRanges(dec)
		case "JB_ja_a_h_ids":
			job.ArrayHold, err = readRanges(dec)
		case "JB_ja_tasks":
			err = forEachChild(dec, func(string) error {
				task, err := decodeTask(dec)
				if err != nil {
					return err
				}
				job.Tasks = append(job.Tasks, task)
				return nil
			})
		default:
			// JB_env_list, JB_context, JB_job_args, JB_script_file, the
			// path lists, the mail list and every field not listed above.
			err = dec.Skip()
		}
		return err
	})
	if hasLegacy {
		job.Requests = append(job.Requests, legacy)
	}
	return err
}
