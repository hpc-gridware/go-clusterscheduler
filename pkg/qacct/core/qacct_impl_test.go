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

package core_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

var _ = Describe("NativeSpecification errors", func() {

	// fakeQacct writes a script that stands in for qacct, so the spec needs
	// no cluster.
	fakeQacct := func(body string) *core.QAcctImpl {
		path := filepath.Join(GinkgoT().TempDir(), "qacct")
		Expect(os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700)).To(Succeed())
		q, err := core.NewCommandLineQAcct(core.CommandLineQAcctConfig{Executable: path})
		Expect(err).NotTo(HaveOccurred())
		return q
	}

	It("carries qacct's own message, not only the exit status", func() {
		// What qacct prints for "-e 196912312359": a blank line, the
		// message, the version banner, then its usage text.
		q := fakeQacct(`echo >&2
echo "Couldn't generate date from input. Perhaps a date before 1970 was specified." >&2
echo "OCS 9.1.5 (250826-0734)" >&2
echo "usage: qacct [options]" >&2
echo " [-ar [ar_id]]                     list [matching] advance reservations" >&2
exit 1`)

		_, err := q.NativeSpecification([]string{"-e", "196912312359"})
		Expect(err).To(MatchError("failed to run command (Couldn't generate date from input. " +
			"Perhaps a date before 1970 was specified.): exit status 1"))
	})

	It("does not take the version banner for a message", func() {
		// What qacct prints for an unknown option or a missing argument:
		// no message, only the banner and the usage text.
		q := fakeQacct(`echo "OCS 9.1.5 (250826-0734)" >&2
echo "usage: qacct [options]" >&2
echo " [-ar [ar_id]]                     list [matching] advance reservations" >&2
exit 1`)

		_, err := q.NativeSpecification([]string{"-nosuchoption"})
		Expect(err).To(MatchError("failed to run command (): exit status 1"))
	})

	It("keeps the exit error in the chain", func() {
		q := fakeQacct("echo 'error: job id 4711 not found' >&2\nexit 1")

		_, err := q.NativeSpecification([]string{"-j", "4711"})
		var exitErr *exec.ExitError
		Expect(errors.As(err, &exitErr)).To(BeTrue())
		Expect(exitErr.ExitCode()).To(Equal(1))
	})

	It("cuts an overlong stderr line", func() {
		q := fakeQacct("printf '%2000s\\n' x | tr ' ' y >&2\nexit 1")

		_, err := q.NativeSpecification([]string{"-j"})
		Expect(err).To(HaveOccurred())
		Expect(len(err.Error())).To(BeNumerically("<", 1200))
		Expect(strings.HasPrefix(err.Error(), "failed to run command (yyy")).To(BeTrue())
	})

	It("has an empty message part when qacct printed nothing", func() {
		q := fakeQacct("exit 3")

		_, err := q.NativeSpecification([]string{"-j"})
		Expect(err).To(MatchError("failed to run command (): exit status 3"))
	})
})
