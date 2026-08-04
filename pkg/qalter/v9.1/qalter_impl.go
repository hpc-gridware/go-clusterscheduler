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

package qalter

import (
	"strconv"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
)

// CommandLineQAlter extends the core implementation with v9.1-specific
// binding options.
type CommandLineQAlter struct {
	core.CommandLineQAlter
}

// CommandLineQAlterConfig is a type alias to the core configuration.
type CommandLineQAlterConfig = core.CommandLineQAlterConfig

// NewCommandLineQAlter creates a new instance of CommandLineQAlter.
func NewCommandLineQAlter(config CommandLineQAlterConfig) (*CommandLineQAlter, error) {
	c, err := core.NewCommandLineQAlter(config)
	if err != nil {
		return nil, err
	}
	return &CommandLineQAlter{CommandLineQAlter: *c}, nil
}

// --- Binding ---

// runBinding validates the job/task list (matching the core qalter methods,
// which the v9.1 binding methods otherwise bypass) then runs qalter with the
// binding flag and value followed by the job/task list.
func (c *CommandLineQAlter) runBinding(jobTaskList string, flagAndValue ...string) (string, error) {
	if err := validate.Enforce(validate.JobTaskList(jobTaskList)); err != nil {
		return "", err
	}
	args := c.GlobalArgs()
	args = append(args, flagAndValue...)
	args = append(args, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingAmount(jobTaskList string, amount int) (string, error) {
	return c.runBinding(jobTaskList, "-bamount", strconv.Itoa(amount))
}

func (c *CommandLineQAlter) SetBindingFilter(jobTaskList, topology string) (string, error) {
	return c.runBinding(jobTaskList, "-bfilter", topology)
}

func (c *CommandLineQAlter) SetBindingInstance(jobTaskList, instance string) (string, error) {
	return c.runBinding(jobTaskList, "-binstance", instance)
}

func (c *CommandLineQAlter) SetBindingSortOrder(jobTaskList, order string) (string, error) {
	return c.runBinding(jobTaskList, "-bsort", order)
}

func (c *CommandLineQAlter) SetBindingStart(jobTaskList, position string) (string, error) {
	return c.runBinding(jobTaskList, "-bstart", position)
}

func (c *CommandLineQAlter) SetBindingStop(jobTaskList, position string) (string, error) {
	return c.runBinding(jobTaskList, "-bstop", position)
}

func (c *CommandLineQAlter) SetBindingStrategy(jobTaskList, strategy string) (string, error) {
	return c.runBinding(jobTaskList, "-bstrategy", strategy)
}

func (c *CommandLineQAlter) SetBindingType(jobTaskList, bindingType string) (string, error) {
	return c.runBinding(jobTaskList, "-btype", bindingType)
}

func (c *CommandLineQAlter) SetBindingUnit(jobTaskList, unit string) (string, error) {
	return c.runBinding(jobTaskList, "-bunit", unit)
}
