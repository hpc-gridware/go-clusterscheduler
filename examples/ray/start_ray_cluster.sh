#!/bin/bash
#
# Script to start a Ray cluster on SGE
# Usage: ./start_ray_cluster.sh [num_workers]
#

set -e

NUM_WORKERS=${1:-5}
RAY_PORT=6379
HEAD_MEMORY="16G"
WORKER_MEMORY="8G"

echo "Starting Ray cluster with $NUM_WORKERS workers..."

# Submit Ray head node
echo "Submitting Ray head node..."
HEAD_JOB_OUTPUT=$(qsub -b y -cwd -j y -o ray_head.log -N ray-head \
  -l mem_free=$HEAD_MEMORY \
  ray start --head --port=$RAY_PORT --dashboard-host=0.0.0.0 --block 2>&1)

# Extract job ID - try multiple patterns for different SGE versions
HEAD_JOB_ID=$(echo "$HEAD_JOB_OUTPUT" | grep -oP '(?<=Your job )\d+' || \
              echo "$HEAD_JOB_OUTPUT" | grep -oP '(?<=Your job-ID is )\d+' || \
              echo "$HEAD_JOB_OUTPUT" | grep -oP '\d+' | head -1)

if [ -z "$HEAD_JOB_ID" ]; then
  echo "Error: Could not extract job ID from qsub output:"
  echo "$HEAD_JOB_OUTPUT"
  exit 1
fi

echo "Head node submitted with job ID: $HEAD_JOB_ID"
echo "Waiting for head node to start..."

# Wait for head node to start
sleep 15

# Get head node hostname
HEAD_HOST=""
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
  HEAD_HOST=$(qstat -j $HEAD_JOB_ID 2>/dev/null | grep "exec_host" | awk -F'[@:]' '{print $2}' || echo "")
  
  if [ ! -z "$HEAD_HOST" ]; then
    echo "Head node running on: $HEAD_HOST"
    break
  fi
  
  ATTEMPT=$((ATTEMPT + 1))
  sleep 2
done

if [ -z "$HEAD_HOST" ]; then
  echo "Error: Could not determine head node hostname"
  echo "Check the job status with: qstat -j $HEAD_JOB_ID"
  exit 1
fi

# Submit Ray workers as job array
echo "Submitting $NUM_WORKERS Ray workers..."
WORKER_JOB_OUTPUT=$(qsub -b y -cwd -j y -o ray_worker.\$TASK_ID.log -N ray-worker \
  -t 1-$NUM_WORKERS \
  -l mem_free=$WORKER_MEMORY \
  ray start --address=$HEAD_HOST:$RAY_PORT --block 2>&1)

# Extract job ID - try multiple patterns for different SGE versions
WORKER_JOB_ID=$(echo "$WORKER_JOB_OUTPUT" | grep -oP '(?<=Your job-array )\d+' || \
                echo "$WORKER_JOB_OUTPUT" | grep -oP '(?<=Your job )\d+' || \
                echo "$WORKER_JOB_OUTPUT" | grep -oP '\d+' | head -1)

if [ -z "$WORKER_JOB_ID" ]; then
  echo "Warning: Could not extract worker job ID from qsub output:"
  echo "$WORKER_JOB_OUTPUT"
  echo "Workers may not have been submitted correctly."
fi

echo "Workers submitted with job array ID: $WORKER_JOB_ID"

# Save cluster information
cat > ray_cluster_info.txt <<EOF
Ray Cluster Information
=======================
Head Job ID: $HEAD_JOB_ID
Worker Job Array ID: $WORKER_JOB_ID
Head Node: $HEAD_HOST
Ray Address: $HEAD_HOST:$RAY_PORT
Dashboard: http://$HEAD_HOST:8265

To connect from Python:
  import ray
  ray.init(address='$HEAD_HOST:$RAY_PORT')

To check cluster status:
  qstat -f
  ray status --address=$HEAD_HOST:$RAY_PORT

To shutdown the cluster:
  qdel $HEAD_JOB_ID $WORKER_JOB_ID
EOF

cat ray_cluster_info.txt

echo ""
echo "Ray cluster started successfully!"
echo "Cluster information saved to: ray_cluster_info.txt"
