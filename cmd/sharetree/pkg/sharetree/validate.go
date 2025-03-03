/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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

package sharetree

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidationError represents a sharetree validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidateSharetree performs comprehensive validation on a sharetree
func ValidateSharetree(nodes []SharetreeNode) error {
	if len(nodes) == 0 {
		return ValidationError{Message: "Sharetree is empty"}
	}

	// Check for unique IDs
	idMap := make(map[int]bool)
	for _, node := range nodes {
		if idMap[node.ID] {
			return ValidationError{Message: fmt.Sprintf("Duplicate node ID found: %d", node.ID)}
		}
		idMap[node.ID] = true

		// Validate required fields
		if node.Name == "" {
			return ValidationError{Message: fmt.Sprintf("Node with ID %d is missing a name", node.ID)}
		}
	}

	// Validate parent-child relationships
	childParentMap := make(map[int]bool) // tracks nodes that are children
	for _, node := range nodes {
		if node.ChildNodes != "NONE" && node.ChildNodes != "" {
			childIDs := strings.Split(node.ChildNodes, ",")
			for _, idStr := range childIDs {
				idStr = strings.TrimSpace(idStr)
				if idStr == "" {
					continue
				}

				id, err := strconv.Atoi(idStr)
				if err != nil {
					return ValidationError{Message: fmt.Sprintf("Invalid child ID format: %s", idStr)}
				}

				// Check if child exists
				if !idMap[id] {
					return ValidationError{Message: fmt.Sprintf("Node %d references child %d which doesn't exist", node.ID, id)}
				}

				childParentMap[id] = true
			}
		}
	}

	// Check for orphaned nodes (except root with ID 0)
	for _, node := range nodes {
		if node.ID != 0 && !childParentMap[node.ID] {
			return ValidationError{Message: fmt.Sprintf("Orphaned node found: %d", node.ID)}
		}
	}

	return nil
}
