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

// Package qrdel deletes advance reservations with qrdel. All types and
// functions are aliases of the shared core implementation; qrdel behaves the
// same in 9.0 and 9.1.
package qrdel

import "github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/core"

// QRDel is the interface for deleting advance reservations.
type QRDel = core.QRDel

// CommandLineQRDel implements QRDel by running the qrdel binary.
type CommandLineQRDel = core.CommandLineQRDel

// CommandLineQRDelConfig configures CommandLineQRDel.
type CommandLineQRDelConfig = core.CommandLineQRDelConfig

// CommandError is returned when qrdel exits with a non-zero exit code.
type CommandError = core.CommandError

// DeleteResult is the outcome of a qrdel call as reported in its output.
type DeleteResult = core.DeleteResult

// ErrNotFound is wrapped when none of the selected ARs exists.
var ErrNotFound = core.ErrNotFound

// ParseDeleteOutput collects the deleted, registered, missing and denied
// ARs from the output of qrdel.
func ParseDeleteOutput(output string) DeleteResult {
	return core.ParseDeleteOutput(output)
}

// NewCommandLineQRDel creates a QRDel client.
func NewCommandLineQRDel(config CommandLineQRDelConfig) (*CommandLineQRDel, error) {
	return core.NewCommandLineQRDel(config)
}

// ValidateUsers checks a user list for qrdel -u, independent of
// GCS_VALIDATION.
func ValidateUsers(users []string) error {
	return core.ValidateUsers(users)
}

// ValidateExactUsers checks a user list for qrdel -u without patterns,
// independent of GCS_VALIDATION.
func ValidateExactUsers(users []string) error {
	return core.ValidateExactUsers(users)
}

// ValidateReservationName checks an AR name or pattern for qrdel,
// independent of GCS_VALIDATION.
func ValidateReservationName(name string) error {
	return core.ValidateReservationName(name)
}
