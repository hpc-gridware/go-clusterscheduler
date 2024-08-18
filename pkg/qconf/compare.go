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
	"reflect"
)

// ClusterConfigComparison contains the differences between two
// ClusterConfig structs.
type ClusterConfigComparison struct {
	IsSame              bool
	GlobalConfigChanged bool
	// Contains added objects
	DiffAdded *ClusterConfig
	// Modified objects
	DiffModified *ClusterConfig
	// Contains removed objects
	DiffRemoved *ClusterConfig
}

func NewClusterConfigComparison() *ClusterConfigComparison {
	return &ClusterConfigComparison{
		DiffAdded:    &ClusterConfig{},
		DiffModified: &ClusterConfig{},
		DiffRemoved:  &ClusterConfig{},
	}
}

func (c *ClusterConfigComparison) String() string {
	return fmt.Sprintf("IsSame: %t, GlobalConfigChanged: %t, Added: %v, Modified: %v, Removed: %v",
		c.IsSame, c.GlobalConfigChanged, c.DiffAdded, c.DiffModified, c.DiffRemoved)
}

// CompareTo compares the current ClusterConfig with the new ClusterConfig and
// returns a struct containing the differences between the two ClusterConfig structs.
// It returns an error if there is an issue comparing the two ClusterConfig structs.
// It does not compare the ClusterEnvironment.
func (c *ClusterConfig) CompareTo(new ClusterConfig) (*ClusterConfigComparison, error) {
	// Compare the two ClusterConfig structs and return a list of differences
	comparison := NewClusterConfigComparison()

	// Global config comparison
	if !reflect.DeepEqual(c.GlobalConfig, new.GlobalConfig) {
		comparison.GlobalConfigChanged = true
		comparison.DiffModified.GlobalConfig = new.GlobalConfig
	}

	// Calendars comparison
	resultCalendars, err := FindDifferences(c.Calendars, new.Calendars, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for calendars: %w", err)
	}

	comparison.DiffAdded.Calendars = append(comparison.DiffAdded.Calendars, resultCalendars.Added...)
	comparison.DiffModified.Calendars = append(comparison.DiffModified.Calendars, resultCalendars.Modified...)
	comparison.DiffRemoved.Calendars = append(comparison.DiffRemoved.Calendars, resultCalendars.Removed...)

	// Complexes comparison
	resultComplexes, err := FindDifferences(c.ComplexEntries, new.ComplexEntries, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for complex entries: %w", err)
	}

	comparison.DiffAdded.ComplexEntries = append(comparison.DiffAdded.ComplexEntries, resultComplexes.Added...)
	comparison.DiffModified.ComplexEntries = append(comparison.DiffModified.ComplexEntries, resultComplexes.Modified...)
	comparison.DiffRemoved.ComplexEntries = append(comparison.DiffRemoved.ComplexEntries, resultComplexes.Removed...)

	// CkptInterfaces comparison
	resultCkptInterfaces, err := FindDifferences(c.CkptInterfaces, new.CkptInterfaces, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for checkpoint interfaces: %w", err)
	}

	comparison.DiffAdded.CkptInterfaces = append(comparison.DiffAdded.CkptInterfaces, resultCkptInterfaces.Added...)
	comparison.DiffModified.CkptInterfaces = append(comparison.DiffModified.CkptInterfaces, resultCkptInterfaces.Modified...)
	comparison.DiffRemoved.CkptInterfaces = append(comparison.DiffRemoved.CkptInterfaces, resultCkptInterfaces.Removed...)

	// HostConfigurations comparison
	resultHostConfigurations, err := FindDifferences(c.HostConfigurations, new.HostConfigurations, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for host configurations: %w", err)
	}

	comparison.DiffAdded.HostConfigurations = append(comparison.DiffAdded.HostConfigurations, resultHostConfigurations.Added...)
	comparison.DiffModified.HostConfigurations = append(comparison.DiffModified.HostConfigurations, resultHostConfigurations.Modified...)
	comparison.DiffRemoved.HostConfigurations = append(comparison.DiffRemoved.HostConfigurations, resultHostConfigurations.Removed...)

	// ExecHosts comparison
	resultExecHosts, err := FindDifferences(c.ExecHosts, new.ExecHosts, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for exec hosts: %w", err)
	}

	comparison.DiffAdded.ExecHosts = append(comparison.DiffAdded.ExecHosts, resultExecHosts.Added...)
	comparison.DiffModified.ExecHosts = append(comparison.DiffModified.ExecHosts, resultExecHosts.Modified...)
	comparison.DiffRemoved.ExecHosts = append(comparison.DiffRemoved.ExecHosts, resultExecHosts.Removed...)

	// AdminHosts comparison
	resultAdminHosts, err := FindDifferences(c.AdminHosts, new.AdminHosts, "")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for admin hosts: %w", err)
	}

	comparison.DiffAdded.AdminHosts = append(comparison.DiffAdded.AdminHosts, resultAdminHosts.Added...)
	comparison.DiffRemoved.AdminHosts = append(comparison.DiffRemoved.AdminHosts, resultAdminHosts.Removed...)

	// HostGroups comparison
	resultHostGroups, err := FindDifferences(c.HostGroups, new.HostGroups, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for host groups: %w", err)
	}
	comparison.DiffAdded.HostGroups = append(comparison.DiffAdded.HostGroups, resultHostGroups.Added...)
	comparison.DiffModified.HostGroups = append(comparison.DiffModified.HostGroups, resultHostGroups.Modified...)
	comparison.DiffRemoved.HostGroups = append(comparison.DiffRemoved.HostGroups, resultHostGroups.Removed...)

	// ResourceQuotaSets comparison
	resultResourceQuotaSets, err := FindDifferences(c.ResourceQuotaSets, new.ResourceQuotaSets, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for resource quota sets: %w", err)
	}
	comparison.DiffAdded.ResourceQuotaSets = append(comparison.DiffAdded.ResourceQuotaSets, resultResourceQuotaSets.Added...)
	comparison.DiffModified.ResourceQuotaSets = append(comparison.DiffModified.ResourceQuotaSets, resultResourceQuotaSets.Modified...)
	comparison.DiffRemoved.ResourceQuotaSets = append(comparison.DiffRemoved.ResourceQuotaSets, resultResourceQuotaSets.Removed...)

	// Managers comparison
	resultManagers, err := FindDifferences(c.Managers, new.Managers, "")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for managers: %w", err)
	}
	comparison.DiffAdded.Managers = append(comparison.DiffAdded.Managers, resultManagers.Added...)
	comparison.DiffRemoved.Managers = append(comparison.DiffRemoved.Managers, resultManagers.Removed...)

	// Operators comparison
	resultOperators, err := FindDifferences(c.Operators, new.Operators, "")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for operators: %w", err)
	}
	comparison.DiffAdded.Operators = append(comparison.DiffAdded.Operators, resultOperators.Added...)
	comparison.DiffRemoved.Operators = append(comparison.DiffRemoved.Operators, resultOperators.Removed...)

	// ParallelEnvironments comparison
	resultParallelEnvironments, err := FindDifferences(c.ParallelEnvironments, new.ParallelEnvironments, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for parallel environments: %w", err)
	}
	comparison.DiffAdded.ParallelEnvironments = append(comparison.DiffAdded.ParallelEnvironments, resultParallelEnvironments.Added...)
	comparison.DiffModified.ParallelEnvironments = append(comparison.DiffModified.ParallelEnvironments, resultParallelEnvironments.Modified...)
	comparison.DiffRemoved.ParallelEnvironments = append(comparison.DiffRemoved.ParallelEnvironments, resultParallelEnvironments.Removed...)

	// Projects comparison
	resultProjects, err := FindDifferences(c.Projects, new.Projects, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for projects: %w", err)
	}
	comparison.DiffAdded.Projects = append(comparison.DiffAdded.Projects, resultProjects.Added...)
	comparison.DiffModified.Projects = append(comparison.DiffModified.Projects, resultProjects.Modified...)
	comparison.DiffRemoved.Projects = append(comparison.DiffRemoved.Projects, resultProjects.Removed...)

	// Users comparison
	resultUsers, err := FindDifferences(c.Users, new.Users, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for users: %w", err)
	}
	comparison.DiffAdded.Users = append(comparison.DiffAdded.Users, resultUsers.Added...)
	comparison.DiffModified.Users = append(comparison.DiffModified.Users, resultUsers.Modified...)
	comparison.DiffRemoved.Users = append(comparison.DiffRemoved.Users, resultUsers.Removed...)

	// ClusterQueues comparison
	resultClusterQueues, err := FindDifferences(c.ClusterQueues, new.ClusterQueues, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for cluster queues: %w", err)
	}
	comparison.DiffAdded.ClusterQueues = append(comparison.DiffAdded.ClusterQueues, resultClusterQueues.Added...)
	comparison.DiffModified.ClusterQueues = append(comparison.DiffModified.ClusterQueues, resultClusterQueues.Modified...)
	comparison.DiffRemoved.ClusterQueues = append(comparison.DiffRemoved.ClusterQueues, resultClusterQueues.Removed...)

	// UserSetLists comparison
	resultUserSetLists, err := FindDifferences(c.UserSetLists, new.UserSetLists, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for user set lists: %w", err)
	}
	comparison.DiffAdded.UserSetLists = append(comparison.DiffAdded.UserSetLists, resultUserSetLists.Added...)
	comparison.DiffModified.UserSetLists = append(comparison.DiffModified.UserSetLists, resultUserSetLists.Modified...)
	comparison.DiffRemoved.UserSetLists = append(comparison.DiffRemoved.UserSetLists, resultUserSetLists.Removed...)

	// UserConfig comparison (if necessary, adjust accordingly)
	resultUserConfig, err := FindDifferences(c.UserConfig, new.UserConfig, "Name")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for user config: %w", err)
	}
	comparison.DiffAdded.UserConfig = append(comparison.DiffAdded.UserConfig, resultUserConfig.Added...)
	comparison.DiffModified.UserConfig = append(comparison.DiffModified.UserConfig, resultUserConfig.Modified...)
	comparison.DiffRemoved.UserConfig = append(comparison.DiffRemoved.UserConfig, resultUserConfig.Removed...)

	return comparison, nil
}

