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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

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
}

var _ ConfigContextReader = (*CommandLineQConf)(nil)

// msgNoResourceQuotaSet is qconf's answer (exit 1) when no resource quota
// set exists (MSG_NORQSFOUND in the scheduler's msg_clients_common.h).
const msgNoResourceQuotaSet = "No resource quota set found"

// RunCommandContext runs qconf like RunCommand, but ends with ctx (and
// still at the configured Timeout, whichever comes first), and runs with
// env instead of the process environment when env is not nil. A caller
// that passes env should set LC_ALL=C: some answers are recognised by
// their English text. SGE_SINGLE_LINE=true is always added.
func (c *CommandLineQConf) RunCommandContext(ctx context.Context, env []string, args ...string) (string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", err
	}
	if c.config.DryRun {
		fmt.Printf("Executing: %s, %v", c.config.Executable, args)
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.config.Executable, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	// See RunCommand: without WaitDelay a grandchild holding the pipes
	// keeps Run waiting after ctx has ended.
	cmd.WaitDelay = time.Second
	if env == nil {
		env = cmd.Environ()
	}
	cmd.Env = append(append([]string{}, env...), "SGE_SINGLE_LINE=true")
	err := cmd.Run()
	if ctx.Err() != nil {
		return out.String(), fmt.Errorf("qconf %s: %w", strings.Join(args, " "), ctx.Err())
	}
	if err != nil {
		return out.String(), fmt.Errorf("failed to run command (%s): %v", out.String(), err)
	}
	return out.String(), nil
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
