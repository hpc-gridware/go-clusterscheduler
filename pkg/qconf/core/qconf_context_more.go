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

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ErrNotSupported is returned when the installed qconf does not know the
// option a method needs ("invalid option argument" answer), so a caller
// can fall back to an older way of getting the same information.
var ErrNotSupported = errors.New("qconf option not supported by this version")

// The *Context methods below read the rest of the cluster configuration
// under the caller's context and environment, like ConfigContextReader
// (see there for the rationale). They are concrete methods only: a
// consumer that wants a fake declares the small interface it needs.
//
// Each "ShowAll..." method answers "none defined" with an empty, non-nil
// collection and not with an error, and leaves out an object that was
// deleted between the list call and its show call.

// ThreadInfo is one row of "qconf -stl": a qmaster thread pool and the
// number of threads in it.
type ThreadInfo struct {
	// Name is the pool name, for example "listener" or "worker".
	Name string `json:"name"`
	// Count is the number of threads in the pool.
	Count int `json:"count"`
}

// msgInvalidOption is the start of qconf's answer to an option it does
// not know (MSG_ANSWER_INVALIDOPTIONARGX_S in the scheduler's
// msg_common.h).
const msgInvalidOption = "invalid option argument"

// noneDefined reports whether out carries qconf's "no <what> defined"
// answer (MSG_QCONF_NOXDEFINED_S), which qconf gives with exit code 1 when
// a list is empty. what is the object name qconf uses for the option, for
// example "parallel environment" for -spl.
func noneDefined(out, what string) bool {
	return strings.Contains(out, "no "+what+" defined")
}

// listContext runs a list option (-spl, -sul, ...) and returns one name
// per line. qconf's "no <what> defined" answer is an empty list.
func (c *CommandLineQConf) listContext(ctx context.Context, env []string, flag, what string) ([]string, error) {
	out, err := c.RunCommandContext(ctx, env, flag)
	if err != nil {
		if ctx.Err() == nil && noneDefined(out, what) {
			return []string{}, nil
		}
		return nil, err
	}
	names := splitWithoutEmptyLines(out, "\n")
	if names == nil {
		return []string{}, nil
	}
	return names, nil
}

// The answers of the single-object show options for a name that does not
// exist (any more); %s is the name, cut like the scheduler's SFN and SFQ
// formats (see nameInMessage). The message ids are those of msg_common.h,
// except MSG_HGROUP_NOTEXIST_S (msg_qconf.h).
const (
	// msgNotAParallelEnvironment is MSG_PARALLEL_XNOTAPARALLELEVIRONMENT_S (-sp).
	msgNotAParallelEnvironment = "%s is not a parallel environment"
	// msgNotACalendar is MSG_CALENDAR_XISNOTACALENDAR_S (-scal).
	msgNotACalendar = "%s is not a calendar"
	// msgNotAProject is MSG_PROJECT_XISNOKNWOWNPROJECT_S (-sprj).
	msgNotAProject = "%s is not known as project"
	// msgNotACkptInterface is MSG_CKPT_XISNOTCHKPINTERFACEDEF_S (-sckpt).
	msgNotACkptInterface = "%s is not a checkpointing interface definition"
	// msgHostGroupNotExist is MSG_HGROUP_NOTEXIST_S (-shgrp_resolved).
	msgHostGroupNotExist = `Host group "%s" does not exist`
	// msgAccessListNotExist is MSG_SGETEXT_DOESNOTEXIST_SS with "access
	// list" (-su prints one such line per missing set).
	msgAccessListNotExist = `denied: access list "%s" does not exist`
)

// maxNameInMessage is the precision of the scheduler's SFN and SFQ
// formats ("%-.100s"): qconf prints at most this many bytes of a name.
const maxNameInMessage = 100

// nameInMessage returns name as qconf prints it in a message.
func nameInMessage(name string) string {
	if len(name) > maxNameInMessage {
		return name[:maxNameInMessage]
	}
	return name
}

// maxListArgument bounds one comma-joined name list argument. Linux refuses
// a single argument above 128 KiB (MAX_ARG_STRLEN) with E2BIG before qconf
// runs, so a longer list is asked for in several calls.
const maxListArgument = 96 * 1024

// chunkNames splits names into lists whose comma-joined length stays at or
// below max; a single longer name gets a list of its own.
func chunkNames(names []string, max int) [][]string {
	var chunks [][]string
	start, size := 0, 0
	for i, name := range names {
		add := len(name)
		if i > start {
			add++
		}
		if i > start && size+add > max {
			chunks = append(chunks, names[start:i])
			start, size, add = i, 0, len(name)
		}
		size += add
	}
	return append(chunks, names[start:])
}

