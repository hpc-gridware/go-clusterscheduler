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
func AddAllEntries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var appliedConfig ClusterConfig

	// Add all user set lists
	for _, elem := range q.UserSetLists {
		if err := qc.AddUserSetList(elem.Name, elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.UserSetLists = append(appliedConfig.UserSetLists, elem)
	}

	// Add all projects - can have userset lists
	for _, elem := range q.Projects {
		if err := qc.AddProject(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.Projects = append(appliedConfig.Projects, elem)
	}

	// Add all users (can reference projects)
	for _, elem := range q.Users {
		if err := qc.AddUser(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.Users = append(appliedConfig.Users, elem)
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

	// Add all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.AddHostConfiguration(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.HostConfigurations = append(appliedConfig.HostConfigurations, elem)
	}

	// Add all host groups
	for _, elem := range q.HostGroups {
		if err := qc.AddHostGroup(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.HostGroups = append(appliedConfig.HostGroups, elem)
	}

	// Add all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.AddExecHost(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ExecHosts = append(appliedConfig.ExecHosts, elem)
	}

	// Add all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.AddComplexEntry(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ComplexEntries = append(appliedConfig.ComplexEntries, elem)
	}

	// Add all calendars
	for _, elem := range q.Calendars {
		if err := qc.AddCalendar(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.Calendars = append(appliedConfig.Calendars, elem)
	}

	// Add all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.AddCkptInterface(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.CkptInterfaces = append(appliedConfig.CkptInterfaces, elem)
	}

	// Add all admin hosts
	if err := qc.AddAdminHost(q.AdminHosts); err != nil {
		return appliedConfig, err
	}
	appliedConfig.AdminHosts = q.AdminHosts

	// Add all resource quota sets
	for _, elem := range q.ResourceQuotaSets {
		if err := qc.AddResourceQuotaSet(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ResourceQuotaSets = append(appliedConfig.ResourceQuotaSets, elem)
	}

	// Add all parallel environments
	for _, elem := range q.ParallelEnvironments {
		if err := qc.AddParallelEnvironment(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ParallelEnvironments = append(appliedConfig.ParallelEnvironments, elem)
	}

	// Add all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.AddClusterQueue(elem); err != nil {
			return appliedConfig, err
		}
		appliedConfig.ClusterQueues = append(appliedConfig.ClusterQueues, elem)
	}

	return appliedConfig, nil
}

// ModifyAllEntries modifies all elements in the cluster configuration
// and returns the applied cluster configuration. The elements must exist
// before; otherwise, an error is returned.
//
// The global config is ignored.
//
// In case of an error, it returns the modified cluster configuration
// and the error. The modified cluster configuration can be used
// for rollback.
func ModifyAllEntries(qc QConf, q ClusterConfig) (ClusterConfig, error) {
	var modifiedConfig ClusterConfig

	// Modify all user set lists
	for _, elem := range q.UserSetLists {
		if err := qc.ModifyUserset(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.UserSetLists = append(modifiedConfig.UserSetLists, elem)
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

	// Modify all managers
	for _, elem := range q.Managers {
		if err := qc.AddUserToManagerList([]string{elem}); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.Managers = append(modifiedConfig.Managers, elem)
	}

	// Modify all operators
	for _, elem := range q.Operators {
		if err := qc.AddUserToOperatorList([]string{elem}); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.Operators = append(modifiedConfig.Operators, elem)
	}

	// Modify all host configurations
	for _, elem := range q.HostConfigurations {
		if err := qc.ModifyHostConfiguration(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.HostConfigurations = append(modifiedConfig.HostConfigurations, elem)
	}

	// Modify all host groups
	for _, elem := range q.HostGroups {
		if err := qc.ModifyHostGroup(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.HostGroups = append(modifiedConfig.HostGroups, elem)
	}

	// Modify all exec hosts
	for _, elem := range q.ExecHosts {
		if err := qc.ModifyExecHost(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ExecHosts = append(modifiedConfig.ExecHosts, elem)
	}

	// Modify all complex entries
	for _, elem := range q.ComplexEntries {
		if err := qc.ModifyAllComplexes([]ComplexEntryConfig{elem}); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ComplexEntries = append(modifiedConfig.ComplexEntries, elem)
	}

	// Modify all calendars
	for _, elem := range q.Calendars {
		if err := qc.ModifyCalendar(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.Calendars = append(modifiedConfig.Calendars, elem)
	}

	// Modify all ckpt interfaces
	for _, elem := range q.CkptInterfaces {
		if err := qc.ModifyCkptInterface(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.CkptInterfaces = append(modifiedConfig.CkptInterfaces, elem)
	}

	// Modify all admin hosts
	if err := qc.AddAdminHost(q.AdminHosts); err != nil {
		return modifiedConfig, err
	}
	modifiedConfig.AdminHosts = q.AdminHosts

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

	// Modify all cluster queues
	for _, elem := range q.ClusterQueues {
		if err := qc.ModifyClusterQueue(elem.Name, elem); err != nil {
			return modifiedConfig, err
		}
		modifiedConfig.ClusterQueues = append(modifiedConfig.ClusterQueues, elem)
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
		deletedConfig.ClusterQueues = append(deletedConfig.ClusterQueues, elem)
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
		deletedConfig.ParallelEnvironments = append(deletedConfig.ParallelEnvironments, elem)
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
		deletedConfig.ResourceQuotaSets = append(deletedConfig.ResourceQuotaSets, elem)
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
		deletedConfig.CkptInterfaces = append(deletedConfig.CkptInterfaces, elem)
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
		deletedConfig.Calendars = append(deletedConfig.Calendars, elem)
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
		deletedConfig.ComplexEntries = append(deletedConfig.ComplexEntries, elem)
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
		deletedConfig.ExecHosts = append(deletedConfig.ExecHosts, elem)
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
		deletedConfig.HostGroups = append(deletedConfig.HostGroups, elem)
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
		deletedConfig.HostConfigurations = append(deletedConfig.HostConfigurations, elem)
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
		deletedConfig.Projects = append(deletedConfig.Projects, elem)
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
		deletedConfig.UserSetLists = append(deletedConfig.UserSetLists, elem)
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
		deletedConfig.Users = append(deletedConfig.Users, elem)
	}

	if len(allErrors) > 0 {
		errMsg := "error deleting multiple objects: "
		for _, err := range allErrors {
			errMsg += err.Error() + "; "
		}
		return deletedConfig, errors.New(errMsg)
	}

	return deletedConfig, nil
}
