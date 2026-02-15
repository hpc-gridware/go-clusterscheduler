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

package core

import (
	"errors"
	"fmt"
	"reflect"
)

// ClusterConfigComparison contains the differences between two
// ClusterConfig structs.
type ClusterConfigComparison struct {
	IsSame              bool `json:"is_same"`
	GlobalConfigChanged bool `json:"global_config_changed"`
	// Contains added objects
	DiffAdded *ClusterConfig `json:"diff_added,omitempty"`
	// Modified objects
	DiffModified *ClusterConfig `json:"diff_modified,omitempty"`
	// Contains removed objects
	DiffRemoved *ClusterConfig `json:"diff_removed,omitempty"`
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

	// Scheduler config comparison
	if !reflect.DeepEqual(c.SchedulerConfig, new.SchedulerConfig) {
		comparison.DiffModified.SchedulerConfig = new.SchedulerConfig
	}

	// Calendars comparison
	resultCalendars, err := FindDifferencesMap(c.Calendars, new.Calendars)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for calendars: %w", err)
	}

	comparison.DiffAdded.Calendars = resultCalendars.Added
	comparison.DiffModified.Calendars = resultCalendars.Modified
	comparison.DiffRemoved.Calendars = resultCalendars.Removed

	// Complexes comparison
	resultComplexes, err := FindDifferencesMap(c.ComplexEntries, new.ComplexEntries)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for complex entries: %w", err)
	}

	comparison.DiffAdded.ComplexEntries = resultComplexes.Added
	comparison.DiffModified.ComplexEntries = resultComplexes.Modified
	comparison.DiffRemoved.ComplexEntries = resultComplexes.Removed

	// CkptInterfaces comparison
	resultCkptInterfaces, err := FindDifferencesMap(c.CkptInterfaces, new.CkptInterfaces)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for checkpoint interfaces: %w", err)
	}

	comparison.DiffAdded.CkptInterfaces = resultCkptInterfaces.Added
	comparison.DiffModified.CkptInterfaces = resultCkptInterfaces.Modified
	comparison.DiffRemoved.CkptInterfaces = resultCkptInterfaces.Removed

	// HostConfigurations comparison
	resultHostConfigurations, err := FindDifferencesMap(c.HostConfigurations, new.HostConfigurations)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for host configurations: %w", err)
	}

	comparison.DiffAdded.HostConfigurations = resultHostConfigurations.Added
	comparison.DiffModified.HostConfigurations = resultHostConfigurations.Modified
	comparison.DiffRemoved.HostConfigurations = resultHostConfigurations.Removed

	// ExecHosts comparison
	resultExecHosts, err := FindDifferencesMap(c.ExecHosts, new.ExecHosts)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for exec hosts: %w", err)
	}

	comparison.DiffAdded.ExecHosts = resultExecHosts.Added
	comparison.DiffModified.ExecHosts = resultExecHosts.Modified
	comparison.DiffRemoved.ExecHosts = resultExecHosts.Removed

	// AdminHosts comparison
	resultAdminHosts, err := FindDifferences(c.AdminHosts, new.AdminHosts, "")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for admin hosts: %w", err)
	}

	comparison.DiffAdded.AdminHosts = append(comparison.DiffAdded.AdminHosts, resultAdminHosts.Added...)
	comparison.DiffRemoved.AdminHosts = append(comparison.DiffRemoved.AdminHosts, resultAdminHosts.Removed...)

	// HostGroups comparison
	resultHostGroups, err := FindDifferencesMap(c.HostGroups, new.HostGroups)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for host groups: %w", err)
	}

	comparison.DiffAdded.HostGroups = resultHostGroups.Added
	comparison.DiffModified.HostGroups = resultHostGroups.Modified
	comparison.DiffRemoved.HostGroups = resultHostGroups.Removed

	// ResourceQuotaSets comparison
	resultResourceQuotaSets, err := FindDifferencesMap(c.ResourceQuotaSets, new.ResourceQuotaSets)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for resource quota sets: %w", err)
	}

	comparison.DiffAdded.ResourceQuotaSets = resultResourceQuotaSets.Added
	comparison.DiffModified.ResourceQuotaSets = resultResourceQuotaSets.Modified
	comparison.DiffRemoved.ResourceQuotaSets = resultResourceQuotaSets.Removed

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
	resultParallelEnvironments, err := FindDifferencesMap(c.ParallelEnvironments, new.ParallelEnvironments)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for parallel environments: %w", err)
	}

	comparison.DiffAdded.ParallelEnvironments = resultParallelEnvironments.Added
	comparison.DiffModified.ParallelEnvironments = resultParallelEnvironments.Modified
	comparison.DiffRemoved.ParallelEnvironments = resultParallelEnvironments.Removed

	// Projects comparison
	resultProjects, err := FindDifferencesMap(c.Projects, new.Projects)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for projects: %w", err)
	}

	comparison.DiffAdded.Projects = resultProjects.Added
	comparison.DiffModified.Projects = resultProjects.Modified
	comparison.DiffRemoved.Projects = resultProjects.Removed

	// Users comparison
	resultUsers, err := FindDifferencesMap(c.Users, new.Users)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for users: %w", err)
	}

	comparison.DiffAdded.Users = resultUsers.Added
	comparison.DiffModified.Users = resultUsers.Modified
	comparison.DiffRemoved.Users = resultUsers.Removed

	// ClusterQueues comparison
	resultClusterQueues, err := FindDifferencesMap(c.ClusterQueues, new.ClusterQueues)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for cluster queues: %w", err)
	}

	comparison.DiffAdded.ClusterQueues = resultClusterQueues.Added
	comparison.DiffModified.ClusterQueues = resultClusterQueues.Modified
	comparison.DiffRemoved.ClusterQueues = resultClusterQueues.Removed

	// UserSetLists comparison
	resultUserSetLists, err := FindDifferencesMap(c.UserSetLists, new.UserSetLists)
	if err != nil {
		return nil, fmt.Errorf("error finding differences for user set lists: %w", err)
	}

	comparison.DiffAdded.UserSetLists = resultUserSetLists.Added
	comparison.DiffModified.UserSetLists = resultUserSetLists.Modified
	comparison.DiffRemoved.UserSetLists = resultUserSetLists.Removed

	// SubmitHosts comparison
	resultSubmitHosts, err := FindDifferences(c.SubmitHosts, new.SubmitHosts, "")
	if err != nil {
		return nil, fmt.Errorf("error finding differences for submit hosts: %w", err)
	}

	comparison.DiffAdded.SubmitHosts = append(comparison.DiffAdded.SubmitHosts,
		resultSubmitHosts.Added...)
	comparison.DiffRemoved.SubmitHosts = append(comparison.DiffRemoved.SubmitHosts,
		resultSubmitHosts.Removed...)

	return comparison, nil
}

type DiffResult[T any] struct {
	Added    []T
	Modified []T
	Removed  []T
}

type DiffResultMap[T any] struct {
	Added    map[string]T
	Modified map[string]T
	Removed  map[string]T
}

func FindDifferencesMap[T any](oldMap, newMap map[string]T) (DiffResultMap[T], error) {
	added := make(map[string]T)
	modified := make(map[string]T)
	removed := make(map[string]T)

	for key, newItem := range newMap {
		oldItem, ok := oldMap[key]
		if !ok {
			added[key] = newItem
		} else if !reflect.DeepEqual(oldItem, newItem) {
			modified[key] = newItem
		}
	}

	for key, oldItem := range oldMap {
		_, ok := newMap[key]
		if !ok {
			removed[key] = oldItem
		}
	}

	return DiffResultMap[T]{Added: added, Modified: modified, Removed: removed}, nil
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