// namedContext runs a single-object show option (-sp, -scal, ...) for one
// name returned by qmaster, so the name is validated like runNamed does.
// None of these options takes a list (each looks up exactly one name), so
// a "ShowAll..." method costs one qconf process per object. found is false,
// without an error, when qconf answers notExist for the name, i.e. the
// object was deleted after it was listed, and ctx has not ended.
func (c *CommandLineQConf) namedContext(ctx context.Context, env []string, flag, name, notExist string) (out string, found bool, err error) {
	if err := validate.Enforce(validate.Operand(name), validate.StrictObjectName(name)); err != nil {
		return "", false, err
	}
	out, err = c.RunCommandContext(ctx, env, flag, name)
	if err != nil {
		if ctx.Err() == nil && strings.Contains(out, fmt.Sprintf(notExist, nameInMessage(name))) {
			return "", false, nil
		}
		return "", false, err
	}
	return out, true, nil
}

// ShowSchedulerConfigurationContext returns the scheduler configuration
// ("qconf -ssconf").
func (c *CommandLineQConf) ShowSchedulerConfigurationContext(ctx context.Context, env []string) (*SchedulerConfig, error) {
	out, err := c.RunCommandContext(ctx, env, "-ssconf")
	if err != nil {
		return nil, err
	}
	cfg := parseSchedulerConfig(strings.Split(out, "\n"))
	return &cfg, nil
}

// ShowGlobalConfigurationContext returns the global configuration ("qconf
// -sconf global").
func (c *CommandLineQConf) ShowGlobalConfigurationContext(ctx context.Context, env []string) (*GlobalConfig, error) {
	out, err := c.RunCommandContext(ctx, env, "-sconf", "global")
	if err != nil {
		return nil, err
	}
	cfg := ParseGlobalConfigFromLines(strings.Split(out, "\n"))
	return &cfg, nil
}

// ShowAllParallelEnvironmentsContext returns every parallel environment
// ("qconf -spl", then one "qconf -sp <name>" per PE, as -sp takes a single
// name: one process per PE).
func (c *CommandLineQConf) ShowAllParallelEnvironmentsContext(ctx context.Context, env []string) ([]ParallelEnvironmentConfig, error) {
	names, err := c.listContext(ctx, env, "-spl", "parallel environment")
	if err != nil {
		return nil, err
	}
	pes := make([]ParallelEnvironmentConfig, 0, len(names))
	for _, name := range names {
		out, found, err := c.namedContext(ctx, env, "-sp", name, msgNotAParallelEnvironment)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		cfg, _ := ParseParallelEnvironmentConfigFromLines(strings.Split(out, "\n"))
		if cfg.Name == "" {
			cfg.Name = name
		}
		pes = append(pes, cfg)
	}
	return pes, nil
}

// ShowAllUserSetListsContext returns every user set list ("qconf -sul",
// then "qconf -su <a,b,...>": one call per maxListArgument bytes of
// names). A set deleted after the list call is left out.
func (c *CommandLineQConf) ShowAllUserSetListsContext(ctx context.Context, env []string) ([]UserSetListConfig, error) {
	names, err := c.listContext(ctx, env, "-sul", "userset list")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return []UserSetListConfig{}, nil
	}
	if err := validate.Enforce(validateListElements("userset list", names)); err != nil {
		return nil, err
	}
	sets := []UserSetListConfig{}
	for _, chunk := range chunkNames(names, maxListArgument) {
		stdout, stderr, err := c.runContext(ctx, env, "-su", strings.Join(chunk, ","))
		// -su prints every set it finds and one line per missing set,
		// then exits 1; only missing sets is not a failure.
		if err != nil && (ctx.Err() != nil || !onlyMissingAccessLists(stderr, chunk)) {
			return nil, err
		}
		sets = append(sets, ParseUserSetLists(stdout)...)
	}
	return sets, nil
}

// onlyMissingAccessLists reports whether stderr holds nothing but -su's
// "does not exist" lines for names it was asked for.
func onlyMissingAccessLists(stderr string, names []string) bool {
	lines := splitWithoutEmptyLines(stderr, "\n")
	if len(lines) == 0 {
		return false
	}
	missing := make(map[string]bool, len(names))
	for _, name := range names {
		missing[fmt.Sprintf(msgAccessListNotExist, nameInMessage(name))] = true
	}
	for _, line := range lines {
		if !missing[line] {
			return false
		}
	}
	return true
}

// ParseUserSetLists reads "qconf -su" output that holds any number of user
// set lists; each starts with its "name" line.
func ParseUserSetLists(output string) []UserSetListConfig {
	sets := []UserSetListConfig{}
	var block []string
	flush := func() {
		if len(block) == 0 {
			return
		}
		if cfg := parseUserSetList(block, ""); cfg.Name != "" {
			sets = append(sets, cfg)
		}
		block = nil
	}
	for _, line := range strings.Split(output, "\n") {
		if f := strings.Fields(line); len(f) > 0 && f[0] == "name" {
			flush()
		}
		block = append(block, line)
	}
	flush()
	return sets
}

