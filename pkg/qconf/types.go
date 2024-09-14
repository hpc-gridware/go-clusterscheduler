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

// ClusterConfig represents the complete configuration of a cluster.
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

// ClusterEnvironment provides information about the
// specific cluster, like the installation path, the
// "cell" (which is used to separate different clusters
// sharing the same installation directory) and the
// ports used by the qmaster and execd.
type ClusterEnvironment struct {
	Name        string `json:"sge_name"`
	Root        string `json:"sge_root"`
	Cell        string `json:"sge_cell"`
	QmasterPort int    `json:"sge_qmaster_port"`
	ExecdPort   int    `json:"sge_execd_port"`
}

// CalendarConfig represents the configuration for resource.
type CalendarConfig struct {
	Name string `json:"calendar_name"`
	Year string `json:"year"`
	Week string `json:"week"`
}

// ComplexEntryConfig represents the configuration for a complex entry.
type ComplexEntryConfig struct {
	Name        string `json:"name"`
	Shortcut    string `json:"shortcut"`
	Type        string `json:"type"`
	Relop       string `json:"relop"`
	Requestable string `json:"requestable"`
	Consumable  string `json:"consumable"`
	Default     string `json:"default"`
	Urgency     int    `json:"urgency"`
}

// CkptInterfaceConfig represents the configuration for a checkpointing interface.
type CkptInterfaceConfig struct {
	Name              string `json:"ckpt_name"`
	Interface         string `json:"interface"`
	CleanCommand      string `json:"clean_command"`
	CheckpointCommand string `json:"ckpt_command"`
	MigrCommand       string `json:"migr_command"`
	RestartCommand    string `json:"restart_command"`
	CheckpointDir     string `json:"ckpt_dir"`
	Signal            string `json:"signal"`
	When              string `json:"when"`
}

type GlobalConfig struct {
	ExecdSpoolDir string `json:"execd_spool_dir"`
	Mailer        string `json:"mailer"`
	Xterm         string `json:"xterm"`
	// CommaSeparatedList
	LoadSensors       []string `json:"load_sensor"`
	Prolog            string   `json:"prolog"`
	Epilog            string   `json:"epilog"`
	ShellStartMode    string   `json:"shell_start_mode"`
	LoginShells       []string `json:"login_shells"`
	MinUID            int      `json:"min_uid"`
	MinGID            int      `json:"min_gid"`
	UserLists         []string `json:"user_lists"`
	XUserLists        []string `json:"xuser_lists"`
	Projects          []string `json:"projects"`
	XProjects         []string `json:"xprojects"`
	EnforceProject    string   `json:"enforce_project"`
	EnforceUser       string   `json:"enforce_user"`
	LoadReportTime    string   `json:"load_report_time"`
	MaxUnheard        string   `json:"max_unheard"`
	RescheduleUnknown string   `json:"reschedule_unknown"`
	LogLevel          string   `json:"loglevel"`
	AdministratorMail string   `json:"administrator_mail"`
	SetTokenCmd       string   `json:"set_token_cmd"`
	PagCmd            string   `json:"pag_cmd"`
	TokenExtendTime   string   `json:"token_extend_time"`
	ShepherdCmd       string   `json:"shepherd_cmd"`
	QmasterParams     []string `json:"qmaster_params"`
	ExecdParams       []string `json:"execd_params"`
	ReportingParams   []string `json:"reporting_params"`
	FinishedJobs      int      `json:"finished_jobs"`
	// CommaSeparatedList
	GidRange               []string `json:"gid_range"`
	QloginCommand          string   `json:"qlogin_command"`
	QloginDaemon           string   `json:"qlogin_daemon"`
	RloginCommand          string   `json:"rlogin_command"`
	RloginDaemon           string   `json:"rlogin_daemon"`
	RshCommand             string   `json:"rsh_command"`
	RshDaemon              string   `json:"rsh_daemon"`
	MaxAJInstances         int      `json:"max_aj_instances"`
	MaxAJTasks             int      `json:"max_aj_tasks"`
	MaxUJobs               int      `json:"max_u_jobs"`
	MaxJobs                int      `json:"max_jobs"`
	MaxAdvanceReservations int      `json:"max_advance_reservations"`
	AutoUserOTicket        int      `json:"auto_user_oticket"`
	AutoUserFshare         int      `json:"auto_user_fshare"`
	AutoUserDefaultProject string   `json:"auto_user_default_project"`
	AutoUserDeleteTime     int      `json:"auto_user_delete_time"`
	DelegatedFileStaging   bool     `json:"delegated_file_staging"`
	Reprioritize           int      `json:"reprioritize"`
	JsvURL                 string   `json:"jsv_url"`
	// CommaSeparatedList
	JsvAllowedMod []string `json:"jsv_allowed_mod"`
}

