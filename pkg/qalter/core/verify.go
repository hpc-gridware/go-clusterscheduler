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

// Context-aware access to "qalter -w v|p <job>": the scheduler's own answer
// to "why is this job pending?".
//
// qmaster runs the dispatch algorithm for the job and prints one line per
// reason a queue instance, host, PE or resource quota rules the job out,
// followed by exactly one result line ("verification: ..."). The exit code
// is 1 both for "no suitable queues" and for a denial ("denied: ..."), and
// 0 otherwise, so the outcome is decided from the text, never from the exit
// code. The texts are the SCHEDD_INFO_* messages of libs/sched/msg_schedd.h
// and the MSG_JOB_VERIFY* messages of daemons/qmaster/msg_qmaster.h; the
// parser understands the English (LC_ALL=C) wording and keeps every line it
// does not recognise as a reason of kind VerifyReasonUnknown.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ContextRunner runs qalter under a caller's context and environment. It
// is deliberately not part of QAlter, so adapters that implement QAlter are
// not affected.
type ContextRunner interface {
	// RunContext runs qalter with args under ctx and returns stdout and
	// stderr separately. When env is non-nil it becomes the child's whole
	// environment; a nil env inherits the parent's. SGE_SINGLE_LINE=true is
	// appended in both cases, as RunCommand does. A non-zero exit returns
	// the captured output together with an error of the form "failed to
	// run command (<stderr>): <cause>"; when ctx ended the process, ctx.Err()
	// is the cause.
	RunContext(ctx context.Context, env []string, args ...string) (stdout, stderr string, err error)
}

// Verifier asks qmaster why a job is pending ("qalter -w v|p"). Like
// ContextRunner it is not part of QAlter.
type Verifier interface {
	// VerifyContext runs "qalter -w <mode> <job>" and parses the answer.
	// job must name exactly one job: a numeric id with an optional task
	// range ("42", "42.1-10:2"). Lists, wildcards and job names are
	// rejected before any process starts (see validate.SingleJob), because
	// qmaster would answer for every matching job and the answers would
	// merge into one result. The single job also bounds the output: one
	// line per queue instance, host, PE or rule at most, so it is read
	// into memory whole.
	//
	// Exit code 1 with a result line ("no suitable queues") is a regular
	// result, not an error. A denial returns ErrNotPermitted or
	// ErrNoSuchJob (wrapped), or a plain error for other denials, together
	// with the result that carries the denial text. A nil error always
	// comes with a non-nil result, also when ctx ends after qalter has
	// answered.
	VerifyContext(ctx context.Context, env []string, job string, mode VerifyMode, opts VerifyOptions) (*VerifyResult, error)
}

var (
	_ ContextRunner = (*CommandLineQAlter)(nil)
	_ Verifier      = (*CommandLineQAlter)(nil)
)

// ErrNotPermitted is returned when qmaster refuses the verification because
// the caller is neither owner of the job nor operator or manager, or the
// host is not an admin or submit host.
var ErrNotPermitted = errors.New("not permitted")

// permissionDenials are the parts of the qmaster denial texts that refuse a
// job modification for lack of rights: MSG_SGETEXT_MUST_BE_JOB_OWN_TO_SUS,
// MSG_SGETEXT_MUST_BE_MGR_TO_SS, MSG_SGETEXT_MUST_BE_OPR_TO_SS (msg_qmaster.h),
// the MSG_SGETEXT_MUSTBEMANAGER* family (msg_common.h) and the host checks
// MSG_SGETEXT_NOSUBMITORADMINHOST_S and MSG_SGETEXT_NOADMINHOST_S.
var permissionDenials = []string{
	" must be at least owner of job ",
	" must be manager",
	" must be operator to ",
	" is neither submit nor admin host",
	" is no admin host",
}

// ErrNoSuchJob is returned when qmaster does not know the job. It is the
// same value as helper.ErrNoSuchJob and the qstat packages' ErrNoSuchJob,
// so errors.Is matches across packages.
var ErrNoSuchJob = helper.ErrNoSuchJob