// ShowHostGroupResolvedContext returns the hosts of one host group with
// its sub-groups resolved ("qconf -shgrp_resolved <group>"). A group that
// does not exist is an error.
func (c *CommandLineQConf) ShowHostGroupResolvedContext(ctx context.Context, env []string, group string) ([]string, error) {
	if err := validate.Enforce(validate.Operand(group), validate.StrictObjectName(group)); err != nil {
		return nil, err
	}
	out, err := c.RunCommandContext(ctx, env, "-shgrp_resolved", group)
	if err != nil {
		return nil, err
	}
	return resolvedHosts(out), nil
}

// resolvedHosts reads "qconf -shgrp_resolved" output. See
// ShowHostGroupResolved: one space-separated line, or one host per line;
// Fields handles both.
func resolvedHosts(out string) []string {
	hosts := strings.Fields(out)
	if hosts == nil {
		return []string{}
	}
	return hosts
}

// ShowAllHostGroupsResolvedContext returns the resolved hosts of every
// host group, keyed by group name ("qconf -shgrpl", then one "qconf
// -shgrp_resolved <group>" per group, as the option takes a single name:
// one process per group). A group deleted after the list call is left out.
func (c *CommandLineQConf) ShowAllHostGroupsResolvedContext(ctx context.Context, env []string) (map[string][]string, error) {
	groups, err := c.listContext(ctx, env, "-shgrpl", "host group list")
	if err != nil {
		return nil, err
	}
	resolved := make(map[string][]string, len(groups))
	for _, group := range groups {
		out, found, err := c.namedContext(ctx, env, "-shgrp_resolved", group, msgHostGroupNotExist)
		if err != nil {
			return nil, err
		}
		if found {
			resolved[group] = resolvedHosts(out)
		}
	}
	return resolved, nil
}

// ShowManagersContext returns the manager list ("qconf -sm").
func (c *CommandLineQConf) ShowManagersContext(ctx context.Context, env []string) ([]string, error) {
	return c.listContext(ctx, env, "-sm", "manager")
}

// ShowOperatorsContext returns the operator list ("qconf -so").
func (c *CommandLineQConf) ShowOperatorsContext(ctx context.Context, env []string) ([]string, error) {
	return c.listContext(ctx, env, "-so", "operator")
}

// ShowAdminHostsContext returns the administrative hosts ("qconf -sh").
func (c *CommandLineQConf) ShowAdminHostsContext(ctx context.Context, env []string) ([]string, error) {
	return c.listContext(ctx, env, "-sh", "administrative host")
}

// ShowSubmitHostsContext returns the submit hosts ("qconf -ss").
func (c *CommandLineQConf) ShowSubmitHostsContext(ctx context.Context, env []string) ([]string, error) {
	return c.listContext(ctx, env, "-ss", "submit host")
}

// ShowSchedulerHostContext returns the host the scheduler thread is
// registered on ("qconf -sss"). No registered scheduler is an error
// carrying qconf's words ("no scheduling host defined").
func (c *CommandLineQConf) ShowSchedulerHostContext(ctx context.Context, env []string) (string, error) {
	out, err := c.RunCommandContext(ctx, env, "-sss")
	if err != nil {
		return "", err
	}
	hosts := splitWithoutEmptyLines(out, "\n")
	if len(hosts) == 0 {
		return "", fmt.Errorf("qconf -sss: no scheduling host in output %q", out)
	}
	return hosts[0], nil
}

// ShowThreadsContext returns the qmaster thread pools and their sizes
// ("qconf -stl"). "qconf -stl"
// prints a "POOL SIZE" header, a dash line and then one "<name> <count>"
// row per thread pool; the two header lines are skipped and a row whose
// count is not a number is left out.
//
// qconf -stl exits 1 even when it prints the whole list (9.1.7, scheduler
// running or not), so a printed header is the answer whatever the exit
// code; only output without it is a failure.
func (c *CommandLineQConf) ShowThreadsContext(ctx context.Context, env []string) ([]ThreadInfo, error) {
	out, err := c.RunCommandContext(ctx, env, "-stl")
	if err != nil && (ctx.Err() != nil || !hasThreadListHeader(out)) {
		return nil, err
	}
	return ParseThreadList(out), nil
}

// hasThreadListHeader reports whether out carries the "POOL SIZE" header of
// a "qconf -stl" thread list.
func hasThreadListHeader(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == "POOL" && f[1] == "SIZE" {
			return true
		}
	}
	return false
}

