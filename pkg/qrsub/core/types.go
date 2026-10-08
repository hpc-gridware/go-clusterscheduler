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

// ToPtr returns a pointer to v. It simplifies filling ReservationOptions.
func ToPtr[T any](v T) *T {
	return &v
}

// ReservationOptions describes an AR for qrsub. Nil and empty fields are
// not passed to qrsub. Either EndTime or Duration must be set.
//
// The options file switch -@ is deliberately not available here; use
// SubmitWithNativeSpecification for it. The verify mode -w v is available
// through Verify.
type ReservationOptions struct {
	// StartTime of the AR (-a). Defaults to now.
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime of the AR (-e).
	EndTime *time.Time `json:"end_time,omitempty"`
	// Duration of the AR (-d), in whole seconds. JSON encodes it as
	// nanoseconds (time.Duration), e.g. 3600000000000 for one hour.
	Duration *time.Duration `json:"duration,omitempty"`
	// Name of the AR (-N).
	Name *string `json:"name,omitempty"`
	// Account string for the accounting record (-A).
	Account *string `json:"account,omitempty"`
	// Resources to reserve (-l name=value,...).
	Resources map[string]string `json:"resources,omitempty"`
	// Queues which may be reserved (-q).
	Queues []string `json:"queues,omitempty"`
	// MasterQueues for the master task of a parallel AR (-masterq).
	MasterQueues []string `json:"master_queues,omitempty"`
	// PEName and PESlots request a parallel environment (-pe name slots).
	// Both must be set together. PESlots is a slot range such as "4",
	// "2-8", "-8" or "4-".
	PEName  *string `json:"pe_name,omitempty"`
	PESlots *string `json:"pe_slots,omitempty"`
	// Checkpoint environment the AR jobs may request (-ckpt).
	Checkpoint *string `json:"checkpoint,omitempty"`
	// Users allowed to submit jobs into the AR (-u): user, @access_list,
	// and with a leading '!' users or lists which are not allowed. Defaults
	// to the AR owner.
	Users []string `json:"users,omitempty"`
	// MailOptions select when mail is sent (-m), a combination of the
	// letters b (begin), e (end), a (error) or n (none).
	MailOptions *string `json:"mail_options,omitempty"`
	// MailList receives the mails (-M user[@host]).
	MailList []string `json:"mail_list,omitempty"`
	// HardErrorHandling stops scheduling jobs into the AR while it is in
	// error state (-he y). The default is soft error handling.
	HardErrorHandling *bool `json:"hard_error_handling,omitempty"`
	// Immediate reserves only interactive queues (-now y).
	Immediate *bool `json:"immediate,omitempty"`
}
