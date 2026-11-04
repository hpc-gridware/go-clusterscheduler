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

package helper_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunContext", func() {
	const sh = "/bin/sh"

	It("returns stdout and stderr of a successful run", func() {
		stdout, stderr, err := helper.RunContext(context.Background(), sh, nil, nil,
			"-c", "echo out; echo err >&2")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(Equal("out\n"))
		Expect(stderr).To(Equal("err\n"))
	})

	It("names stderr and wraps the exit error of a failed run", func() {
		stdout, _, err := helper.RunContext(context.Background(), sh, nil, nil,
			"-c", "echo partial; echo '  broken  ' >&2; exit 3")
		Expect(stdout).To(Equal("partial\n"))
		Expect(err).To(MatchError(HavePrefix("failed to run command (broken): exit status 3")))
	})

	It("replaces the process environment with env", func() {
		os.Setenv("RUN_CONTEXT_PARENT", "parent")
		defer os.Unsetenv("RUN_CONTEXT_PARENT")
		stdout, _, err := helper.RunContext(context.Background(), sh, []string{"ONLY=1"}, nil,
			"-c", `echo "$ONLY:$RUN_CONTEXT_PARENT"`)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(Equal("1:\n"))
	})

	It("appends extraEnv to env, or to the process environment when env is nil", func() {
		os.Setenv("RUN_CONTEXT_PARENT", "parent")
		defer os.Unsetenv("RUN_CONTEXT_PARENT")
		script := `echo "$ONLY:$RUN_CONTEXT_PARENT:$EXTRA"`
		stdout, _, err := helper.RunContext(context.Background(), sh, []string{"ONLY=1"}, []string{"EXTRA=x"}, "-c", script)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(Equal("1::x\n"))
		stdout, _, err = helper.RunContext(context.Background(), sh, nil, []string{"EXTRA=x"}, "-c", script)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(Equal(":parent:x\n"))
	})

	It("reports the context error when ctx ends the command", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, _, err := helper.RunContext(ctx, sh, nil, nil, "-c", "sleep 10")
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "got: %v", err)
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	It("keeps at most MaxStderrBytes of stderr", func() {
		_, stderr, err := helper.RunContext(context.Background(), sh, nil, nil,
			"-c", "i=0; while [ $i -lt 2000 ]; do echo 0123456789012345678901234567890123456789 >&2; i=$((i+1)); done")
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(HaveLen(helper.MaxStderrBytes))
		Expect(strings.HasPrefix(stderr, "0123")).To(BeTrue())
	})
})
