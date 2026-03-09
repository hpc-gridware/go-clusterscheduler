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
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
)

type ResourceRequest = core.ResourceRequest
type CommandLineQSubConfig = core.CommandLineQSubConfig

const ResourceRequestTypeHard = core.ResourceRequestTypeHard
const ResourceRequestTypeSoft = core.ResourceRequestTypeSoft

const ResourceRequestScopeGlobal = core.ResourceRequestScopeGlobal
const ResourceRequestScopeMaster = core.ResourceRequestScopeMaster
const ResourceRequestScopeSlave = core.ResourceRequestScopeSlave

func ToPtr[T any](v T) *T {
	return core.ToPtr(v)
}

func SimpleLRequest(resources map[string]string) map[string]map[string]ResourceRequest {
	return core.SimpleLRequest(resources)
}

func ConvertTimeToQsubDateTime(t time.Time) string {
	return core.ConvertTimeToQsubDateTime(t)
}

// JobOptions extends core.JobOptions with GCS 9.1 binding options.
// The legacy ProcessorBinding field (inherited from core) is ignored;
// use the new granular binding fields instead.
type JobOptions struct {
	core.JobOptions

	// BindingAmount defines the number of binding units (-bamount).
	BindingAmount *int `flag:"-bamount"`
	// BindingStop defines the stop position for binding (-bstop).
	// Values: S, s, C, c, E, e, N, n, X, x, Y, y
	BindingStop *string `flag:"-bstop"`
	// BindingFilter specifies a binding filter to mask binding units (-bfilter).
	BindingFilter *string `flag:"-bfilter"`
	// BindingInstance defines the instance applying the binding (-binstance).
	// Values: set, env, pe
	BindingInstance *string `flag:"-binstance"`
	// BindingSort enables and specifies binding sort order (-bsort).
	// Values: S, s, C, c, E, e, N, n, X, x, Y, y
	BindingSort *string `flag:"-bsort"`
	// BindingStart defines the start position for binding (-bstart).
	// Values: S, s, C, c, E, e, N, n, X, x, Y, y
	BindingStart *string `flag:"-bstart"`
	// BindingStrategy defines the binding strategy (-bstrategy).
	BindingStrategy *string `flag:"-bstrategy"`
	// BindingType sets the type of binding (-btype).
	// Values: host, slot
	BindingType *string `flag:"-btype"`
	// BindingUnit sets the binding unit (-bunit).
	// Values: T, ET, C, E, S, ES, X, EX, Y, EY, N, EN
	BindingUnit *string `flag:"-bunit"`
}
