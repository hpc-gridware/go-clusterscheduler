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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// listCall is one of qconf's list commands and qconf's answer when the
// list is empty: "no <name> defined" on stderr, exit 1
// (MSG_QCONF_NOXDEFINED_S, show_object_list in ocs_qconf_parse.cc).
type listCall struct {
	name  string
	flag  string
	empty string
	call  func(*core.CommandLineQConf) ([]string, error)
}

// listCalls is every list a fresh cluster may have empty. Each one used
// to check for the cluster-queue message, so every empty list but the
// queues failed (calendars, PEs, quota sets, ...).
var listCalls = []listCall{
	{"ShowCalendars", "-scall", "no calendar defined", (*core.CommandLineQConf).ShowCalendars},
	{"ShowComplexEntries", "-scel", "no complex entry defined", (*core.CommandLineQConf).ShowComplexEntries},
	{"ShowCkptInterfaces", "-sckptl", "no ckpt interface definition defined", (*core.CommandLineQConf).ShowCkptInterfaces},
	{"ShowHostConfigurations", "-sconfl", "no config defined", (*core.CommandLineQConf).ShowHostConfigurations},
	{"ShowExecHosts", "-sel", "no execution host defined", (*core.CommandLineQConf).ShowExecHosts},
	{"ShowHostGroups", "-shgrpl", "no host group list defined", (*core.CommandLineQConf).ShowHostGroups},
	{"ShowResourceQuotaSets", "-srqsl", "no resource quota set list defined", (*core.CommandLineQConf).ShowResourceQuotaSets},
	{"ShowParallelEnvironments", "-spl", "no parallel environment defined", (*core.CommandLineQConf).ShowParallelEnvironments},
	{"ShowProjects", "-sprjl", "no project list defined", (*core.CommandLineQConf).ShowProjects},
	{"ShowClusterQueues", "-sql", "no cqueue list defined", (*core.CommandLineQConf).ShowClusterQueues},
	{"ShowUsers", "-suserl", "no user list defined", (*core.CommandLineQConf).ShowUsers},
	{"ShowUserSetLists", "-sul", "no userset list defined", (*core.CommandLineQConf).ShowUserSetLists},
	{"ShowOperators", "-so", "no operator defined", (*core.CommandLineQConf).ShowOperators},
	{"ShowSubmitHosts", "-ss", "no submit host defined", (*core.CommandLineQConf).ShowSubmitHosts},
}

// answeringQconf is a qconf that prints out (on stderr, exit 1) only for
// flag and fails loudly for any other command, so a list call that runs
// the wrong command cannot pass.
func answeringQconf(flag, out string, rc int) *core.CommandLineQConf {
	stream := ""
	if rc != 0 {
		stream = " >&2"
	}
	return contextQconf(fakeQconf(`[ "$*" = "`+flag+`" ] || { echo "unexpected: $*" >&2; exit 2; }
printf '%s\n' "`+out+`"`+stream+`
exit `+strconv.Itoa(rc)), 0)
}

var _ = Describe("qconf list commands", func() {
	for _, lc := range listCalls {
		lc := lc
		Context(lc.name+" ("+lc.flag+")", func() {
			It("reads qconf's own empty-list answer as an empty list", func() {
				names, err := lc.call(answeringQconf(lc.flag, lc.empty, 1))
				Expect(err).NotTo(HaveOccurred())
				Expect(names).To(BeEmpty())
			})

			It("returns the names qconf prints", func() {
				names, err := lc.call(answeringQconf(lc.flag, "a\nb", 0))
				Expect(err).NotTo(HaveOccurred())
				Expect(names).To(Equal([]string{"a", "b"}))
			})

			// The two ways this broke or could break: matching ANOTHER list's
			// message (the cluster-queue one, everywhere), and matching any
			// "no ... defined" -- which reads a real failure as "empty".
			It("does not take another list's or another error's message for empty", func() {
				for _, other := range []string{
					otherEmptyMessage(lc.empty),
					"error: no permission, user root not defined as manager",
				} {
					names, err := lc.call(answeringQconf(lc.flag, other, 1))
					Expect(err).To(HaveOccurred(), "qconf said %q", other)
					Expect(names).To(BeNil())
				}
			})
		})
	}

	It("covers both of qconf's submit host messages", func() {
		names, err := answeringQconf("-ss", "no submit defined", 1).ShowSubmitHosts()
		Expect(err).NotTo(HaveOccurred())
		Expect(names).To(BeEmpty())
	})
})

// otherEmptyMessage is an empty-list message that belongs to a different
// list than own.
func otherEmptyMessage(own string) string {
	if own == "no cqueue list defined" {
		return "no calendar defined"
	}
	return "no cqueue list defined"
}
