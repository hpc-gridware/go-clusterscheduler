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

package qstat_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// schedulerJobInfoContexter is the 9.1 method set a caller reaches through
// a qstat.QStat value. The embedded 9.0 method of the same name returns the
// 9.0 type and does not satisfy it.
type schedulerJobInfoContexter interface {
	SchedulerJobInfoContext(ctx context.Context, env []string, ids ...string) ([]qstat.SchedulerJobInfo, error)
}

var _ = Describe("QStatImpl (9.1 client)", func() {
	var impl *qstat.QStatImpl
	abs := func(parts ...string) string {
		p, err := filepath.Abs(filepath.Join(parts...))
		Expect(err).NotTo(HaveOccurred())
		return p
	}
	env := func(kv ...string) []string {
		return append([]string{"PATH=" + os.Getenv("PATH")}, kv...)
	}

	BeforeEach(func() {
		var err error
		impl, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{
			Executable: abs("..", "v9.0", "testdata", "fake-qstat.sh"),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("parses qstat -j with the 9.1 parser, called directly", func() {
		jobs, err := impl.SchedulerJobInfoContext(context.Background(),
			env("FAKE_QSTAT_FILE="+abs("testdata", "qstat-j-held-9.1.7.txt")), "2")
		Expect(err).NotTo(HaveOccurred())
		Expect(jobs).To(HaveLen(1))
		Expect(jobs[0].JobNumber).To(Equal(2))
		Expect(jobs[0].CategoryID).To(Equal(1), "category_id is read by the 9.1 parser only")
		Expect(jobs[0].PendingTasks).To(Equal(1))
	})

	It("dispatches to the 9.1 method through a QStat interface value", func() {
		var q qstat.QStat = impl
		via, ok := q.(schedulerJobInfoContexter)
		Expect(ok).To(BeTrue(), "the 9.1 SchedulerJobInfoContext must shadow the 9.0 one")
		jobs, err := via.SchedulerJobInfoContext(context.Background(),
			env("FAKE_QSTAT_FILE="+abs("testdata", "qstat-j-held-9.1.7.txt")), "2")
		Expect(err).NotTo(HaveOccurred())
		Expect(jobs[0].CategoryID).To(Equal(1))
	})

	It("maps an unknown job to ErrNoSuchJob", func() {
		_, err := impl.SchedulerJobInfoContext(context.Background(),
			env("FAKE_QSTAT_EXIT=1", "FAKE_QSTAT_STDERR=Following jobs do not exist: \n99999999"), "99999999")
		Expect(errors.Is(err, qstat.ErrNoSuchJob)).To(BeTrue(), "got: %v", err)
	})

	It("fails to construct when the executable cannot be found", func() {
		bad, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{
			Executable: filepath.Join(GinkgoT().TempDir(), "no-such-qstat"),
		})
		Expect(err).To(MatchError(ContainSubstring("executable not found")))
		Expect(bad).To(BeNil())
	})
})