// VerifyMode selects which picture of the cluster qmaster verifies against.
type VerifyMode string

const (
	// VerifyEmptyCluster ("-w v") verifies against the static configuration
	// as if the cluster were empty: no load, no running jobs, no consumed
	// resources. A "no suitable queues" here is a configuration or request
	// problem that will never resolve by itself.
	VerifyEmptyCluster VerifyMode = "v"
	// VerifyNow ("-w p", poke) verifies against the current usage and
	// states: load alarms, consumed resources, suspended or disabled queue
	// instances, resource quota usage.
	VerifyNow VerifyMode = "p"
)

// VerifyOutcome is the result line qmaster prints last.
type VerifyOutcome string

const (
	// VerifyNoSuitableQueues: no queue instance can take the job.
	VerifyNoSuitableQueues VerifyOutcome = "no suitable queues"
	// VerifyFoundSuitableQueues: a sequential job could be dispatched.
	VerifyFoundSuitableQueues VerifyOutcome = "found suitable queues"
	// VerifyFoundAssignment: a parallel job could be dispatched;
	// VerifyResult.Slots holds the slot count of the assignment.
	VerifyFoundAssignment VerifyOutcome = "found assignment"
	// VerifyAlreadyRunning: qmaster counted no idle, unheld task, so it
	// verified nothing. Printed for running jobs, for held jobs and for
	// array jobs whose unheld tasks all run.
	VerifyAlreadyRunning VerifyOutcome = "already running"
	// VerifyOutcomeUnknown: no result line was found (denial, transport
	// error, dry run).
	VerifyOutcomeUnknown VerifyOutcome = "unknown"
)

// VerifyReasonKind names the condition a reason line reports. The set
// follows the SCHEDD_INFO_* messages that name a target or a resource;
// everything else is VerifyReasonUnknown and still carries its Text.
type VerifyReasonKind string

