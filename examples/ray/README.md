# Ray on SGE/Grid Engine Integration Example

This directory contains examples and tools for integrating Ray (distributed computing framework) with SGE/Grid Engine using the go-clusterscheduler API.

## Overview

Ray is a unified framework for scaling AI and Python applications. This integration allows you to:
- Launch Ray clusters on SGE/Grid Engine infrastructure
- Manage Ray head and worker nodes as SGE jobs
- Leverage SGE's resource management and scheduling capabilities
- Run distributed Ray applications on HPC clusters

## Files in This Directory

- **`ray_cluster_manager.go`**: Go application that manages Ray clusters using SGE
- **`start_ray_cluster.sh`**: Bash script to start a Ray cluster on SGE
- **`stop_ray_cluster.sh`**: Bash script to stop a Ray cluster on SGE
- **`example_ray_job.py`**: Example Python application demonstrating Ray usage on SGE
- **`README.md`**: This file

## Prerequisites

### On All SGE Execution Hosts

1. **Ray Installation**:
   ```bash
   pip install ray[default]
   ```

2. **Python 3.7+**:
   ```bash
   python --version
   ```

3. **Network Connectivity**: Ensure all execution hosts can communicate on:
   - Port 6379 (Ray default port for Redis)
   - Port 8265 (Ray dashboard, optional)

### For Go Application

1. **Go 1.19+**:
   ```bash
   go version
   ```

2. **SGE Client Tools**: `qsub`, `qstat`, `qdel` must be available

## Usage

### Method 1: Using Shell Scripts (Recommended for Quick Start)

#### Start a Ray Cluster

```bash
# Start with default settings (5 workers)
./start_ray_cluster.sh

# Start with custom number of workers
./start_ray_cluster.sh 10
```

This will:
1. Submit the Ray head node as an SGE job
2. Wait for the head node to start
3. Submit worker nodes as an SGE job array
4. Create `ray_cluster_info.txt` with connection details

#### Run a Ray Application

After the cluster starts, use the connection information:

```bash
# Check the cluster info
cat ray_cluster_info.txt

# Run the example application
python example_ray_job.py --address <head_host>:6379
```

#### Stop the Ray Cluster

```bash
./stop_ray_cluster.sh
```

This will delete all Ray-related SGE jobs.

### Method 2: Using Go Application

#### Build the Application

```bash
go build -o ray-cluster-manager ray_cluster_manager.go
```

#### Run the Manager

```bash
./ray-cluster-manager
```

The Go application will:
1. Start a Ray head node
2. Launch 5 worker nodes (configurable in code)
3. Monitor the cluster status
4. Provide connection information
5. Wait for Ctrl+C to shutdown

You can modify the configuration in the `main()` function:
```go
const (
    rayPort      = 6379      // Ray communication port
    headMemory   = "16G"     // Memory for head node
    workerMemory = "8G"      // Memory per worker
    workerCount  = 5         // Number of workers
)
```

## Example Ray Application

The included `example_ray_job.py` demonstrates several Ray features:

### 1. Pi Estimation (Monte Carlo)

Distributed Monte Carlo sampling to estimate π:
```python
python example_ray_job.py --address <head_host>:6379
```

### 2. Matrix Operations

Distributed matrix multiplications:
```python
python example_ray_job.py --address <head_host>:6379 --no-pi --no-actor
```

### 3. Ray Actors

Stateful distributed actors:
```python
python example_ray_job.py --address <head_host>:6379 --no-pi --no-matrix
```

## Advanced Configuration

### Customizing Resource Requests

Edit the scripts or Go code to change SGE resource requests:

**In Shell Script**:
```bash
# Edit start_ray_cluster.sh
HEAD_MEMORY="32G"      # Increase head node memory
WORKER_MEMORY="16G"    # Increase worker memory

# Add more SGE options
qsub ... -l h_rt=48:00:00 -pe smp 4 ...
```

**In Go Code**:
```go
// Edit ray_cluster_manager.go
jobId, _, err := r.qsub.Submit(ctx, qsub.JobOptions{
    // ... existing options ...
    MemoryLimit: qsub.ToPtr("32G"),
    ParallelEnvironment: qsub.ToPtr("smp"),
    Slots: qsub.ToPtr(4),
})
```

### Specifying SGE Queue

```bash
# In shell script
qsub -q high_priority.q ...

# In Go code
Queue: qsub.ToPtr("high_priority.q"),
```

### Using Parallel Environments

For multi-core Ray workers:

```bash
qsub -pe smp 4 ...  # Request 4 cores per worker
```

## Monitoring

### Check Cluster Status

```bash
# View all jobs
qstat -f

# View specific job details
qstat -j <job_id>

# Ray cluster status (from head node)
ray status --address=<head_host>:6379
```

### View Logs

```bash
# Head node logs
tail -f ray_head.log

# Worker logs (for task array)
tail -f ray_worker.*.log

# Ray internal logs (on the head node)
ls ~/ray/session_*/logs/
```

### Access Ray Dashboard

The Ray dashboard provides a web interface for monitoring:

