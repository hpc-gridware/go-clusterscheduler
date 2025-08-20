/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024-2025 HPC-Gridware GmbH
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

package main

import (
	"fmt"
	"log"
	"os"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Initialize qconf client
	qconfClient, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
		Executable: "qconf",
		DryRun:     false,
		DelayAfter: 0,
	})
	if err != nil {
		log.Fatalf("Failed to create qconf client: %v", err)
	}

	command := os.Args[1]
	switch command {
	case "list":
		listAllResources(qconfClient)
	case "show":
		if len(os.Args) < 3 {
			fmt.Println("Usage: show <resource_name>")
			os.Exit(1)
		}
		showResource(qconfClient, os.Args[2])
	case "add":
		addExampleResource(qconfClient)
	case "add-rsmap":
		addRSMAPResource(qconfClient)
	case "modify":
		if len(os.Args) < 3 {
			fmt.Println("Usage: modify <resource_name>")
			os.Exit(1)
		}
		modifyResource(qconfClient, os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: delete <resource_name>")
			os.Exit(1)
		}
		deleteResource(qconfClient, os.Args[2])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("HPC Resource (Complex Entry) Management Example")
	fmt.Println("Usage: resources <command> [arguments]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list                    - List all resources")
	fmt.Println("  show <name>            - Show details of a specific resource")
	fmt.Println("  add                    - Add an example GPU resource (INT type)")
	fmt.Println("  add-rsmap              - Add an example GPU resource (RSMAP type)")
	fmt.Println("  modify <name>          - Modify a resource's urgency")
	fmt.Println("  delete <name>          - Delete a resource")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ./resources list")
	fmt.Println("  ./resources show gpu")
	fmt.Println("  ./resources add")
	fmt.Println("  ./resources add-rsmap")
	fmt.Println("  ./resources modify gpu")
	fmt.Println("  ./resources delete gpu")
}

// listAllResources displays all complex entries (resources) in the cluster
func listAllResources(qc qconf.QConf) {
	fmt.Println("=== All Cluster Resources (Complex Entries) ===")

	// Get list of resource names
	resourceNames, err := qc.ShowComplexEntries()
	if err != nil {
		log.Fatalf("Failed to get resource list: %v", err)
	}

	fmt.Printf("Found %d resources:\n\n", len(resourceNames))

	// Get all complex entries with details
	allResources, err := qc.ShowAllComplexes()
	if err != nil {
		log.Fatalf("Failed to get resource details: %v", err)
	}

	// Display in formatted table
	fmt.Printf("%-15s %-8s %-10s %-8s %-12s %-12s %-10s %-8s\n",
		"NAME", "SHORTCUT", "TYPE", "RELOP", "REQUESTABLE", "CONSUMABLE", "DEFAULT", "URGENCY")
	fmt.Println("---------------------------------------------------------------------------------------------")

	for _, resource := range allResources {
		fmt.Printf("%-15s %-8s %-10s %-8s %-12s %-12s %-10s %-8d\n",
			resource.Name,
			resource.Shortcut,
			resource.Type,
			resource.Relop,
			resource.Requestable,
			resource.Consumable,
			resource.Default,
			resource.Urgency)
	}
}

// showResource displays detailed information about a specific resource
func showResource(qc qconf.QConf, resourceName string) {
	fmt.Printf("=== Resource Details: %s ===\n", resourceName)

	resource, err := qc.ShowComplexEntry(resourceName)
	if err != nil {
		log.Fatalf("Failed to get resource '%s': %v", resourceName, err)
	}

	fmt.Printf("Name:        %s\n", resource.Name)
	fmt.Printf("Shortcut:    %s\n", resource.Shortcut)
	fmt.Printf("Type:        %s\n", resource.Type)
	fmt.Printf("Relop:       %s\n", resource.Relop)
	fmt.Printf("Requestable: %s\n", resource.Requestable)
	fmt.Printf("Consumable:  %s\n", resource.Consumable)
	fmt.Printf("Default:     %s\n", resource.Default)
	fmt.Printf("Urgency:     %d\n", resource.Urgency)

	fmt.Printf("\nResource Type Explanation:\n")
	switch resource.Type {
	case qconf.ResourceTypeInt:
		fmt.Println("  - Integer type resource (e.g., CPU cores, memory MB)")
	case qconf.ResourceTypeDouble:
		fmt.Println("  - Floating-point resource (e.g., CPU utilization)")
	case qconf.ResourceTypeMemory:
		fmt.Println("  - Memory resource with units (e.g., 1G, 512M)")
	case qconf.ResourceTypeTime:
		fmt.Println("  - Time duration resource (e.g., wallclock time)")
	case qconf.ResourceTypeString:
		fmt.Println("  - String-based resource (e.g., architecture)")
	case qconf.ResourceTypeBool:
		fmt.Println("  - Boolean resource (true/false)")
	case qconf.ResourceTypeRSMAP:
		fmt.Println("  - Resource Map type - manages specific instances of resources")
		fmt.Println("    (e.g., individual GPU IDs, specific network devices)")
		fmt.Println("    Provides exclusive access to named resource instances")
	}
}