const (
	// VerifyReasonNotInHardQueueList: the job's hard -q list excludes the
	// cluster queue.
	VerifyReasonNotInHardQueueList VerifyReasonKind = "not in hard queue list"
	// VerifyReasonQueueInstanceNotAvailable: the queue instance is in an
	// unknown, error or calendar-off state, or its host is down.
	VerifyReasonQueueInstanceNotAvailable VerifyReasonKind = "queue instance not available"
	// VerifyReasonQueueDisabled: the queue instance is disabled.
	VerifyReasonQueueDisabled VerifyReasonKind = "queue instance disabled"
	// VerifyReasonQueueSuspended: the queue instance is suspended.
	VerifyReasonQueueSuspended VerifyReasonKind = "queue instance suspended"
	// VerifyReasonQueueFull: the queue instance has no free slot.
	VerifyReasonQueueFull VerifyReasonKind = "queue instance full"
	// VerifyReasonLoadAlarm: the queue instance is, or the job would put
	// it, in load alarm (load_thresholds).
	VerifyReasonLoadAlarm VerifyReasonKind = "load alarm"
	// VerifyReasonSuspendAlarm: the queue instance is in suspend alarm
	// (suspend_thresholds).
	VerifyReasonSuspendAlarm VerifyReasonKind = "suspend alarm"
	// VerifyReasonQueueType: the queue's type (batch, interactive,
	// parallel, checkpointing) or its requestability excludes the job.
	VerifyReasonQueueType VerifyReasonKind = "queue type"
	// VerifyReasonNoPermission: user_lists/xuser_lists of the target, or
	// the PE's or advance reservation's access lists, exclude the owner.
	VerifyReasonNoPermission VerifyReasonKind = "no permission"
	// VerifyReasonProjectAccess: projects/xprojects of the target exclude
	// the job's project (or the job has none).
	VerifyReasonProjectAccess VerifyReasonKind = "project access"
	// VerifyReasonForcedResourceNotRequested: the target defines a FORCED
	// complex the job does not request.
	VerifyReasonForcedResourceNotRequested VerifyReasonKind = "forced resource not requested"
	// VerifyReasonUnknownResource: the requested complex is not defined
	// on the target.
	VerifyReasonUnknownResource VerifyReasonKind = "unknown resource"
	// VerifyReasonNonRequestableResource: the complex is defined but
	// requestable NO.
	VerifyReasonNonRequestableResource VerifyReasonKind = "non requestable resource"
	// VerifyReasonResourceInsufficient: the target offers less than the
	// job requests ("it offers only <dominance>:<resource>=<available>").
	VerifyReasonResourceInsufficient VerifyReasonKind = "resource insufficient"
	// VerifyReasonExclusiveInUse: an EXCL complex of the target is held by
	// another job.
	VerifyReasonExclusiveInUse VerifyReasonKind = "exclusive resource in use"
	// VerifyReasonRQSLimit: a resource quota rule limits the job.
	VerifyReasonRQSLimit VerifyReasonKind = "resource quota limit"
	// VerifyReasonPESlots: the PE cannot provide a slot count in the job's
	// range, or its allocation rule does not fit.
	VerifyReasonPESlots VerifyReasonKind = "pe slots"
	// VerifyReasonNotInQueueList: the queue's pe_list or ckpt_list lacks
	// the requested object (Resource names it).
	VerifyReasonNotInQueueList VerifyReasonKind = "not in queue list"
	// VerifyReasonNotReservedByAR: the job requests an advance reservation
	// that did not reserve the target.
	VerifyReasonNotReservedByAR VerifyReasonKind = "not reserved by advance reservation"
	// VerifyReasonCalendar: a calendar disables the queue instance before
	// the job's runtime would end.
	VerifyReasonCalendar VerifyReasonKind = "calendar"
	// VerifyReasonHostCleanup: a previous run of the job is still being
	// cleaned up on the host.
	VerifyReasonHostCleanup VerifyReasonKind = "host cleanup"
	// VerifyReasonUnknown: a line the parser does not recognise. Text is
	// kept verbatim.
	VerifyReasonUnknown VerifyReasonKind = "unknown"
)

// VerifyTargetKind says what kind of object a reason's Target names.
type VerifyTargetKind string

const (
	// VerifyTargetGlobal is the global host ("global").
	VerifyTargetGlobal VerifyTargetKind = "global"
	// VerifyTargetHost is an execution host.
	VerifyTargetHost VerifyTargetKind = "host"
	// VerifyTargetClusterQueue is a cluster queue ("all.q").
	VerifyTargetClusterQueue VerifyTargetKind = "cluster queue"
	// VerifyTargetQueueInstance is a queue instance ("all.q@node01").
	VerifyTargetQueueInstance VerifyTargetKind = "queue instance"
	// VerifyTargetPE is a parallel environment.
	VerifyTargetPE VerifyTargetKind = "pe"
	// VerifyTargetRQSRule is a resource quota rule ("set/1" or
	// "set/rulename").
	VerifyTargetRQSRule VerifyTargetKind = "rqs rule"
	// VerifyTargetAdvanceReservation is an advance reservation id.
	VerifyTargetAdvanceReservation VerifyTargetKind = "advance reservation"
	// VerifyTargetJob: the line names no target (job-wide conditions and
	// unknown lines).
	VerifyTargetJob VerifyTargetKind = "job"
)

