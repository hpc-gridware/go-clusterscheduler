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
	v90 "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

// Constants

const QTypeBatch string = "BATCH"
const QTypeInteractive string = "INTERACTIVE"

// Complex entry resource types
const ResourceTypeInt string = "INT"
const ResourceTypeDouble string = "DOUBLE"
const ResourceTypeMemory string = "MEMORY"
const ResourceTypeTime string = "TIME"
const ResourceTypeString string = "STRING"
const ResourceTypeBool string = "BOOL"
const ResourceTypeRSMAP string = "RSMAP"

// Complex entry consumable types
// ConsumableYES indicates a per-slot consumable resource where the limit
// is multiplied by the number of slots being used by the job before being applied.
const ConsumableYES string = "YES"

// ConsumableNO indicates the resource is not managed as a consumable.
const ConsumableNO string = "NO"

// ConsumableJOB indicates a per-job consumable resource where the resource
// is debited as requested (without slot multiplication) from the allocated master queue.
const ConsumableJOB string = "JOB"

// ConsumableHOST indicates a per-host consumable resource (new in Open Cluster Scheduler).
const ConsumableHOST string = "HOST"

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

type ClusterEnvironment struct {
	v90.ClusterEnvironment
}

type CalendarConfig struct {
	v90.CalendarConfig
}

type ComplexEntryConfig struct {
	v90.ComplexEntryConfig
}

type CkptInterfaceConfig struct {
	v90.CkptInterfaceConfig
}

type GlobalConfig struct {
	v90.GlobalConfig

	// Reprioritize shadows the v9.0 field; removed in v9.1.
	// omitempty ensures the zero value is not serialized.
	Reprioritize int `json:"reprioritize,omitempty"`

	// New fields in v9.1
	JsvParams        string            `json:"jsv_params"`
	TopologyFile     string            `json:"topology_file"`
	MailTag          string            `json:"mail_tag"`
	GDIRequestLimits string            `json:"gdi_request_limits"`
	BindingParams    map[string]string `json:"binding_params"` // key-value pairs
}

type SchedulerConfig struct {
	v90.SchedulerConfig
}

type HostConfiguration struct {
	v90.HostConfiguration
}

type HostExecConfig struct {
	v90.HostExecConfig
}

type HostGroupConfig struct {
	v90.HostGroupConfig
}

type ResourceQuotaSetConfig struct {
	v90.ResourceQuotaSetConfig
}

type ParallelEnvironmentConfig struct {
	v90.ParallelEnvironmentConfig
}

type ProjectConfig struct {
	v90.ProjectConfig
}

type UserConfig struct {
	v90.UserConfig
}

type ClusterQueueConfig struct {
	v90.ClusterQueueConfig
}

type UserSetListConfig struct {
	v90.UserSetListConfig
}
