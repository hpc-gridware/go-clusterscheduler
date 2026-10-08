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
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// exitCodeLimitReached is the qrsub exit code when the cluster wide
// max_advance_reservations limit is reached.
const exitCodeLimitReached = 25

// dryRunPrefix starts the output of a dry run.
const dryRunPrefix = "Dry run:"

var grantedPattern = regexp.MustCompile(`(?m)^Your advance reservation ([0-9]+) has been granted$`)

// Lines of qrsub -w v. "found possible assignment" is printed for ARs with
// a parallel environment.
const (
	verifyFoundQueues     = "verification: found suitable queue(s)"
	verifyFoundAssignment = "verification: found possible assignment with "
	verifyNoQueues        = "verification: no suitable queues"
)

// CommandError is returned when qrsub exits with a non-zero exit code. It
// keeps the exit code and the combined stdout/stderr output, which holds
// the reason (for example "advance_reservation: no suitable queues").
type CommandError struct {
	ExitCode int
	Output   string
}

// Error implements the error interface.
func (e *CommandError) Error() string {
	return fmt.Sprintf("qrsub exited with code %d: %s", e.ExitCode, strings.TrimSpace(e.Output))
}

// limitReachedText is part of the qmaster message for the AR limit.
const limitReachedText = "advance reservations are allowed per cluster"

// IsLimitReached reports whether err says that the cluster wide limit of
// advance reservations (max_advance_reservations) is reached. The exit code
// 25 is used for other "try again" rejections too, so the message is
// checked as well.
func IsLimitReached(err error) bool {
	var cmdErr *CommandError
	return errors.As(err, &cmdErr) && cmdErr.ExitCode == exitCodeLimitReached &&
		strings.Contains(cmdErr.Output, limitReachedText)
}

// CommandLineQRSubConfig configures CommandLineQRSub.
type CommandLineQRSubConfig struct {
	// Executable is the qrsub binary. Defaults to "qrsub".
	Executable string
	// DryRun runs nothing; the output is the command line instead.
	DryRun bool
}

// CommandLineQRSub implements QRSub by running the qrsub binary.
type CommandLineQRSub struct {
	config CommandLineQRSubConfig
}

// NewCommandLineQRSub creates a QRSub client. Unless DryRun is set the
// executable must be found in PATH.
func NewCommandLineQRSub(config CommandLineQRSubConfig) (*CommandLineQRSub, error) {
	if config.Executable == "" {
		config.Executable = "qrsub"
	}
	if !config.DryRun {
		if _, err := exec.LookPath(config.Executable); err != nil {
			return nil, fmt.Errorf("executable not found: %w", err)
		}
	}
	return &CommandLineQRSub{config: config}, nil
}

// FormatDateTime formats t in the local time zone as qrsub date_time
// [[CC]YY]MMDDhhmm[.SS]. qrsub has no zone offset in its date format, so
// an instant within the repeated hour at the end of daylight saving time
// is ambiguous; use Duration rather than an end time near such a switch.
func FormatDateTime(t time.Time) string {
	return t.In(time.Local).Format("200601021504.05")
}

// formatDuration formats d as qrsub time hh:mm:ss.
func formatDuration(d time.Duration) string {
	return helper.FormatSecondsToTimeResourceValue(int64(d / time.Second))
}

func yesNo(b bool) string {
	if b {
		return "y"
	}
	return "n"
}

// BuildQrsubArgs returns the qrsub arguments for opts. Requests qmaster
// would reject (no end time or duration, PE name without slots, ...) are
// always errors; the argument injection checks follow GCS_VALIDATION.
func BuildQrsubArgs(opts ReservationOptions) ([]string, error) {
	if err := checkPreconditions(opts); err != nil {
		return nil, err
	}
	if err := validate.Enforce(checkFields(opts)); err != nil {
		return nil, err
	}
	var args []string
	if opts.StartTime != nil {
		args = append(args, "-a", FormatDateTime(*opts.StartTime))
	}
	if opts.EndTime != nil {
		args = append(args, "-e", FormatDateTime(*opts.EndTime))
	}
	if opts.Duration != nil {
		args = append(args, "-d", formatDuration(*opts.Duration))
	}
	if opts.Name != nil {
		args = append(args, "-N", *opts.Name)
	}
	if opts.Account != nil {
		args = append(args, "-A", *opts.Account)
	}
	if len(opts.Resources) > 0 {
		pairs := make([]string, 0, len(opts.Resources))
		for _, name := range sortedKeys(opts.Resources) {
			pairs = append(pairs, name+"="+opts.Resources[name])
		}
		args = append(args, "-l", strings.Join(pairs, ","))
	}
	if len(opts.Queues) > 0 {
		args = append(args, "-q", strings.Join(opts.Queues, ","))
	}
	if len(opts.MasterQueues) > 0 {
		args = append(args, "-masterq", strings.Join(opts.MasterQueues, ","))
	}
	if opts.PEName != nil {
		args = append(args, "-pe", *opts.PEName, *opts.PESlots)
	}
	if opts.Checkpoint != nil {
		args = append(args, "-ckpt", *opts.Checkpoint)
	}
	if len(opts.Users) > 0 {
		args = append(args, "-u", strings.Join(opts.Users, ","))
	}
	if opts.MailOptions != nil {
		args = append(args, "-m", *opts.MailOptions)
	}
	if len(opts.MailList) > 0 {
		args = append(args, "-M", strings.Join(opts.MailList, ","))
	}
	if opts.HardErrorHandling != nil {
		args = append(args, "-he", yesNo(*opts.HardErrorHandling))
	}
	if opts.Immediate != nil {
		args = append(args, "-now", yesNo(*opts.Immediate))
	}
	return args, nil
}

