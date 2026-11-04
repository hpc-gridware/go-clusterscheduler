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

package qalter

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
)

// The "qalter -w v|p" verification API of the core package, re-exported so
// callers of this package need not import core. CommandLineQAlter embeds
// core.CommandLineQAlter, so its RunContext and VerifyContext are promoted.

// ContextRunner runs qalter under a caller's context and environment.
type ContextRunner = core.ContextRunner

// Verifier asks qmaster why a job is pending ("qalter -w v|p").
type Verifier = core.Verifier

// VerifyMode selects which picture of the cluster qmaster verifies against.
type VerifyMode = core.VerifyMode

// VerifyOutcome is the result line qmaster prints last.
type VerifyOutcome = core.VerifyOutcome

// VerifyReasonKind names the condition a reason line reports.
type VerifyReasonKind = core.VerifyReasonKind

// VerifyTargetKind says what kind of object a reason's Target names.
type VerifyTargetKind = core.VerifyTargetKind

// VerifyReason is one parsed reason line.
type VerifyReason = core.VerifyReason

// VerifyResult is the parsed answer of one "qalter -w v|p" run.
type VerifyResult = core.VerifyResult

// VerifyOptions bounds what ParseVerifyOutput keeps.
type VerifyOptions = core.VerifyOptions

// Verify modes, see core.VerifyEmptyCluster and core.VerifyNow.
const (
	VerifyEmptyCluster = core.VerifyEmptyCluster
	VerifyNow          = core.VerifyNow
)

// Verify outcomes, see the core constants of the same names.
const (
	VerifyNoSuitableQueues    = core.VerifyNoSuitableQueues
	VerifyFoundSuitableQueues = core.VerifyFoundSuitableQueues
	VerifyFoundAssignment     = core.VerifyFoundAssignment
	VerifyAlreadyRunning      = core.VerifyAlreadyRunning
	VerifyOutcomeUnknown      = core.VerifyOutcomeUnknown
)

// Reason kinds, see the core constants of the same names.
const (
	VerifyReasonNotInHardQueueList         = core.VerifyReasonNotInHardQueueList
	VerifyReasonQueueInstanceNotAvailable  = core.VerifyReasonQueueInstanceNotAvailable
	VerifyReasonQueueDisabled              = core.VerifyReasonQueueDisabled
	VerifyReasonQueueSuspended             = core.VerifyReasonQueueSuspended
	VerifyReasonQueueFull                  = core.VerifyReasonQueueFull
	VerifyReasonLoadAlarm                  = core.VerifyReasonLoadAlarm
	VerifyReasonSuspendAlarm               = core.VerifyReasonSuspendAlarm
	VerifyReasonQueueType                  = core.VerifyReasonQueueType
	VerifyReasonNoPermission               = core.VerifyReasonNoPermission
	VerifyReasonProjectAccess              = core.VerifyReasonProjectAccess
	VerifyReasonForcedResourceNotRequested = core.VerifyReasonForcedResourceNotRequested
	VerifyReasonUnknownResource            = core.VerifyReasonUnknownResource
	VerifyReasonNonRequestableResource     = core.VerifyReasonNonRequestableResource
	VerifyReasonResourceInsufficient       = core.VerifyReasonResourceInsufficient
	VerifyReasonExclusiveInUse             = core.VerifyReasonExclusiveInUse
	VerifyReasonRQSLimit                   = core.VerifyReasonRQSLimit
	VerifyReasonPESlots                    = core.VerifyReasonPESlots
	VerifyReasonNotInQueueList             = core.VerifyReasonNotInQueueList
	VerifyReasonNotReservedByAR            = core.VerifyReasonNotReservedByAR
	VerifyReasonCalendar                   = core.VerifyReasonCalendar
	VerifyReasonHostCleanup                = core.VerifyReasonHostCleanup
	VerifyReasonUnknown                    = core.VerifyReasonUnknown
)

// Target kinds, see the core constants of the same names.
const (
	VerifyTargetGlobal             = core.VerifyTargetGlobal
	VerifyTargetHost               = core.VerifyTargetHost
	VerifyTargetClusterQueue       = core.VerifyTargetClusterQueue
	VerifyTargetQueueInstance      = core.VerifyTargetQueueInstance
	VerifyTargetPE                 = core.VerifyTargetPE
	VerifyTargetRQSRule            = core.VerifyTargetRQSRule
	VerifyTargetAdvanceReservation = core.VerifyTargetAdvanceReservation
	VerifyTargetJob                = core.VerifyTargetJob
)

// DefaultMaxReasons is the cap when VerifyOptions.MaxReasons is 0.
const DefaultMaxReasons = core.DefaultMaxReasons

// ErrNotPermitted is core.ErrNotPermitted: qmaster refused the verification
// for lack of rights.
var ErrNotPermitted = core.ErrNotPermitted

// ErrNoSuchJob is core.ErrNoSuchJob: qmaster does not know the job.
var ErrNoSuchJob = core.ErrNoSuchJob

var (
	_ ContextRunner = (*CommandLineQAlter)(nil)
	_ Verifier      = (*CommandLineQAlter)(nil)
)

// ParseVerifyOutput parses what "qalter -w <mode>" printed; see
// core.ParseVerifyOutput.
func ParseVerifyOutput(mode VerifyMode, stdout, stderr string, opts VerifyOptions) (*VerifyResult, error) {
	return core.ParseVerifyOutput(mode, stdout, stderr, opts)
}
