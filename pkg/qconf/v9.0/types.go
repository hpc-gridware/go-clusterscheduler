/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024-2025 HPC-Gridware GmbH
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

// Type aliases - all types delegate to the core package.

type BootstrapFile = core.BootstrapFile
type ClusterConfig = core.ClusterConfig
type ClusterEnvironment = core.ClusterEnvironment
type CalendarConfig = core.CalendarConfig
type ComplexEntryConfig = core.ComplexEntryConfig
type CkptInterfaceConfig = core.CkptInterfaceConfig
type GlobalConfig = core.GlobalConfig
type HostConfiguration = core.HostConfiguration
type HostGroupConfig = core.HostGroupConfig
type HostExecConfig = core.HostExecConfig
type ResourceQuotaSetConfig = core.ResourceQuotaSetConfig
type ParallelEnvironmentConfig = core.ParallelEnvironmentConfig
type ProjectConfig = core.ProjectConfig
type ClusterQueueConfig = core.ClusterQueueConfig
type UserSetListConfig = core.UserSetListConfig
type UserConfig = core.UserConfig
type ComplexAttributeConfig = core.ComplexAttributeConfig
type SchedulerConfig = core.SchedulerConfig
type ShareTreeNode = core.ShareTreeNode
type StructuredShareTree = core.StructuredShareTree
type StructuredShareTreeNode = core.StructuredShareTreeNode
type ShareTreeNodeType = core.ShareTreeNodeType
type ShareTreeMonitoring = core.ShareTreeMonitoring
type ShareTreeNodeStats = core.ShareTreeNodeStats
type ShareTreeValidationError = core.ShareTreeValidationError
type ShareTreeValidationErrors = core.ShareTreeValidationErrors
type ShareTreeValidationOptions = core.ShareTreeValidationOptions
type ClusterSchedulerProduct = core.ClusterSchedulerProduct
type ClusterSchedulerVersion = core.ClusterSchedulerVersion

// Constants - re-exported from core.
const QTypeBatch = core.QTypeBatch
const QTypeInteractive = core.QTypeInteractive

const ResourceTypeInt = core.ResourceTypeInt
const ResourceTypeDouble = core.ResourceTypeDouble
const ResourceTypeMemory = core.ResourceTypeMemory
const ResourceTypeTime = core.ResourceTypeTime
const ResourceTypeString = core.ResourceTypeString
const ResourceTypeBool = core.ResourceTypeBool
const ResourceTypeRSMAP = core.ResourceTypeRSMAP

const ShareTreeNodeUser = core.ShareTreeNodeUser
const ShareTreeNodeProject = core.ShareTreeNodeProject

// Function re-exports for the share-tree parser and formatter so consumers
// can import only v9.0 and still use the structured helpers.
var ParseShareTreeText = core.ParseShareTreeText
var FormatShareTreeText = core.FormatShareTreeText
var ParseShareMonOutput = core.ParseShareMonOutput

// Subtree helper re-exports.
var NormalizeSharePath = core.NormalizeSharePath
var SplitSharePath = core.SplitSharePath
var FindNodeByPath = core.FindNodeByPath
var CloneShareTreeSubtree = core.CloneShareTreeSubtree
var IsDescendant = core.IsDescendant
var ValidateShareTree = core.ValidateShareTree
var ApplySubtreeReplace = core.ApplySubtreeReplace
var ApplySubtreeAdd = core.ApplySubtreeAdd
var ApplySubtreeDelete = core.ApplySubtreeDelete
var ApplySubtreeMove = core.ApplySubtreeMove

// Sentinel error re-exports.
var ErrNoShareTree = core.ErrNoShareTree
var ErrShareTreeMonNotAvail = core.ErrShareTreeMonNotAvail
var ErrShareTreeNodeNotFound = core.ErrShareTreeNodeNotFound

// ShareCode* re-exports so qontrol handlers can branch on validation
// errors without typing the raw string literals. Keep in lock-step with
// go-clusterscheduler/pkg/qconf/core/share_tree_validate.go.
const (
	ShareCodeDuplicatePath          = core.ShareCodeDuplicatePath
	ShareCodeProjectDuplicate       = core.ShareCodeProjectDuplicate
	ShareCodeUserDuplicateInProject = core.ShareCodeUserDuplicateInProject
	ShareCodeUserDuplicateOutside   = core.ShareCodeUserDuplicateOutside
	ShareCodeUserAsInterior         = core.ShareCodeUserAsInterior
	ShareCodeLeafUnknownName        = core.ShareCodeLeafUnknownName
	ShareCodeProjectNested          = core.ShareCodeProjectNested
	ShareCodeUserNoProjectAccess    = core.ShareCodeUserNoProjectAccess
	ShareCodeNegativeShares         = core.ShareCodeNegativeShares
	ShareCodeDefaultReserved        = core.ShareCodeDefaultReserved
	ShareCodeCycle                  = core.ShareCodeCycle
	ShareCodePathNotFound           = core.ShareCodePathNotFound
	ShareCodeRootDelete             = core.ShareCodeRootDelete
	ShareCodeEmptyName              = core.ShareCodeEmptyName
	ShareCodeNilNode                = core.ShareCodeNilNode
)

const ConsumableYES = core.ConsumableYES
const ConsumableNO = core.ConsumableNO
const ConsumableJOB = core.ConsumableJOB
const ConsumableHOST = core.ConsumableHOST

const ClusterSchedulerProductGCS = core.ClusterSchedulerProductGCS
const ClusterSchedulerProductOCS = core.ClusterSchedulerProductOCS
