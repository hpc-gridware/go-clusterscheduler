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

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ReservationBuilder provides a fluent API for constructing and submitting
// advance reservations, in the style of the qsub JobBuilder. It builds raw
// qrsub arguments and submits them via SubmitWithNativeSpecification, so it
// works with every QRSub version. Use Flag for the 9.1 binding switches.
//
// Setters cannot return errors; the first invalid argument is recorded and
// reported by Submit or Verify before qrsub runs.
type ReservationBuilder struct {
	submitter NativeSubmitter
	args      []string
	start     *time.Time
	end       *time.Time
	// hasDuration is set by Duration.
	hasDuration bool
	// structuralErr is a request qmaster would reject; it is always
	// reported. injectionErr is an argument injection check failure and
	// follows GCS_VALIDATION, like the options path.
	structuralErr error
	injectionErr  error
}

// NewReservationBuilder creates a ReservationBuilder which submits through
// the given client (any QRSub version).
func NewReservationBuilder(submitter NativeSubmitter) *ReservationBuilder {
	return &ReservationBuilder{submitter: submitter}
}

func (b *ReservationBuilder) failStructural(err error) {
	if err != nil && b.structuralErr == nil {
		b.structuralErr = err
	}
}

func (b *ReservationBuilder) failInjection(err error) {
	if err != nil && b.injectionErr == nil {
		b.injectionErr = err
	}
}

// Name sets the name of the AR (-N).
func (b *ReservationBuilder) Name(name string) *ReservationBuilder {
	b.failInjection(checkName(name))
	b.args = append(b.args, "-N", name)
	return b
}

// Account sets the account string of the AR (-A).
func (b *ReservationBuilder) Account(account string) *ReservationBuilder {
	b.failInjection(checkAccount(account))
	b.args = append(b.args, "-A", account)
	return b
}

// Start sets the start time of the AR (-a). Without it the AR starts now.
func (b *ReservationBuilder) Start(t time.Time) *ReservationBuilder {
	b.start = &t
	b.args = append(b.args, "-a", FormatDateTime(t))
	return b
}

// End sets the end time of the AR (-e).
func (b *ReservationBuilder) End(t time.Time) *ReservationBuilder {
	b.end = &t
	b.args = append(b.args, "-e", FormatDateTime(t))
	return b
}

// Duration sets the duration of the AR (-d), in whole seconds.
func (b *ReservationBuilder) Duration(d time.Duration) *ReservationBuilder {
	b.hasDuration = true
	b.failStructural(checkDuration(d))
	b.args = append(b.args, "-d", formatDuration(d))
	return b
}

// Resource adds a resource request (-l name=value).
func (b *ReservationBuilder) Resource(name, value string) *ReservationBuilder {
	b.failInjection(checkResource(name, value))
	b.args = append(b.args, "-l", name+"="+value)
	return b
}

// Queue sets the queues which may be reserved (-q).
func (b *ReservationBuilder) Queue(queues ...string) *ReservationBuilder {
	b.failInjection(checkList("queue", queues))
	b.args = append(b.args, "-q", strings.Join(queues, ","))
	return b
}

// MasterQueue sets the queues for the master task of a parallel AR
// (-masterq).
func (b *ReservationBuilder) MasterQueue(queues ...string) *ReservationBuilder {
	b.failInjection(checkList("master queue", queues))
	b.args = append(b.args, "-masterq", strings.Join(queues, ","))
	return b
}

// PE requests a parallel environment with a slot range such as "4",
// "2-8", "-8" or "4-" (-pe name slots).
func (b *ReservationBuilder) PE(name, slots string) *ReservationBuilder {
	b.failStructural(checkPESlots(slots))
	b.failInjection(checkObjectName("pe name", name))
	b.args = append(b.args, "-pe", name, slots)
	return b
}

// Checkpoint sets the checkpoint environment AR jobs may request (-ckpt).
func (b *ReservationBuilder) Checkpoint(name string) *ReservationBuilder {
	b.failInjection(checkObjectName("checkpoint", name))
	b.args = append(b.args, "-ckpt", name)
	return b
}

// Users sets the users and access lists (@name) allowed to submit jobs into
// the AR; a leading '!' excludes them (-u).
func (b *ReservationBuilder) Users(users ...string) *ReservationBuilder {
	b.failInjection(checkList("user", users))
	b.args = append(b.args, "-u", strings.Join(users, ","))
	return b
}

// MailOptions sets when mail is sent, a combination of b, e, a and n (-m).
func (b *ReservationBuilder) MailOptions(options string) *ReservationBuilder {
	b.failInjection(checkMailOptions(options))
	b.args = append(b.args, "-m", options)
	return b
}

// MailTo sets the mail recipients (-M).
func (b *ReservationBuilder) MailTo(addresses ...string) *ReservationBuilder {
	b.failInjection(checkList("mail address", addresses))
	b.args = append(b.args, "-M", strings.Join(addresses, ","))
	return b
}

// HardErrorHandling stops scheduling jobs into the AR while it is in error
// state (-he y).
func (b *ReservationBuilder) HardErrorHandling() *ReservationBuilder {
	b.args = append(b.args, "-he", "y")
	return b
}

// Immediate reserves only interactive queues (-now y).
func (b *ReservationBuilder) Immediate() *ReservationBuilder {
	b.args = append(b.args, "-now", "y")
	return b
}

// Flag appends one of the qrsub 9.1 core binding switches (-bamount,
// -bunit, -bfilter, -bsort, -bstart, -bstop, -bstrategy, -binstance) with
// its value, checked against the same grammar as the v9.1
// ReservationOptions. Every other switch has a named method; Flag refuses
// them so that their checks cannot be bypassed. qrsub 9.0 rejects the
// binding switches.
func (b *ReservationBuilder) Flag(name, value string) *ReservationBuilder {
	if _, ok := bindingSwitches[name]; !ok {
		b.failStructural(fmt.Errorf("flag %q is not allowed: Flag only takes the binding switches", name))
	}
	b.failInjection(ValidateBindingSwitch(name, value))
	b.args = append(b.args, name, value)
	return b
}

// Args returns a copy of the accumulated arguments.
func (b *ReservationBuilder) Args() []string {
	return slices.Clone(b.args)
}

// check reports the first recorded error before qrsub runs.
func (b *ReservationBuilder) check() error {
	if b.structuralErr != nil {
		return b.structuralErr
	}
	if err := checkTimeWindow(b.start, b.end, b.hasDuration); err != nil {
		return err
	}
	return validate.Enforce(b.injectionErr)
}

// Submit creates the AR and returns its id and the qrsub output.
func (b *ReservationBuilder) Submit(ctx context.Context) (int64, string, error) {
	if err := b.check(); err != nil {
		return 0, "", err
	}
	return SubmitArgs(ctx, b.submitter, b.args)
}

// Verify checks whether the AR could be granted right now without creating
// it (qrsub -w v).
func (b *ReservationBuilder) Verify(ctx context.Context) (bool, string, error) {
	if err := b.check(); err != nil {
		return false, "", err
	}
	return VerifyArgs(ctx, b.submitter, b.args)
}
