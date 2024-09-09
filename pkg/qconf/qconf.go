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

// QConf defines the methods for interacting with the Open Cluster Scheduler
// configuration. The methods are named after the qconf command line tool
// which is used to interact with the Open Cluster Scheduler configuration.
type QConf interface {
	GetClusterConfiguration() (ClusterConfig, error)
	ApplyClusterConfiguration(c ClusterConfig) error

	AddCalendar(c CalendarConfig) error
	DeleteCalendar(calendarName string) error
	ShowCalendar(calendarName string) (CalendarConfig, error)
	ShowCalendars() ([]string, error)
	ModifyCalendar(calendarName string, c CalendarConfig) error

	AddComplexEntry(e ComplexEntryConfig) error
	DeleteComplexEntry(entryName string) error
	ShowComplexEntry(entryName string) (ComplexEntryConfig, error)
	ShowComplexEntries() ([]string, error)
	ShowAllComplexes() ([]ComplexEntryConfig, error)
	ModifyAllComplexes(complexAttr []ComplexEntryConfig) error

	AddCkptInterface(c CkptInterfaceConfig) error
	DeleteCkptInterface(interfaceName string) error
	ShowCkptInterface(interfaceName string) (CkptInterfaceConfig, error)
	ShowCkptInterfaces() ([]string, error)
	ModifyCkptInterface(ckptName string, c CkptInterfaceConfig) error

	AddHostConfiguration(config HostConfiguration) error
	DeleteHostConfiguration(configName string) error
	ShowHostConfiguration(hostName string) (HostConfiguration, error)
	ShowHostConfigurations() ([]string, error)
	ModifyHostConfiguration(configName string, c HostConfiguration) error

	ShowGlobalConfiguration() (*GlobalConfig, error)
	ModifyGlobalConfig(g GlobalConfig) error

	AddExecHost(hostExecConfig HostExecConfig) error
	DeleteExecHost(hostList string) error
	ModifyExecHost(execHostName string, h HostExecConfig) error
	ShowExecHost(hostName string) (HostExecConfig, error)
	ShowExecHosts() ([]string, error)

	AddAdminHost(hosts []string) error
	DeleteAdminHost(hosts []string) error
	ShowAdminHosts() ([]string, error)

	AddHostGroup(hostGroup HostGroupConfig) error
	ModifyHostGroup(hostGroupName string, hg HostGroupConfig) error
	DeleteHostGroup(groupName string) error
	ShowHostGroup(groupName string) (HostGroupConfig, error)
	ShowHostGroups() ([]string, error)

	AddResourceQuotaSet(rqs ResourceQuotaSetConfig) error
	DeleteResourceQuotaSet(rqsList string) error
	ShowResourceQuotaSet(rqsList string) (ResourceQuotaSetConfig, error)
	ShowResourceQuotaSets() ([]string, error)
	ModifyResourceQuotaSet(rqsName string, rqs ResourceQuotaSetConfig) error

	AddUserToManagerList(users []string) error
	DeleteUserFromManagerList(users []string) error
	ShowManagers() ([]string, error)

	AddUserToOperatorList(users []string) error
	DeleteUserFromOperatorList(users []string) error
	ShowOperators() ([]string, error)

	AddParallelEnvironment(pe ParallelEnvironmentConfig) error
	DeleteParallelEnvironment(peName string) error
	ShowParallelEnvironment(peName string) (ParallelEnvironmentConfig, error)
	ShowParallelEnvironments() ([]string, error)
	ModifyParallelEnvironment(peName string, pe ParallelEnvironmentConfig) error

	AddProject(project ProjectConfig) error
	DeleteProject(projects []string) error
	ShowProject(projectName string) (ProjectConfig, error)
	ShowProjects() ([]string, error)
	ModifyProject(projectName string, p ProjectConfig) error

	AddClusterQueue(queue ClusterQueueConfig) error
	ModifyClusterQueue(queueName string, q ClusterQueueConfig) error
	DeleteClusterQueue(queueName string) error
	ShowClusterQueue(queueName string) (ClusterQueueConfig, error)
	ShowClusterQueues() ([]string, error)

	AddSubmitHosts(hostnames []string) error
	DeleteSubmitHost(hostnames []string) error
	ShowSubmitHosts() ([]string, error)

	/* TODO: implement
	AddShareTreeNode(nodeShareList string) (string, error)
	DeleteShareTreeNode(nodeList string) (string, error)
	ShowShareTreeNodes(nodeList string) ([]string, error)
	ShowShareTree() (string, error)
	ModifyShareTree(shareTreeConfig ShareTreeConfig) (string, error)
	*/

	AddUserSetList(listnameList string, u UserSetListConfig) error
	AddUserToUserSetList(userList, listnameList string) error
	DeleteUserFromUserSetList(userList, listnameList string) error
	DeleteUserSetList(userList string) error
	ShowUserSetList(listnameList string) (UserSetListConfig, error)
	ShowUserSetLists() ([]string, error)
	ModifyUserset(listnameList string, u UserSetListConfig) error

	AddUser(userConfig UserConfig) error
	DeleteUser(users []string) error
	ShowUser(userName string) (UserConfig, error)
	ShowUsers() ([]string, error)
	ModifyUser(userName string, u UserConfig) error

	ClearUsage() error
	CleanQueue(destinID []string) error
	ShutdownExecDaemons(hosts []string) error
	ShutdownMasterDaemon() error
	ShutdownSchedulingDaemon() error
	KillEventClient(evids []string) error
	KillQmasterThread(threadName string) error

	ModifyAttribute(objName, attrName, val, objIDList string) error
	DeleteAttribute(objName, attrName, val, objIDList string) error
	AddAttribute(objName, attrName, val, objIDList string) error

	ModifySchedulerConfig(cfg SchedulerConfig) error
	ShowSchedulerConfiguration() (*SchedulerConfig, error)
}
