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

// Regression tests for CS-2478: qconf -shgrp_resolved prints the whole
// membership space-separated on a single line, but ShowHostGroupResolved
// used to split on newlines and returned all members as one element.

package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core/internal/fakeqconf"
)

var _ = Describe("ShowHostGroupResolved parsing (CS-2478)", func() {
	BeforeEach(func() {
		if !fakeqconf.Available() {
			Skip("fakeqconf uses a bash script; skip on this platform")
		}
	})

	newQConfWithOutput := func(stdout string) core.QConf {
		f := fakeqconf.New(GinkgoT(), stdout, 0)
		DeferCleanup(f.Cleanup)
		qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{
			Executable: f.Path(),
		})
		Expect(err).NotTo(HaveOccurred())
		return qc
	}

	It("splits a space-separated single-line membership into hosts", func() {
		// This is the real qconf -shgrp_resolved output format.
		qc := newQConfWithOutput("master sim1 sim2 sim3\n")

		hosts, err := qc.ShowHostGroupResolved("@allhosts")
		Expect(err).NotTo(HaveOccurred())
		Expect(hosts).To(Equal([]string{"master", "sim1", "sim2", "sim3"}))
	})

	It("splits a newline-separated membership into hosts", func() {
		qc := newQConfWithOutput("master\nsim1\nsim2\nsim3\n")

		hosts, err := qc.ShowHostGroupResolved("@allhosts")
		Expect(err).NotTo(HaveOccurred())
		Expect(hosts).To(Equal([]string{"master", "sim1", "sim2", "sim3"}))
	})

	It("handles mixed separators, repeated whitespace, and CRLF", func() {
		qc := newQConfWithOutput("master sim1\r\n  sim2\t sim3 \n\n")

		hosts, err := qc.ShowHostGroupResolved("@allhosts")
		Expect(err).NotTo(HaveOccurred())
		Expect(hosts).To(Equal([]string{"master", "sim1", "sim2", "sim3"}))
	})

	It("returns an empty slice for empty output", func() {
		qc := newQConfWithOutput("")

		hosts, err := qc.ShowHostGroupResolved("@empty")
		Expect(err).NotTo(HaveOccurred())
		Expect(hosts).To(BeEmpty())
	})
})
