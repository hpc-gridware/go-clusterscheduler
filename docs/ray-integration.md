# Integrating SGE/Grid Engine with Ray

This guide explains how to integrate Sun Grid Engine (SGE) / Open Grid Engine / Gridware Cluster Scheduler with [Ray](https://www.ray.io/), a distributed computing framework.

## Overview

Ray is a unified framework for scaling AI and Python applications. It provides distributed computing capabilities through a cluster of workers. While Ray has built-in support for cloud providers and schedulers like SLURM, it can also be integrated with SGE/Grid Engine clusters.

## Architecture

The integration follows a similar pattern to Ray's SLURM integration:

1. **Head Node**: A Ray head node runs on one SGE execution host
2. **Worker Nodes**: Ray workers are submitted as SGE jobs
3. **Auto-scaling**: Workers can be dynamically added or removed based on workload
4. **Resource Management**: SGE handles resource allocation and job scheduling

## Integration Approaches

### Approach 1: Manual Cluster Setup

Launch Ray head and workers manually using SGE job submission:

```bash
# Submit Ray head node
qsub -b y -cwd -o ray_head.log -e ray_head.err \
  ray start --head --port=6379 --dashboard-host=0.0.0.0

# Wait for head node to start and get its address
HEAD_NODE_IP=$(qstat -j <job_id> | grep exec_host | cut -d'@' -f2 | cut -d'.' -f1)

# Submit Ray workers
qsub -b y -cwd -t 1-10 -o ray_worker.\$TASK_ID.log -e ray_worker.\$TASK_ID.err \
  ray start --address=${HEAD_NODE_IP}:6379
```

### Approach 2: Ray Cluster Launcher with SGE Backend

Ray provides a cluster launcher that can be extended to support SGE. This approach uses a YAML configuration file to define the cluster.

#### Prerequisites

- SGE/Grid Engine cluster properly configured
- Ray installed on all execution hosts
- Python 3.7+ on all nodes
- Network connectivity between all SGE execution hosts

## Configuration

### Example Ray Cluster Configuration (ray-sge-cluster.yaml)

```yaml
# A unique identifier for the cluster
cluster_name: ray-sge-cluster

# The maximum number of workers the cluster will have at any given time
max_workers: 10

# Cloud provider-specific configuration
# For SGE, we use a custom node provider
provider:
  type: external
  module: ray_sge_provider  # Custom module for SGE integration
  
# How Ray will authenticate with newly launched nodes
auth:
  ssh_user: your_username
  ssh_private_key: ~/.ssh/id_rsa

# SGE-specific settings
sge:
  # Queue to submit jobs to
  queue: all.q
  
  # Parallel environment (for multi-slot jobs)
  parallel_environment: smp
  
  # Project to charge resources to
  project: your_project
  
  # Additional qsub options
  qsub_options: "-l h_rt=24:00:00 -l mem_free=4G"

# Configuration for the head node
head_node:
  # Resources allocated to head node
  resources:
    slots: 4
    memory: "16G"

# Configuration for worker nodes  
worker_nodes:
  # Resources per worker
  resources:
    slots: 2
    memory: "8G"
  
  # Minimum number of workers to maintain
  min_workers: 2
  
  # Maximum number of workers to scale to
  max_workers: 10

# Commands to run on each node
setup_commands:
  - pip install -U ray[default]
  - pip install -U numpy pandas

# Command to start Ray on the head node
head_start_ray_commands:
  - ray stop
  - ray start --head --port=6379 --dashboard-host=0.0.0.0 --autoscaling-config=~/ray_bootstrap_config.yaml

# Command to start Ray on worker nodes
worker_start_ray_commands:
  - ray stop
  - ray start --address=$RAY_HEAD_IP:6379
```

## Implementation with Go API

This repository provides a Go API for SGE that can be used to build Ray integration tools. Here's an example:

### Example: Ray Worker Launcher

Create a Go application that manages Ray workers as SGE jobs:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    
    qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
    qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
    "github.com/hpc-gridware/go-clusterscheduler/pkg/qdel/v9.0"
)

type RayClusterManager struct {
    qsub     qsub.QSub
    qstat    qstat.QStat
    qdel     qdel.QDel
    headIP   string
    workerJobs []int64
}

func NewRayClusterManager(headIP string) (*RayClusterManager, error) {
    qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
    if err != nil {
        return nil, err
    }
    
    qst, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
    if err != nil {
        return nil, err
    }
    
    qd, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{})
    if err != nil {
        return nil, err
    }
    
    return &RayClusterManager{
        qsub:   qs,
        qstat:  qst,
        qdel:   qd,
        headIP: headIP,
        workerJobs: make([]int64, 0),
    }, nil
}

func (r *RayClusterManager) StartHead(ctx context.Context) (int64, error) {
    jobId, _, err := r.qsub.Submit(ctx, qsub.JobOptions{
        Command:     "ray",
        CommandArgs: []string{"start", "--head", "--port=6379", "--dashboard-host=0.0.0.0"},
        Binary:      qsub.ToPtr(true),
        JobName:     qsub.ToPtr("ray-head"),
        OutputPath:  qsub.ToPtr("ray_head.log"),
        ErrorPath:   qsub.ToPtr("ray_head.err"),
        MemoryLimit: qsub.ToPtr("16G"),
    })
    
    if err != nil {
        return 0, fmt.Errorf("failed to submit head node: %w", err)
    }
    
    fmt.Printf("Ray head node submitted with job ID: %d\n", jobId)
    return jobId, nil
}

