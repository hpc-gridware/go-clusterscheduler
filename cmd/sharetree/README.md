# Simple Sharetree Editor

A web-based visualization and editing tool for Open Cluster
Scheduler (OCS) and Gridware Cluster Scheduler (GCS) sharetree
configurations.

## Overview

Sharetree Editor provides a simple interface for creating,
viewing, and modifying sharetree hierarchies used by the
Open Cluster Scheduler (OCS) and Gridware Cluster Scheduler
(GCS). This tool is designed to make managing complex
sharetree structures more intuitive through a visual tree
representation.

> **Note:** This is a very basic share tree editor currently
not making any interaction with Cluster Scheduler. Based on
feedback, more functionalities are being added.

## Features

- Visual tree representation of sharetree structures
- Create, edit, and delete sharetree nodes
- Automatic calculation of level and total percentages
- Support for both user and project node types
- Upload and download sharetree configurations in SGE format
- Real-time validation of sharetree structures
- Temporary file storage for session persistence

## Installation

### Prerequisites

- Go 1.23 or higher
- Web browser with JavaScript enabled

### Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/hpc-gridware/go-clusterscheduler.git
   ```

2. Build and run the application:

   ```bash
   cd go-clusterscheduler/cmd/sharetree
   go run main.go
   ```

3. Open your browser and navigate to:

   ```
   http://localhost:8080
   ```

## Usage

### Creating a New Sharetree

- Click "New Sharetree" to start with a clean Root node
- Use the "Add Child Node" button to add nodes under the currently selected node
- Use the "Add Sibling Node" to add nodes at the same level

### Editing Nodes

- Click on any node to select it and view its properties in the right panel
- Modify node properties (name, type, shares) and click "Save Node"

### Uploading and Downloading

- Click "Upload" to import an existing SGE format sharetree file (`qconf -sstree  > <sharetree_file>`)
- Click "Download Sharetree" to export your current sharetree configuration
 (after storing the original sharetree configuration it can be applied
 with `qconf -Astree <sharetree_file>`)

## Development

### Project Structure

- `/pkg/sharetree` - Core sharetree data structure and validation logic
- `/pkg/app` - Application state management
- `/pkg/api` - API handlers for the web interface
- `/templates` - HTML templates for the web UI
- `/static` - Static assets for the web UI

### Running Tests

```bash
go test ./...
```

### Possible Improvements

- Load sharetree directly from Gridware Cluster Scheduler
- Load users and projects from Gridware Cluster Scheduler.
- Use $SGE_ROOT/utilbin/lx-amd64/sge_share_mon for retrieving the sharetree usage.
- Semantic validation of the sharetree configuration.
