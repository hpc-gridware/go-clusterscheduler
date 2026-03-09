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

func (c *CommandLineQAlter) SetBindingAmount(jobTaskList string, amount int) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bamount", strconv.Itoa(amount), jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingFilter(jobTaskList, topology string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bfilter", topology, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingInstance(jobTaskList, instance string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-binstance", instance, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingSortOrder(jobTaskList, order string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bsort", order, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingStart(jobTaskList, position string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bstart", position, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingStop(jobTaskList, position string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bstop", position, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingStrategy(jobTaskList, strategy string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bstrategy", strategy, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingType(jobTaskList, bindingType string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-btype", bindingType, jobTaskList)
	return c.RunCommand(args...)
}

func (c *CommandLineQAlter) SetBindingUnit(jobTaskList, unit string) (string, error) {
	args := c.GlobalArgs()
	args = append(args, "-bunit", unit, jobTaskList)
	return c.RunCommand(args...)
}
