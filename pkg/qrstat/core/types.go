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

import "time"

// ReservationState is the state letter qrstat prints for an AR. The letter
// is also the JSON value, for example "w" or "E".
type ReservationState string

const (
	// StateWaiting means the AR is waiting for its start time.
	StateWaiting ReservationState = "w"
	// StateRunning means the AR is active.
	StateRunning ReservationState = "r"
	// StateExited means the AR has ended.
	StateExited ReservationState = "x"
	// StateDeleted means the AR is being deleted.
	StateDeleted ReservationState = "d"
	// StateError means the AR is active but in error state.
	StateError ReservationState = "E"
	// StateWarning means the AR is waiting and in error state.
	StateWarning ReservationState = "W"
	// StateUnknown means the AR state is unknown.
	StateUnknown ReservationState = "u"
)

// Description returns a human readable description of the state, for
// display; the JSON value stays the letter.
func (s ReservationState) Description() string {
	switch s {
	case StateWaiting:
		return "waiting"
	case StateRunning:
		return "running"
	case StateExited:
		return "exited"
	case StateDeleted:
		return "deleted"
	case StateError:
		return "error"
	case StateWarning:
		return "warning"
	case StateUnknown:
		return "unknown"
	default:
		return string(s)
	}
}

// ListOptions selects the ARs listed by ListReservations.
type ListOptions struct {
	// Users limits the list to ARs of these users (qrstat -u). Empty means
	// the calling user, "*" means all users.
	Users []string `json:"users,omitempty"`
	// Explain adds the reasons for an error state (qrstat -explain).
	Explain bool `json:"explain,omitempty"`
}

// ReservationSummary is one row of the qrstat summary output.
type ReservationSummary struct {
	ID int64 `json:"id"`
	// Name is truncated to 10 characters by qrstat.
	Name string `json:"name"`
	// Owner is truncated to 12 characters by qrstat.
	Owner     string           `json:"owner"`
	State     ReservationState `json:"state"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	// Duration is encoded in JSON as nanoseconds.
	Duration time.Duration `json:"duration"`
	// Messages holds the error reasons shown with ListOptions.Explain.
	Messages []string `json:"messages,omitempty"`
}

// GrantedPE is the parallel environment granted to an AR.
type GrantedPE struct {
	Name string `json:"name"`
	// Range is the slot range the AR was submitted with, e.g. "4" or "2-8".
	Range string `json:"range"`
}

// GrantedResource is one resource map entry an AR holds on a host.
type GrantedResource struct {
	Name string `json:"name"`
	// Amount is the number of granted instances as printed by qrstat.
	Amount string `json:"amount"`
	// IDs are the granted instance identifiers. Empty when qrstat prints
	// only the amount.
	IDs []string `json:"ids,omitempty"`
}

// Reservation holds the details of an AR as shown by qrstat -ar.
// Attributes which do not apply to an AR are left empty.
type Reservation struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	Owner     string           `json:"owner"`
	State     ReservationState `json:"state"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	// Duration is encoded in JSON as nanoseconds.
	Duration       time.Duration `json:"duration"`
	Messages       []string      `json:"messages,omitempty"`
	SubmissionTime time.Time     `json:"submission_time"`
	Group          string        `json:"group"`
	Account        string        `json:"account"`
	// ResourceList holds the requested resources (qrsub -l).
	ResourceList map[string]string `json:"resource_list,omitempty"`
	// HardErrorHandling is true for ARs submitted with qrsub -he yes.
	HardErrorHandling bool `json:"hard_error_handling"`
	// ExecQueueList maps each reserved queue instance to its slot count.
	// 9.0 prints it as granted_slots_list, 9.1 as exec_queue_list.
	ExecQueueList              map[string]int `json:"exec_queue_list,omitempty"`
	GrantedParallelEnvironment *GrantedPE     `json:"granted_parallel_environment,omitempty"`
	// MasterQueueList holds the queues requested with qrsub -masterq.
	MasterQueueList []string `json:"master_queue_list,omitempty"`
	CheckpointName  string   `json:"checkpoint_name,omitempty"`
	MailOptions     string   `json:"mail_options,omitempty"`
	MailList        []string `json:"mail_list,omitempty"`
	// ACLList holds the users and access lists allowed to use the AR.
	ACLList []string `json:"acl_list,omitempty"`
	// XACLList holds the users and access lists not allowed to use the AR.
	XACLList []string `json:"xacl_list,omitempty"`

	// Binding is the core binding request of the AR (9.1 and later).
	Binding string `json:"binding,omitempty"`
	// ExecBindingList holds the binding of the AR per host (9.1 and later).
	ExecBindingList map[string]string `json:"exec_binding_list,omitempty"`
	// GrantedResourcesList holds the granted resource maps per host
	// (9.1.6 and later).
	GrantedResourcesList map[string][]GrantedResource `json:"granted_resource_list,omitempty"`

	// ExtraFields holds attributes this parser does not know, so newer
	// qrstat versions do not lose information.
	ExtraFields map[string]string `json:"extra_fields,omitempty"`
}