// VerifyReason is one parsed reason line. Text is always set; the other
// fields are filled when the line names them.
type VerifyReason struct {
	// JobID is the id from the "Job N " prefix; 0 when the line had none.
	JobID uint64
	// Kind classifies the condition.
	Kind VerifyReasonKind
	// TargetKind says what Target names.
	TargetKind VerifyTargetKind
	// Target is the named object: "all.q", "all.q@node01", "node01",
	// "global", "smp", "max_slots_per_user/1", "2" (an AR id).
	Target string
	// Request is the job's request from the "(-l R=V)" prefix, "gpu=99".
	Request string
	// Resource is the complex the line names ("gpu", "slots", "matlab"),
	// or the PE or ckpt object for VerifyReasonNotInQueueList.
	Resource string
	// Dominance is the two-letter source of Available ("hc", "qc", "gc",
	// "qf", ...): layer g|h|q, then value v|f|l|L|c (template, fixed,
	// load, corrected load, consumable).
	Dominance string
	// Available is what the target offers, as printed ("4.000000").
	Available string
	// Limit is the resource quota limit scope ("carol/////"), or the rule
	// name when the line names only a rule and a queue instance.
	Limit string
	// Slots is the slot count the line mentions (PE offer, load alarm).
	Slots int
	// Text is the line without the "Job N " prefix, as printed.
	Text string
}

// VerifyResult is the parsed answer of one "qalter -w v|p" run.
type VerifyResult struct {
	// Mode is the mode the output was produced with.
	Mode VerifyMode
	// Outcome is the result line.
	Outcome VerifyOutcome
	// Slots is the assignment size for VerifyFoundAssignment.
	Slots int
	// Reasons holds the reason lines in output order: every line Keep
	// accepts, plus the other lines up to the MaxReasons cap.
	Reasons []VerifyReason
	// Dropped counts the reason lines left out because of MaxReasons.
	Dropped int
	// Denied is the "denied: ..." text when qmaster refused the request;
	// Outcome is then VerifyOutcomeUnknown.
	Denied string
}

// VerifyOptions bounds what ParseVerifyOutput keeps. A busy cluster prints
// one line per queue instance, so a job on a 10,000-host cluster may yield
// tens of thousands of lines.
type VerifyOptions struct {
	// MaxReasons caps the reasons Keep does not accept; 0 means
	// DefaultMaxReasons. Lines beyond the cap are counted in
	// VerifyResult.Dropped.
	MaxReasons int
	// Keep, when set, accepts the lines that are always kept. They are
	// never counted against MaxReasons, so a caller can keep every line
	// about the host or queue it asks about on a large cluster. Keep sees
	// the parsed reason and must not retain it.
	Keep func(VerifyReason) bool
}

// DefaultMaxReasons is the cap when VerifyOptions.MaxReasons is 0.
const DefaultMaxReasons = 2000

// RunContext implements ContextRunner.
func (c *CommandLineQAlter) RunContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if c.config.DryRun {
		fmt.Printf("Executing: %s %v\n", c.config.Executable, args)
		return "", "", nil
	}
	// helper.RunContext maps a context that ended while qalter ran to
	// ctx.Err() before DelayAfter starts, so the delay never changes the
	// reported cause.
	stdout, stderr, err := helper.RunContext(ctx, c.config.Executable, env, []string{"SGE_SINGLE_LINE=true"}, args...)
	if c.config.DelayAfter != 0 {
		select {
		case <-time.After(c.config.DelayAfter):
		case <-ctx.Done():
		}
	}
	return stdout, stderr, err
}

// VerifyContext implements Verifier.
func (c *CommandLineQAlter) VerifyContext(ctx context.Context, env []string, job string, mode VerifyMode, opts VerifyOptions) (*VerifyResult, error) {
	if mode != VerifyEmptyCluster && mode != VerifyNow {
		return nil, fmt.Errorf("verify mode %q: must be %q or %q", mode, VerifyEmptyCluster, VerifyNow)
	}
	// A contract of the call, not a policy: applied directly, not through
	// validate.Enforce.
	if err := validate.SingleJob(job); err != nil {
		return nil, err
	}
	stdout, stderr, runErr := c.RunContext(ctx, env, "-w", string(mode), job)
	// The output is incomplete only when the context ended the process. A
	// context that ends after qalter finished (during DelayAfter, say)
	// leaves a complete answer, which is parsed like any other.
	if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(runErr, ctxErr) {
		return nil, runErr
	}
	result, err := ParseVerifyOutput(mode, stdout, stderr, opts)
	if err != nil {
		return result, err
	}
	// Exit code 1 means "no suitable queues"; only a run without a result
	// line (qmaster unreachable, unknown option) is a failure.
	if runErr != nil && result.Outcome == VerifyOutcomeUnknown {
		return result, runErr
	}
	return result, nil
}

