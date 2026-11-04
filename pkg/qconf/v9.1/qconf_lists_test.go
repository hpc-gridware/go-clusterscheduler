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

package qconf

import (
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The v9.1 adapter inherits the list calls from core. This pins that an
// empty list is an empty list through THIS adapter too, so a future v9.1
// override cannot bring back the wrong empty-list message (core
// qconf_lists_test.go has the full matrix).
var _ = Describe("v9.1 qconf list commands on an empty cluster", func() {
	BeforeEach(func() {
		if runtime.GOOS == "windows" {
			Skip("the fake qconf is a shell script")
		}
	})

	empty := map[string]string{
		"-scall": "calendar", "-scel": "complex entry", "-sckptl": "ckpt interface definition",
		"-sconfl": "config", "-sel": "execution host", "-shgrpl": "host group list",
		"-srqsl": "resource quota set list", "-spl": "parallel environment",
		"-sprjl": "project list", "-sql": "cqueue list", "-suserl": "user list",
		"-sul": "userset list", "-so": "operator", "-ss": "submit host",
	}

	It("returns an empty list, not an error, for every list", func() {
		script := "#!/bin/sh\ncase \"$*\" in\n"
		for flag, what := range empty {
			script += flag + `) echo "no ` + what + ` defined" >&2; exit 1;;` + "\n"
		}
		script += "*) echo \"unexpected: $*\" >&2; exit 2;;\nesac\n"
		exe := filepath.Join(GinkgoT().TempDir(), "qconf")
		Expect(os.WriteFile(exe, []byte(script), 0o755)).To(Succeed())
		q, err := NewCommandLineQConf(CommandLineQConfConfig{Executable: exe})
		Expect(err).NotTo(HaveOccurred())

		calls := map[string]func() ([]string, error){
			"ShowCalendars": q.ShowCalendars, "ShowComplexEntries": q.ShowComplexEntries,
			"ShowCkptInterfaces": q.ShowCkptInterfaces, "ShowHostConfigurations": q.ShowHostConfigurations,
			"ShowExecHosts": q.ShowExecHosts, "ShowHostGroups": q.ShowHostGroups,
			"ShowResourceQuotaSets": q.ShowResourceQuotaSets, "ShowParallelEnvironments": q.ShowParallelEnvironments,
			"ShowProjects": q.ShowProjects, "ShowClusterQueues": q.ShowClusterQueues,
			"ShowUsers": q.ShowUsers, "ShowUserSetLists": q.ShowUserSetLists,
			"ShowOperators": q.ShowOperators, "ShowSubmitHosts": q.ShowSubmitHosts,
		}
		Expect(calls).To(HaveLen(len(empty)))
		for name, call := range calls {
			names, err := call()
			Expect(err).NotTo(HaveOccurred(), name)
			Expect(names).To(BeEmpty(), name)
		}
	})
})