type DiffResult[T any] struct {
	Added    []T
	Modified []T
	Removed  []T
}

// FindDifferences finds differences between old and new lists. In case
// of structs, keyField is used to determine the key field. If keyField is
// empty, the item itself is used as the key (for slices).
func FindDifferences[T any](oldList, newList []T, keyField string) (DiffResult[T], error) {
	added := []T{}
	modified := []T{}
	removed := []T{}

	newMap := make(map[interface{}]T)
	oldMap := make(map[interface{}]T)

	// if T is a struct, we can use reflection to get the key field
	if keyField != "" {
		for _, item := range newList {
			key, err := getKey(item, keyField)
			if err != nil {
				return DiffResult[T]{}, err
			}
			newMap[key] = item
		}
	} else {
		// if T is not a struct, we can use the item itself as the key
		for _, item := range newList {
			newMap[item] = item
		}
	}

	if keyField != "" {
		for _, item := range oldList {
			key, err := getKey(item, keyField)
			if err != nil {
				return DiffResult[T]{}, err
			}
			oldMap[key] = item
		}
	} else {
		// if T is not a struct, we can use the item itself as the key
		for _, item := range oldList {
			oldMap[item] = item
		}
	}

	for key, newItem := range newMap {
		if oldItem, exists := oldMap[key]; exists {
			delete(oldMap, key) // Prevents double-checking it in the next loop
			if !reflect.DeepEqual(oldItem, newItem) {
				modified = append(modified, newItem)
			}
		} else {
			added = append(added, newItem)
		}
	}

	for _, oldItem := range oldMap {
		removed = append(removed, oldItem)
	}

	return DiffResult[T]{
		Added:    added,
		Modified: modified,
		Removed:  removed,
	}, nil
}

// getKey extracts the key field from the item.
func getKey[T any](item T, keyField string) (interface{}, error) {
	value := reflect.ValueOf(item)
	if reflect.TypeOf(item).Kind() != reflect.Struct {
		return nil, errors.New("item must be a struct")
	}
	field := value.FieldByName(keyField)
	if !field.IsValid() {
		return nil, fmt.Errorf("field %s not found in the item", keyField)
	}
	return field.Interface(), nil
}
