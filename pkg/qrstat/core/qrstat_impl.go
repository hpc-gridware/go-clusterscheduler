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
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ErrNotFound is wrapped by GetReservations when none of the requested ARs
// exists.
var ErrNotFound = errors.New("advance reservation not found")

// CommandError is returned when qrstat exits with a non-zero exit code. It
// keeps the exit code and the combined stdout/stderr output.
type CommandError struct {
	ExitCode int
	Output   string
}

// Error implements the error interface.
func (e *CommandError) Error() string {
	return fmt.Sprintf("qrstat exited with code %d: %s", e.ExitCode, strings.TrimSpace(e.Output))
}

// CommandLineQRStatConfig configures CommandLineQRStat.
type CommandLineQRStatConfig struct {
	// Executable is the qrstat binary. Defaults to "qrstat".
	Executable string
	// DryRun prints nothing and runs nothing: NativeSpecification returns
	// the command line it would run, the typed methods return no ARs.
	DryRun bool
}

// CommandLineQRStat implements QRStat by running the qrstat binary.
type CommandLineQRStat struct {
	config CommandLineQRStatConfig
}

// NewCommandLineQRStat creates a QRStat client. Unless DryRun is set the
// executable must be found in PATH.
func NewCommandLineQRStat(config CommandLineQRStatConfig) (*CommandLineQRStat, error) {
	if config.Executable == "" {
		config.Executable = "qrstat"
	}
	if !config.DryRun {
		if _, err := exec.LookPath(config.Executable); err != nil {
			return nil, fmt.Errorf("executable not found: %w", err)
		}
	}
	return &CommandLineQRStat{config: config}, nil
}

// ValidateUsers checks a user list for qrstat -u. Each entry must be a
// single list element: no comma, no space, no leading '-' and no control
// character. Patterns such as "*" (all users) or "a*" are allowed. Trust
// boundaries which take a user name from a request should use
// ValidateExactUsers. Neither is subject to GCS_VALIDATION.
func ValidateUsers(users []string) error {
	return validate.List("user", users)
}

// ValidateExactUsers is ValidateUsers without patterns: it also rejects
// '*', '?', '[' and ']', so every entry names exactly one user.
func ValidateExactUsers(users []string) error {
	if err := ValidateUsers(users); err != nil {
		return err
	}
	for _, u := range users {
		if i := strings.IndexAny(u, "*?[]"); i >= 0 {
			return fmt.Errorf("user %q: must not contain the pattern character %q", u, u[i])
		}
	}
	return nil
}

// buildListArgs returns the qrstat arguments for ListReservations.
func buildListArgs(opts ListOptions) ([]string, error) {
	if err := validate.Enforce(ValidateUsers(opts.Users)); err != nil {
		return nil, err
	}
	var args []string
	if len(opts.Users) > 0 {
		args = append(args, "-u", strings.Join(opts.Users, ","))
	}
	if opts.Explain {
		args = append(args, "-explain")
	}
	return args, nil
}

// buildGetArgs returns the qrstat arguments for GetReservations.
func buildGetArgs(ids ...int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, errors.New("no advance reservation ids specified")
	}
	list := make([]string, 0, len(ids))
	for _, id := range ids {
		// qmaster reads AR ids as 32 bit unsigned values; larger ids would
		// address other ARs.
		if id <= 0 || id > math.MaxUint32 {
			return nil, fmt.Errorf("invalid advance reservation id %d", id)
		}
		list = append(list, strconv.FormatInt(id, 10))
	}
	return []string{"-ar", strings.Join(list, ",")}, nil
}

// run executes qrstat and returns stdout. A non-zero exit code yields a
// *CommandError holding stdout and stderr. Unlike qrsub and qrdel, stdout
// and stderr are kept apart so that diagnostics on stderr never reach the
// parsers.
func (c *CommandLineQRStat) run(ctx context.Context, args []string) (string, error) {
	// Layer 1 guard: reject control characters and invalid UTF-8 in any
	// argv token before spawning qrstat or printing the dry-run line.
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", err
	}
	if c.config.DryRun {
		return fmt.Sprintf("Dry run: %s %s", c.config.Executable, strings.Join(args, " ")), nil
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.config.Executable, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdout.String(), &CommandError{
				ExitCode: exitErr.ExitCode(),
				Output:   stdout.String() + stderr.String(),
			}
		}
		return "", fmt.Errorf("failed to run %s: %w", c.config.Executable, err)
	}
	return stdout.String(), nil
}

// ListReservations implements QRStat.
func (c *CommandLineQRStat) ListReservations(ctx context.Context, opts ListOptions) ([]ReservationSummary, error) {
	args, err := buildListArgs(opts)
	if err != nil {
		return nil, err
	}
	out, err := c.run(ctx, args)
	if err != nil {
		return nil, err
	}
	if c.config.DryRun {
		return []ReservationSummary{}, nil
	}
	return ParseSummary(out)
}

// GetReservations implements QRStat.
func (c *CommandLineQRStat) GetReservations(ctx context.Context, ids ...int64) ([]Reservation, error) {
	args, err := buildGetArgs(ids...)
	if err != nil {
		return nil, err
	}
	out, err := c.run(ctx, args)
	if err != nil {
		if strings.HasPrefix(strings.TrimSpace(out), notFoundHeader) {
			return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return nil, err
	}
	if c.config.DryRun {
		return []Reservation{}, nil
	}
	return ParseDetail(out)
}

// NativeSpecification implements QRStat.
func (c *CommandLineQRStat) NativeSpecification(ctx context.Context, args []string) (string, error) {
	return c.run(ctx, args)
}
