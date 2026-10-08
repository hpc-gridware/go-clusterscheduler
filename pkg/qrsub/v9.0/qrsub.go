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

// Package qrsub creates advance reservations with qrsub on 9.0 clusters.
// All types and functions are aliases of the shared core implementation.
package qrsub

import (
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

// QRSub is the interface for creating advance reservations.
type QRSub = core.QRSub

// CommandLineQRSub implements QRSub by running the qrsub binary.
type CommandLineQRSub = core.CommandLineQRSub

// CommandLineQRSubConfig configures CommandLineQRSub.
type CommandLineQRSubConfig = core.CommandLineQRSubConfig

// CommandError is returned when qrsub exits with a non-zero exit code.
type CommandError = core.CommandError

// ReservationOptions describes an AR for qrsub.
type ReservationOptions = core.ReservationOptions

// ReservationBuilder provides a fluent API for creating ARs.
type ReservationBuilder = core.ReservationBuilder

// NewCommandLineQRSub creates a QRSub client.
func NewCommandLineQRSub(config CommandLineQRSubConfig) (*CommandLineQRSub, error) {
	return core.NewCommandLineQRSub(config)
}

// NativeSubmitter runs qrsub with raw arguments; the clients of both
// versions implement it. Mock it to test code using ReservationBuilder.
type NativeSubmitter = core.NativeSubmitter

// NewReservationBuilder creates a ReservationBuilder for the given client.
func NewReservationBuilder(submitter NativeSubmitter) *ReservationBuilder {
	return core.NewReservationBuilder(submitter)
}

// BuildQrsubArgs returns the qrsub arguments for opts.
func BuildQrsubArgs(opts ReservationOptions) ([]string, error) {
	return core.BuildQrsubArgs(opts)
}

// ValidateReservationOptions checks opts independent of GCS_VALIDATION.
func ValidateReservationOptions(opts ReservationOptions) error {
	return core.ValidateReservationOptions(opts)
}

// IsLimitReached reports whether err says that the cluster wide AR limit
// is reached.
func IsLimitReached(err error) bool {
	return core.IsLimitReached(err)
}

// FormatDateTime formats t in the local time zone as qrsub date_time.
func FormatDateTime(t time.Time) string {
	return core.FormatDateTime(t)
}

// ToPtr returns a pointer to v.
func ToPtr[T any](v T) *T {
	return core.ToPtr(v)
}

// ParseQrsubOutput returns the id from the qrsub success message, for
// output of SubmitWithNativeSpecification.
func ParseQrsubOutput(output string) (int64, error) {
	return core.ParseQrsubOutput(output)
}

// ParseVerifyOutput interprets the output of qrsub -w v.
func ParseVerifyOutput(output string) (bool, error) {
	return core.ParseVerifyOutput(output)
}
