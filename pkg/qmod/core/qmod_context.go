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

// Context-aware qmod for callers that act on behalf of someone else (a web
// console, an agent) and must therefore bound the call, choose the child's
// environment, and read qmod's answer themselves.
//
// RunCommand is the opposite on all three counts: no deadline, the whole
// parent environment, and stdout and stderr in one buffer with any non-zero
// exit turned into an error. qmod's exit status does not say what happened
// per target anyway: it exits 0 when a queue instance "is already in the
// specified state", exits 1 for "invalid queue" and for "no permission to
// modify queue", and prints one answer line per queue instance either way
// (verified on 9.1.7). The answer lines are the result; this runner hands
// them over untouched together with the exit error.

import (
	"context"
	"fmt"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ContextRunner runs qmod under a context with a chosen environment.
// *CommandLineQMod implements it.
type ContextRunner interface {
	// RunContext runs qmod with args under ctx. When env is non-nil it is
	// the child's whole environment. stdout is returned in full, stderr
	// up to its first 64 KiB; a non-zero exit (or the context ending) also
	// returns an error of the form "failed to run command (<stderr>):
	// <cause>".
	RunContext(ctx context.Context, env []string, args ...string) (stdout, stderr string, err error)
	// ActionContext runs one action flag ("-d", "-e", "-c", "-s", "-us",
	// "-cq", ...) on targets, validating every target before it is
	// comma-joined, and honours the configured Force.
	ActionContext(ctx context.Context, env []string, flag string, targets []string) (stdout, stderr string, err error)
}

var _ ContextRunner = (*CommandLineQMod)(nil)

// RunContext implements ContextRunner. stderr is capped by
// helper.RunContext; stdout is not: its answer lines are the result, one per
// target, and a wildcard target on a large cluster prints tens of thousands
// of them.
func (c *CommandLineQMod) RunContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if c.config.DryRun {
		return fmt.Sprintf("Dry run: qmod %v", args), "", nil
	}
	return helper.RunContext(ctx, c.config.Executable, env, nil, args...)
}

// ActionContext implements ContextRunner.
func (c *CommandLineQMod) ActionContext(ctx context.Context, env []string, flag string, targets []string) (string, string, error) {
	args, err := c.actionArgs(flag, targets)
	if err != nil {
		return "", "", err
	}
	return c.RunContext(ctx, env, args...)
}