// ParseQrsubOutput returns the id from the qrsub success message
// "Your advance reservation <id> has been granted".
func ParseQrsubOutput(output string) (int64, error) {
	match := grantedPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, fmt.Errorf("qrsub did not report a granted advance reservation: %s", strings.TrimSpace(output))
	}
	id, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid advance reservation id %q", match[1])
	}
	return id, nil
}

// ParseVerifyOutput interprets the output of qrsub -w v. It returns true
// if suitable resources were found, false if not, and an error if the
// output is not recognised (so an unexpected answer never reads as yes).
func ParseVerifyOutput(output string) (bool, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == verifyFoundQueues, strings.HasPrefix(line, verifyFoundAssignment):
			return true, nil
		case line == verifyNoQueues:
			return false, nil
		}
	}
	return false, fmt.Errorf("unrecognised qrsub verification output: %s", strings.TrimSpace(output))
}

// NativeSubmitter runs qrsub with raw arguments. CommandLineQRSub and the
// version specific clients implement it.
type NativeSubmitter interface {
	SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error)
}

// SubmitArgs runs qrsub with args and returns the id of the granted AR.
// It is exported for the version packages and ReservationBuilder. In dry
// run mode the id is 0 and the output is the command line; a dry run is
// recognised by the "Dry run:" prefix of the output, which real qrsub
// output never has.
func SubmitArgs(ctx context.Context, s NativeSubmitter, args []string) (int64, string, error) {
	output, err := s.SubmitWithNativeSpecification(ctx, args)
	if err != nil {
		return 0, output, err
	}
	if strings.HasPrefix(output, dryRunPrefix) {
		return 0, output, nil
	}
	id, err := ParseQrsubOutput(output)
	return id, output, err
}

// VerifyArgs runs qrsub -w v with args. It is exported for the version
// packages and ReservationBuilder. A request which cannot be granted yields
// false and no error; qrsub exits with 1 in that case. A "found" answer
// counts only together with a successful exit. In dry run mode the result
// is false and the output is the command line.
func VerifyArgs(ctx context.Context, s NativeSubmitter, args []string) (bool, string, error) {
	verifyArgs := append(slices.Clone(args), "-w", "v")
	output, err := s.SubmitWithNativeSpecification(ctx, verifyArgs)
	if err == nil && strings.HasPrefix(output, dryRunPrefix) {
		return false, output, nil
	}
	found, parseErr := ParseVerifyOutput(output)
	if parseErr == nil && !found {
		return false, output, nil
	}
	if err != nil {
		return false, output, err
	}
	return found, output, parseErr
}

// SubmitWithNativeSpecification implements QRSub.
func (c *CommandLineQRSub) SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no arguments provided")
	}
	// Layer 1 guard: reject control characters and invalid UTF-8 in any
	// argv token. A control character in a name could also forge the
	// "has been granted" line parsed from the output.
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", err
	}
	if c.config.DryRun {
		return fmt.Sprintf("%s %s %s", dryRunPrefix, c.config.Executable, strings.Join(args, " ")), nil
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, c.config.Executable, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out.String(), &CommandError{ExitCode: exitErr.ExitCode(), Output: out.String()}
		}
		return "", fmt.Errorf("failed to run %s: %w", c.config.Executable, err)
	}
	return out.String(), nil
}

// Submit implements QRSub.
func (c *CommandLineQRSub) Submit(ctx context.Context, opts ReservationOptions) (int64, string, error) {
	args, err := BuildQrsubArgs(opts)
	if err != nil {
		return 0, "", err
	}
	return SubmitArgs(ctx, c, args)
}

// Verify implements QRSub.
func (c *CommandLineQRSub) Verify(ctx context.Context, opts ReservationOptions) (bool, string, error) {
	args, err := BuildQrsubArgs(opts)
	if err != nil {
		return false, "", err
	}
	return VerifyArgs(ctx, c, args)
}
