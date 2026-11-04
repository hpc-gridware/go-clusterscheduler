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

package helper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// MaxStderrBytes bounds the stderr RunContext keeps. The messages that
// matter ("unable to send message to qmaster", "Following jobs do not
// exist") are short; a client that floods stderr must not grow memory.
const MaxStderrBytes = 64 * 1024

// CommandWaitDelay bounds how long RunContext waits after ctx has ended
// when a grandchild of the command still holds its pipes.
const CommandWaitDelay = time.Second

// RunContext runs executable with args until it exits or ctx ends, and
// returns its stdout and the first MaxStderrBytes of its stderr. It is the
// shared runner of the context-aware command wrappers; argument validation
// and dry-run handling stay with the callers.
//
// The command runs with env when env is not nil and with the process
// environment otherwise; extraEnv is appended in both cases (qalter and
// qconf add SGE_SINGLE_LINE=true). A failed run returns
// "failed to run command (<stderr>): <cause>", where the cause is ctx.Err()
// when ctx ended while the command ran and the exit error otherwise.
func RunContext(ctx context.Context, executable string, env, extraEnv []string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.WaitDelay = CommandWaitDelay
	switch {
	case len(extraEnv) > 0:
		if env == nil {
			env = os.Environ()
		}
		cmd.Env = append(append([]string{}, env...), extraEnv...)
	case env != nil:
		cmd.Env = env
	}
	var stdout bytes.Buffer
	stderr := NewCappedBuffer(MaxStderrBytes)
	cmd.Stdout = &stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return stdout.String(), stderr.String(),
			fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), err)
	}
	return stdout.String(), stderr.String(), nil
}
