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

// Package core wraps the qrstat command, which shows the advance
// reservations (ARs) of the cluster.
//
// The plain text output of qrstat 9.0 and 9.1 is parsed into typed structs.
// One parser handles both versions: 9.0 calls the reserved queue list
// granted_slots_list, 9.1 calls it exec_queue_list and adds binding and
// granted resource attributes.
//
// qrstat prints times in the local time zone of the qrstat process, which is
// the time zone of this Go process (time.Local), without a zone offset. A
// time within the repeated hour at the end of daylight saving time is
// ambiguous and is read as its first occurrence. The parsers rely on the
// English output of qrstat.
//
// Limits of the qrstat output, verified on 9.0 and 9.1:
//
//   - The summary truncates the name to 10 and the owner to 12 bytes (a cut
//     inside a multi-byte character is dropped), so summary rows must not be
//     used to decide about ownership; use GetReservations.
//   - The detail view (-ar, and -xml) prints at most 100 characters of the
//     name, although qrsub accepts up to 511.
//   - Leading and trailing spaces of names and accounts are not preserved.
//
// Like qrsub, qrstat first reads default options from
// $SGE_ROOT/$SGE_CELL/common/sge_qrstat and ~/.sge_qrstat. An option there
// which changes the output format (-xml) makes the typed methods fail with a
// parse error; services should run under an account without ~/.sge_qrstat.
package core

import "context"

// QRStat is the interface for querying advance reservations with qrstat.
type QRStat interface {
	// ListReservations returns one summary row per AR (qrstat [-u users]
	// [-explain]). Without users only the ARs of the calling user are
	// listed; the user "*" lists the ARs of all users. qrstat truncates the
	// name and the owner in this view; see the package documentation.
	ListReservations(ctx context.Context, opts ListOptions) ([]ReservationSummary, error)

	// GetReservations returns the full details of the given ARs
	// (qrstat -ar id,...). Ids which do not exist are left out of the
	// result. If none of them exists, the error wraps ErrNotFound.
	GetReservations(ctx context.Context, ids ...int64) ([]Reservation, error)

	// NativeSpecification runs qrstat with the given raw arguments and
	// returns its output. It applies only the control character check, so
	// it must not be used with untrusted input.
	NativeSpecification(ctx context.Context, args []string) (string, error)
}
