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

// Package core wraps the qrsub command, which creates advance reservations
// (ARs).
//
// ARs can be described with ReservationOptions (for structured callers) or
// built with the fluent ReservationBuilder. Both apply the same argument
// validation before qrsub runs.
//
// Things to know about qrsub:
//
//   - Start and end times are interpreted in the local time zone of the
//     qrsub process. Times are converted to time.Local before they are
//     formatted, so any time.Time denotes the intended instant.
//   - The caller must be a manager or a member of the "arusers" access list,
//     and qrsub must run on a submit host.
//   - qrsub reads default options from
//     $SGE_ROOT/$SGE_CELL/common/sge_ar_request and ~/.sge_ar_request
//     before its command line, and has no option to ignore them. Services
//     should run under an account without ~/.sge_ar_request.
//   - Verify (qrsub -w v) and Submit are not atomic: resources found by
//     Verify can be taken before Submit runs.
//   - If ctx is cancelled after qmaster accepted the request, the AR exists
//     although Submit returns an error. Use unique names to find it again
//     with qrstat.
//   - The output parsing relies on the English messages of qrsub.
package core

import "context"

// QRSub is the interface for creating advance reservations with qrsub.
type QRSub interface {
	// Submit creates an AR and returns its id and the qrsub output.
	Submit(ctx context.Context, opts ReservationOptions) (int64, string, error)

	// Verify checks whether the AR could be granted right now without
	// creating it (qrsub -w v). It returns false and no error if no
	// suitable resources were found. In dry run mode it returns false and
	// the command line as output.
	Verify(ctx context.Context, opts ReservationOptions) (bool, string, error)

	// SubmitWithNativeSpecification runs qrsub with the given raw arguments
	// and returns its output. It applies only the control character check,
	// so it must not be used with untrusted input. It is the only way to
	// use the options file switch -@.
	SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error)
}
