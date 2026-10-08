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
	"fmt"
	"regexp"
	"strconv"
)

// Value grammars of the qrsub core binding switches (9.1 and later). The
// -btype switch is missing on purpose: qrsub 9.1 rejects every value of it.
var (
	bindingUnits     = map[string]bool{"T": true, "ET": true, "C": true, "E": true, "S": true, "ES": true, "X": true, "EX": true, "Y": true, "EY": true, "N": true, "EN": true}
	bindingInstances = map[string]bool{"set": true, "env": true, "pe": true}
	bindingPosition  = regexp.MustCompile(`^[SsCcEeNnXxYy]$`)
	bindingSortOrder = regexp.MustCompile(`^[SsCcEeNnXxYy]+$`)
	bindingTopology  = regexp.MustCompile(`^[A-Za-z]+$`)
)

// bindingSwitches maps each binding switch to the check of its value.
var bindingSwitches = map[string]func(string) error{
	"-bamount": func(v string) error {
		if n, err := strconv.Atoi(v); err != nil || n <= 0 {
			return fmt.Errorf("must be a positive number")
		}
		return nil
	},
	"-bunit": func(v string) error {
		if !bindingUnits[v] {
			return fmt.Errorf("must be one of T, ET, C, E, S, ES, X, EX, Y, EY, N, EN")
		}
		return nil
	},
	"-binstance": func(v string) error {
		if !bindingInstances[v] {
			return fmt.Errorf("must be set, env or pe")
		}
		return nil
	},
	"-bstrategy": func(v string) error {
		if v != "packed" {
			return fmt.Errorf("qrsub supports packed")
		}
		return nil
	},
	"-bstart": checkBindingPosition,
	"-bstop":  checkBindingPosition,
	"-bsort": func(v string) error {
		if !bindingSortOrder.MatchString(v) {
			return fmt.Errorf("must consist of S, s, C, c, E, e, N, n, X, x, Y, y")
		}
		return nil
	},
	"-bfilter": func(v string) error {
		if !bindingTopology.MatchString(v) {
			return fmt.Errorf("must be a topology string")
		}
		return nil
	},
}

func checkBindingPosition(v string) error {
	if !bindingPosition.MatchString(v) {
		return fmt.Errorf("must be one of S, s, C, c, E, e, N, n, X, x, Y, y")
	}
	return nil
}

// ValidateBindingSwitch checks one qrsub core binding switch (-bamount,
// -bunit, -bfilter, -bsort, -bstart, -bstop, -bstrategy, -binstance) and its
// value against the qrsub grammar. It is not subject to GCS_VALIDATION.
func ValidateBindingSwitch(flag, value string) error {
	check, ok := bindingSwitches[flag]
	if !ok {
		return fmt.Errorf("%s is not a qrsub binding switch", flag)
	}
	if err := check(value); err != nil {
		return fmt.Errorf("binding %s %q: %w", flag, value, err)
	}
	return nil
}
