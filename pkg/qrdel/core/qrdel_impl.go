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
	"regexp"
	"strconv"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ErrNotFound is wrapped by the delete methods when none of the selected
// ARs exists and nothing was deleted.
var ErrNotFound = errors.New("advance reservation not found")

// CommandError is returned when qrdel exits with a non-zero exit code. It
// keeps the exit code and the combined stdout/stderr output, which also
// lists the ARs deleted before the failure.
type CommandError struct {
	ExitCode int
	Output   string
}

// Error implements the error interface.
func (e *CommandError) Error() string {
	return fmt.Sprintf("qrdel exited with code %d: %s", e.ExitCode, strings.TrimSpace(e.Output))
}

// DeleteResult is the outcome of a qrdel call as reported in its output.
type DeleteResult struct {
	// Deleted holds the ids of the deleted ARs.
	Deleted []int64 `json:"deleted,omitempty"`
	// Registered holds the ids of ARs registered for deletion: they still
	// have running jobs and disappear once the jobs are gone.
	Registered []int64 `json:"registered,omitempty"`
	// Missing holds the selectors (ids, names or users) which matched no AR.
	Missing []string `json:"missing,omitempty"`
	// Denied holds the ids of ARs the caller may not delete.
	Denied []string `json:"denied,omitempty"`
}

var (
	deletedPattern    = regexp.MustCompile(`^\S+ has deleted advance_reservation ([0-9]+)$`)
	registeredPattern = regexp.MustCompile(`^\S+ has registered the advance_reservation ([0-9]+) for deletion$`)
	missingPattern    = regexp.MustCompile(`^denied: advance_reservation "(.*)" does not exist$`)
	deniedPattern     = regexp.MustCompile(`^\S+ - you do not have the necessary privileges to delete the advance_reservation "(.*)"$`)
)

// noUserARsPrefix starts the qrdel line for users without ARs (-u).
const noUserARsPrefix = "There is no advance_reservation registered for the following users: "

// ParseDeleteOutput collects the deleted, registered, missing and denied
// ARs from the output of qrdel. Lines it does not know are ignored.
func ParseDeleteOutput(output string) DeleteResult {
	var r DeleteResult
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := deletedPattern.FindStringSubmatch(line); m != nil {
			if id, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				r.Deleted = append(r.Deleted, id)
			}
		} else if m := registeredPattern.FindStringSubmatch(line); m != nil {
			if id, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				r.Registered = append(r.Registered, id)
			}
		} else if m := missingPattern.FindStringSubmatch(line); m != nil {
			r.Missing = append(r.Missing, m[1])
		} else if m := deniedPattern.FindStringSubmatch(line); m != nil {
			r.Denied = append(r.Denied, m[1])
		} else if users, ok := strings.CutPrefix(line, noUserARsPrefix); ok {
			for _, u := range strings.Split(users, ",") {
				if u = strings.TrimSpace(u); u != "" {
					r.Missing = append(r.Missing, u)
				}
			}
		}
	}
	return r
}

// CommandLineQRDelConfig configures CommandLineQRDel.
type CommandLineQRDelConfig struct {
	// Executable is the qrdel binary. Defaults to "qrdel".
	Executable string
	// DryRun runs nothing; the methods return the command line instead.
	DryRun bool
	// Force adds -f to every delete: the ARs and their jobs are removed
	// even if the execution daemons do not respond. It is a client setting
	// rather than a per-call option so that callers cannot escalate a
	// single request.
	Force bool
}

// CommandLineQRDel implements QRDel by running the qrdel binary.
type CommandLineQRDel struct {
	config CommandLineQRDelConfig
}

// NewCommandLineQRDel creates a QRDel client. Unless DryRun is set the
// executable must be found in PATH.
func NewCommandLineQRDel(config CommandLineQRDelConfig) (*CommandLineQRDel, error) {
	if config.Executable == "" {
		config.Executable = "qrdel"
	}
	if !config.DryRun {
		if _, err := exec.LookPath(config.Executable); err != nil {
			return nil, fmt.Errorf("executable not found: %w", err)
		}
	}
	return &CommandLineQRDel{config: config}, nil
}

// ValidateUsers checks a user list for qrdel -u: no comma, no space, no
// leading '-' and no control character per entry. Patterns such as "*" or
// "a*" are allowed; under a manager account they select the ARs of every
// matching user. Trust boundaries which take a user name from a request
// should use ValidateExactUsers. Neither is subject to GCS_VALIDATION.
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

// ValidateReservationName checks an AR name or pattern for qrdel. A comma
// would split it into two selectors and a leading '-' would turn it into a
// flag; qrdel itself refuses spaces and brackets. A leading digit or '+'
// is refused because qmaster treats any selector strtol can parse ("42",
// "+42", "0x2a", "052") as an AR id; AR names never start with a digit.
// It is not subject to GCS_VALIDATION, so trust boundaries can rely on it.
func ValidateReservationName(name string) error {
	if err := validate.ListElement(name); err != nil {
		return fmt.Errorf("advance reservation name %q: %w", name, err)
	}
	if c := name[0]; c == '+' || (c >= '0' && c <= '9') {
		return fmt.Errorf("advance reservation name %q: must not start with a digit or '+' (would select an AR id)", name)
	}
	return nil
}

