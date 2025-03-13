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
	"errors"
	"fmt"
)

// Apply compares the current cluster configuration with the new configuration
// and applies the changes. If dryRun is true, it only prints the plan of the actions,
// otherwise it applies the changes.
func Apply(qc QConf, newConfig ClusterConfig, dryRun bool) error {
	currentConfig, err := qc.GetClusterConfiguration()
	if err != nil {
		return fmt.Errorf("failed to get current cluster configuration: %w", err)
	}

	comparison, err := currentConfig.CompareTo(newConfig)
	if err != nil {
		return fmt.Errorf("failed to compare configurations: %w", err)
	}

	if comparison.IsSame {
		fmt.Println("No changes to apply.")
		return nil
	}

	if dryRun {
		fmt.Println("Dry run - planned changes:")
		qc, err = NewCommandLineQConf(CommandLineQConfConfig{
			Executable: "qconf",
			DryRun:     true,
		})
		if err != nil {
			return fmt.Errorf("failed to create qconf: %w", err)
		}
	}

	if comparison.DiffAdded != nil {
		if _, err := AddAllEntries(qc, *comparison.DiffAdded); err != nil {
			return fmt.Errorf("failed to add elements: %w", err)
		}
	}

	if comparison.DiffModified != nil {
		if _, err := ModifyAllEntries(qc, *comparison.DiffModified); err != nil {
			return fmt.Errorf("failed to modify elements: %w", err)
		}
	}

	if comparison.DiffRemoved != nil {
		if _, err := DeleteAllEnries(qc, *comparison.DiffRemoved, false); err != nil {
			return fmt.Errorf("failed to delete elements: %w", err)
		}
	}

	return nil
}

