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

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
)

// RayClusterManager manages a Ray cluster using SGE as the backend scheduler
type RayClusterManager struct {
	qsub       qsub.Qsub
	qstat      qstat.QStat
	headJobID  int64
	workerJobs []int64
}

// NewRayClusterManager creates a new RayClusterManager
func NewRayClusterManager() (*RayClusterManager, error) {
	qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create qsub client: %w", err)
	}

	qst, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create qstat client: %w", err)
	}

	return &RayClusterManager{
		qsub:       qs,
		qstat:      qst,
		workerJobs: make([]int64, 0),
	}, nil
}

// StartHead starts the Ray head node as an SGE job
func (r *RayClusterManager) StartHead(ctx context.Context, port int, memory string) (int64, error) {
	rayCommand := fmt.Sprintf("ray start --head --port=%d --dashboard-host=0.0.0.0 --block", port)

	jobId, _, err := r.qsub.Submit(ctx, qsub.JobOptions{
		Command:         "bash",
		CommandArgs:     []string{"-c", rayCommand},
		Binary:          qsub.ToPtr(true),
		JobName:         qsub.ToPtr("ray-head"),
		StdOut:          []string{"ray_head.log"},
		StdErr:          []string{"ray_head.err"},
		ScopedResources: qsub.SimpleLRequest(map[string]string{"mem_free": memory}),
		Shell:           qsub.ToPtr(false),
		MergeStdOutErr:  qsub.ToPtr(true),
	})

	if err != nil {
		return 0, fmt.Errorf("failed to submit head node job: %w", err)
	}

	r.headJobID = jobId
	fmt.Printf("✓ Ray head node submitted with job ID: %d\n", jobId)
	fmt.Printf("  Monitor with: qstat -j %d\n", jobId)
	fmt.Printf("  View logs: tail -f ray_head.log\n")
	return jobId, nil
}

