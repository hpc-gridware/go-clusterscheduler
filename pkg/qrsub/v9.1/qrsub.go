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

// Package qrsub creates advance reservations with qrsub on 9.1 clusters.
// It extends the shared core with the 9.1 core binding switches; see
// package core for the general behaviour of qrsub.
package qrsub

import (
	"context"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

// QRSub is the interface for creating advance reservations on 9.1.
type QRSub interface {
	// Submit creates an AR and returns its id and the qrsub output.
	Submit(ctx context.Context, opts ReservationOptions) (int64, string, error)

	// Verify checks whether the AR could be granted right now without
	// creating it (qrsub -w v). It returns false and no error if no
	// suitable resources were found.
	Verify(ctx context.Context, opts ReservationOptions) (bool, string, error)

	// SubmitWithNativeSpecification runs qrsub with the given raw
	// arguments. It must not be used with untrusted input.
	SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error)
}

// CommandLineQRSubConfig configures the qrsub client.
type CommandLineQRSubConfig = core.CommandLineQRSubConfig

// CommandError is returned when qrsub exits with a non-zero exit code.
type CommandError = core.CommandError

// ReservationBuilder provides a fluent API for creating ARs. Use Flag for
// the binding switches; it checks them with the same grammar as
// ReservationOptions.
type ReservationBuilder = core.ReservationBuilder

// NativeSubmitter runs qrsub with raw arguments; the clients of both
// versions implement it. Mock it to test code using ReservationBuilder.
type NativeSubmitter = core.NativeSubmitter

// NewReservationBuilder creates a ReservationBuilder for the given client.
func NewReservationBuilder(submitter NativeSubmitter) *ReservationBuilder {
	return core.NewReservationBuilder(submitter)
}

// IsLimitReached reports whether err says that the cluster wide AR limit
// is reached.
func IsLimitReached(err error) bool {
	return core.IsLimitReached(err)
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