// AddAllEntries adds all elements to the cluster configuration
// and returns the applied cluster configuration. Note, that
// the configuration elements must not exist before otherwise an
// error is returned.
//
// The global config is ignored.
//
// In case of an error, it returns the applied cluster configuration
// and the error. The applied cluster configuration can be used
// for rollback.
//
// Order:
// 1. UserSetLists
// 2. Projects
// 3. Users
// 4. Managers
// 5. Operators
// 6. HostConfigurations
// 7. HostGroups
// 8. ExecHosts
// 9. ComplexEntries
// 10. Calendars
// 11. CkptInterfaces
// 12. AdminHosts
// 13. ResourceQuotaSets
// 14. ParallelEnvironments
// 15. ClusterQueues
// 16. SubmitHosts
func AddAllEntries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var appliedConfig ClusterConfig

	// Add all user set lists
	for k, elem := range q.UserSetLists {
		if err := qc.AddUserSetList(elem.Name, elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add user set list %s: %w",
				elem.Name, err)
		}
		if appliedConfig.UserSetLists == nil {
			appliedConfig.UserSetLists = make(map[string]UserSetListConfig)
		}
		appliedConfig.UserSetLists[k] = elem
	}

	// Add all projects - can have userset lists
	for _, elem := range q.Projects {
		if err := qc.AddProject(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add project %s: %w",
				elem.Name, err)
		}
		if appliedConfig.Projects == nil {
			appliedConfig.Projects = make(map[string]ProjectConfig)
		}
		appliedConfig.Projects[elem.Name] = elem
	}

	// Add all users (can reference projects)
	for _, elem := range q.Users {
		if err := qc.AddUser(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add user %s: %w",
				elem.Name, err)
		}
		if appliedConfig.Users == nil {
			appliedConfig.Users = make(map[string]UserConfig)
		}
		appliedConfig.Users[elem.Name] = elem
	}

	// Add all managers
	if err := qc.AddUserToManagerList(q.Managers); err != nil {
		return appliedConfig, fmt.Errorf("failed to add managers: %w", err)
	}
	appliedConfig.Managers = q.Managers

	// Add all operators
	if err := qc.AddUserToOperatorList(q.Operators); err != nil {
		return appliedConfig, fmt.Errorf("failed to add operators: %w", err)
	}
	appliedConfig.Operators = q.Operators

	// Add all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.AddHostConfiguration(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add host configuration %s: %w",
				elem.Name, err)
		}
		if appliedConfig.HostConfigurations == nil {
			appliedConfig.HostConfigurations = make(map[string]HostConfiguration, 0)
		}
		appliedConfig.HostConfigurations[elem.Name] = elem
	}

	// Add all host groups
	for _, elem := range q.HostGroups {
		if err := qc.AddHostGroup(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add host group %s: %w",
				elem.Name, err)
		}
		if appliedConfig.HostGroups == nil {
			appliedConfig.HostGroups = make(map[string]HostGroupConfig, 0)
		}
		appliedConfig.HostGroups[elem.Name] = elem
	}

	// Add all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.AddExecHost(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add exec host %s: %w",
				elem.Name, err)
		}
		if appliedConfig.ExecHosts == nil {
			appliedConfig.ExecHosts = make(map[string]HostExecConfig, 0)
		}
		appliedConfig.ExecHosts[elem.Name] = elem
	}

	// Add all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.AddComplexEntry(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add complex entry %s: %w",
				elem.Name, err)
		}
		if appliedConfig.ComplexEntries == nil {
			appliedConfig.ComplexEntries = make(map[string]ComplexEntryConfig, 0)
		}
		appliedConfig.ComplexEntries[elem.Name] = elem

	}

	// Add all calendars
	for _, elem := range q.Calendars {
		if err := qc.AddCalendar(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add calendar %s: %w",
				elem.Name, err)
		}
		if appliedConfig.Calendars == nil {
			appliedConfig.Calendars = make(map[string]CalendarConfig, 0)
		}
		appliedConfig.Calendars[elem.Name] = elem
	}

	// Add all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.AddCkptInterface(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add ckpt interface %s: %w",
				elem.Name, err)
		}
		if appliedConfig.CkptInterfaces == nil {
			appliedConfig.CkptInterfaces = make(map[string]CkptInterfaceConfig, 0)
		}
		appliedConfig.CkptInterfaces[elem.Name] = elem
	}

	// Add all admin hosts
	if err := qc.AddAdminHost(q.AdminHosts); err != nil {
		return appliedConfig, fmt.Errorf("failed to add admin hosts: %w", err)
	}
	appliedConfig.AdminHosts = q.AdminHosts

	// Add all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.AddResourceQuotaSet(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add resource quota set %s: %w",
				elem.Name, err)
		}
		if appliedConfig.ResourceQuotaSets == nil {
			appliedConfig.ResourceQuotaSets = make(map[string]ResourceQuotaSetConfig, 0)
		}
		appliedConfig.ResourceQuotaSets[elem.Name] = elem
	}

	// Add all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.AddParallelEnvironment(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add parallel environment %s: %w",
				elem.Name, err)
		}
		if appliedConfig.ParallelEnvironments == nil {
			appliedConfig.ParallelEnvironments = make(map[string]ParallelEnvironmentConfig, 0)
		}
		appliedConfig.ParallelEnvironments[elem.Name] = elem
	}

	// Add all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.AddClusterQueue(elem); err != nil {
			return appliedConfig, fmt.Errorf("failed to add cluster queue %s: %w",
				elem.Name, err)
		}
		if appliedConfig.ClusterQueues == nil {
			appliedConfig.ClusterQueues = make(map[string]ClusterQueueConfig, 0)
		}
		appliedConfig.ClusterQueues[elem.Name] = elem
	}

	// Add all submit hosts
	if err := qc.AddSubmitHosts(q.SubmitHosts); err != nil {
		return appliedConfig, fmt.Errorf("failed to add submit hosts: %w", err)
	}
	appliedConfig.SubmitHosts = q.SubmitHosts

	return appliedConfig, nil
}

