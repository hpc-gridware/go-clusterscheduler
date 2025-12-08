#!/usr/bin/env python3
"""
Example Ray application to run on SGE cluster

This example demonstrates how to connect to a Ray cluster
running on SGE and execute distributed tasks.

Usage:
  1. Start Ray cluster with start_ray_cluster.sh
  2. Get the Ray address from ray_cluster_info.txt
  3. Run: python example_ray_job.py --address <head_host>:6379
"""

import argparse
import time
import ray
import numpy as np


@ray.remote
def compute_pi_sample(n_samples):
    """
    Monte Carlo estimation of Pi using random sampling.
    Each task generates n_samples random points.
    """
    inside_circle = 0
    for _ in range(n_samples):
        x, y = np.random.random(), np.random.random()
        if x*x + y*y <= 1:
            inside_circle += 1
    return inside_circle


@ray.remote
def matrix_multiply(matrix_a, matrix_b):
    """
    Simple matrix multiplication task.
    """
    return np.dot(matrix_a, matrix_b)


@ray.remote
class Counter:
    """
    Example Ray actor for stateful computation.
    """
    def __init__(self):
        self.count = 0
    
    def increment(self):
        self.count += 1
        return self.count
    
    def get_count(self):
        return self.count


def estimate_pi(n_tasks=100, samples_per_task=1000000):
    """
    Estimate Pi using distributed Monte Carlo sampling.
    """
    print(f"\nEstimating Pi using {n_tasks} tasks...")
    print(f"Each task samples {samples_per_task} points")
    
    start_time = time.time()
    
    # Submit tasks in parallel
    futures = [compute_pi_sample.remote(samples_per_task) for _ in range(n_tasks)]
    
    # Gather results
    results = ray.get(futures)
    
    # Calculate Pi estimate
    total_inside = sum(results)
    total_samples = n_tasks * samples_per_task
    pi_estimate = 4.0 * total_inside / total_samples
    
    elapsed = time.time() - start_time
    
    print(f"Pi estimate: {pi_estimate:.6f}")
    print(f"Actual Pi:   {np.pi:.6f}")
    print(f"Error:       {abs(pi_estimate - np.pi):.6f}")
    print(f"Time taken:  {elapsed:.2f} seconds")
    
    return pi_estimate


def matrix_operations():
    """
    Demonstrate distributed matrix operations.
    """
    print("\nPerforming distributed matrix operations...")
    
    # Create random matrices
    size = 1000
    n_operations = 20
    
    print(f"Matrix size: {size}x{size}")
    print(f"Number of operations: {n_operations}")
    
    start_time = time.time()
    
    # Submit matrix multiplication tasks
    futures = []
    for i in range(n_operations):
        matrix_a = np.random.rand(size, size)
        matrix_b = np.random.rand(size, size)
        futures.append(matrix_multiply.remote(matrix_a, matrix_b))
    
    # Wait for all results
    results = ray.get(futures)
    
    elapsed = time.time() - start_time
    
    print(f"Completed {n_operations} matrix multiplications")
    print(f"Time taken: {elapsed:.2f} seconds")
    print(f"Throughput: {n_operations/elapsed:.2f} operations/second")


def actor_example():
    """
    Demonstrate Ray actors for stateful computation.
    """
    print("\nDemonstrating Ray actors...")
    
    # Create multiple counter actors
    counters = [Counter.remote() for _ in range(5)]
    
    # Increment each counter multiple times
    futures = []
    for counter in counters:
        for _ in range(10):
            futures.append(counter.increment.remote())
    
    # Wait for all increments
    ray.get(futures)
    
    # Get final counts
    counts = ray.get([counter.get_count.remote() for counter in counters])
    
    print(f"Created {len(counters)} counter actors")
    print(f"Final counts: {counts}")
    print(f"Total count: {sum(counts)}")


def main():
    parser = argparse.ArgumentParser(description='Example Ray application on SGE')
    parser.add_argument('--address', type=str, default='auto',
                      help='Ray cluster address (e.g., head_host:6379)')
    parser.add_argument('--no-pi', action='store_true',
                      help='Skip Pi estimation')
    parser.add_argument('--no-matrix', action='store_true',
                      help='Skip matrix operations')
    parser.add_argument('--no-actor', action='store_true',
                      help='Skip actor example')
    
    args = parser.parse_args()
    
    print("=" * 60)
    print("Ray on SGE - Example Application")
    print("=" * 60)
    
    # Connect to Ray cluster
    print(f"\nConnecting to Ray cluster at: {args.address}")
    
    try:
        ray.init(address=args.address, ignore_reinit_error=True)
    except Exception as e:
        print(f"Error connecting to Ray cluster: {e}")
        print("\nMake sure:")
        print("1. Ray cluster is running (check ray_cluster_info.txt)")
        print("2. You're on a host that can connect to the head node")
        print("3. The address is correct (format: hostname:port)")
        return 1
    
    # Display cluster info
    print("\nCluster Information:")
    print(f"Available resources: {ray.available_resources()}")
    print(f"Cluster nodes: {len(ray.nodes())}")
    
    # Run examples
    try:
        if not args.no_pi:
            estimate_pi(n_tasks=50, samples_per_task=1000000)
        
        if not args.no_matrix:
            matrix_operations()
        
        if not args.no_actor:
            actor_example()
        
        print("\n" + "=" * 60)
        print("All tasks completed successfully!")
        print("=" * 60)
        
    except Exception as e:
        print(f"\nError during execution: {e}")
        return 1
    
    finally:
        # Note: Don't shutdown Ray if using an existing cluster
        # ray.shutdown()
        pass
    
    return 0


if __name__ == '__main__':
    exit(main())
