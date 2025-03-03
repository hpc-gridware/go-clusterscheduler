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

package app

import (
	"log"
	"strconv"
	"strings"

	st "github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

// CalculatePercentages calculates the level and total percentages for all nodes
func (a *App) CalculatePercentages(nodes []st.SharetreeNode) []st.SharetreeNode {
	// Create a copy of the nodes to avoid modifying the original slice
	result := make([]st.SharetreeNode, len(nodes))
	copy(result, nodes)

	// Create a map of nodes by ID for easy access
	nodeMap := make(map[int]*st.SharetreeNode)
	for i := range result {
		nodeMap[result[i].ID] = &result[i]
	}

	// Build a parent-child relationship map
	parentMap := make(map[int]int) // child ID -> parent ID
	for _, node := range result {
		if node.ChildNodes == "NONE" || node.ChildNodes == "" {
			continue
		}

		childIDs := strings.Split(node.ChildNodes, ",")
		for _, childIDStr := range childIDs {
			childIDStr = strings.TrimSpace(childIDStr)
			if childIDStr == "" {
				continue
			}

			childID, err := strconv.Atoi(childIDStr)
			if err != nil {
				log.Printf("Warning: Invalid child ID: %s", childIDStr)
				continue
			}

			parentMap[childID] = node.ID
		}
	}

	// Map of parent ID -> total shares of all children
	levelSharesMap := make(map[int]int)

	// Calculate level shares (sum of shares for all children of a parent)
	for _, node := range result {
		// Skip root node for level shares calculation
		if node.ID == 0 {
			continue
		}

		parentID, hasParent := parentMap[node.ID]
		if hasParent {
			levelSharesMap[parentID] += node.Shares
		}
	}

	// Calculate total shares in the whole tree (excluding root node)
	totalTreeShares := 0
	hasChildNodes := false

	for _, node := range result {
		// Skip root node for total shares calculation
		if node.ID == 0 {
			// Check if root has child nodes for total percentage calculation later
			if node.ChildNodes != "NONE" && node.ChildNodes != "" {
				hasChildNodes = true
			}
			continue
		}

		totalTreeShares += node.Shares
	}

	// Now calculate percentages for each node
	for i := range result {
		// Special handling for root node (ID 0)
		if result[i].ID == 0 {
			// Root node always has 100% level percentage
			result[i].LevelPercentage = 100.0

			// Root node's total percentage is 0% if it has children, 100% otherwise
			if hasChildNodes {
				result[i].TotalPercentage = 0.0
			} else {
				result[i].TotalPercentage = 100.0
			}
		} else {
			// For non-root nodes, calculate normally

			// Total percentage (of the entire tree excluding root)
			if totalTreeShares > 0 {
				result[i].TotalPercentage = float64(result[i].Shares) / float64(totalTreeShares) * 100
			} else {
				result[i].TotalPercentage = 0
			}

			// Level percentage (relative to siblings)
			parentID, hasParent := parentMap[result[i].ID]
			if hasParent {
				levelShares := levelSharesMap[parentID]
				if levelShares > 0 {
					result[i].LevelPercentage = float64(result[i].Shares) / float64(levelShares) * 100
				} else {
					result[i].LevelPercentage = 0
				}
			} else {
				// Orphaned non-root node
				result[i].LevelPercentage = 100.0
			}
		}

		// Log the calculated percentages for debugging
		log.Printf("Node %d (%s): Level Percentage = %.2f%%, Total Percentage = %.2f%%",
			result[i].ID, result[i].Name, result[i].LevelPercentage, result[i].TotalPercentage)
	}

	return result
}