// ModifyAllEntries modifies all elements in the cluster configuration
// and returns the applied cluster configuration. The elements must exist
// before; otherwise, an error is returned.
//
// The global config is NOT ignored.
//
// In case of an error, it returns the modified cluster configuration
// and the error. The modified cluster configuration can be used
// for rollback.
func ModifyAllEntries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var modifiedConfig ClusterConfig

	// Modify all user set lists
	for _, elem := range q.UserSetLists {
		if err := qc.ModifyUserset(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify userset %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.UserSetLists == nil {
			modifiedConfig.UserSetLists = make(map[string]UserSetListConfig, 0)
		}
		modifiedConfig.UserSetLists[elem.Name] = elem
	}

	// Modify all projects
	for _, elem := range q.Projects {
		if err := qc.ModifyProject(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify project %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.Projects == nil {
			modifiedConfig.Projects = make(map[string]ProjectConfig, 0)
		}
		modifiedConfig.Projects[elem.Name] = elem
	}

	// Modify all users
	for _, elem := range q.Users {
		if err := qc.ModifyUser(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify user %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.Users == nil {
			modifiedConfig.Users = make(map[string]UserConfig, 0)
		}
		modifiedConfig.Users[elem.Name] = elem
	}

	// Modify all managers
	for _, elem := range q.Managers {
		if err := qc.AddUserToManagerList([]string{elem}); err != nil {
			return modifiedConfig, fmt.Errorf("failed to add user to manager list %s: %w",
				elem, err)
		}
		modifiedConfig.Managers = append(modifiedConfig.Managers, elem)
	}

	// Modify all operators
	for _, elem := range q.Operators {
		if err := qc.AddUserToOperatorList([]string{elem}); err != nil {
			return modifiedConfig, fmt.Errorf("failed to add user to operator list %s: %w",
				elem, err)
		}
		modifiedConfig.Operators = append(modifiedConfig.Operators, elem)
	}

	// Modify all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.ModifyHostConfiguration(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify host configuration %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.HostConfigurations == nil {
			modifiedConfig.HostConfigurations = make(map[string]HostConfiguration, 0)
		}
		modifiedConfig.HostConfigurations[elem.Name] = elem
	}

	// Modify all host groups
	for _, elem := range q.HostGroups {
		if err := qc.ModifyHostGroup(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify host group %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.HostGroups == nil {
			modifiedConfig.HostGroups = make(map[string]HostGroupConfig, 0)
		}
		modifiedConfig.HostGroups[elem.Name] = elem
	}

	// Modify all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.ModifyExecHost(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify exec host %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.ExecHosts == nil {
			modifiedConfig.ExecHosts = make(map[string]HostExecConfig, 0)
		}
		modifiedConfig.ExecHosts[elem.Name] = elem
	}

	// Modify all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.ModifyAllComplexes([]ComplexEntryConfig{elem}); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify complex entry %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.ComplexEntries == nil {
			modifiedConfig.ComplexEntries = make(map[string]ComplexEntryConfig, 0)
		}
		modifiedConfig.ComplexEntries[elem.Name] = elem
	}

	// Modify all calendars
	for _, elem := range q.Calendars {
		if err := qc.ModifyCalendar(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify calendar %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.Calendars == nil {
			modifiedConfig.Calendars = make(map[string]CalendarConfig, 0)
		}
		modifiedConfig.Calendars[elem.Name] = elem
	}

	// Modify all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.ModifyCkptInterface(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify ckpt interface %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.CkptInterfaces == nil {
			modifiedConfig.CkptInterfaces = make(map[string]CkptInterfaceConfig, 0)
		}
		modifiedConfig.CkptInterfaces[elem.Name] = elem
	}

	// Modify all admin hosts
	if err := qc.AddAdminHost(q.AdminHosts); err != nil {
		return modifiedConfig, fmt.Errorf("failed to add admin hosts: %w", err)
	}
	modifiedConfig.AdminHosts = q.AdminHosts

	// Modify all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.ModifyResourceQuotaSet(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify resource quota set %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.ResourceQuotaSets == nil {
			modifiedConfig.ResourceQuotaSets = make(map[string]ResourceQuotaSetConfig, 0)
		}
		modifiedConfig.ResourceQuotaSets[elem.Name] = elem
	}

	// Modify all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.ModifyParallelEnvironment(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify parallel environment %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.ParallelEnvironments == nil {
			modifiedConfig.ParallelEnvironments = make(map[string]ParallelEnvironmentConfig, 0)
		}
		modifiedConfig.ParallelEnvironments[elem.Name] = elem
	}

	// Modify all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.ModifyClusterQueue(elem.Name, elem); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify cluster queue %s: %w",
				elem.Name, err)
		}
		if modifiedConfig.ClusterQueues == nil {
			modifiedConfig.ClusterQueues = make(map[string]ClusterQueueConfig, 0)
		}
		modifiedConfig.ClusterQueues[elem.Name] = elem
	}

	// Modify global config if set
	if q.GlobalConfig != nil {
		if err := qc.ModifyGlobalConfig(*q.GlobalConfig); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify global config: %w", err)
		}
		modifiedConfig.GlobalConfig = q.GlobalConfig
	}

	// Modify scheduler config if set
	if q.SchedulerConfig != nil {
		if err := qc.ModifySchedulerConfig(*q.SchedulerConfig); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify scheduler config: %w", err)
		}
		modifiedConfig.SchedulerConfig = q.SchedulerConfig
	}

	// Modify submit hosts if set
	if q.SubmitHosts != nil {
		if err := qc.AddSubmitHosts(q.SubmitHosts); err != nil {
			return modifiedConfig, fmt.Errorf("failed to modify submit hosts: %w", err)
		}
		modifiedConfig.SubmitHosts = q.SubmitHosts
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
func DeleteAllEnries(qc QConf, q ClusterConfig, continueOnError bool) (ClusterConfig, error) {
	var deletedConfig ClusterConfig

	var allErrors []error

	// Delete all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.DeleteClusterQueue(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting cluster queue %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting cluster queue %s: %w", elem.Name, err)
		}
		if deletedConfig.ClusterQueues == nil {
			deletedConfig.ClusterQueues = make(map[string]ClusterQueueConfig, 0)
		}
		deletedConfig.ClusterQueues[elem.Name] = elem
	}

	// Delete all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.DeleteParallelEnvironment(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting parallel environment %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting parallel environment %s: %w", elem.Name, err)
		}
		if deletedConfig.ParallelEnvironments == nil {
			deletedConfig.ParallelEnvironments = make(map[string]ParallelEnvironmentConfig, 0)
		}
		deletedConfig.ParallelEnvironments[elem.Name] = elem
	}

	// Delete all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.DeleteResourceQuotaSet(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting resource quota set %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting resource quota set %s: %w", elem.Name, err)
		}
		if deletedConfig.ResourceQuotaSets == nil {
			deletedConfig.ResourceQuotaSets = make(map[string]ResourceQuotaSetConfig, 0)
		}
		deletedConfig.ResourceQuotaSets[elem.Name] = elem
	}

	// Delete all admin hosts
	if err := qc.DeleteAdminHost(q.AdminHosts); err != nil {
		if continueOnError {
			allErrors = append(allErrors,
				fmt.Errorf("error deleting admin hosts: %w", err))
		} else {
			return deletedConfig,
				fmt.Errorf("error deleting admin hosts: %w", err)
		}
	}
	deletedConfig.AdminHosts = q.AdminHosts

	// Delete all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.DeleteCkptInterface(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting ckpt interface %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting ckpt interface %s: %w", elem.Name, err)
		}
		if deletedConfig.CkptInterfaces == nil {
			deletedConfig.CkptInterfaces = make(map[string]CkptInterfaceConfig, 0)
		}
		deletedConfig.CkptInterfaces[elem.Name] = elem
	}

	// Delete all calendars
	for _, elem := range q.Calendars {
		if err := qc.DeleteCalendar(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting calendar %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting calendar %s: %w", elem.Name, err)
		}
		if deletedConfig.Calendars == nil {
			deletedConfig.Calendars = make(map[string]CalendarConfig, 0)
		}
		deletedConfig.Calendars[elem.Name] = elem
	}

	// Delete all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.DeleteComplexEntry(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting complex entry %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting complex entry %s: %w", elem.Name, err)
		}
		if deletedConfig.ComplexEntries == nil {
			deletedConfig.ComplexEntries = make(map[string]ComplexEntryConfig, 0)
		}
		deletedConfig.ComplexEntries[elem.Name] = elem
	}

	// Delete all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.DeleteExecHost(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting exec host %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting exec host %s: %w", elem.Name, err)
		}
		if deletedConfig.ExecHosts == nil {
			deletedConfig.ExecHosts = make(map[string]HostExecConfig, 0)
		}
		deletedConfig.ExecHosts[elem.Name] = elem
	}

	// Delete all host groups
	for _, elem := range q.HostGroups {
		if err := qc.DeleteHostGroup(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting host group %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting host group %s: %w", elem.Name, err)
		}
		if deletedConfig.HostGroups == nil {
			deletedConfig.HostGroups = make(map[string]HostGroupConfig, 0)
		}
		deletedConfig.HostGroups[elem.Name] = elem
	}

	// Delete all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.DeleteHostConfiguration(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting host configuration %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting host configuration %s: %w", elem.Name, err)
		}
		if deletedConfig.HostConfigurations == nil {
			deletedConfig.HostConfigurations = make(map[string]HostConfiguration, 0)
		}
		deletedConfig.HostConfigurations[elem.Name] = elem
	}

	// Delete all operators
	for _, elem := range q.Operators {
		if err := qc.DeleteUserFromOperatorList([]string{elem}); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting operator %s: %w", elem, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting operator %s: %w", elem, err)
		}
		deletedConfig.Operators = q.Operators
	}

	// Delete all managers
	for _, elem := range q.Managers {
		if err := qc.DeleteUserFromManagerList([]string{elem}); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting manager %s: %w", elem, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting manager %s: %w", elem, err)
		}
		deletedConfig.Managers = append(deletedConfig.Managers, elem)
	}

	// Delete all projects
	for _, elem := range q.Projects {
		if err := qc.DeleteProject([]string{elem.Name}); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting project %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting project %s: %w", elem.Name, err)
		}
		if deletedConfig.Projects == nil {
			deletedConfig.Projects = make(map[string]ProjectConfig, 0)
		}
		deletedConfig.Projects[elem.Name] = elem
	}

	// Delete all user set lists
	for _, elem := range q.UserSetLists {
		if err := qc.DeleteUserSetList(elem.Name); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting user set list %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting user set list %s: %w", elem.Name, err)
		}
		if deletedConfig.UserSetLists == nil {
			deletedConfig.UserSetLists = make(map[string]UserSetListConfig, 0)
		}
		deletedConfig.UserSetLists[elem.Name] = elem
	}

	// Delete all users
	for _, elem := range q.Users {
		if err := qc.DeleteUser([]string{elem.Name}); err != nil {
			if continueOnError {
				allErrors = append(allErrors,
					fmt.Errorf("error deleting user %s: %w", elem.Name, err))
				continue
			}
			return deletedConfig,
				fmt.Errorf("error deleting user %s: %w", elem.Name, err)
		}
		if deletedConfig.Users == nil {
			deletedConfig.Users = make(map[string]UserConfig, 0)
		}
		deletedConfig.Users[elem.Name] = elem
	}

	// Delete all submit hosts
	if err := qc.DeleteSubmitHost(q.SubmitHosts); err != nil {
		if continueOnError {
			allErrors = append(allErrors, fmt.Errorf("error deleting submit hosts: %w", err))
		} else {
			return deletedConfig, fmt.Errorf("error deleting submit hosts: %w", err)
		}
	}
	deletedConfig.SubmitHosts = q.SubmitHosts

	if len(allErrors) > 0 {
		errMsg := "error deleting multiple objects: "
		for _, err := range allErrors {
			errMsg += err.Error() + "; "
		}
		return deletedConfig, errors.New(errMsg)
	}

	return deletedConfig, nil
}