// GetHeadNodeAddress retrieves the hostname where the head node is running
func (r *RayClusterManager) GetHeadNodeAddress(ctx context.Context) (string, error) {
	if r.headJobID == 0 {
		return "", fmt.Errorf("head node not started")
	}

	// Poll for the job to be running
	for i := 0; i < 30; i++ {
		jobs, err := r.qstat.ViewJobsOfUser([]string{})
		if err != nil {
			return "", fmt.Errorf("failed to get job info: %w", err)
		}

		for _, job := range jobs {
			if job.JobID == int(r.headJobID) && job.State == "r" {
				// Extract hostname from queue info
				// Format is typically: queue@hostname
				if job.Queue != "" {
					return job.Queue, nil
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for head node to start")
}

// AddWorkers adds Ray worker nodes as SGE jobs
func (r *RayClusterManager) AddWorkers(ctx context.Context, count int, headAddress string, port int, memory string) error {
	rayAddress := fmt.Sprintf("%s:%d", headAddress, port)
	rayCommand := fmt.Sprintf("ray start --address=%s --block", rayAddress)

	jobId, _, err := r.qsub.Submit(ctx, qsub.JobOptions{
		Command:         "bash",
		CommandArgs:     []string{"-c", rayCommand},
		Binary:          qsub.ToPtr(true),
		JobName:         qsub.ToPtr("ray-worker"),
		StdOut:          []string{"ray_worker.$TASK_ID.log"},
		StdErr:          []string{"ray_worker.$TASK_ID.err"},
		JobArray:        qsub.ToPtr(fmt.Sprintf("1-%d", count)),
		ScopedResources: qsub.SimpleLRequest(map[string]string{"mem_free": memory}),
		Shell:           qsub.ToPtr(false),
		MergeStdOutErr:  qsub.ToPtr(true),
	})

	if err != nil {
		return fmt.Errorf("failed to submit worker jobs: %w", err)
	}

	r.workerJobs = append(r.workerJobs, jobId)
	fmt.Printf("✓ Submitted %d Ray workers with job array ID: %d\n", count, jobId)
	fmt.Printf("  Monitor with: qstat -j %d\n", jobId)
	return nil
}

// GetClusterStatus returns the status of all Ray cluster jobs
func (r *RayClusterManager) GetClusterStatus(ctx context.Context) error {
	jobs, err := r.qstat.ViewJobsOfUser([]string{})
	if err != nil {
		return fmt.Errorf("failed to get job info: %w", err)
	}

	fmt.Println("\n=== Ray Cluster Status ===")
	
	// Head node status
	if r.headJobID != 0 {
		found := false
		for _, job := range jobs {
			if job.JobID == int(r.headJobID) {
				fmt.Printf("Head Node (Job %d): State=%s, Queue=%s\n", 
					r.headJobID, job.State, job.Queue)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Head Node (Job %d): Completed or not found\n", r.headJobID)
		}
	}

	// Worker status
	for _, workerJobID := range r.workerJobs {
		workerCount := 0
		for _, job := range jobs {
			if job.JobID == int(workerJobID) {
				workerCount++
			}
		}
		if workerCount > 0 {
			fmt.Printf("Workers (Job Array %d): %d tasks running\n", workerJobID, workerCount)
		}
	}
	
	fmt.Println("=========================\n")
	return nil
}

// Shutdown terminates all Ray cluster jobs using qdel command
func (r *RayClusterManager) Shutdown(ctx context.Context) error {
	fmt.Println("\nShutting down Ray cluster...")

	// Delete worker jobs first
	for _, jobId := range r.workerJobs {
		cmd := exec.Command("qdel", fmt.Sprintf("%d", jobId))
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠ Warning: failed to delete worker job %d: %v\n", jobId, err)
		} else {
			fmt.Printf("✓ Deleted worker job array %d\n", jobId)
		}
	}

	// Delete head node
	if r.headJobID != 0 {
		cmd := exec.Command("qdel", fmt.Sprintf("%d", r.headJobID))
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠ Warning: failed to delete head job %d: %v\n", r.headJobID, err)
		} else {
			fmt.Printf("✓ Deleted head node job %d\n", r.headJobID)
		}
	}

	fmt.Println("✓ Ray cluster shutdown complete")
	return nil
}

func main() {
	// Configuration
	const (
		rayPort      = 6379
		headMemory   = "16G"
		workerMemory = "8G"
		workerCount  = 5
	)

	ctx := context.Background()

	// Create cluster manager
	manager, err := NewRayClusterManager()
	if err != nil {
		fmt.Printf("Error creating cluster manager: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start head node
	fmt.Println("Starting Ray cluster...")
	headJobID, err := manager.StartHead(ctx, rayPort, headMemory)
	if err != nil {
		fmt.Printf("Error starting head node: %v\n", err)
		os.Exit(1)
	}

	// Wait for head node to be scheduled and get its address
	fmt.Println("\nWaiting for head node to start...")
	time.Sleep(10 * time.Second)

	headAddress, err := manager.GetHeadNodeAddress(ctx)
	if err != nil {
		fmt.Printf("Error getting head node address: %v\n", err)
		fmt.Println("You may need to manually check qstat and get the hostname")
		// Continue anyway with a placeholder
		headAddress = "localhost"
	} else {
		fmt.Printf("✓ Head node running at: %s:%d\n", headAddress, rayPort)
	}

	// Start workers
	fmt.Println("\nStarting Ray workers...")
	if err := manager.AddWorkers(ctx, workerCount, headAddress, rayPort, workerMemory); err != nil {
		fmt.Printf("Error adding workers: %v\n", err)
		manager.Shutdown(ctx)
		os.Exit(1)
	}

	// Display status
	time.Sleep(5 * time.Second)
	manager.GetClusterStatus(ctx)

	// Display connection information
	fmt.Println("=== Ray Cluster Information ===")
	fmt.Printf("Head node job ID: %d\n", headJobID)
	fmt.Printf("Ray address: %s:%d\n", headAddress, rayPort)
	fmt.Printf("Dashboard: http://%s:8265\n", headAddress)
	fmt.Println("\nTo connect from Python:")
	fmt.Println("  import ray")
	fmt.Printf("  ray.init(address='%s:%d')\n", headAddress, rayPort)
	fmt.Println("\nPress Ctrl+C to shutdown the cluster...")
	fmt.Println("================================\n")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nReceived shutdown signal...")

	// Shutdown cluster
	if err := manager.Shutdown(ctx); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		os.Exit(1)
	}
}