const (
	deniedPrefix = "denied: "
	resultPrefix = "verification: "
)

var (
	reJobLine     = regexp.MustCompile(`^Job (\d+) (.*)$`)
	reRequest     = regexp.MustCompile(`^\(-l ([^)]*)\) (.*)$`)
	reFoundSlots  = regexp.MustCompile(`^found possible assignment with (\d+) slots$`)
	reDominance   = regexp.MustCompile(`^([ghq][vflLc]):([^=]+)=(\S+)`)
	reNameValue   = regexp.MustCompile(`^([^=\s]+)=(\S+)`)
	reParenthesis = regexp.MustCompile(`\(([^)]*)\)`)
	reQuoted      = regexp.MustCompile(`"([^"]*)"`)
	reNumber      = regexp.MustCompile(`(\d+)`)
)

// ParseVerifyOutput parses what "qalter -w <mode>" printed on stdout and
// stderr. It is pure. A denial returns the result (Denied filled, Outcome
// VerifyOutcomeUnknown) together with an error carrying the denial text:
// ErrNoSuchJob for an unknown job, ErrNotPermitted for a missing right
// (owner, operator, manager, admin or submit host), a plain error for any
// other denial.
func ParseVerifyOutput(mode VerifyMode, stdout, stderr string, opts VerifyOptions) (*VerifyResult, error) {
	maxReasons := opts.MaxReasons
	if maxReasons <= 0 {
		maxReasons = DefaultMaxReasons
	}

	result := &VerifyResult{Mode: mode, Outcome: VerifyOutcomeUnknown, Reasons: []VerifyReason{}}
	regular := 0
	// stdout and stderr are walked in place, one after the other, instead
	// of concatenating and splitting them, so no copy of the output is made.
	for _, output := range [2]string{stdout, stderr} {
		for line := range strings.Lines(output) {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if text, ok := strings.CutPrefix(line, deniedPrefix); ok {
				result.Denied = text
				continue
			}
			if text, ok := strings.CutPrefix(line, resultPrefix); ok {
				result.Outcome, result.Slots = parseResultLine(text)
				continue
			}
			// Without Keep a line past the cap can only be dropped, so it
			// is counted without running the patterns over it.
			if opts.Keep == nil && regular >= maxReasons {
				result.Dropped++
				continue
			}
			reason := parseReasonLine(line)
			switch {
			case opts.Keep != nil && opts.Keep(reason):
				result.Reasons = append(result.Reasons, reason)
			case regular < maxReasons:
				result.Reasons = append(result.Reasons, reason)
				regular++
			default:
				result.Dropped++
			}
		}
	}

	if result.Denied != "" {
		result.Outcome = VerifyOutcomeUnknown
		return result, deniedError(result.Denied)
	}
	return result, nil
}

// deniedError maps the text of a "denied: " line to an error: ErrNoSuchJob
// for an unknown job, ErrNotPermitted for the permission texts, a plain
// error carrying the text for every other denial (a malformed request, an
// unsupported modification).
func deniedError(text string) error {
	if strings.Contains(text, "does not exist") {
		return fmt.Errorf("%w: %s", ErrNoSuchJob, text)
	}
	for _, permission := range permissionDenials {
		if strings.Contains(text, permission) {
			return fmt.Errorf("%w: %s", ErrNotPermitted, text)
		}
	}
	return fmt.Errorf("denied: %s", text)
}

