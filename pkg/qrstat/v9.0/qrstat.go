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

// Package qrstat queries advance reservations with qrstat. All types and
// functions are aliases of the shared core implementation, which parses the
// qrstat output of 9.0 and 9.1.
package qrstat

import "github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/core"

// QRStat is the interface for querying advance reservations.
type QRStat = core.QRStat

// CommandLineQRStat implements QRStat by running the qrstat binary.
type CommandLineQRStat = core.CommandLineQRStat

// CommandLineQRStatConfig configures CommandLineQRStat.
type CommandLineQRStatConfig = core.CommandLineQRStatConfig

// CommandError is returned when qrstat exits with a non-zero exit code.
type CommandError = core.CommandError

// ListOptions selects the ARs listed by ListReservations.
type ListOptions = core.ListOptions

// ReservationSummary is one row of the qrstat summary output.
type ReservationSummary = core.ReservationSummary

// Reservation holds the details of an AR as shown by qrstat -ar.
type Reservation = core.Reservation

// ReservationState is the state letter qrstat prints for an AR.
type ReservationState = core.ReservationState

// GrantedPE is the parallel environment granted to an AR.
type GrantedPE = core.GrantedPE

// GrantedResource is one resource map entry an AR holds on a host.
type GrantedResource = core.GrantedResource

// AR states, see core.ReservationState.
const (
	StateWaiting = core.StateWaiting
	StateRunning = core.StateRunning
	StateExited  = core.StateExited
	StateDeleted = core.StateDeleted
	StateError   = core.StateError
	StateWarning = core.StateWarning
	StateUnknown = core.StateUnknown
)

// ErrNotFound is wrapped by GetReservations when none of the requested ARs
// exists.
var ErrNotFound = core.ErrNotFound

// NewCommandLineQRStat creates a QRStat client.
func NewCommandLineQRStat(config CommandLineQRStatConfig) (*CommandLineQRStat, error) {
	return core.NewCommandLineQRStat(config)
}

// ParseSummary parses the qrstat summary output.
func ParseSummary(output string) ([]ReservationSummary, error) {
	return core.ParseSummary(output)
}

// ParseDetail parses the output of qrstat -ar.
func ParseDetail(output string) ([]Reservation, error) {
	return core.ParseDetail(output)
}

// ValidateUsers checks a user list for qrstat -u, independent of
// GCS_VALIDATION.
func ValidateUsers(users []string) error {
	return core.ValidateUsers(users)
}

// ValidateExactUsers checks a user list for qrstat -u without patterns,
// independent of GCS_VALIDATION.
func ValidateExactUsers(users []string) error {
	return core.ValidateExactUsers(users)
}