```bash
# Find the head node hostname
qstat -j <head_job_id> | grep exec_host

# Access dashboard at:
# http://<head_host>:8265
```

## Troubleshooting

### Workers Cannot Connect to Head Node

**Symptoms**: Workers fail to connect, logs show connection errors

**Solutions**:
1. Verify network connectivity:
   ```bash
   # From a worker node
   telnet <head_host> 6379
   ```

2. Check firewall settings:
   ```bash
   # Ensure port 6379 is open
   ```

3. Verify head node is running:
   ```bash
   qstat -j <head_job_id>
   ```

### Jobs Stuck in Queue

**Symptoms**: Jobs remain in "qw" state

**Solutions**:
1. Check resource availability:
   ```bash
   qhost
   qstat -g c
   ```

2. Verify queue configuration:
   ```bash
   qconf -sq <queue_name>
   ```

3. Reduce resource requests (memory, slots)

### Out of Memory Errors

**Symptoms**: Jobs die with OOM errors in logs

**Solutions**:
1. Increase memory request:
   ```bash
   qsub -l mem_free=32G ...
   ```

2. Reduce workload per task

3. Check actual memory usage:
   ```bash
   qacct -j <job_id>
   ```

### Ray Connection Timeout

**Symptoms**: `ray.init()` times out

**Solutions**:
1. Verify Ray is running on head node:
   ```bash
   # On head node
   ray status
   ```

2. Check head node logs:
   ```bash
   cat ray_head.log
   ```

3. Ensure using correct address:
   ```bash
   cat ray_cluster_info.txt
   ```

## Best Practices

1. **Resource Planning**: 
   - Head node: 4+ cores, 16+ GB RAM
   - Workers: 2+ cores, 8+ GB RAM
   - Adjust based on workload

2. **Network Configuration**:
   - Ensure low latency between nodes
   - Use high-speed interconnect if available

3. **Job Arrays**:
   - Use SGE job arrays for launching multiple identical workers
   - More efficient than individual job submissions

4. **Cleanup**:
   - Always shutdown clusters when done
   - Use `stop_ray_cluster.sh` or Ctrl+C in Go app

5. **Logging**:
   - Monitor logs for errors
   - Keep logs for debugging
   - Use unique log names for each cluster

6. **Testing**:
   - Start with small clusters (1-2 workers)
   - Scale up after verifying connectivity
   - Test with simple workloads first

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    SGE Cluster                           │
│                                                          │
│  ┌──────────────┐                                       │
│  │  SGE Qmaster │  (Job Scheduler)                      │
│  └──────┬───────┘                                       │
│         │                                                │
│         │ Submits Jobs                                  │
│         │                                                │
│    ┌────▼────────────────────────────────┐             │
│    │                                      │             │
│  ┌─▼────────────┐         ┌──────────────▼──┐         │
│  │ Exec Host 1  │         │ Exec Host 2      │         │
│  │              │         │                  │         │
│  │ ┌──────────┐ │         │ ┌──────────────┐│         │
│  │ │ Ray Head │◄├─────────┼─┤ Ray Worker 1 ││         │
│  │ │  (qsub)  │ │  6379   │ │   (qsub)     ││         │
│  │ └──────────┘ │         │ └──────────────┘│         │
│  │              │         │ ┌──────────────┐│         │
│  │              │◄────────┼─┤ Ray Worker 2 ││         │
│  │              │         │ │   (qsub)     ││         │
│  └──────────────┘         │ └──────────────┘│         │
│                           └──────────────────┘         │
│                                                         │
└─────────────────────────────────────────────────────────┘

User Application (Python/Go)
      │
      │ ray.init()
      ▼
   Ray Head ─────► Ray Workers
   (Port 6379)
```

## Integration with Existing Workflows

### Submitting Ray Jobs via SGE

You can wrap Ray applications in SGE jobs:

```bash
#!/bin/bash
#$ -N my-ray-job
#$ -cwd
#$ -j y

# Connect to existing Ray cluster
export RAY_ADDRESS="<head_host>:6379"

# Run your application
python my_ray_app.py
```

### Auto-scaling (Advanced)

For dynamic worker scaling based on load, you can:
1. Monitor Ray metrics
2. Use `qsub` to add workers
3. Use `qdel` to remove idle workers

Example monitoring script:
```bash
# Check Ray load
ray status --address=$RAY_ADDRESS

# Add workers if needed
if [ $LOAD -gt 80 ]; then
  qsub -t 1-5 ... ray start --address=$RAY_ADDRESS
fi
```

## Further Reading

- [Complete Integration Guide](../../docs/ray-integration.md)
- [Ray Documentation](https://docs.ray.io/)
- [SGE/Grid Engine](https://github.com/hpc-gridware/clusterscheduler)
- [go-clusterscheduler API](../../README.md)

## Support

For issues or questions:
- SGE Integration: Open an issue in this repository
- Ray Core: [Ray GitHub](https://github.com/ray-project/ray)
- Grid Engine: [Gridware Cluster Scheduler](https://github.com/hpc-gridware/clusterscheduler)