// parseResultLine maps the text after "verification: " to an outcome.
func parseResultLine(text string) (VerifyOutcome, int) {
	switch text {
	case "no suitable queues":
		return VerifyNoSuitableQueues, 0
	case "found suitable queue(s)":
		return VerifyFoundSuitableQueues, 0
	case "job is already running":
		return VerifyAlreadyRunning, 0
	}
	if m := reFoundSlots.FindStringSubmatch(text); m != nil {
		slots, _ := strconv.Atoi(m[1])
		return VerifyFoundAssignment, slots
	}
	return VerifyOutcomeUnknown, 0
}

// parseReasonLine splits off the "Job N " prefix and the "(-l R=V) "
// request and classifies the rest. A line without the prefix is kept as
// VerifyReasonUnknown.
func parseReasonLine(line string) VerifyReason {
	r := VerifyReason{Kind: VerifyReasonUnknown, TargetKind: VerifyTargetJob, Text: line}
	m := reJobLine.FindStringSubmatch(line)
	if m == nil {
		return r
	}
	r.JobID, _ = strconv.ParseUint(m[1], 10, 64)
	r.Text = m[2]
	text := r.Text
	if m := reRequest.FindStringSubmatch(text); m != nil {
		r.Request = m[1]
		if name, _, ok := strings.Cut(m[1], "="); ok {
			r.Resource = name
		} else {
			r.Resource = m[1]
		}
		text = m[2]
	}
	// MSG_SCHEDD_INFO_CANNOTRUNRQS_SSS starts with "Job " itself.
	text = strings.TrimPrefix(text, "Job ")
	classifyReason(&r, text)
	return r
}

// reasonPattern is one SCHEDD_INFO_* message shape; fill completes the
// reason from the submatches.
type reasonPattern struct {
	re   *regexp.Regexp
	fill func(r *VerifyReason, m []string)
}

// droppedStateKinds maps the state of a "queue instance X dropped because it
// is <state>" line to its reason kind.
var droppedStateKinds = map[string]VerifyReasonKind{
	"temporarily not available": VerifyReasonQueueInstanceNotAvailable,
	"full":                      VerifyReasonQueueFull,
	"suspended":                 VerifyReasonQueueSuspended,
	"disabled":                  VerifyReasonQueueDisabled,
	"overloaded":                VerifyReasonLoadAlarm,
}

