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
	"fmt"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ConfigContextReader reads configuration under the caller's context and
// with the caller's environment: for callers that run qconf in a loop with
// their own budget and shutdown (a background indexer), where the
// context-free QConf methods would block past both. It is deliberately not
// part of QConf, so adapters that implement QConf are not affected.
type ConfigContextReader interface {
	// ShowAllResourceQuotaSetsContext returns every resource quota set in
	// one qconf call ("qconf -srqs" without a name), so a set deleted
	// meanwhile cannot fail a per-set read. No set defined is an empty
	// list, not an error.
	ShowAllResourceQuotaSetsContext(ctx context.Context, env []string) ([]ResourceQuotaSetConfig, error)
	// ShowAllComplexesContext returns the complex table ("qconf -sc").
	ShowAllComplexesContext(ctx context.Context, env []string) ([]ComplexEntryConfig, error)
	// ShowExecHostContext returns one execution host's configuration
	// ("qconf -se <host>"), the "global" host included.
	ShowExecHostContext(ctx context.Context, env []string, host string) (HostExecConfig, error)
	// ShowAllClusterQueuesContext returns every cluster queue's
	// configuration in two qconf calls ("qconf -sql", then one "qconf -sq
	// <q1,q2,...>"). No cluster queue defined is an empty list. A queue
	// deleted between the two calls fails the whole read; retry it.
	ShowAllClusterQueuesContext(ctx context.Context, env []string) ([]ClusterQueueConfig, error)
}

var _ ConfigContextReader = (*CommandLineQConf)(nil)

// msgNoResourceQuotaSet is qconf's answer (exit 1) when no resource quota
// set exists (MSG_NORQSFOUND in the scheduler's msg_clients_common.h).
const msgNoResourceQuotaSet = "No resource quota set found"

// msgNoClusterQueue is "qconf -sql"'s answer (exit 1) when no cluster queue
// exists (MSG_QCONF_NOXDEFINED_S with "cqueue list" in msg_common.h).
const msgNoClusterQueue = "no cqueue list defined"

// RunCommandContext runs qconf like RunCommand, but ends with ctx (and
// still at the configured Timeout, whichever comes first), and runs with
// env instead of the process environment when env is not nil. A caller
// that passes env should set LC_ALL=C: some answers are recognised by
// their English text. SGE_SINGLE_LINE=true is always added.
//
// stdout and stderr are kept apart. On success only stdout is returned, so
// a warning on stderr never becomes parser input. On failure stdout
// followed by stderr is returned, so a caller can still recognise an
// answer such as "no <what> defined" whichever stream qconf used, and the
// error carries only the first helper.MaxStderrBytes of stderr, never the
// configuration printed on stdout.
func (c *CommandLineQConf) RunCommandContext(ctx context.Context, env []string, args ...string) (string, error) {
	stdout, stderr, err := c.runContext(ctx, env, args...)
	if err != nil {
		return stdout + stderr, err
	}
	return stdout, nil
}

// runContext is RunCommandContext returning stdout and the capped stderr
// separately, for a caller that parses stdout of a failed call.
func (c *CommandLineQConf) runContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if c.config.DryRun {
		fmt.Printf("Executing: %s, %v", c.config.Executable, args)
		return "", "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	return helper.RunContext(ctx, c.config.Executable, env, []string{"SGE_SINGLE_LINE=true"}, args...)
}

// ShowAllResourceQuotaSetsContext implements ConfigContextReader.
func (c *CommandLineQConf) ShowAllResourceQuotaSetsContext(ctx context.Context, env []string) ([]ResourceQuotaSetConfig, error) {
	out, err := c.RunCommandContext(ctx, env, "-srqs")
	if err != nil {
		if ctx.Err() == nil && strings.Contains(out, msgNoResourceQuotaSet) {
			return []ResourceQuotaSetConfig{}, nil
		}
		return nil, err
	}
	return ParseResourceQuotaSets(out), nil
}

// ShowAllComplexesContext implements ConfigContextReader.
func (c *CommandLineQConf) ShowAllComplexesContext(ctx context.Context, env []string) ([]ComplexEntryConfig, error) {
	out, err := c.RunCommandContext(ctx, env, "-sc")
	if err != nil {
		return nil, err
	}
	return parseAllComplexes(out)
}

// ParseResourceQuotaSets reads "qconf -srqs" output that holds any number
// of sets, each in its own "{ ... }" block. A block without a name is not
// a set and is left out.
func ParseResourceQuotaSets(output string) []ResourceQuotaSetConfig {
	sets := []ResourceQuotaSetConfig{}
	var block []string
	in := false
	for _, line := range strings.Split(output, "\n") {
		switch strings.TrimSpace(line) {
		case "{":
			in, block = true, nil
		case "}":
			if in {
				if cfg := parseResourceQuotaSet(block, ""); cfg.Name != "" {
					sets = append(sets, cfg)
				}
			}
			in = false
		default:
			if in {
				block = append(block, line)
			}
		}
	}
	return sets
}

// ShowExecHostContext implements ConfigContextReader.
func (c *CommandLineQConf) ShowExecHostContext(ctx context.Context, env []string, host string) (HostExecConfig, error) {
	if err := validate.Enforce(validate.Operand(host), validate.StrictObjectName(host)); err != nil {
		return HostExecConfig{}, err
	}
	out, err := c.RunCommandContext(ctx, env, "-se", host)
	if err != nil {
		return HostExecConfig{}, err
	}
	cfg, err := ParseExecHostConfigFromLines(strings.Split(out, "\n"))
	if err != nil {
		return HostExecConfig{}, fmt.Errorf("failed to parse exec host %s: %w", host, err)
	}
	if cfg.Name == "" {
		cfg.Name = host
	}
	return cfg, nil
}

// ShowAllClusterQueuesContext implements ConfigContextReader.
func (c *CommandLineQConf) ShowAllClusterQueuesContext(ctx context.Context, env []string) ([]ClusterQueueConfig, error) {
	out, err := c.RunCommandContext(ctx, env, "-sql")
	if err != nil {
		if ctx.Err() == nil && strings.Contains(out, msgNoClusterQueue) {
			return []ClusterQueueConfig{}, nil
		}
		return nil, err
	}
	names := splitWithoutEmptyLines(out, "\n")
	if len(names) == 0 {
		return []ClusterQueueConfig{}, nil
	}
	if err := validate.Enforce(validate.List("cluster queue", names)); err != nil {
		return nil, err
	}
	out, err = c.RunCommandContext(ctx, env, "-sq", strings.Join(names, ","))
	if err != nil {
		return nil, err
	}
	return ParseClusterQueues(out), nil
}

// ParseClusterQueues reads "qconf -sq" output that holds any number of
// cluster queues; each starts with its "qname" line.
func ParseClusterQueues(output string) []ClusterQueueConfig {
	queues := []ClusterQueueConfig{}
	var block []string
	flush := func() {
		if len(block) == 0 {
			return
		}
		if cfg := parseClusterQueue(block, ""); cfg.Name != "" {
			queues = append(queues, cfg)
		}
		block = nil
	}
	for _, line := range strings.Split(output, "\n") {
		if f := strings.Fields(line); len(f) > 0 && f[0] == "qname" {
			flush()
		}
		block = append(block, line)
	}
	flush()
	return queues
}