func (r *RayClusterManager) AddWorkers(ctx context.Context, count int) error {
    rayAddress := fmt.Sprintf("%s:6379", r.headIP)
    
    jobId, _, err := r.qsub.Submit(ctx, qsub.JobOptions{
        Command:     "ray",
        CommandArgs: []string{"start", "--address=" + rayAddress},
        Binary:      qsub.ToPtr(true),
        JobName:     qsub.ToPtr("ray-worker"),
        OutputPath:  qsub.ToPtr("ray_worker.$JOB_ID.log"),
        ErrorPath:   qsub.ToPtr("ray_worker.$JOB_ID.err"),
        JobArray:    qsub.ToPtr(fmt.Sprintf("1-%d", count)),
        MemoryLimit: qsub.ToPtr("8G"),
    })
    
    if err != nil {
        return fmt.Errorf("failed to submit workers: %w", err)
    }
    
    r.workerJobs = append(r.workerJobs, jobId)
    fmt.Printf("Submitted %d Ray workers with job ID: %d\n", count, jobId)
    return nil
}

func (r *RayClusterManager) Shutdown(ctx context.Context) error {
    // Delete all worker jobs
    for _, jobId := range r.workerJobs {
        if err := r.qdel.DeleteJobs(ctx, []int64{jobId}); err != nil {
            fmt.Printf("Warning: failed to delete job %d: %v\n", jobId, err)
        }
    }
    
    fmt.Println("Ray cluster shutdown complete")
    return nil
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: ray-cluster-manager <head-node-ip>")
        os.Exit(1)
    }
    
    ctx := context.Background()
    manager, err := NewRayClusterManager(os.Args[1])
    if err != nil {
        fmt.Printf("Error creating cluster manager: %v\n", err)
        os.Exit(1)
    }
    
    // Start head node
    headJobID, err := manager.StartHead(ctx)
    if err != nil {
        fmt.Printf("Error starting head node: %v\n", err)
        os.Exit(1)
    }
    
    // Wait for head node to be ready
    fmt.Println("Waiting for head node to start...")
    time.Sleep(30 * time.Second)
    
    // Add workers
    if err := manager.AddWorkers(ctx, 5); err != nil {
        fmt.Printf("Error adding workers: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Ray cluster is running. Head node job ID: %d\n", headJobID)
    fmt.Println("Press Ctrl+C to shutdown...")
    
    // Keep running until interrupted
    select {}
}
```

## Best Practices

1. **Resource Specification**: Always specify memory and CPU requirements to help SGE schedule jobs efficiently
2. **Job Arrays**: Use SGE job arrays for launching multiple identical workers
3. **Error Handling**: Implement proper error handling and logging for job submission failures
4. **Cleanup**: Always clean up SGE jobs when shutting down the Ray cluster
5. **Monitoring**: Use `qstat` to monitor job status and implement health checks
6. **Network Configuration**: Ensure all SGE execution hosts can communicate on Ray's ports (default: 6379 for Redis)

## Monitoring and Debugging

### Check Ray Cluster Status

```bash
# From the head node
ray status

# Check SGE job status
qstat -f
```

### View Ray Dashboard

The Ray dashboard is accessible at `http://<head-node>:8265` by default.

### Logs

- Ray logs: `~/ray/session_*/logs/`
- SGE job output: `ray_head.log`, `ray_worker.*.log`
- SGE job errors: `ray_head.err`, `ray_worker.*.err`

## Resource Management

SGE provides fine-grained resource management that can be leveraged:

- **Slots**: Map to Ray CPUs using `-pe smp N`
- **Memory**: Specify using `-l mem_free=XG`
- **Runtime**: Set maximum runtime using `-l h_rt=HH:MM:SS`
- **Queues**: Direct jobs to specific queues using `-q queue_name`

## Example: Submitting a Ray Application

Once your Ray cluster is running on SGE:

```python
import ray

# Connect to existing cluster
ray.init(address="auto")

# Your Ray application
@ray.remote
def compute_task(x):
    return x * x

# Submit tasks
futures = [compute_task.remote(i) for i in range(100)]
results = ray.get(futures)

print(f"Results: {results}")
```

## Troubleshooting

### Workers Cannot Connect to Head

- Verify network connectivity between execution hosts
- Check that port 6379 is open
- Ensure `RAY_HEAD_IP` is correctly set

### Jobs Stuck in Queue

- Check SGE queue status: `qstat -f`
- Verify resource availability: `qhost`
- Review queue configuration: `qconf -sq <queue_name>`

### Out of Memory Errors

- Increase memory request in job submission: `-l mem_free=16G`
- Monitor actual memory usage: `qacct -j <job_id>`

## Further Reading

- [Ray Documentation](https://docs.ray.io/)
- [Ray Cluster Launcher](https://docs.ray.io/en/latest/cluster/launcher.html)
- [SGE/Grid Engine Documentation](https://github.com/hpc-gridware/clusterscheduler)
- [go-clusterscheduler Examples](../examples/)

## Support

For issues related to:
- **SGE Integration**: Open an issue in this repository
- **Ray Core**: Visit [Ray GitHub](https://github.com/ray-project/ray)
- **SGE/Grid Engine**: Visit [Gridware Cluster Scheduler](https://github.com/hpc-gridware/clusterscheduler)
