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
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// Comparison types re-exported from core.
type ClusterConfigComparison = core.ClusterConfigComparison
type DiffResult[T any] = core.DiffResult[T]
type DiffResultMap[T any] = core.DiffResultMap[T]

var NewClusterConfigComparison = core.NewClusterConfigComparison

// FindDifferencesMap finds differences between two maps.
func FindDifferencesMap[T any](oldMap, newMap map[string]T) (DiffResultMap[T], error) {
	return core.FindDifferencesMap(oldMap, newMap)
}

// FindDifferences finds differences between two slices.
func FindDifferences[T any](oldList, newList []T, keyField string) (DiffResult[T], error) {
	return core.FindDifferences(oldList, newList, keyField)
}
