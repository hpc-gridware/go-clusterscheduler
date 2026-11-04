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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core/internal/fakeqconf"
)

// loggingQconf is fakeQconf with an argv log: every call appends its
// "$*" to the returned log file, so multi-call flows can be asserted.
func loggingQconf(body string) (exe, logPath string) {
	logPath = filepath.Join(GinkgoT().TempDir(), "argv.log")
	exe = fakeQconf(`echo "$*" >> ` + logPath + "\n" + body)
	return exe, logPath
}

func argvLines(logPath string) []string {
	raw, err := os.ReadFile(logPath)
	Expect(err).NotTo(HaveOccurred())
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

// noneDefinedQconf answers every call with qconf's "no <what> defined".
func noneDefinedQconf(what string) *core.CommandLineQConf {
	return contextQconf(fakeQconf(`echo "no `+what+` defined" >&2; exit 1`), 0)
}

// singleCallQconf points a CommandLineQConf at a fakeqconf stub that
// answers every call with stdout and exits with rc.
func singleCallQconf(stdout string, rc int) (*core.CommandLineQConf, *fakeqconf.Fake) {
	f := fakeqconf.New(GinkgoT(), stdout, rc)
	DeferCleanup(f.Cleanup)
	return contextQconf(f.Path(), 0), f
}

const ssconfSample = `algorithm                         default
schedule_interval                 0:0:15
maxujobs                          0
queue_sort_method                 load
job_load_adjustments              np_load_avg=0.50
load_adjustment_decay_time        0:7:30
load_formula                      np_load_avg
schedd_job_info                   true
flush_submit_sec                  0
flush_finish_sec                  0
params                            MONITOR=1 PROFILE=0
reprioritize_interval             0:0:0
halftime                          168
weight_user                       0.250000
max_reservation                   32
default_duration                  INFINITY
`

const suMultiSample = `name       arusers
type       ACL
fshare     0
oticket    0
entries    NONE

name       deadlineusers
type       ACL
fshare     0
oticket    0
entries    alice,bob

name       defaultdepartment
type       ACL DEPT
fshare     100
oticket    0
entries    NONE
`

var _ = Describe("Reading the rest of the configuration under the caller's context", func() {
	BeforeEach(func() {
		if !fakeqconf.Available() {
			Skip("the fakes are shell scripts; skip on this platform")
		}
	})

	Describe("scheduler and global configuration", func() {
		It("reads the scheduler configuration with one qconf -ssconf", func() {
			c, f := singleCallQconf(ssconfSample, 0)
			cfg, err := c.ShowSchedulerConfigurationContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-ssconf"}))
			Expect(cfg.Algorithm).To(Equal("default"))
			Expect(cfg.ScheduleInterval).To(Equal("0:0:15"))
			Expect(cfg.ScheddJobInfo).To(Equal("true"))
			Expect(cfg.Params).To(Equal([]string{"MONITOR=1", "PROFILE=0"}))
			Expect(cfg.Halftime).To(Equal(168))
			Expect(cfg.MaxReservation).To(Equal(32))
			Expect(cfg.WeightUser).To(BeNumerically("~", 0.25))
		})

		It("reads the global configuration with one qconf -sconf global", func() {
			c, f := singleCallQconf("execd_spool_dir   /opt/cs/default/spool\nmax_jobs          0\nshell_start_mode  unix_behavior\njsv_params        NONE\n", 0)
			cfg, err := c.ShowGlobalConfigurationContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-sconf global"}))
			Expect(cfg.ExecdSpoolDir).To(Equal("/opt/cs/default/spool"))
			Expect(cfg.ShellStartMode).To(Equal("unix_behavior"))
			Expect(cfg.ExtraFields).To(HaveKeyWithValue("jsv_params", "NONE"))
		})
	})

	Describe("parallel environments", func() {
		It("lists with -spl and reads each PE with its own -sp", func() {
			exe, log := loggingQconf(`case "$*" in
"-spl") printf 'mpi\nsmp\n' ;;
"-sp mpi") printf 'pe_name            mpi\nslots              999\nallocation_rule    $round_robin\ncontrol_slaves     TRUE\n' ;;
"-sp smp") printf 'pe_name            smp\nslots              64\nallocation_rule    $pe_slots\n' ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			pes, err := contextQconf(exe, 0).ShowAllParallelEnvironmentsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(argvLines(log)).To(Equal([]string{"-spl", "-sp mpi", "-sp smp"}))
			Expect(pes).To(HaveLen(2))
			Expect(pes[0].Name).To(Equal("mpi"))
			Expect(pes[0].Slots).To(Equal(999))
			Expect(pes[0].AllocationRule).To(Equal("$round_robin"))
			Expect(pes[1].Name).To(Equal("smp"))
			Expect(pes[1].Slots).To(Equal(64))
		})

		It("answers no PE defined with an empty list", func() {
			pes, err := noneDefinedQconf("parallel environment").ShowAllParallelEnvironmentsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(pes).NotTo(BeNil())
			Expect(pes).To(BeEmpty())
		})

		It("returns the context error, not an empty list, when ctx is already cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := noneDefinedQconf("parallel environment").ShowAllParallelEnvironmentsContext(ctx, nil)
			Expect(err).To(MatchError(context.Canceled))
		})
	})

	Describe("user set lists", func() {
		It("splits a multi-set qconf -su answer into one struct per set", func() {
			sets := core.ParseUserSetLists(suMultiSample)
			Expect(sets).To(HaveLen(3))
			// The shared parser reads "entries NONE" as no entries.
			Expect(sets[0]).To(Equal(core.UserSetListConfig{Name: "arusers", Type: "ACL"}))
			Expect(sets[1].Name).To(Equal("deadlineusers"))
			Expect(sets[1].Entries).To(Equal([]string{"alice", "bob"}))
			Expect(sets[2].Type).To(Equal("ACL DEPT"))
			Expect(sets[2].FShare).To(Equal(100))
			Expect(core.ParseUserSetLists("")).To(BeEmpty())
		})

		It("lists with -sul and reads all sets with one -su", func() {
			src := filepath.Join(GinkgoT().TempDir(), "su.txt")
			Expect(os.WriteFile(src, []byte(suMultiSample), 0o644)).To(Succeed())
			exe, log := loggingQconf(`case "$*" in
"-sul") printf 'arusers\ndeadlineusers\ndefaultdepartment\n' ;;
"-su arusers,deadlineusers,defaultdepartment") cat ` + src + ` ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			sets, err := contextQconf(exe, 0).ShowAllUserSetListsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(argvLines(log)).To(Equal([]string{"-sul", "-su arusers,deadlineusers,defaultdepartment"}))
			Expect(sets).To(HaveLen(3))
			Expect(sets[2].Name).To(Equal("defaultdepartment"))
		})

		It("answers no user set list defined with an empty list", func() {
			sets, err := noneDefinedQconf("userset list").ShowAllUserSetListsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sets).NotTo(BeNil())
			Expect(sets).To(BeEmpty())
		})
	})

	Describe("host groups", func() {
		It("resolves one group with -shgrp_resolved", func() {
			c, f := singleCallQconf("master sim1 sim2\n", 0)
			hosts, err := c.ShowHostGroupResolvedContext(context.Background(), nil, "@allhosts")
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-shgrp_resolved @allhosts"}))
			Expect(hosts).To(Equal([]string{"master", "sim1", "sim2"}))
		})

		It("lists with -shgrpl and resolves each group", func() {
			exe, log := loggingQconf(`case "$*" in
"-shgrpl") printf '@allhosts\n@gpu\n' ;;
"-shgrp_resolved @allhosts") echo "master sim1 sim2" ;;
"-shgrp_resolved @gpu") echo "sim2" ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			groups, err := contextQconf(exe, 0).ShowAllHostGroupsResolvedContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(argvLines(log)).To(Equal([]string{"-shgrpl", "-shgrp_resolved @allhosts", "-shgrp_resolved @gpu"}))
			Expect(groups).To(Equal(map[string][]string{
				"@allhosts": {"master", "sim1", "sim2"},
				"@gpu":      {"sim2"},
			}))
		})

		It("answers no host group defined with an empty map", func() {
			groups, err := noneDefinedQconf("host group list").ShowAllHostGroupsResolvedContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(groups).NotTo(BeNil())
			Expect(groups).To(BeEmpty())
		})
	})

	Describe("user and host lists", func() {
		type listCase struct {
			flag string
			what string
			call func(*core.CommandLineQConf) ([]string, error)
		}
		cases := []listCase{
			{"-sm", "manager", func(c *core.CommandLineQConf) ([]string, error) {
				return c.ShowManagersContext(context.Background(), nil)
			}},
			{"-so", "operator", func(c *core.CommandLineQConf) ([]string, error) {
				return c.ShowOperatorsContext(context.Background(), nil)
			}},
			{"-sh", "administrative host", func(c *core.CommandLineQConf) ([]string, error) {
				return c.ShowAdminHostsContext(context.Background(), nil)
			}},
			{"-ss", "submit host", func(c *core.CommandLineQConf) ([]string, error) {
				return c.ShowSubmitHostsContext(context.Background(), nil)
			}},
		}
		for _, tc := range cases {
			tc := tc
			It("reads "+tc.flag+" as one name per line, and none defined as an empty list", func() {
				c, f := singleCallQconf("root\n\nalice\n", 0)
				names, err := tc.call(c)
				Expect(err).NotTo(HaveOccurred())
				Expect(f.AllArgvLines()).To(Equal([]string{tc.flag}))
				Expect(names).To(Equal([]string{"root", "alice"}))

				names, err = tc.call(noneDefinedQconf(tc.what))
				Expect(err).NotTo(HaveOccurred())
				Expect(names).NotTo(BeNil())
				Expect(names).To(BeEmpty())
			})
		}

		It("reports any other failure with qconf's words", func() {
			c := contextQconf(fakeQconf(`echo "error: commlib error: got select error (Connection refused)" >&2; exit 1`), 0)
			_, err := c.ShowManagersContext(context.Background(), nil)
			Expect(err).To(MatchError(ContainSubstring("commlib error")))
		})
	})

	Describe("scheduler host and threads", func() {
		It("reads the scheduler host with -sss", func() {
			c, f := singleCallQconf("master\n", 0)
			host, err := c.ShowSchedulerHostContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-sss"}))
			Expect(host).To(Equal("master"))
		})

		It("reports no registered scheduler as an error with qconf's words", func() {
			_, err := noneDefinedQconf("scheduling host").ShowSchedulerHostContext(context.Background(), nil)
			Expect(err).To(MatchError(ContainSubstring("no scheduling host defined")))
		})

		It("reads the thread pools with -stl, skipping the header", func() {
			c, f := singleCallQconf("POOL            SIZE\n--------------------\nlistener        4\nworker          2\nscheduler       1\n", 0)
			threads, err := c.ShowThreadsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-stl"}))
			Expect(threads).To(Equal([]core.ThreadInfo{
				{Name: "listener", Count: 4},
				{Name: "worker", Count: 2},
				{Name: "scheduler", Count: 1},
			}))
			Expect(core.ParseThreadList("")).To(BeEmpty())
		})
	})

	Describe("my roles (GCS 9.1 RBAC)", func() {
		It("reads the roles with -smyrole, one per line", func() {
			c, f := singleCallQconf("manager\njob_user\n", 0)
			roles, err := c.ShowMyRolesContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-smyrole"}))
			Expect(roles).To(Equal([]string{"manager", "job_user"}))
		})

		It("answers no role with an empty list", func() {
			c, _ := singleCallQconf("", 0)
			roles, err := c.ShowMyRolesContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(roles).NotTo(BeNil())
			Expect(roles).To(BeEmpty())
		})

		It("returns ErrNotSupported when qconf does not know the option", func() {
			c, _ := singleCallQconf("qconf: invalid option argument \"-smyrole\"\nUsage: qconf -help\n", 1)
			_, err := c.ShowMyRolesContext(context.Background(), nil)
			Expect(err).To(MatchError(core.ErrNotSupported))
		})

		It("returns the context error when ctx ends", func() {
			c := contextQconf(fakeQconf(`sleep 30`), time.Minute)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			_, err := c.ShowMyRolesContext(ctx, nil)
			Expect(err).To(MatchError(context.DeadlineExceeded))
		})
	})

	Describe("version", func() {
		It("parses the -help banner of the configured executable", func() {
			c, f := singleCallQconf("GCS 9.1.7 (260101-1200)\nusage: qconf [options]\n", 1)
			v, err := c.GetVersionContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-help"}))
			Expect(v.Product).To(Equal(core.ClusterSchedulerProductGCS))
			Expect(v.Version).To(Equal("9.1.7"))
			Expect(v.Patch).To(Equal(7))
		})

		It("answers GetVersion through the same single call", func() {
			c, f := singleCallQconf("OCS 9.0.12 (260101-1200)\nusage: qconf [options]\n", 1)
			v, err := c.GetVersion()
			Expect(err).NotTo(HaveOccurred())
			Expect(f.AllArgvLines()).To(Equal([]string{"-help"}))
			Expect(v.Product).To(Equal(core.ClusterSchedulerProductOCS))
			Expect(v.Version).To(Equal("9.0.12"))
		})

		It("prints one terminated line and runs nothing in dry-run mode", func() {
			c, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{Executable: "/nonexistent/qconf", DryRun: true})
			Expect(err).NotTo(HaveOccurred())
			read, write, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			stdout := os.Stdout
			os.Stdout = write
			v, verr := c.GetVersion()
			os.Stdout = stdout
			Expect(write.Close()).To(Succeed())
			printed, err := io.ReadAll(read)
			Expect(err).NotTo(HaveOccurred())
			Expect(verr).NotTo(HaveOccurred())
			Expect(v).To(Equal(core.ClusterSchedulerVersion{}))
			Expect(string(printed)).To(Equal("Executing: /nonexistent/qconf -help\n"))
		})

		It("ends with the caller's context", func() {
			c := contextQconf(fakeQconf(`sleep 30`), time.Minute)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			start := time.Now()
			_, err := c.GetVersionContext(ctx, nil)
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
		})
	})

	Describe("calendars, projects and checkpointing interfaces", func() {
		It("lists with -scall and reads each calendar with -scal", func() {
			exe, log := loggingQconf(`case "$*" in
"-scall") printf 'night\nweekend\n' ;;
"-scal night") printf 'calendar_name    night\nyear             NONE\nweek             mon-fri=6-20=off\n' ;;
"-scal weekend") printf 'calendar_name    weekend\nyear             NONE\nweek             sat-sun=off\n' ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			cals, err := contextQconf(exe, 0).ShowAllCalendarsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(argvLines(log)).To(Equal([]string{"-scall", "-scal night", "-scal weekend"}))
			Expect(cals).To(HaveLen(2))
			Expect(cals[0].Name).To(Equal("night"))
			Expect(cals[0].Week).To(Equal("mon-fri=6-20=off"))
			Expect(cals[1].Name).To(Equal("weekend"))
		})

		It("lists with -sprjl and reads each project with -sprj", func() {
			exe, log := loggingQconf(`case "$*" in
"-sprjl") printf 'finance\n' ;;
"-sprj finance") printf 'name    finance\noticket 0\nfshare  100\nacl     finusers\nxacl    NONE\n' ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			projects, err := contextQconf(exe, 0).ShowAllProjectsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(argvLines(log)).To(Equal([]string{"-sprjl", "-sprj finance"}))
			Expect(projects).To(HaveLen(1))
			Expect(projects[0].Name).To(Equal("finance"))
			Expect(projects[0].FShare).To(Equal(100))
			Expect(projects[0].ACL).To(Equal([]string{"finusers"}))
		})

		It("lists with -sckptl and reads each interface with -sckpt", func() {
			exe, log := loggingQconf(`case "$*" in
"-sckptl") printf 'blcr\n' ;;
"-sckpt blcr") printf 'ckpt_name          blcr\ninterface          APPLICATION-LEVEL\nckpt_command       NONE\nmigr_command       NONE\nrestart_command    NONE\nclean_command      NONE\nckpt_dir           /tmp\nsignal             NONE\nwhen               sx\n' ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			ckpts, err := contextQconf(exe, 0).ShowAllCkptInterfacesContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(argvLines(log)).To(Equal([]string{"-sckptl", "-sckpt blcr"}))
			Expect(ckpts).To(HaveLen(1))
			Expect(ckpts[0].Name).To(Equal("blcr"))
			Expect(ckpts[0].Interface).To(Equal("APPLICATION-LEVEL"))
			Expect(ckpts[0].When).To(Equal("sx"))
		})

		It("answers none defined with an empty list for each", func() {
			cals, err := noneDefinedQconf("calendar").ShowAllCalendarsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(cals).NotTo(BeNil())
			Expect(cals).To(BeEmpty())

			projects, err := noneDefinedQconf("project list").ShowAllProjectsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(projects).NotTo(BeNil())
			Expect(projects).To(BeEmpty())

			ckpts, err := noneDefinedQconf("ckpt interface definition").ShowAllCkptInterfacesContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(ckpts).NotTo(BeNil())
			Expect(ckpts).To(BeEmpty())
		})
	})
	Describe("objects deleted between the list call and the show call", func() {
		// Each fake lists "a" and "gone"; "gone" was deleted meanwhile and
		// qconf answers it with its "does not exist" message on stderr and
		// exit 1, the way the scheduler's C client does.
		DescribeTable("leave the vanished object out instead of failing the call",
			func(list, show, notExist string, call func(*core.CommandLineQConf) (int, error), body string) {
				exe, log := loggingQconf(`case "$*" in
"` + list + `") printf 'a\ngone\n' ;;
"` + show + ` a") ` + body + ` ;;
"` + show + ` gone") echo '` + notExist + `' >&2; exit 1 ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
				n, err := call(contextQconf(exe, 0))
				Expect(err).NotTo(HaveOccurred())
				Expect(n).To(Equal(1))
				Expect(argvLines(log)).To(Equal([]string{list, show + " a", show + " gone"}))
			},
			Entry("-sp", "-spl", "-sp", "gone is not a parallel environment",
				func(c *core.CommandLineQConf) (int, error) {
					pes, err := c.ShowAllParallelEnvironmentsContext(context.Background(), nil)
					return len(pes), err
				}, `printf 'pe_name a\nslots 1\n'`),
			Entry("-scal", "-scall", "-scal", "gone is not a calendar",
				func(c *core.CommandLineQConf) (int, error) {
					cals, err := c.ShowAllCalendarsContext(context.Background(), nil)
					return len(cals), err
				}, `printf 'calendar_name a\nyear NONE\nweek NONE\n'`),
			Entry("-sprj", "-sprjl", "-sprj", "gone is not known as project",
				func(c *core.CommandLineQConf) (int, error) {
					projects, err := c.ShowAllProjectsContext(context.Background(), nil)
					return len(projects), err
				}, `printf 'name a\nfshare 0\n'`),
			Entry("-sckpt", "-sckptl", "-sckpt", "gone is not a checkpointing interface definition",
				func(c *core.CommandLineQConf) (int, error) {
					ckpts, err := c.ShowAllCkptInterfacesContext(context.Background(), nil)
					return len(ckpts), err
				}, `printf 'ckpt_name a\ninterface userdefined\n'`),
			Entry("-shgrp_resolved", "-shgrpl", "-shgrp_resolved", `Host group "gone" does not exist`,
				func(c *core.CommandLineQConf) (int, error) {
					groups, err := c.ShowAllHostGroupsResolvedContext(context.Background(), nil)
					return len(groups), err
				}, `echo master`),
		)

		It("still fails on any other answer for an object", func() {
			exe, _ := loggingQconf(`case "$*" in
"-spl") printf 'a\n' ;;
*) echo "error: no permission, user root not defined as manager" >&2; exit 1 ;;
esac`)
			_, err := contextQconf(exe, 0).ShowAllParallelEnvironmentsContext(context.Background(), nil)
			Expect(err).To(MatchError(ContainSubstring("no permission")))
		})

		It("still reports a missing group read on its own as an error", func() {
			c := contextQconf(fakeQconf(`echo 'Host group "@gone" does not exist' >&2; exit 1`), 0)
			_, err := c.ShowHostGroupResolvedContext(context.Background(), nil, "@gone")
			Expect(err).To(MatchError(ContainSubstring("does not exist")))
		})

		It("keeps the user sets -su printed when one of them is gone", func() {
			src := filepath.Join(GinkgoT().TempDir(), "su.txt")
			Expect(os.WriteFile(src, []byte(suMultiSample), 0o644)).To(Succeed())
			exe, _ := loggingQconf(`case "$*" in
"-sul") printf 'arusers\ndeadlineusers\ngone\ndefaultdepartment\n' ;;
"-su arusers,deadlineusers,gone,defaultdepartment") cat ` + src + `; echo 'denied: access list "gone" does not exist' >&2; exit 1 ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			sets, err := contextQconf(exe, 0).ShowAllUserSetListsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sets).To(HaveLen(3))
		})

		It("fails -su when stderr holds anything but missing sets", func() {
			exe, _ := loggingQconf(`case "$*" in
"-sul") printf 'arusers\n' ;;
*) echo 'denied: access list "other" does not exist' >&2; echo "error: something else" >&2; exit 1 ;;
esac`)
			_, err := contextQconf(exe, 0).ShowAllUserSetListsContext(context.Background(), nil)
			Expect(err).To(MatchError(ContainSubstring("something else")))
		})

		It("returns the context error, not a skip, when ctx ends during a show call", func() {
			exe, _ := loggingQconf(`case "$*" in
"-spl") printf 'a\n' ;;
*) sleep 30 ;;
esac`)
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			_, err := contextQconf(exe, time.Minute).ShowAllParallelEnvironmentsContext(ctx, nil)
			Expect(err).To(MatchError(context.DeadlineExceeded))
		})
	})

	Describe("a long user set list", func() {
		It("is asked for in several -su calls, none with an argument above 96 KiB", func() {
			names := make([]string, 0, 2000)
			for i := 0; i < 2000; i++ {
				names = append(names, fmt.Sprintf("userset_%04d_%s", i, strings.Repeat("x", 80)))
			}
			list := filepath.Join(GinkgoT().TempDir(), "sul.txt")
			Expect(os.WriteFile(list, []byte(strings.Join(names, "\n")+"\n"), 0o644)).To(Succeed())
			exe, log := loggingQconf(`case "$1" in
-sul) cat ` + list + ` ;;
-su) echo "$2" | tr ',' '\n' | while read -r n; do printf 'name %s\ntype ACL\nfshare 0\noticket 0\nentries NONE\n\n' "$n"; done ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
			sets, err := contextQconf(exe, 0).ShowAllUserSetListsContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(sets).To(HaveLen(2000))
			Expect(sets[0].Name).To(Equal(names[0]))
			Expect(sets[1999].Name).To(Equal(names[1999]))
			calls := argvLines(log)[1:]
			Expect(len(calls)).To(BeNumerically(">", 1))
			for _, call := range calls {
				Expect(len(strings.TrimPrefix(call, "-su "))).To(BeNumerically("<=", 96*1024))
			}
		})
	})

	Describe("hostile names returned by qmaster", func() {
		DescribeTable("are rejected before they become an argument",
			func(list string, call func(*core.CommandLineQConf) error) {
				exe, log := loggingQconf(`case "$*" in
"` + list + `") printf -- '-x\n' ;;
*) echo "unexpected $*" >&2; exit 2 ;;
esac`)
				Expect(call(contextQconf(exe, 0))).To(MatchError(ContainSubstring("must not start with '-'")))
				Expect(argvLines(log)).To(Equal([]string{list}), "no follow-up qconf call")
			},
			Entry("-spl", "-spl", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllParallelEnvironmentsContext(context.Background(), nil)
				return err
			}),
			Entry("-sul", "-sul", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllUserSetListsContext(context.Background(), nil)
				return err
			}),
			Entry("-shgrpl", "-shgrpl", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllHostGroupsResolvedContext(context.Background(), nil)
				return err
			}),
			Entry("-scall", "-scall", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllCalendarsContext(context.Background(), nil)
				return err
			}),
			Entry("-sprjl", "-sprjl", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllProjectsContext(context.Background(), nil)
				return err
			}),
			Entry("-sckptl", "-sckptl", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllCkptInterfacesContext(context.Background(), nil)
				return err
			}),
		)
	})

	Describe("a failing qconf", func() {
		// The fake prints configuration on stdout and the failure on
		// stderr; the error carries stderr only.
		DescribeTable("is an error carrying qconf's stderr, never its stdout",
			func(call func(*core.CommandLineQConf) error) {
				c := contextQconf(fakeQconf(`echo "secret_value 42"; echo "error: commlib error: got select error (Connection refused)" >&2; exit 1`), 0)
				err := call(c)
				Expect(err).To(MatchError(ContainSubstring("commlib error")))
				Expect(err.Error()).NotTo(ContainSubstring("secret_value"))
			},
			Entry("scheduler configuration", func(c *core.CommandLineQConf) error {
				_, err := c.ShowSchedulerConfigurationContext(context.Background(), nil)
				return err
			}),
			Entry("global configuration", func(c *core.CommandLineQConf) error {
				_, err := c.ShowGlobalConfigurationContext(context.Background(), nil)
				return err
			}),
			Entry("parallel environments", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllParallelEnvironmentsContext(context.Background(), nil)
				return err
			}),
			Entry("user set lists", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllUserSetListsContext(context.Background(), nil)
				return err
			}),
			Entry("one host group", func(c *core.CommandLineQConf) error {
				_, err := c.ShowHostGroupResolvedContext(context.Background(), nil, "@allhosts")
				return err
			}),
			Entry("all host groups", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllHostGroupsResolvedContext(context.Background(), nil)
				return err
			}),
			Entry("managers", func(c *core.CommandLineQConf) error {
				_, err := c.ShowManagersContext(context.Background(), nil)
				return err
			}),
			Entry("operators", func(c *core.CommandLineQConf) error {
				_, err := c.ShowOperatorsContext(context.Background(), nil)
				return err
			}),
			Entry("admin hosts", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAdminHostsContext(context.Background(), nil)
				return err
			}),
			Entry("submit hosts", func(c *core.CommandLineQConf) error {
				_, err := c.ShowSubmitHostsContext(context.Background(), nil)
				return err
			}),
			Entry("scheduler host", func(c *core.CommandLineQConf) error {
				_, err := c.ShowSchedulerHostContext(context.Background(), nil)
				return err
			}),
			Entry("threads", func(c *core.CommandLineQConf) error {
				_, err := c.ShowThreadsContext(context.Background(), nil)
				return err
			}),
			Entry("my roles", func(c *core.CommandLineQConf) error {
				_, err := c.ShowMyRolesContext(context.Background(), nil)
				return err
			}),
			Entry("calendars", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllCalendarsContext(context.Background(), nil)
				return err
			}),
			Entry("projects", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllProjectsContext(context.Background(), nil)
				return err
			}),
			Entry("checkpointing interfaces", func(c *core.CommandLineQConf) error {
				_, err := c.ShowAllCkptInterfacesContext(context.Background(), nil)
				return err
			}),
		)

		It("caps the stderr it carries", func() {
			c := contextQconf(fakeQconf(`head -c 200000 /dev/zero | tr '\0' 'e' >&2; exit 1`), 0)
			_, err := c.ShowManagersContext(context.Background(), nil)
			Expect(err).To(HaveOccurred())
			Expect(len(err.Error())).To(BeNumerically("<", 70*1024))
		})

		It("keeps a stderr warning of a successful call out of the parser", func() {
			c := contextQconf(fakeQconf(`echo "warning: a b c d e f g h" >&2; printf 'slots s INT <= YES YES 1 1000\n'`), 0)
			cx, err := c.ShowAllComplexesContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(cx).To(HaveLen(1))
			Expect(cx[0].Name).To(Equal("slots"))
		})
	})

	Describe("the scheduler configuration parser", func() {
		It("reads every key of a 9.1 qconf -ssconf answer (synthesized from the C field list, not captured)", func() {
			src, err := filepath.Abs(filepath.Join("testdata", "qconf-ssconf-9.1-synthesized.txt"))
			Expect(err).NotTo(HaveOccurred())
			c := contextQconf(fakeQconf(`cat `+src), 0)
			cfg, err := c.ShowSchedulerConfigurationContext(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(*cfg).To(Equal(core.SchedulerConfig{
				Algorithm:                   "default",
				ScheduleInterval:            "0:0:10",
				MaxUJobs:                    20,
				QueueSortMethod:             "seqno",
				JobLoadAdjustments:          []string{"np_load_avg=0.500000", "mem_used=0.100000"},
				LoadAdjustmentDecayTime:     "0:7:30",
				LoadFormula:                 "np_load_avg+mem_used",
				ScheddJobInfo:               "false",
				FlushSubmitSec:              1,
				FlushFinishSec:              2,
				Params:                      []string{"MONITOR=1", "PROFILE=0"},
				ReprioritizeInterval:        "0:2:0",
				Halftime:                    336,
				UsageWeightList:             []string{"cpu=0.700000", "mem=0.200000", "io=0.100000"},
				CompensationFactor:          5,
				WeightUser:                  0.1,
				WeightProject:               0.2,
				WeightDepartment:            0.3,
				WeightJob:                   0.4,
				WeightTicketsFunctional:     10000,
				WeightTicketsShare:          20000,
				ShareOverrideTickets:        true,
				ShareFunctionalShares:       true,
				MaxFunctionalJobsToSchedule: 300,
				ReportPJobTickets:           true,
				MaxPendingTasksPerJob:       60,
				HalflifeDecayList:           []string{"cpu=168:mem=0:io=-1"},
				PolicyHierarchy:             "OFS",
				WeightTicket:                0.01,
				WeightWaitingTime:           0.5,
				WeightDeadline:              3600000,
				WeightUrgency:               0.1,
				WeightPriority:              1,
				MaxReservation:              32,
				DefaultDuration:             "8:0:0",
			}))
			Expect(cfg.ExtraFields).To(BeEmpty())
		})
	})

	Describe("version under the caller's context", func() {
		It("runs -help with the given environment instead of the process environment", func() {
			GinkgoT().Setenv("QCONF_TEST_SECRET", "leak")
			c := contextQconf(fakeQconf(`[ "$MARK" = yes ] && [ -z "$QCONF_TEST_SECRET" ] && echo "GCS 9.1.7 (260101-1200)"`), 0)
			v, err := c.GetVersionContext(context.Background(), []string{"MARK=yes"})
			Expect(err).NotTo(HaveOccurred())
			Expect(v.Version).To(Equal("9.1.7"))
		})

		It("says timed out when the deadline passes", func() {
			c := contextQconf(fakeQconf(`sleep 30`), time.Minute)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			_, err := c.GetVersionContext(ctx, nil)
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(err.Error()).To(ContainSubstring("timed out"))
		})

		It("says canceled, not timed out, when the caller cancels", func() {
			c := contextQconf(fakeQconf(`sleep 30`), time.Minute)
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(200*time.Millisecond, cancel)
			_, err := c.GetVersionContext(ctx, nil)
			Expect(err).To(MatchError(context.Canceled))
			Expect(err.Error()).To(ContainSubstring("canceled"))
			Expect(err.Error()).NotTo(ContainSubstring("timed out"))
		})
	})
})
