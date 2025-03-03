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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LoadSharetreeFromFile loads sharetree data from the specified file
func LoadSharetreeFromFile(filename string) ([]SharetreeNode, error) {
	// Ensure the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Printf("Sharetree file %s does not exist. Creating default sharetree with Root node.", filename)
		return CreateDefaultSharetree(), nil
	}

	// Read the file
	fileData, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading sharetree file: %v. Creating default sharetree.", err)
		return CreateDefaultSharetree(), nil
	}

	// Parse the data
	var sharetree []SharetreeNode
	if err := json.Unmarshal(fileData, &sharetree); err != nil {
		log.Printf("Error parsing sharetree file: %v. Creating default sharetree.", err)
		return CreateDefaultSharetree(), nil
	}

	if len(sharetree) == 0 {
		log.Printf("Sharetree file %s is empty. Creating default sharetree.", filename)
		return CreateDefaultSharetree(), nil
	}

	log.Printf("Loaded sharetree configuration from %s (%d nodes)", filename, len(sharetree))
	return sharetree, nil
}

// createDefaultSharetree initializes a default sharetree with a root node
func CreateDefaultSharetree() []SharetreeNode {
	return []SharetreeNode{
		{
			ID:         0,
			Name:       DefaultRootNodeName,
			Type:       1, // Project type
			Shares:     1,
			ChildNodes: "NONE",
		},
	}
}

// SaveSharetreeToFile saves sharetree data to the specified file
func SaveSharetreeToFile(filename string, sharetree []SharetreeNode) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal the data to JSON
	jsonData, err := json.MarshalIndent(sharetree, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return err
	}

	log.Printf("Saved sharetree configuration to %s (%d nodes)", filename, len(sharetree))
	return nil
}

// ParseSgeContent parses sharetree data from SGE format string content
func ParseSgeContent(content string) ([]SharetreeNode, error) {
	log.Printf("Parsing SGE content: %s", content)
	lines := strings.Split(content, "\n")

	var nodes []SharetreeNode
	var currentNode SharetreeNode
	var nodeParsing bool = false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if it's a key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Start of a new node
		if key == "id" {
			if nodeParsing {
				// Save the previous node
				nodes = append(nodes, currentNode)
			}
			nodeParsing = true
			currentNode = SharetreeNode{}
			id, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			currentNode.ID = id
		} else if key == "name" {
			currentNode.Name = value
		} else if key == "type" {
			typeVal, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			currentNode.Type = typeVal
		} else if key == "shares" {
			shares, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			currentNode.Shares = shares
		} else if key == "childnodes" {
			// Properly handle childnodes - can be "NONE" or comma-separated IDs
			currentNode.ChildNodes = value
		}
	}

	// Add the last node if exists
	if nodeParsing {
		nodes = append(nodes, currentNode)
	}

	// If no nodes were parsed, return an error
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid nodes found in sharetree content")
	}

	log.Printf("Parsed SGE sharetree content: %v", nodes)

	return nodes, nil
}

// LoadSharetreeFromSgeFile loads sharetree data from the specified file in SGE format
func LoadSharetreeFromSgeFile(filename string) ([]SharetreeNode, error) {
	// Ensure the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Printf("Sharetree file %s does not exist. Creating default sharetree with Root node.", filename)
		return CreateDefaultSharetree(), nil
	}

	// Read the file
	fileData, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading sharetree file: %v. Creating default sharetree.", err)
		return CreateDefaultSharetree(), nil
	}

	// Parse the SGE format using the shared parsing function
	nodes, err := ParseSgeContent(string(fileData))
	if err != nil {
		log.Printf("Error parsing SGE sharetree file: %v. Creating default sharetree.", err)
		return CreateDefaultSharetree(), nil
	}

	log.Printf("Loaded SGE format sharetree configuration from %s (%d nodes)", filename, len(nodes))
	return nodes, nil
}

// SaveSharetreeToSgeFile saves sharetree data to the specified file in SGE format
func SaveSharetreeToSgeFile(filename string, sharetree []SharetreeNode) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Convert to SGE format
	content, err := ConvertToSgeFormat(sharetree)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return err
	}

	log.Printf("Saved SGE format sharetree configuration to %s (%d nodes)", filename, len(sharetree))
	return nil
}

func ConvertToSgeFormat(nodes []SharetreeNode) (string, error) {
	var sb strings.Builder
	sb.WriteString("# OCS Sharetree Configuration\n")
	sb.WriteString("# Generated on " + time.Now().Format(time.RFC3339) + "\n\n")

	for _, node := range nodes {
		sb.WriteString(fmt.Sprintf("id=%d\n", node.ID))
		sb.WriteString(fmt.Sprintf("name=%s\n", node.Name))
		sb.WriteString(fmt.Sprintf("type=%d\n", node.Type))
		sb.WriteString(fmt.Sprintf("shares=%d\n", node.Shares))
		sb.WriteString(fmt.Sprintf("childnodes=%s\n\n", node.ChildNodes))
	}

	return sb.String(), nil
}

// ConvertToJsTreeFormat converts a slice of SharetreeNode to jsTree format
func ConvertToJsTreeFormat(nodes []SharetreeNode) []map[string]interface{} {
	if len(nodes) == 0 {
		return nil
	}

	// First, create a map of all nodes by ID for quick lookup
	nodeMap := make(map[int]*SharetreeNode)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	// Find root node (usually ID 0)
	var rootNode *SharetreeNode
	for i := range nodes {
		if nodes[i].ID == 0 {
			rootNode = &nodes[i]
			break
		}
	}

	// If no root found, use the first node
	if rootNode == nil && len(nodes) > 0 {
		rootNode = &nodes[0]
	}

	// Build the tree recursively
	result := []map[string]interface{}{}
	if rootNode != nil {
		result = append(result, buildJsTreeNode(rootNode, nodeMap))
	}

	return result
}

// buildJsTreeNode recursively builds a jsTree node from a SharetreeNode
func buildJsTreeNode(node *SharetreeNode, nodeMap map[int]*SharetreeNode) map[string]interface{} {
	jsTreeNode := map[string]interface{}{
		"id":   strconv.Itoa(node.ID), // jsTree expects string IDs
		"text": node.Name,
		"data": map[string]interface{}{
			"type":   node.Type,
			"shares": node.Shares,
		},
	}

	// Process children if any
	if node.ChildNodes != "NONE" {
		childIDs := strings.Split(node.ChildNodes, ",")
		children := []map[string]interface{}{}

		for _, idStr := range childIDs {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err != nil {
				continue
			}

			if childNode, ok := nodeMap[id]; ok {
				children = append(children, buildJsTreeNode(childNode, nodeMap))
			}
		}

		if len(children) > 0 {
			jsTreeNode["children"] = children
		}
	}

	return jsTreeNode
}
