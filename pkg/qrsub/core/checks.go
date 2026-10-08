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

// Field checks shared by BuildQrsubArgs and ReservationBuilder, so both
// paths reject the same input with the same error.
//
// Single-value switches (-N, -A) are taken verbatim by the qrsub parser, so
// a leading '-' there is not reinterpreted as a flag (verified on 9.0 and
// 9.1: "qrsub -N -foo" creates an AR named "-foo"). They only need the
// control character check every argv token gets. List and name=value
// switches need the element checks, because an embedded ',' or '=' would
// forge additional entries.

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// mailOptionsPattern is the -m grammar of qrsub; unlike qsub it has no 's'.
var mailOptionsPattern = regexp.MustCompile(`^[aben]+$`)

// slotRangePattern is the slot range grammar of -pe: n, n-m, n-, -m and
// comma separated lists of those. A leading '-' is legitimate here.
var slotRangePattern = regexp.MustCompile(`^([0-9]+(-[0-9]*)?|-[0-9]+)(,([0-9]+(-[0-9]*)?|-[0-9]+))*$`)

func checkName(name string) error {
	if name == "" {
		return errors.New("name: must not be empty")
	}
	if err := validate.NoControl(name); err != nil {
		return fmt.Errorf("name %q: %w", name, err)
	}
	if err := validate.StrictJobName(name); err != nil {
		return fmt.Errorf("name %q: %w", name, err)
	}
	return nil
}

func checkAccount(account string) error {
	if account == "" {
		return errors.New("account: must not be empty")
	}
	if err := validate.NoControl(account); err != nil {
		return fmt.Errorf("account %q: %w", account, err)
	}
	return nil
}

// checkObjectName checks a PE or checkpoint name, which become their own
// argv token.
func checkObjectName(what, name string) error {
	if err := validate.Operand(name); err != nil {
		return fmt.Errorf("%s %q: %w", what, name, err)
	}
	if err := validate.StrictObjectName(name); err != nil {
		return fmt.Errorf("%s %q: %w", what, name, err)
	}
	return nil
}

func checkPESlots(slots string) error {
	if !slotRangePattern.MatchString(slots) {
		return fmt.Errorf("pe slot range %q: must be a slot range such as 4, 2-8, -8 or 4-", slots)
	}
	return nil
}

func checkMailOptions(options string) error {
	if !mailOptionsPattern.MatchString(options) {
		return fmt.Errorf("mail options %q: must be a combination of b, e, a and n", options)
	}
	return nil
}

// checkResource checks one -l name=value pair. The resource list parser
// splits on ',' and on ' ', so neither name nor value may contain a space
// (verified: -l "h_rt=600 arch=x" requests arch=x as well). A '[' starts a
// bracketed expression which runs up to the next ']', so brackets in a
// value must be balanced, and a name (a complex name, which never contains
// brackets) must have none; otherwise the following pairs are swallowed.
func checkResource(name, value string) error {
	if err := validate.NameValueKey(name); err != nil {
		return fmt.Errorf("resource name %q: %w", name, err)
	}
	if i := strings.IndexAny(name, " []"); i >= 0 {
		return fmt.Errorf("resource name %q: must not contain %q", name, name[i])
	}
	if err := validate.NameValueValue(value); err != nil {
		return fmt.Errorf("resource %s value %q: %w", name, value, err)
	}
	if strings.Contains(value, " ") {
		return fmt.Errorf("resource %s value %q: must not contain a space", name, value)
	}
	if !balancedBrackets(value) {
		return fmt.Errorf("resource %s value %q: unbalanced brackets", name, value)
	}
	return nil
}

// balancedBrackets reports whether every '[' in s is closed by a ']' and
// no ']' appears without an open '['. Brackets do not nest.
func balancedBrackets(s string) bool {
	open := false
	for _, r := range s {
		switch r {
		case '[':
			if open {
				return false
			}
			open = true
		case ']':
			if !open {
				return false
			}
			open = false
		}
	}
	return !open
}

// checkList checks a list switch; an empty list would produce an empty
// argv token.
func checkList(what string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("%s: list must not be empty", what)
	}
	return validate.List(what, items)
}

func checkDuration(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("duration %s: must be positive", d)
	}
	if d%time.Second != 0 {
		return fmt.Errorf("duration %s: must be whole seconds", d)
	}
	return nil
}

func checkTimeWindow(start, end *time.Time, hasDuration bool) error {
	if end == nil && !hasDuration {
		return errors.New("either an end time or a duration is required")
	}
	// qrsub receives whole seconds, so compare what it will see.
	if start != nil && end != nil && !end.Truncate(time.Second).After(start.Truncate(time.Second)) {
		return fmt.Errorf("end time %s is not after start time %s", end.Format(time.RFC3339), start.Format(time.RFC3339))
	}
	return nil
}

// sortedKeys returns the keys of m in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// checkPreconditions checks the request logic which qmaster would reject.
// These errors are never relaxed by GCS_VALIDATION.
func checkPreconditions(opts ReservationOptions) error {
	if err := checkTimeWindow(opts.StartTime, opts.EndTime, opts.Duration != nil); err != nil {
		return err
	}
	if opts.Duration != nil {
		if err := checkDuration(*opts.Duration); err != nil {
			return err
		}
	}
	if (opts.PEName == nil) != (opts.PESlots == nil) {
		return errors.New("pe name and pe slots must be set together")
	}
	if opts.PESlots != nil {
		return checkPESlots(*opts.PESlots)
	}
	return nil
}

// checkFields applies the argument injection checks to every field.
func checkFields(opts ReservationOptions) error {
	if opts.Name != nil {
		if err := checkName(*opts.Name); err != nil {
			return err
		}
	}
	if opts.Account != nil {
		if err := checkAccount(*opts.Account); err != nil {
			return err
		}
	}
	// Sorted, so that the first reported error does not depend on the map
	// iteration order.
	for _, name := range sortedKeys(opts.Resources) {
		if err := checkResource(name, opts.Resources[name]); err != nil {
			return err
		}
	}
	lists := []struct {
		what  string
		items []string
	}{
		{"queue", opts.Queues},
		{"master queue", opts.MasterQueues},
		{"user", opts.Users},
		{"mail address", opts.MailList},
	}
	for _, l := range lists {
		// Nil and empty lists are both "not set" (JSON [] decodes to an
		// empty, non-nil slice).
		if len(l.items) == 0 {
			continue
		}
		if err := checkList(l.what, l.items); err != nil {
			return err
		}
	}
	if opts.PEName != nil {
		if err := checkObjectName("pe name", *opts.PEName); err != nil {
			return err
		}
	}
	if opts.Checkpoint != nil {
		if err := checkObjectName("checkpoint", *opts.Checkpoint); err != nil {
			return err
		}
	}
	if opts.MailOptions != nil {
		if err := checkMailOptions(*opts.MailOptions); err != nil {
			return err
		}
	}
	return nil
}

// ValidateReservationOptions applies all checks of BuildQrsubArgs to opts
// and returns the first error. Unlike BuildQrsubArgs it is not subject to
// GCS_VALIDATION, so trust boundaries (for example a network service which
// maps requests onto ReservationOptions) can rely on it.
func ValidateReservationOptions(opts ReservationOptions) error {
	if err := checkPreconditions(opts); err != nil {
		return err
	}
	return checkFields(opts)
}