// reasonPatterns is tried in order; the first match wins. The generic
// "cannot run in queue/at host/globally because ..." shape comes last
// because the specific shapes above it share its prefix.
var reasonPatterns = []reasonPattern{
	{regexp.MustCompile(`^cannot run in queue "([^"]*)" because it is not contained in its hard queue list`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonNotInHardQueueList
			setQueueTarget(r, m[1], false)
		}},
	{regexp.MustCompile(`^queue instance "([^"]*)" dropped because it is (temporarily not available|full|suspended|disabled|overloaded)`),
		func(r *VerifyReason, m []string) {
			r.Kind = droppedStateKinds[m[2]]
			setQueueTarget(r, m[1], true)
		}},
	{regexp.MustCompile(`^queue instance "([^"]*)" is in suspend alarm`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonSuspendAlarm
			setQueueTarget(r, m[1], true)
		}},
	{regexp.MustCompile(`^\((\d+) slots\) would set queue instance "([^"]*)" in load alarm state`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonLoadAlarm
			r.Slots, _ = strconv.Atoi(m[1])
			setQueueTarget(r, m[2], true)
		}},
	{regexp.MustCompile(`^does not request globally 'forced' resource "([^"]*)"`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonForcedResourceNotRequested
			r.Resource = m[1]
			r.TargetKind, r.Target = VerifyTargetGlobal, "global"
		}},
	{regexp.MustCompile(`^does not request 'forced' resource "([^"]*)" of (queue instance|host) (\S+)`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonForcedResourceNotRequested
			r.Resource = m[1]
			if m[2] == "host" {
				setHostTarget(r, m[3])
			} else {
				setQueueTarget(r, m[3], true)
			}
		}},
	{regexp.MustCompile(`^cannot run because it exceeds limit "([^"]*)" in rule "([^"]*)"`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonRQSLimit
			r.Limit = m[1]
			r.TargetKind, r.Target = VerifyTargetRQSRule, m[2]
		}},
	{regexp.MustCompile(`^cannot run in queue instance "([^"]*)" because exceeds limit in rule (\S+)`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonRQSLimit
			r.Limit = m[2]
			setQueueTarget(r, m[1], true)
		}},
	{regexp.MustCompile(`^dropped because of invalid dynamical limit (\S+) in rule (\S+)`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonRQSLimit
			r.Limit = m[1]
			r.TargetKind, r.Target = VerifyTargetRQSRule, m[2]
		}},
	{regexp.MustCompile(`^has no permission for (cluster queue|queue instance|queue|host|pe|PE|advance_reservation) "?([^"\s]*)"?`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonNoPermission
			setNamedTarget(r, m[1], m[2])
		}},
	{regexp.MustCompile(`^\((?:project (\S+)|no project)\) (?:does not have the correct project to run in|is not allowed to run in) (cluster queue|queue instance|queue|host) "([^"]*)"`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonProjectAccess
			setNamedTarget(r, m[2], m[3])
		}},
	{regexp.MustCompile(`^cannot run because no access to pe "([^"]*)"`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonNoPermission
			r.TargetKind, r.Target = VerifyTargetPE, m[1]
		}},
	{regexp.MustCompile(`^cannot run in PE "([^"]*)" because it only offers (\d+) slots`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonPESlots
			r.TargetKind, r.Target = VerifyTargetPE, m[1]
			r.Slots, _ = strconv.Atoi(m[2])
		}},
	{regexp.MustCompile(`^cannot run because total slots of pe "([^"]*)" not in range of job`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonPESlots
			r.TargetKind, r.Target = VerifyTargetPE, m[1]
		}},
	{regexp.MustCompile(`^pe "([^"]*)" dropped because allocation rule is not suitable`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonPESlots
			r.TargetKind, r.Target = VerifyTargetPE, m[1]
		}},
	{regexp.MustCompile(`^cannot run on host "([^"]*)" until clean up of a previous run has finished`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonHostCleanup
			setHostTarget(r, m[1])
		}},
	{regexp.MustCompile(`^resource "([^"]*)" could not be granted on host "([^"]*)"`),
		func(r *VerifyReason, m []string) {
			r.Kind = VerifyReasonResourceInsufficient
			r.Resource = m[1]
			setHostTarget(r, m[2])
		}},
	{regexp.MustCompile(`^cannot run (?:in queue (instance )?"([^"]*)"|(?:at|on) host "([^"]*)"|globally) because (.*)$`),
		func(r *VerifyReason, m []string) {
			switch {
			case m[3] != "":
				setHostTarget(r, m[3])
			case m[2] != "" || m[1] != "":
				setQueueTarget(r, m[2], m[1] != "")
			default:
				r.TargetKind, r.Target = VerifyTargetGlobal, "global"
			}
			classifyBecause(r, m[4])
		}},
}

// classifyReason fills Kind and the target fields from the text after the
// job id and request prefixes.
func classifyReason(r *VerifyReason, text string) {
	for _, p := range reasonPatterns {
		if m := p.re.FindStringSubmatch(text); m != nil {
			p.fill(r, m)
			return
		}
	}
}