// HostConfiguration represents the configuration for a host.
type HostConfiguration struct {
	Name   string // Inconsistency
	Mailer string `json:"mailer"`
	Xterm  string `json:"xterm"`
	// @TODO Add additional fields here as needed
}

// HostGroupConfig represents the configuration for a host group.
type HostGroupConfig struct {
	Name string `json:"group_name"`
	// Hosts are space separated.
	Hosts []string `json:"hostlist"`
}

// HostExecConfig represents the execution host configuration.
type HostExecConfig struct {
	Name string `json:"hostname"`
	// LoadScaling scales the reported load of the resources.
	LoadScaling map[string]float64 `json:"load_scaling"`
	// UsageScaling scales the reported usage of the resources.
	UsageScaling  map[string]float64 `json:"usage_scaling"`
	ComplexValues map[string]string  `json:"complex_values"`
	UserLists     []string           `json:"user_lists"`
	XUserLists    []string           `json:"xuser_lists"`
	Projects      []string           `json:"projects"`
	XProjects     []string           `json:"xprojects"`
	// ReportVariables includes the resources that are reported by the execution host.
	ReportVariables []string `json:"report_variables"`
}

// ResourceQuotaSetConfig represents the configuration for a resource quota set.
type ResourceQuotaSetConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Limits      []string `json:"limits"`
}

// ParallelEnvironmentConfig represents the configuration for a parallel environment.
type ParallelEnvironmentConfig struct {
	Name              string   `json:"pe_name"`
	Slots             int      `json:"slots"`
	UserLists         []string `json:"user_lists"`
	XUserLists        []string `json:"xuser_lists"`
	StartProcArgs     string   `json:"start_proc_args"`
	StopProcArgs      string   `json:"stop_proc_args"`
	AllocationRule    string   `json:"allocation_rule"`
	ControlSlaves     string   `json:"control_slaves"`
	JobIsFirstTask    bool     `json:"job_is_first_task"`
	UrgencySlots      string   `json:"urgency_slots"`
	AccountingSummary bool     `json:"accounting_summary"`
}

// ProjectConfig represents the configuration for a project.
type ProjectConfig struct {
	Name    string   `json:"name"`
	OTicket int      `json:"oticket"`
	FShare  int      `json:"fshare"`
	ACL     []string `json:"acl"`  // user_list space separated
	XACL    []string `json:"xacl"` // user_list space separated
}

const QTypeBatch string = "BATCH"
const QTypeInteractive string = "INTERACTIVE"

// ClusterQueueConfig represents the configuration for a cluster queue.
type ClusterQueueConfig struct {
	Name              string   `json:"qname"`
	HostList          []string `json:"hostlist"`
	SeqNo             int      `json:"seq_no"`
	LoadThresholds    string   `json:"load_thresholds"`
	SuspendThresholds string   `json:"suspend_thresholds"`
	NSuspend          int      `json:"nsuspend"`
	SuspendInterval   string   `json:"suspend_interval"`
	Priority          int      `json:"priority"`
	MinCpuInterval    string   `json:"min_cpu_interval"`
	Processors        string   `json:"processors"`
	QType             []string `json:"qtype"`
	CkptList          []string `json:"ckpt_list"`
	PeList            []string `json:"pe_list"`
	Rerun             bool     `json:"rerun"`
	Slots             int      `json:"slots"`
	TmpDir            string   `json:"tmpdir"`
	Shell             string   `json:"shell"`
	Prolog            string   `json:"prolog"`
	Epilog            string   `json:"epilog"`
	ShellStartMode    string   `json:"shell_start_mode"`
	StarterMethod     string   `json:"starter_method"`
	SuspendMethod     string   `json:"suspend_method"`
	ResumeMethod      string   `json:"resume_method"`
	TerminateMethod   string   `json:"terminate_method"`
	Notify            string   `json:"notify"`
	OwnerList         []string `json:"owner_list"`
	UserLists         []string `json:"user_lists"`
	XUserLists        []string `json:"xuser_lists"`
	SubordinateList   []string `json:"subordinate_list"`
	ComplexValues     []string `json:"complex_values"`
	Projects          []string `json:"projects"`
	XProjects         []string `json:"xprojects"`
	Calendar          string   `json:"calendar"`
	InitialState      string   `json:"initial_state"`
	SRt               string   `json:"s_rt"`
	HRt               string   `json:"h_rt"`
	SCpu              string   `json:"s_cpu"`
	HCpu              string   `json:"h_cpu"`
	SSize             string   `json:"s_fsize"`
	HSize             string   `json:"h_fsize"`
	SData             string   `json:"s_data"`
	HData             string   `json:"h_data"`
	SStack            string   `json:"s_stack"`
	HStack            string   `json:"h_stack"`
	SCore             string   `json:"s_core"`
	HCore             string   `json:"h_core"`
	SRss              string   `json:"s_rss"`
	HRss              string   `json:"h_rss"`
	SVmem             string   `json:"s_vmem"`
	HVmem             string   `json:"h_vmem"`
}

