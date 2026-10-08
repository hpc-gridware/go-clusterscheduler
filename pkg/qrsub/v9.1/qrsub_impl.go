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

package qrsub

import (
	"context"
	"strconv"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
)

// CommandLineQRSub implements the 9.1 QRSub by running the qrsub binary.
// It builds on the core client and adds the binding switches.
type CommandLineQRSub struct {
	coreClient *core.CommandLineQRSub
}

// NewCommandLineQRSub creates a 9.1 QRSub client. Unless DryRun is set the
// executable must be found in PATH.
func NewCommandLineQRSub(config CommandLineQRSubConfig) (*CommandLineQRSub, error) {
	c, err := core.NewCommandLineQRSub(config)
	if err != nil {
		return nil, err
	}
	return &CommandLineQRSub{coreClient: c}, nil
}

// validateBinding checks the binding switches against the qrsub grammar.
func validateBinding(opts ReservationOptions) error {
	for _, b := range bindingArgs(opts) {
		if err := core.ValidateBindingSwitch(b.flag, b.value); err != nil {
			return err
		}
	}
	return nil
}

type bindingArg struct {
	flag, value string
}

// bindingArgs lists the set binding switches in argv order.
func bindingArgs(opts ReservationOptions) []bindingArg {
	var args []bindingArg
	if opts.BindingAmount != nil {
		args = append(args, bindingArg{"-bamount", strconv.Itoa(*opts.BindingAmount)})
	}
	for _, s := range []struct {
		flag  string
		value *string
	}{
		{"-bunit", opts.BindingUnit},
		{"-bfilter", opts.BindingFilter},
		{"-bsort", opts.BindingSort},
		{"-bstart", opts.BindingStart},
		{"-bstop", opts.BindingStop},
		{"-bstrategy", opts.BindingStrategy},
		{"-binstance", opts.BindingInstance},
	} {
		if s.value != nil {
			args = append(args, bindingArg{s.flag, *s.value})
		}
	}
	return args
}

// ValidateReservationOptions applies all checks of BuildQrsubArgs to opts,
// independent of GCS_VALIDATION, so trust boundaries can rely on it.
func ValidateReservationOptions(opts ReservationOptions) error {
	if err := core.ValidateReservationOptions(opts.ReservationOptions); err != nil {
		return err
	}
	return validateBinding(opts)
}

// BuildQrsubArgs returns the qrsub arguments for opts: the common options
// followed by the binding switches.
func BuildQrsubArgs(opts ReservationOptions) ([]string, error) {
	args, err := core.BuildQrsubArgs(opts.ReservationOptions)
	if err != nil {
		return nil, err
	}
	if err := validate.Enforce(validateBinding(opts)); err != nil {
		return nil, err
	}
	for _, b := range bindingArgs(opts) {
		args = append(args, b.flag, b.value)
	}
	return args, nil
}

// Submit implements QRSub.
func (c *CommandLineQRSub) Submit(ctx context.Context, opts ReservationOptions) (int64, string, error) {
	args, err := BuildQrsubArgs(opts)
	if err != nil {
		return 0, "", err
	}
	return core.SubmitArgs(ctx, c.coreClient, args)
}

// Verify implements QRSub.
func (c *CommandLineQRSub) Verify(ctx context.Context, opts ReservationOptions) (bool, string, error) {
	args, err := BuildQrsubArgs(opts)
	if err != nil {
		return false, "", err
	}
	return core.VerifyArgs(ctx, c.coreClient, args)
}

// SubmitWithNativeSpecification implements QRSub.
func (c *CommandLineQRSub) SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error) {
	return c.coreClient.SubmitWithNativeSpecification(ctx, args)
}