// checkIDs rejects ids qmaster cannot address: qmaster reads AR ids as
// 32 bit unsigned values, so larger ids wrap around to other ARs.
func checkIDs(ids []int64) error {
	if len(ids) == 0 {
		return errors.New("no advance reservation ids specified")
	}
	for _, id := range ids {
		if id <= 0 || id > math.MaxUint32 {
			return fmt.Errorf("invalid advance reservation id %d", id)
		}
	}
	return nil
}

// buildDeleteArgs returns the qrdel arguments to delete ARs by id.
func buildDeleteArgs(force bool, ids ...int64) ([]string, error) {
	if err := checkIDs(ids); err != nil {
		return nil, err
	}
	list := make([]string, 0, len(ids))
	for _, id := range ids {
		list = append(list, strconv.FormatInt(id, 10))
	}
	return append(forceArgs(force), strings.Join(list, ",")), nil
}

// buildDeleteByNameArgs returns the qrdel arguments to delete the ARs of
// owner by name or pattern: qrdel -u owner name,...
func buildDeleteByNameArgs(force bool, owner string, names ...string) ([]string, error) {
	if len(names) == 0 {
		return nil, errors.New("no advance reservation names specified")
	}
	// The owner is what keeps the selection to one user's ARs, so it must
	// name exactly one user, independent of GCS_VALIDATION.
	if err := ValidateExactUsers([]string{owner}); err != nil {
		return nil, fmt.Errorf("owner: %w", err)
	}
	for _, name := range names {
		if err := validate.Enforce(ValidateReservationName(name), validate.StrictObjectName(name)); err != nil {
			return nil, err
		}
	}
	return append(forceArgs(force), "-u", owner, strings.Join(names, ",")), nil
}

// buildDeleteByUserArgs returns the qrdel arguments to delete all ARs of
// the given users.
func buildDeleteByUserArgs(force bool, users ...string) ([]string, error) {
	if len(users) == 0 {
		return nil, errors.New("no users specified")
	}
	if err := validate.Enforce(ValidateUsers(users)); err != nil {
		return nil, err
	}
	return append(forceArgs(force), "-u", strings.Join(users, ",")), nil
}

func forceArgs(force bool) []string {
	if force {
		return []string{"-f"}
	}
	return nil
}

// run executes qrdel and returns its combined output. A non-zero exit code
// yields a *CommandError together with the output; it also wraps
// ErrNotFound if nothing was deleted because nothing matched.
func (c *CommandLineQRDel) run(ctx context.Context, args []string) (string, error) {
	// Layer 1 guard: reject control characters and invalid UTF-8 in any
	// argv token before spawning qrdel or printing the dry-run line.
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", err
	}
	if c.config.DryRun {
		return fmt.Sprintf("Dry run: %s %s", c.config.Executable, strings.Join(args, " ")), nil
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, c.config.Executable, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to run %s: %w", c.config.Executable, err)
		}
		cmdErr := &CommandError{ExitCode: exitErr.ExitCode(), Output: out.String()}
		r := ParseDeleteOutput(out.String())
		if len(r.Missing) > 0 && len(r.Deleted) == 0 && len(r.Registered) == 0 && len(r.Denied) == 0 {
			return out.String(), fmt.Errorf("%w: %w", ErrNotFound, cmdErr)
		}
		return out.String(), cmdErr
	}
	return out.String(), nil
}

// DeleteReservations implements QRDel.
func (c *CommandLineQRDel) DeleteReservations(ctx context.Context, ids ...int64) (string, error) {
	args, err := buildDeleteArgs(c.config.Force, ids...)
	if err != nil {
		return "", err
	}
	return c.run(ctx, args)
}

// DeleteReservationsByName implements QRDel.
func (c *CommandLineQRDel) DeleteReservationsByName(ctx context.Context, owner string, names ...string) (string, error) {
	args, err := buildDeleteByNameArgs(c.config.Force, owner, names...)
	if err != nil {
		return "", err
	}
	return c.run(ctx, args)
}

// DeleteReservationsByUser implements QRDel.
func (c *CommandLineQRDel) DeleteReservationsByUser(ctx context.Context, users ...string) (string, error) {
	args, err := buildDeleteByUserArgs(c.config.Force, users...)
	if err != nil {
		return "", err
	}
	return c.run(ctx, args)
}

// NativeSpecification implements QRDel.
func (c *CommandLineQRDel) NativeSpecification(ctx context.Context, args []string) (string, error) {
	return c.run(ctx, args)
}
