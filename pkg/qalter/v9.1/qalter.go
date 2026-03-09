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
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
)

// QAlter defines the methods for interacting with the GCS 9.1 qalter
// command. It extends the core QAlter interface with v9.1-specific
// binding options.
//
// The -when flag is configured via CommandLineQAlterConfig.When and
// is automatically prepended to all operations when set.
type QAlter interface {
	core.QAlter

	// --- Binding ---

	// SetBindingAmount defines the number of binding units to be
	// used (-bamount number).
	SetBindingAmount(jobTaskList string, amount int) (string, error)

	// SetBindingFilter specifies a binding filter to mask binding
	// units (-bfilter topology_string). Topology string where lower
	// case letters show masked units.
	SetBindingFilter(jobTaskList, topology string) (string, error)

	// SetBindingInstance defines the instance applying the binding
	// (-binstance set|env|pe).
	SetBindingInstance(jobTaskList, instance string) (string, error)

	// SetBindingSortOrder enables and specifies binding sort order
	// (-bsort [SsCcEeNnXxYy]).
	SetBindingSortOrder(jobTaskList, order string) (string, error)

	// SetBindingStart defines the start position for binding
	// (-bstart [S|s|C|c|E|e|N|n|X|x|Y|y]).
	SetBindingStart(jobTaskList, position string) (string, error)

	// SetBindingStop defines the stop position for binding
	// (-bstop [S|s|C|c|E|e|N|n|X|x|Y|y]).
	SetBindingStop(jobTaskList, position string) (string, error)

	// SetBindingStrategy defines the binding strategy
	// (-bstrategy name).
	SetBindingStrategy(jobTaskList, strategy string) (string, error)

	// SetBindingType sets the type of binding
	// (-btype host|slot).
	SetBindingType(jobTaskList, bindingType string) (string, error)

	// SetBindingUnit sets the binding unit
	// (-bunit [T|ET|C|E|S|ES|X|EX|Y|EY|N|EN]).
	SetBindingUnit(jobTaskList, unit string) (string, error)
}
