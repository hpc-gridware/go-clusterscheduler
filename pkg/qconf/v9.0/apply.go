/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
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

package qconf

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// Apply functions re-exported from core.
var Apply = core.Apply
var AddAllEntries = core.AddAllEntries
var ModifyAllEntries = core.ModifyAllEntries
var DeleteAllEnries = core.DeleteAllEnries

// Re-export additional functions used by tests and consumers.
var ParseVersionInfo = core.ParseVersionInfo
var ReadBootstrapFile = core.ReadBootstrapFile
var GetEnvironment = core.GetEnvironment
var GetEnvInt = core.GetEnvInt
var ParseGlobalConfigFromLines = core.ParseGlobalConfigFromLines
var ParseIntoStringFloatMap = core.ParseIntoStringFloatMap
var ParseIntoStringStringMap = core.ParseIntoStringStringMap

// SetDefault* functions re-exported from core.
var SetDefaultComplexEntryValues = core.SetDefaultComplexEntryValues
var SetDefaultParallelEnvironmentValues = core.SetDefaultParallelEnvironmentValues
var SetDefaultProjectValues = core.SetDefaultProjectValues
var SetDefaultQueueValues = core.SetDefaultQueueValues
var SetDefaultUserSetListConfig = core.SetDefaultUserSetListConfig
var SetDefaultExecHostConfig = core.SetDefaultExecHostConfig
var SetResourceQuotaSetDefaults = core.SetResourceQuotaSetDefaults
var SetDefaultUserValues = core.SetDefaultUserValues
var MakeBoolCfg = core.MakeBoolCfg
