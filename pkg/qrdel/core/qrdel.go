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

// Package core wraps the qrdel command, which deletes advance reservations
// (ARs).
//
// Deleting an AR first deletes all jobs bound to it. If such jobs are still
// running, qrdel only registers the AR for deletion (state "d") and
// returns successfully; the AR disappears once its jobs are gone.
//
// qrdel exits with an error as soon as one selector fails, even if it
// already deleted other ARs. The output returned together with the error
// lists every AR that was deleted; ParseDeleteOutput turns it into a
// DeleteResult. If none of the selected ARs exists the error wraps
// ErrNotFound.
//
// Checking an AR with qrstat and then deleting it is not atomic. To act only
// on one user's ARs, delete by owner and name, or by user, which qmaster
// restricts itself, instead of checking the owner first and deleting by id.
//
// Selecting ARs by name always names the owner. Without an owner, qmaster
// matches an exact name against the ARs of all users (CS-2863), so a
// manager would delete other users' ARs of the same name.
package core

import "context"

// QRDel is the interface for deleting advance reservations with qrdel.
type QRDel interface {
	// DeleteReservations deletes the ARs with the given ids. An id is not
	// restricted to the caller's ARs; qmaster only checks permissions.
	DeleteReservations(ctx context.Context, ids ...int64) (string, error)

	// DeleteReservationsByName deletes the ARs of owner whose name matches
	// one of the given names or patterns (* and ?; qmaster refuses
	// brackets and spaces). ARs with a name starting with '-' can only be
	// deleted by id. A caller who is not a manager can only delete its own
	// ARs.
	DeleteReservationsByName(ctx context.Context, owner string, names ...string) (string, error)

	// DeleteReservationsByUser deletes all ARs of the given users (-u). The
	// user "*" deletes the ARs of all users and requires manager rights.
	DeleteReservationsByUser(ctx context.Context, users ...string) (string, error)

	// NativeSpecification runs qrdel with the given raw arguments. It
	// applies only the control character check, so it must not be used
	// with untrusted input.
	NativeSpecification(ctx context.Context, args []string) (string, error)
}
