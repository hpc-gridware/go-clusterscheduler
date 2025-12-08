#!/bin/bash
#
# Script to stop a Ray cluster running on SGE
# Usage: ./stop_ray_cluster.sh
#

set -e

if [ ! -f ray_cluster_info.txt ]; then
  echo "Error: ray_cluster_info.txt not found"
  echo "Cannot determine which jobs to delete"
  echo ""
  echo "To manually delete jobs, use:"
  echo "  qstat -u $USER  # to see your jobs"
  echo "  qdel <job_id>   # to delete specific jobs"
  exit 1
fi

# Extract job IDs from cluster info
HEAD_JOB_ID=$(grep "Head Job ID:" ray_cluster_info.txt | awk '{print $NF}')
WORKER_JOB_ID=$(grep "Worker Job Array ID:" ray_cluster_info.txt | awk '{print $NF}')

echo "Shutting down Ray cluster..."
echo "Head Job ID: $HEAD_JOB_ID"
echo "Worker Job Array ID: $WORKER_JOB_ID"

# Delete worker jobs
if [ ! -z "$WORKER_JOB_ID" ]; then
  echo "Deleting worker jobs..."
  qdel $WORKER_JOB_ID 2>/dev/null || echo "Worker jobs already deleted or not found"
fi

# Delete head job
if [ ! -z "$HEAD_JOB_ID" ]; then
  echo "Deleting head job..."
  qdel $HEAD_JOB_ID 2>/dev/null || echo "Head job already deleted or not found"
fi

echo "Ray cluster shutdown complete"
echo ""
echo "Cluster information archived to: ray_cluster_info.txt.bak"
mv ray_cluster_info.txt ray_cluster_info.txt.bak
