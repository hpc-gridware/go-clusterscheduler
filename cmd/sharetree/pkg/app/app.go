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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	st "github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

// App represents the sharetree application with its state
type App struct {
	CurrentSharetree []st.SharetreeNode
	TempFilePath     string // Path to the temporary file
}

// NewApp creates and initializes a new App
func NewApp() (*App, error) {
	app := &App{}
	if err := app.InitializeTempFile(); err != nil {
		return nil, err
	}
	log.Printf("Initialized temporary sharetree file: %s", app.TempFilePath)
	return app, nil
}

// InitializeTempFile creates a fresh temporary file for this session
func (a *App) InitializeTempFile() error {
	// Create a temp directory if it doesn't exist
	tempDir := filepath.Join(os.TempDir(), "sharetreeeditor")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}

	// Create a unique temp file name with timestamp
	timestamp := time.Now().Format("20060102_150405")
	a.TempFilePath = filepath.Join(tempDir, fmt.Sprintf("sharetree_%s.sge", timestamp))

	// Initialize with a basic root node (User type with ID 0)
	rootNode := []st.SharetreeNode{
		{
			ID:         0,
			Name:       "Root",
			Type:       0, // User type
			Shares:     1,
			ChildNodes: "NONE",
		},
	}

	// Save the initial tree to the temp file
	a.CurrentSharetree = rootNode
	return a.SaveSharetreeAsSGE(rootNode, a.TempFilePath)
}

// ValidateRootNode ensures a single root node
func (a *App) ValidateRootNode(nodes []st.SharetreeNode) error {
	// Check if we have exactly one node with ID 0
	rootCount := 0
	for _, node := range nodes {
		if node.ID == 0 {
			rootCount++
			// Verify root is named "Root"
			if node.Name != "Root" {
				return st.ValidationError{Message: "Root node must be named 'Root'"}
			}
			// Verify root is a user (type 0)
			if node.Type != 0 {
				return st.ValidationError{Message: "Root node must be User type"}
			}
		}
	}

	if rootCount == 0 {
		return st.ValidationError{Message: "Missing Root node (ID 0)"}
	}

	if rootCount > 1 {
		return st.ValidationError{Message: "Multiple root nodes found"}
	}

	// Check if root has siblings (other nodes without parents)
	rootSiblings := 0
	for _, node := range nodes {
		isChild := false
		for _, parent := range nodes {
			if parent.ChildNodes != "NONE" && parent.ChildNodes != "" {
				childIDs := strings.Split(parent.ChildNodes, ",")
				for _, idStr := range childIDs {
					id, err := strconv.Atoi(strings.TrimSpace(idStr))
					if err == nil && id == node.ID {
						isChild = true
						break
					}
				}
			}
			if isChild {
				break
			}
		}

		if !isChild && node.ID != 0 {
			rootSiblings++
		}
	}

	if rootSiblings > 0 {
		return st.ValidationError{Message: "Root node cannot have siblings"}
	}

	return nil
}

// LoadSharetreeFromSGE loads a sharetree from SGE content.
func (a *App) LoadSharetreeFromSGE(content string) ([]st.SharetreeNode, error) {
	return st.ParseSgeContent(content)
}

// SaveSharetreeAsSGE saves a sharetree to an SGE file.
func (a *App) SaveSharetreeAsSGE(nodes []st.SharetreeNode, filename string) error {
	return st.SaveSharetreeToSgeFile(filename, nodes)
}

// GetCurrentTempFileName returns the base filename of the current temp file
func (a *App) GetCurrentTempFileName() string {
	return filepath.Base(a.TempFilePath)
}
