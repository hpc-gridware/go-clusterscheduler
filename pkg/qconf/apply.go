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

// AddAllEntries adds all elements to the cluster configuration
// and returns the updated cluster configuration. Note, that
// the configuration elements must not exist before otherwise an
// error is returned.
//
// The global config is ignored.
//
// In case of an error, it returns the applied cluster configuration
// and the error. The applied cluster configuration can be used
// for rollback.
func AddAllEntries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var appliedConfig ClusterConfig

	// Add all calendars
	for _, elem := range q.Calendars {
		if err := qc.AddCalendar(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.Calendars = append(appliedConfig.Calendars, elem)
	}

	// Add all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.AddComplexEntry(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ComplexEntries = append(appliedConfig.ComplexEntries, elem)
	}

	// Add all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.AddCkptInterface(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.CkptInterfaces = append(appliedConfig.CkptInterfaces, elem)
	}

	// Add all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.AddHostConfiguration(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.HostConfigurations = append(appliedConfig.HostConfigurations, elem)
	}

	// Add all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.AddExecHost(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ExecHosts = append(appliedConfig.ExecHosts, elem)
	}

	// Add all admin hosts
	if err := qc.AddAdminHost(q.AdminHosts); err != nil {
		return appliedConfig, err
	}
	appliedConfig.AdminHosts = q.AdminHosts

	// Add all host groups
	for _, elem := range q.HostGroups {
		if err := qc.AddHostGroup(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.HostGroups = append(appliedConfig.HostGroups, elem)
	}

	// Add all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.AddResourceQuotaSet(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ResourceQuotaSets = append(appliedConfig.ResourceQuotaSets, elem)
	}

	// Add all managers
	if err := qc.AddUserToManagerList(q.Managers); err != nil {
		return appliedConfig, err
	}
	appliedConfig.Managers = q.Managers

	// Add all operators
	if err := qc.AddUserToOperatorList(q.Operators); err != nil {
		return appliedConfig, err
	}
	appliedConfig.Operators = q.Operators

	// Add all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.AddParallelEnvironment(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ParallelEnvironments = append(appliedConfig.ParallelEnvironments, elem)
	}

	// Add all projects
	for _, elem := range q.Projects {
		if err := qc.AddProject(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.Projects = append(appliedConfig.Projects, elem)
	}

	// Add all users
	for _, elem := range q.Users {
		if err := qc.AddUser(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.Users = append(appliedConfig.Users, elem)
	}

	// Add all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.AddClusterQueue(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ClusterQueues = append(appliedConfig.ClusterQueues, elem)
	}

	// Add all user set lists
	for _, elem := range q.UserSetLists {
		listName := "" // Assuming UserSetListConfig has a field or method to get the ListName
		if err := qc.AddUserSetList(listName, elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.UserSetLists = append(appliedConfig.UserSetLists, elem)
	}

	return appliedConfig, nil
}

// ModifyAllEntries modifies all elements in the cluster configuration
// and returns the updated cluster configuration. The elements must exist
// before; otherwise, an error is returned.
//
// The global config is ignored.
//
// In case of an error, it returns the modified cluster configuration
// and the error. The modified cluster configuration can be used
// for rollback.
func ModifyAllEntries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var modifiedConfig ClusterConfig

	// Modify all calendars
	for _, elem := range q.Calendars {
		if err := qc.ModifyCalendar(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.Calendars = append(modifiedConfig.Calendars, elem)
	}

	// Modify all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.ModifyAllComplexes([]ComplexEntryConfig{elem}); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ComplexEntries = append(modifiedConfig.ComplexEntries, elem)
	}

	// Modify all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.ModifyCkptInterface(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.CkptInterfaces = append(modifiedConfig.CkptInterfaces, elem)
	}

	// Modify all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.ModifyHostConfiguration(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.HostConfigurations = append(modifiedConfig.HostConfigurations, elem)
	}

	// Modify all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.ModifyExecHost(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ExecHosts = append(modifiedConfig.ExecHosts, elem)
	}

	// Modify all host groups
	for _, elem := range q.HostGroups {
		if err := qc.ModifyHostGroup(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.HostGroups = append(modifiedConfig.HostGroups, elem)
	}

	// Modify all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.ModifyResourceQuotaSet(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ResourceQuotaSets = append(modifiedConfig.ResourceQuotaSets, elem)
	}

	// Modify all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.ModifyParallelEnvironment(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ParallelEnvironments = append(modifiedConfig.ParallelEnvironments, elem)
	}

	// Modify all projects
	for _, elem := range q.Projects {
		if err := qc.ModifyProject(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.Projects = append(modifiedConfig.Projects, elem)
	}

	// Modify all users
	for _, elem := range q.Users {
		if err := qc.ModifyUser(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.Users = append(modifiedConfig.Users, elem)
	}

	// Modify all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.ModifyClusterQueue(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ClusterQueues = append(modifiedConfig.ClusterQueues, elem)
	}

	// Modify all user set lists
	for _, elem := range q.UserSetLists {
		listName := "" // Assuming UserSetListConfig has a field or method to get the ListName
		if err := qc.ModifyUserset(listName, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.UserSetLists = append(modifiedConfig.UserSetLists, elem)
	}

	return modifiedConfig, nil
}

// DeleteAllEnries deletes all elements from the cluster configuration
// and returns the delete objects of the cluster configuration. The
// elements must exist before; otherwise, an error is returned.
//
// The global config is ignored.
//
// In case of an error, it returns the deleted cluster configuration
// and the error. The deleted cluster configuration can be used
// for rollback.
func DeleteAllEnries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var deletedConfig ClusterConfig

	// Delete all calendars
	for _, elem := range q.Calendars {
		if err := qc.DeleteCalendar(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.Calendars = append(deletedConfig.Calendars, elem)
	}

	// Delete all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.DeleteComplexEntry(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.ComplexEntries = append(deletedConfig.ComplexEntries, elem)
	}

	// Delete all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.DeleteCkptInterface(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.CkptInterfaces = append(deletedConfig.CkptInterfaces, elem)
	}

	// Delete all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.DeleteHostConfiguration(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.HostConfigurations = append(deletedConfig.HostConfigurations, elem)
	}

	// Delete all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.DeleteExecHost(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.ExecHosts = append(deletedConfig.ExecHosts, elem)
	}

	// Delete all admin hosts
	if err := qc.DeleteAdminHost(q.AdminHosts); err != nil {
		return deletedConfig, err
	}
	deletedConfig.AdminHosts = q.AdminHosts

	// Delete all host groups
	for _, elem := range q.HostGroups {
		if err := qc.DeleteHostGroup(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.HostGroups = append(deletedConfig.HostGroups, elem)
	}

	// Delete all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.DeleteResourceQuotaSet(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.ResourceQuotaSets = append(deletedConfig.ResourceQuotaSets, elem)
	}

	// Delete all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.DeleteParallelEnvironment(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.ParallelEnvironments = append(deletedConfig.ParallelEnvironments, elem)
	}

	// Delete all projects
	for _, elem := range q.Projects {
		if err := qc.DeleteProject([]string{elem.Name}); err != nil {
			return deletedConfig, err
		}
		deletedConfig.Projects = append(deletedConfig.Projects, elem)
	}

	// Delete all users
	for _, elem := range q.Users {
		if err := qc.DeleteUser([]string{elem.Name}); err != nil {
			return deletedConfig, err
		}
		deletedConfig.Users = append(deletedConfig.Users, elem)
	}

	// Delete all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.DeleteClusterQueue(elem.Name); err != nil {
			return deletedConfig, err
		}
		deletedConfig.ClusterQueues = append(deletedConfig.ClusterQueues, elem)
	}

	// Delete all user set lists
	for _, elem := range q.UserSetLists {
		listName := "" // Assuming UserSetListConfig has a field or method to get the ListName
		if err := qc.DeleteUserSetList(listName); err != nil {
			return deletedConfig, err
		}
		deletedConfig.UserSetLists = append(deletedConfig.UserSetLists, elem)
	}

	return deletedConfig, nil
}