// ParseThreadList reads "qconf -stl" output; see ShowThreadsContext for
// the format.
func ParseThreadList(output string) []ThreadInfo {
	threads := []ThreadInfo{}
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		count, err := strconv.Atoi(f[1])
		if err != nil {
			continue
		}
		threads = append(threads, ThreadInfo{Name: f[0], Count: count})
	}
	return threads
}

// ShowMyRolesContext returns the RBAC roles the calling user may use
// ("qconf -smyrole", GCS 9.1), one per line; no role is an empty list. On
// a build without RBAC the error wraps ErrNotSupported.
func (c *CommandLineQConf) ShowMyRolesContext(ctx context.Context, env []string) ([]string, error) {
	out, err := c.RunCommandContext(ctx, env, "-smyrole")
	if err != nil {
		if ctx.Err() == nil && strings.Contains(out, msgInvalidOption) {
			return nil, fmt.Errorf("qconf -smyrole: %w", ErrNotSupported)
		}
		if ctx.Err() == nil && noneDefined(out, "role") {
			return []string{}, nil
		}
		return nil, err
	}
	roles := splitWithoutEmptyLines(out, "\n")
	if roles == nil {
		return []string{}, nil
	}
	return roles, nil
}

// GetVersionContext detects the installed product and version from the
// client's "-help" banner, like GetVersion. It tries the
// configured executable first and qhost next to it as a fallback (see
// GetVersion for why), runs them with env instead of the process
// environment when env is not nil, and stops as soon as ctx ends.
func (c *CommandLineQConf) GetVersionContext(ctx context.Context, env []string) (ClusterSchedulerVersion, error) {
	if c.config.DryRun {
		fmt.Printf("Executing: %s -help\n", c.config.Executable)
		return ClusterSchedulerVersion{}, nil
	}
	candidates := []string{c.config.Executable}
	if qhost := deriveQhostPath(c.config.Executable); qhost != c.config.Executable {
		candidates = append(candidates, qhost)
	}
	// The CONFIGURED binary's failure is the one reported. A fallback that
	// is simply absent produces "no output from .../qhost", which would
	// otherwise mask the real cause -- a timeout, or qconf missing.
	var firstErr error
	for _, path := range candidates {
		version, err := c.versionFrom(ctx, env, path)
		if err == nil {
			return version, nil
		}
		if ctx.Err() != nil {
			return ClusterSchedulerVersion{}, err
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return ClusterSchedulerVersion{}, firstErr
}

// ShowAllCalendarsContext returns every calendar ("qconf -scall", then
// one "qconf -scal <name>" per calendar, as -scal takes a single name: one
// process per calendar).
func (c *CommandLineQConf) ShowAllCalendarsContext(ctx context.Context, env []string) ([]CalendarConfig, error) {
	names, err := c.listContext(ctx, env, "-scall", "calendar")
	if err != nil {
		return nil, err
	}
	calendars := make([]CalendarConfig, 0, len(names))
	for _, name := range names {
		out, found, err := c.namedContext(ctx, env, "-scal", name, msgNotACalendar)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		cfg, _ := ParseCalendarConfigFromLines(strings.Split(out, "\n"))
		if cfg.Name == "" {
			cfg.Name = name
		}
		calendars = append(calendars, cfg)
	}
	return calendars, nil
}

// ShowAllProjectsContext returns every project ("qconf -sprjl", then one
// "qconf -sprj <name>" per project, as -sprj takes a single name: one
// process per project).
func (c *CommandLineQConf) ShowAllProjectsContext(ctx context.Context, env []string) ([]ProjectConfig, error) {
	names, err := c.listContext(ctx, env, "-sprjl", "project list")
	if err != nil {
		return nil, err
	}
	projects := make([]ProjectConfig, 0, len(names))
	for _, name := range names {
		out, found, err := c.namedContext(ctx, env, "-sprj", name, msgNotAProject)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		projects = append(projects, parseProject(strings.Split(out, "\n"), name))
	}
	return projects, nil
}

// ShowAllCkptInterfacesContext returns every checkpointing interface
// ("qconf -sckptl", then one "qconf -sckpt <name>" per interface, as
// -sckpt takes a single name: one process per interface).
func (c *CommandLineQConf) ShowAllCkptInterfacesContext(ctx context.Context, env []string) ([]CkptInterfaceConfig, error) {
	names, err := c.listContext(ctx, env, "-sckptl", "ckpt interface definition")
	if err != nil {
		return nil, err
	}
	interfaces := make([]CkptInterfaceConfig, 0, len(names))
	for _, name := range names {
		out, found, err := c.namedContext(ctx, env, "-sckpt", name, msgNotACkptInterface)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		interfaces = append(interfaces, parseCkptInterface(strings.Split(out, "\n"), name))
	}
	return interfaces, nil
}