// addExampleResource creates a new GPU resource as an example (INT type)
func addExampleResource(qc qconf.QConf) {
	fmt.Println("=== Adding Example GPU Resource (INT type) ===")

	gpuResource := qconf.ComplexEntryConfig{
		Name:        "gpu",
		Shortcut:    "gpu",
		Type:        qconf.ResourceTypeInt,
		Relop:       "<=",
		Requestable: "YES",
		Consumable:  "YES",
		Default:     "0",
		Urgency:     500,
	}

	err := qc.AddComplexEntry(gpuResource)
	if err != nil {
		log.Fatalf("Failed to add GPU resource: %v", err)
	}

	fmt.Println(" Successfully added GPU resource (INT type)")
	fmt.Println("  - Name: gpu")
	fmt.Println("  - Type: INT (integer count)")
	fmt.Println("  - Consumable: YES (tracked per job)")
	fmt.Println("  - Default: 0 (no GPUs by default)")
	fmt.Println("  - Urgency: 500 (medium priority)")
	fmt.Println("\nNote: Configure host complex_values to make GPUs available on execution hosts")
}

// addRSMAPResource creates a new GPU resource using RSMAP type for instance-specific allocation
func addRSMAPResource(qc qconf.QConf) {
	fmt.Println("=== Adding Example GPU Resource (RSMAP type) ===")

	gpuResource := qconf.ComplexEntryConfig{
		Name:        "gpu_rsmap",
		Shortcut:    "grs",
		Type:        qconf.ResourceTypeRSMAP,
		Relop:       "<=",
		Requestable: "YES",
		Consumable:  "YES",
		Default:     "0",
		Urgency:     600,
	}

	err := qc.AddComplexEntry(gpuResource)
	if err != nil {
		log.Fatalf("Failed to add GPU RSMAP resource: %v", err)
	}

	fmt.Println(" Successfully added GPU resource (RSMAP type)")
	fmt.Println("  - Name: gpu_rsmap")
	fmt.Println("  - Type: RSMAP (resource map for specific instances)")
	fmt.Println("  - Consumable: YES (tracked per job)")
	fmt.Println("  - Default: 0 (no GPUs by default)")
	fmt.Println("  - Urgency: 600 (higher priority than INT type)")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Configure host complex_values with GPU instances:")
	fmt.Println("     qconf -me <hostname>")
	fmt.Println("     complex_values gpu_rsmap=4(gpu0 gpu1 gpu2 gpu3)")
	fmt.Println("  2. Submit jobs requesting specific GPU instances:")
	fmt.Println("     qsub -l gpu_rsmap=1 myjob.sh")
	fmt.Println("")
	fmt.Println("RSMAP Benefits:")
	fmt.Println("  - Access to specific GPU instances")
	fmt.Println("  - Prevents resource conflicts between jobs")
}

// modifyResource demonstrates modifying an existing resource
func modifyResource(qc qconf.QConf, resourceName string) {
	fmt.Printf("=== Modifying Resource: %s ===\n", resourceName)

	// First get the current resource configuration
	currentResource, err := qc.ShowComplexEntry(resourceName)
	if err != nil {
		log.Fatalf("Failed to get current resource '%s': %v", resourceName, err)
	}

	fmt.Printf("Current urgency: %d\n", currentResource.Urgency)

	// Modify the urgency (example: increase by 100)
	modifiedResource := currentResource
	modifiedResource.Urgency = currentResource.Urgency + 100

	err = qc.ModifyComplexEntry(resourceName, modifiedResource)
	if err != nil {
		log.Fatalf("Failed to modify resource '%s': %v", resourceName, err)
	}

	fmt.Printf(" Successfully modified resource '%s'\n", resourceName)
	fmt.Printf("  - New urgency: %d (increased by 100)\n", modifiedResource.Urgency)
	fmt.Println("\nNote: Higher urgency values give resources more weight in scheduling decisions")
}

// deleteResource removes a resource from the cluster
func deleteResource(qc qconf.QConf, resourceName string) {
	fmt.Printf("=== Deleting Resource: %s ===\n", resourceName)

	// Show what we're about to delete
	resource, err := qc.ShowComplexEntry(resourceName)
	if err != nil {
		log.Fatalf("Failed to get resource '%s': %v", resourceName, err)
	}

	fmt.Printf("About to delete resource:\n")
	fmt.Printf("  - Name: %s\n", resource.Name)
	fmt.Printf("  - Type: %s\n", resource.Type)
	fmt.Printf("  - Consumable: %s\n", resource.Consumable)

	err = qc.DeleteComplexEntry(resourceName)
	if err != nil {
		log.Fatalf("Failed to delete resource '%s': %v", resourceName, err)
	}

	fmt.Printf(" Successfully deleted resource '%s'\n", resourceName)
	fmt.Println("\nWarning: Jobs requiring this resource may fail until they complete or are modified")
}
