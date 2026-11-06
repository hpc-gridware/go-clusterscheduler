/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
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
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ThreadAnswer is qmaster's answer to "qconf -at" or "qconf -kt" for a
// qmaster thread. qconf exits 0 for every one of them, a refusal included
// (verified on 9.1.7), so the answer is read from the output and never
// from the exit code.
type ThreadAnswer string

// The answers qmaster gives, with the text they are read from (9.1.7,
// LC_ALL=C, printed on stderr; "scheduler" is the only thread name -at
// accepts):
//
//	-at scheduler  "scheduler has been started"           ThreadStarted
//	-at scheduler  "scheduler thread is already running"  ThreadAlreadyRunning
//	-kt scheduler  "scheduler thread terminated"          ThreadTerminated
//	-kt scheduler  "scheduler thread is not running"      ThreadNotRunning
const (
	ThreadStarted        ThreadAnswer = "started"
	ThreadAlreadyRunning ThreadAnswer = "already_running"
	ThreadTerminated     ThreadAnswer = "terminated"
	ThreadNotRunning     ThreadAnswer = "not_running"
)

// SchedulerThread is the qmaster thread name of the scheduler.
const SchedulerThread = "scheduler"

// StartQmasterThreadContext runs "qconf -at <name>" and returns qmaster's
// answer. An answer that is none of the known ones (a permission refusal,
// an unknown thread name) is an error carrying qconf's text.
func (c *CommandLineQConf) StartQmasterThreadContext(ctx context.Context, env []string, name string) (ThreadAnswer, error) {
	return c.threadCommand(ctx, env, "-at", name)
}

// KillQmasterThreadContext runs "qconf -kt <name>" and returns qmaster's
// answer, like StartQmasterThreadContext.
func (c *CommandLineQConf) KillQmasterThreadContext(ctx context.Context, env []string, name string) (ThreadAnswer, error) {
	return c.threadCommand(ctx, env, "-kt", name)
}

func (c *CommandLineQConf) threadCommand(ctx context.Context, env []string, flag, name string) (ThreadAnswer, error) {
	if err := validate.Enforce(validate.Operand(name), validate.StrictObjectName(name)); err != nil {
		return "", err
	}
	// qmaster's answer is printed on STDERR, with exit 0 (9.1.7, all four
	// answers), so both streams are read.
	stdout, stderr, err := c.runContext(ctx, env, flag, name)
	out := stdout + stderr
	if answer, ok := ParseThreadAnswer(out); ok {
		return answer, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("qconf %s %s: unexpected answer %q", flag, name, strings.TrimSpace(out))
}

// ParseThreadAnswer reads qmaster's answer to "qconf -at" or "qconf -kt"
// (see the ThreadAnswer constants for the texts).
func ParseThreadAnswer(out string) (ThreadAnswer, bool) {
	switch s := strings.ToLower(out); {
	case strings.Contains(s, "has been started"):
		return ThreadStarted, true
	case strings.Contains(s, "is already running"):
		return ThreadAlreadyRunning, true
	case strings.Contains(s, "is not running"):
		return ThreadNotRunning, true
	case strings.Contains(s, "thread terminated"):
		return ThreadTerminated, true
	}
	return "", false
}

// SchedulerRunningContext reports whether the scheduler thread runs.
// "qconf -stl" lists a "scheduler" pool only while it does (verified on
// 9.1.7: after "qconf -kt scheduler" the pool is gone). It exits 1 in both
// states, so ShowThreadsContext reads the list, not the exit code; "qconf
// -sss" would answer a stopped scheduler with "no scheduling host defined",
// also exit 1, and a running one with the host.
func (c *CommandLineQConf) SchedulerRunningContext(ctx context.Context, env []string) (bool, error) {
	threads, err := c.ShowThreadsContext(ctx, env)
	if err != nil {
		return false, err
	}
	return SchedulerInThreads(threads), nil
}

// SchedulerInThreads reports whether a "qconf -stl" thread list holds a
// running scheduler pool.
func SchedulerInThreads(threads []ThreadInfo) bool {
	for _, t := range threads {
		if t.Name == SchedulerThread && t.Count > 0 {
			return true
		}
	}
	return false
}
