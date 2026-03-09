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

package qsub

import (
	"context"
	"fmt"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
)

type qsubClient struct {
	coreClient *core.QSubClient
}

// NewCommandLineQSub creates a new v9.1 Qsub client.
func NewCommandLineQSub(config CommandLineQSubConfig) (Qsub, error) {
	coreQsub, err := core.NewCommandLineQSub(config)
	if err != nil {
		return nil, err
	}
	return &qsubClient{
		coreClient: coreQsub.(*core.QSubClient),
	}, nil
}

func (c *qsubClient) SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error) {
	return c.coreClient.SubmitWithNativeSpecification(ctx, args)
}

func (c *qsubClient) Submit(ctx context.Context, opts JobOptions) (int64, string, error) {
	if opts.Terse == nil {
		opts.Terse = &True
	}

	// Build common args from core options, but clear legacy binding
	coreOpts := opts.JobOptions
	coreOpts.ProcessorBinding = nil
	cmdArgs, err := core.BuildQsubArgs(coreOpts)
	if err != nil {
		return 0, "", err
	}

	// Append v9.1 binding options
	cmdArgs = appendBindingArgs(cmdArgs, opts)

	output, err := c.coreClient.SubmitWithNativeSpecification(ctx, cmdArgs)
	if err != nil {
		return 0, output, err
	}

	return core.ParseSubmitOutput(output, opts.Terse != nil && *opts.Terse,
		opts.Synchronize != nil && *opts.Synchronize, c.coreClient.IsDryRun())
}

func (c *qsubClient) SubmitSimple(ctx context.Context, additionalOptions *JobOptions, command string, args ...string) (int64, string, error) {
	if additionalOptions == nil {
		additionalOptions = &JobOptions{}
	}
	additionalOptions.Command = command
	additionalOptions.CommandArgs = args
	return c.Submit(ctx, *additionalOptions)
}

func (c *qsubClient) SubmitSimpleBinary(ctx context.Context, command string, args ...string) (int64, string, error) {
	opts := JobOptions{}
	opts.Command = command
	opts.CommandArgs = args
	opts.Binary = core.ToPtr(true)
	return c.Submit(ctx, opts)
}

func appendBindingArgs(args []string, opts JobOptions) []string {
	if opts.BindingAmount != nil {
		args = append(args, "-bamount", fmt.Sprintf("%d", *opts.BindingAmount))
	}
	if opts.BindingStop != nil {
		args = append(args, "-bstop", *opts.BindingStop)
	}
	if opts.BindingFilter != nil {
		args = append(args, "-bfilter", *opts.BindingFilter)
	}
	if opts.BindingInstance != nil {
		args = append(args, "-binstance", *opts.BindingInstance)
	}
	if opts.BindingSort != nil {
		args = append(args, "-bsort", *opts.BindingSort)
	}
	if opts.BindingStart != nil {
		args = append(args, "-bstart", *opts.BindingStart)
	}
	if opts.BindingStrategy != nil {
		args = append(args, "-bstrategy", *opts.BindingStrategy)
	}
	if opts.BindingType != nil {
		args = append(args, "-btype", *opts.BindingType)
	}
	if opts.BindingUnit != nil {
		args = append(args, "-bunit", *opts.BindingUnit)
	}
	return args
}
