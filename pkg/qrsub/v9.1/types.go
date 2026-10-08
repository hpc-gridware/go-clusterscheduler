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

package qrsub

import (
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

// ReservationOptions describes an AR for qrsub 9.1. It embeds the options
// common to all versions and adds the core binding switches of 9.1.
//
// The binding type switch -btype is not offered: qrsub 9.1 rejects every
// -btype value (an inverted check in the qrsub option parser), so the AR
// would never be created. The 9.1 default binding type is "slot".
type ReservationOptions struct {
	core.ReservationOptions

	// BindingAmount is the number of binding units (-bamount).
	BindingAmount *int `json:"binding_amount,omitempty"`
	// BindingUnit is the binding unit (-bunit): T, ET, C, E, S, ES, X, EX,
	// Y, EY, N or EN.
	BindingUnit *string `json:"binding_unit,omitempty"`
	// BindingFilter masks binding units with a topology string (-bfilter).
	BindingFilter *string `json:"binding_filter,omitempty"`
	// BindingSort is the sort order of binding units (-bsort), a sequence of
	// S, s, C, c, E, e, N, n, X, x, Y, y.
	BindingSort *string `json:"binding_sort,omitempty"`
	// BindingStart is the start position for binding (-bstart), one of
	// S, s, C, c, E, e, N, n, X, x, Y, y.
	BindingStart *string `json:"binding_start,omitempty"`
	// BindingStop is the stop position for binding (-bstop), same letters
	// as BindingStart.
	BindingStop *string `json:"binding_stop,omitempty"`
	// BindingStrategy is the binding strategy (-bstrategy); qrsub 9.1
	// supports "packed".
	BindingStrategy *string `json:"binding_strategy,omitempty"`
	// BindingInstance is the instance applying the binding (-binstance):
	// set, env or pe.
	BindingInstance *string `json:"binding_instance,omitempty"`
}

// ToPtr returns a pointer to v.
func ToPtr[T any](v T) *T {
	return core.ToPtr(v)
}

// FormatDateTime formats t in the local time zone as qrsub date_time.
func FormatDateTime(t time.Time) string {
	return core.FormatDateTime(t)
}