// classifyBecause interprets the reason after "cannot run in queue X
// because " (and the host and global variants). The resource reasons may
// be prefixed with "for default request " when a default value of a
// consumable, not an explicit request, fails.
func classifyBecause(r *VerifyReason, because string) {
	because = strings.TrimPrefix(because, "for default request ")
	switch {
	case strings.HasPrefix(because, "job requests unknown resource"):
		r.Kind = VerifyReasonUnknownResource
		setResourceFromParenthesis(r, because)
	case strings.HasPrefix(because, "job requests non requestable resource"):
		r.Kind = VerifyReasonNonRequestableResource
		rest := strings.TrimPrefix(because, "job requests non requestable resource ")
		r.Resource = strings.Trim(strings.TrimSpace(rest), `"`)
	case strings.HasPrefix(because, "exclusive resource"):
		r.Kind = VerifyReasonExclusiveInUse
		setResourceFromParenthesis(r, because)
	case strings.HasPrefix(because, "it offers only "):
		r.Kind = VerifyReasonResourceInsufficient
		offer := strings.TrimPrefix(because, "it offers only ")
		if m := reDominance.FindStringSubmatch(offer); m != nil {
			r.Dominance, r.Resource, r.Available = m[1], m[2], m[3]
		} else if m := reNameValue.FindStringSubmatch(offer); m != nil {
			r.Resource, r.Available = m[1], m[2]
		}
	case strings.HasPrefix(because, `it has "0" slots`):
		r.Kind = VerifyReasonResourceInsufficient
		r.Resource, r.Available = "slots", "0"
	case strings.HasPrefix(because, "consumable "):
		r.Kind = VerifyReasonResourceInsufficient
		setResourceFromQuotes(r, because)
	case strings.HasPrefix(because, "it is not of "),
		strings.HasPrefix(because, "the job is not parallel"),
		strings.HasPrefix(because, "queues are non requestable"):
		r.Kind = VerifyReasonQueueType
	case strings.HasPrefix(because, "it was not reserved by advance reservation"):
		r.Kind = VerifyReasonNotReservedByAR
	case strings.HasPrefix(because, "PE "), strings.HasPrefix(because, "ckpt object "):
		r.Kind = VerifyReasonNotInQueueList
		setResourceFromQuotes(r, because)
	case strings.HasPrefix(because, "the job runtime of"):
		r.Kind = VerifyReasonCalendar
		if m := reNumber.FindStringSubmatch(because); m != nil {
			r.Available = m[1]
		}
	}
}

func setResourceFromParenthesis(r *VerifyReason, text string) {
	if m := reParenthesis.FindStringSubmatch(text); m != nil {
		r.Resource = m[1]
	}
}

func setResourceFromQuotes(r *VerifyReason, text string) {
	if m := reQuoted.FindStringSubmatch(text); m != nil {
		r.Resource = m[1]
	}
}

// setQueueTarget classifies a name printed after "queue": a queue instance
// when the message said so or the name carries "@", the global host when it
// is "global" (the parallel dispatch path prints the global layer that
// way), a cluster queue otherwise.
func setQueueTarget(r *VerifyReason, name string, instance bool) {
	r.Target = name
	switch {
	case instance || strings.Contains(name, "@"):
		r.TargetKind = VerifyTargetQueueInstance
	case name == "global":
		r.TargetKind = VerifyTargetGlobal
	default:
		r.TargetKind = VerifyTargetClusterQueue
	}
}

func setHostTarget(r *VerifyReason, name string) {
	r.Target = name
	if name == "global" {
		r.TargetKind = VerifyTargetGlobal
	} else {
		r.TargetKind = VerifyTargetHost
	}
}

// setNamedTarget maps the object word the permission and project messages
// print ("cluster queue", "queue", "host", "pe", "advance_reservation") to
// a target kind. The patterns that call it admit only these words, so there
// is no default case.
func setNamedTarget(r *VerifyReason, word, name string) {
	switch word {
	case "cluster queue", "queue":
		setQueueTarget(r, name, false)
	case "queue instance":
		setQueueTarget(r, name, true)
	case "host":
		setHostTarget(r, name)
	case "pe", "PE":
		r.TargetKind, r.Target = VerifyTargetPE, name
	case "advance_reservation":
		r.TargetKind, r.Target = VerifyTargetAdvanceReservation, name
	}
}
