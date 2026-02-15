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

const ConsumableYES = core.ConsumableYES
const ConsumableNO = core.ConsumableNO
const ConsumableJOB = core.ConsumableJOB
const ConsumableHOST = core.ConsumableHOST

// Type aliases for types unchanged from v9.0/core.
type BootstrapFile = core.BootstrapFile
type ClusterEnvironment = core.ClusterEnvironment
type CalendarConfig = core.CalendarConfig
type ComplexEntryConfig = core.ComplexEntryConfig
type CkptInterfaceConfig = core.CkptInterfaceConfig
type SchedulerConfig = core.SchedulerConfig
type HostConfiguration = core.HostConfiguration
type HostExecConfig = core.HostExecConfig
type HostGroupConfig = core.HostGroupConfig
type ResourceQuotaSetConfig = core.ResourceQuotaSetConfig
type ParallelEnvironmentConfig = core.ParallelEnvironmentConfig
type ProjectConfig = core.ProjectConfig
type ClusterQueueConfig = core.ClusterQueueConfig
type UserSetListConfig = core.UserSetListConfig
type UserConfig = core.UserConfig
type ComplexAttributeConfig = core.ComplexAttributeConfig
type ShareTreeNode = core.ShareTreeNode
type ClusterSchedulerProduct = core.ClusterSchedulerProduct
type ClusterSchedulerVersion = core.ClusterSchedulerVersion
type CommandLineQConfConfig = core.CommandLineQConfConfig

// ClusterConfig uses the v9.1 GlobalConfig type.
type ClusterConfig struct {
	ClusterEnvironment   *ClusterEnvironment                  `json:"cluster_environment"`
	GlobalConfig         *GlobalConfig                        `json:"global_config"`
	SchedulerConfig      *SchedulerConfig                     `json:"scheduler_config"`
	Calendars            map[string]CalendarConfig            `json:"calendars"`
	ComplexEntries       map[string]ComplexEntryConfig        `json:"complex_entries"`
	CkptInterfaces       map[string]CkptInterfaceConfig       `json:"ckpt_interfaces"`
	HostConfigurations   map[string]HostConfiguration         `json:"host_configurations"`
	ExecHosts            map[string]HostExecConfig            `json:"exec_hosts"`
	AdminHosts           []string                             `json:"admin_hosts"`
	SubmitHosts          []string                             `json:"submit_hosts"`
	HostGroups           map[string]HostGroupConfig           `json:"host_groups"`
	ResourceQuotaSets    map[string]ResourceQuotaSetConfig    `json:"resource_quota_sets"`
	Managers             []string                             `json:"managers"`
	Operators            []string                             `json:"operators"`
	ParallelEnvironments map[string]ParallelEnvironmentConfig `json:"parallel_environments"`
	Projects             map[string]ProjectConfig             `json:"projects"`
	Users                map[string]UserConfig                `json:"users"`
	ClusterQueues        map[string]ClusterQueueConfig        `json:"cluster_queues"`
	UserSetLists         map[string]UserSetListConfig         `json:"user_set_lists"`
}

// GlobalConfig extends the core GlobalConfig with v9.1-specific fields.
type GlobalConfig struct {
	core.GlobalConfig

	// New fields in v9.1
	JsvParams        string            `json:"jsv_params"`
	TopologyFile     string            `json:"topology_file"`
	MailTag          string            `json:"mail_tag"`
	GDIRequestLimits string            `json:"gdi_request_limits"`
	BindingParams    map[string]string `json:"binding_params"`
}