// UserSetListConfig represents the configuration for a user set list.
type UserSetListConfig struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	FShare  int      `json:"fshare"`
	OTicket int      `json:"oticket"`
	Entries []string `json:"entries"`
}

// UserConfig represents the configuration for a user.
type UserConfig struct {
	Name           string `json:"name"`
	OTicket        int    `json:"oticket"`
	FShare         int    `json:"fshare"`
	DeleteTime     int    `json:"delete_time"`
	DefaultProject string `json:"default_project"`
}

// ComplexAttributeConfig represents the configuration for complex attributes.
type ComplexAttributeConfig struct {
	Resources []ComplexEntryConfig `json:"resources"`
}

// SchedulerConfig represents the configuration for the scheduler.
type SchedulerConfig struct {
	Algorithm                   string   `json:"algorithm"`
	ScheduleInterval            string   `json:"schedule_interval"`
	MaxUJobs                    int      `json:"maxujobs"`
	QueueSortMethod             string   `json:"queue_sort_method"`
	JobLoadAdjustments          []string `json:"job_load_adjustments"`
	LoadAdjustmentDecayTime     string   `json:"load_adjustment_decay_time"`
	LoadFormula                 string   `json:"load_formula"`
	ScheddJobInfo               string   `json:"schedd_job_info"`
	FlushSubmitSec              int      `json:"flush_submit_sec"`
	FlushFinishSec              int      `json:"flush_finish_sec"`
	Params                      []string `json:"params"`
	ReprioritizeInterval        string   `json:"reprioritize_interval"`
	Halftime                    int      `json:"halftime"`
	UsageWeightList             []string `json:"usage_weight_list"`
	CompensationFactor          float64  `json:"compensation_factor"`
	WeightUser                  float64  `json:"weight_user"`
	WeightProject               float64  `json:"weight_project"`
	WeightDepartment            float64  `json:"weight_department"`
	WeightJob                   float64  `json:"weight_job"`
	WeightTicketsFunctional     int      `json:"weight_tickets_functional"`
	WeightTicketsShare          int      `json:"weight_tickets_share"`
	ShareOverrideTickets        bool     `json:"share_override_tickets"`
	ShareFunctionalShares       bool     `json:"share_functional_shares"`
	MaxFunctionalJobsToSchedule int      `json:"max_functional_jobs_to_schedule"`
	ReportPJobTickets           bool     `json:"report_pjob_tickets"`
	MaxPendingTasksPerJob       int      `json:"max_pending_tasks_per_job"`
	HalflifeDecayList           []string `json:"halflife_decay_list"`
	PolicyHierarchy             string   `json:"policy_hierarchy"`
	WeightTicket                float64  `json:"weight_ticket"`
	WeightWaitingTime           float64  `json:"weight_waiting_time"`
	WeightDeadline              float64  `json:"weight_deadline"`
	WeightUrgency               float64  `json:"weight_urgency"`
	WeightPriority              float64  `json:"weight_priority"`
	MaxReservation              int      `json:"max_reservation"`
	DefaultDuration             string   `json:"default_duration"`
}

// ShareTreeNodeShare represents a node in the share tree with its share value
type ShareTreeNode struct {
	Node  string
	Share int
}
