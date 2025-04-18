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

package main

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
)

// MPI configuration prompt text - updated based on template
var MPIConfigurationPrompt = "# MPI Configuration for Cluster Scheduler\n\n" +
	"## Description\n" +
	"Configure MPI parallel environments for Cluster Scheduler (Open Cluster Scheduler or Gridware Cluster Scheduler) to enable efficient distributed processing across multiple nodes.\n\n" +
	"## Parallel Environment Integration Types\n\n" +
	"### Tight Integration\n" +
	"- Parallel tasks start under scheduler control\n" +
	"- Resource accounting and limits are enforced for all tasks\n" +
	"- Scheduler ensures all tasks terminate correctly\n" +
	"- Tasks are started using `qrsh -inherit <hostname> <cmd> <args>`\n" +
	"- Most MPI implementations support tight integration\n" +
	"- Recommended for production environments\n\n" +
	"### Loose Integration\n" +
	"- Only master task (job script) is controlled by scheduler\n" +
	"- Slave tasks are not accounted and resource limits not enforced\n" +
	"- Application must handle task startup and termination\n" +
	"- Typically uses SSH for remote task execution\n" +
	"- Less preferred but sometimes necessary for certain applications\n\n" +
	"## Implementation Steps\n\n" +
	"### 1. Check for Existing PE Configurations\n" +
	"First, check if the desired PE already exists:\n\n" +
	"```\n" +
	"# Using command-line tools\n" +
	"qconf -spl  # Lists all configured parallel environments\n\n" +
	"# Using API/programmatic approach\n" +
	"get_cluster_configuration()  # Returns complete cluster config including PEs\n" +
	"```\n\n" +
	"Look for your desired MPI implementation in the parallel_environments section of the configuration.\n\n" +
	"### 2. Select the Appropriate PE Template\n" +
	"Each MPI implementation has its specific configuration:\n\n" +
	"- **Intel MPI**: `$SGE_ROOT/mpi/intel-mpi.pe`\n" +
	"- **MPICH**: `$SGE_ROOT/mpi/mpich/mpich.pe`\n" +
	"- **MVAPICH**: `$SGE_ROOT/mpi/mvapich/mvapich.pe`\n" +
	"- **OpenMPI**: `$SGE_ROOT/mpi/openmpi/openmpi.pe`\n" +
	"- **SSH Wrapper**: `$SGE_ROOT/mpi/ssh-wrapper/ssh-wrapper.pe`\n\n" +
	"### 3. Register the PE Configuration\n" +
	"Install the parallel environment:\n\n" +
	"```\n" +
	"qconf -Ap $SGE_ROOT/mpi/<implementation>/<implementation>.pe\n" +
	"```\n\n" +
	"### 4. Add PE to Target Queue\n" +
	"Make the PE available in your desired queue:\n\n" +
	"```\n" +
	"qconf -aattr queue pe_list <pe_name> <queue_name>\n" +
	"```\n\n" +
	"### 5. Submit Jobs Using the PE\n" +
	"When submitting jobs, specify the PE and number of slots:\n\n" +
	"```\n" +
	"qsub -pe <pe_name> <slot_count> job_script.sh\n" +
	"```\n\n" +
	"## PE Configuration Details\n\n" +
	"### Intel MPI\n" +
	"A high-performance, low-latency MPI implementation optimized for Intel architectures:\n" +
	"```\n" +
	"pe_name              intel-mpi.pe\n" +
	"slots                999\n" +
	"user_lists           NONE\n" +
	"xuser_lists          NONE\n" +
	"start_proc_args      $sge_root/mpi/intel-mpi/pe_start_proc.sh\n" +
	"stop_proc_args       NONE\n" +
	"allocation_rule      $round_robin\n" +
	"control_slaves       TRUE\n" +
	"job_is_first_task    TRUE\n" +
	"urgency_slots        min\n" +
	"accounting_summary   FALSE\n" +
	"ign_sreq_on_mhost    FALSE\n" +
	"master_forks_slaves  TRUE\n" +
	"daemon_forks_slaves  TRUE\n" +
	"```\n\n" +
	"Intel MPI environment variable configuration:\n" +
	"```\n" +
	"I_MPI_HYDRA_BOOTSTRAP=sge\n" +
	"```\n\n" +
	"### OpenMPI\n" +
	"A collaborative, modular, and extensible implementation:\n" +
	"- Supports wide range of interconnects\n" +
	"- Highly customizable for specific needs\n" +
	"- Good choice for heterogeneous environments\n\n" +
	"## Testing and Verification\n" +
	"After configuration, test with a simple MPI job:\n" +
	"```\n" +
	"# Example job script for Intel MPI\n" +
	"#!/bin/bash\n" +
	"#$ -S /bin/bash\n" +
	"#$ -N mpi_test\n" +
	"#$ -pe intel-mpi.pe 16\n\n" +
	"mpirun -np $NSLOTS ./my_mpi_application\n" +
	"```\n\n" +
	"## Troubleshooting Tips\n" +
	"- Verify PE is properly registered: `qconf -sp <pe_name>`\n" +
	"- Check queue has PE enabled: `qconf -sq <queue_name>`\n" +
	"- Examine scheduler logs for configuration issues\n" +
	"- For Intel MPI issues, verify I_MPI_HYDRA_BOOTSTRAP is set to \"sge\"\n" +
	"- If using tight integration, ensure qrsh is working properly\n\n" +
	"## Best Practices\n" +
	"- Use tight integration when possible for better resource control\n" +
	"- Select appropriate MPI implementation for your network (InfiniBand→MVAPICH, etc.)\n" +
	"- Consider allocation rules based on your job communication patterns\n" +
	"- Test with small jobs before scaling to production workloads"

// Scheduling optimization prompt text
var SchedulingOptimizationPrompt = "# Scheduling Optimization Guidelines for the Gridware Cluster Scheduler\n\n" +
	"Optimizing job scheduling in a cluster environment requires understanding the scheduling policies, resource availability, and job requirements. This guide provides best practices for maximizing throughput and minimizing wait time.\n\n" +
	"## Resource Request Optimization\n\n" +
	"When submitting jobs, accurate resource specifications are crucial:\n\n" +
	"1. Request only the resources your job actually needs\n" +
	"2. Specify memory requirements explicitly with `-l h_vmem=VALUE`\n" +
	"3. Define realistic runtime limits with `-l h_rt=HH:MM:SS`\n" +
	"4. Use array jobs for many similar tasks with `-t START-END:STEP`\n\n" +
	"## Queue Selection Strategies\n\n" +
	"Different queues offer various trade-offs between priority, resources, and limits:\n\n" +
	"- **short**: For quick jobs (< 1 hour), higher priority but strict limits\n" +
	"- **long**: For extended runs, lower priority but relaxed limits\n" +
	"- **high_mem**: For memory-intensive workloads\n" +
	"- **gpu**: For GPU-accelerated computations\n\n" +
	"Select the most appropriate queue based on your job's characteristics.\n\n" +
	"## Advanced Scheduling Techniques\n\n" +
	"Improve scheduling efficiency with these advanced techniques:\n\n" +
	"### Job Dependencies\n\n" +
	"Use `-hold_jid JOB_ID` to create job dependency chains:\n" +
	"```bash\n" +
	"job1_id=$(qsub first_job.sh | cut -d' ' -f3)\n" +
	"qsub -hold_jid $job1_id second_job.sh\n" +
	"```\n\n" +
	"### Resource Reservations\n\n" +
	"For critical workloads, consider advance reservations:\n" +
	"```bash\n" +
	"qrsub -a START_TIME -d DURATION -l resource=value\n" +
	"```\n\n" +
	"### Job Priorities\n\n" +
	"Set job priorities based on importance:\n" +
	"```bash\n" +
	"qsub -p -500 high_priority_job.sh  # Higher priority (lower value)\n" +
	"qsub -p 500 low_priority_job.sh    # Lower priority (higher value)\n" +
	"```\n\n" +
	"## Monitoring and Adaptation\n\n" +
	"Regularly monitor cluster utilization and adapt your strategy:\n\n" +
	"1. Use `qstat -g c` to view queue and cluster load\n" +
	"2. Check `qacct -j JOB_ID` for completed job statistics\n" +
	"3. Adjust resource requests based on historical usage\n\n" +
	"## Scheduling Policies\n\n" +
	"Understand these key scheduling policies that affect job placement:\n\n" +
	"- **Backfilling**: Small jobs may run before larger queued jobs if they fit in available resources\n" +
	"- **Fair-share**: Historical usage affects job priority\n" +
	"- **Resource quotas**: Limits on concurrent jobs or resources per user/group\n\n" +
	"For specific policy details and resource quotas in your environment, consult your system administrator."

// RegisterPrompts adds all prompts to the server
func RegisterPrompts(s *SchedulerServer) error {
	// Register MPI Configuration prompt
	s.server.AddPrompt(mcp.NewPrompt("mpi_configuration",
		mcp.WithPromptDescription("Provides comprehensive guidance on configuring MPI parallel environments for the Gridware Cluster Scheduler, including integration types, implementation steps, and best practices."),
		mcp.WithArgument("mpi_implementation",
			mcp.ArgumentDescription("MPI implementation to use (intel-mpi, mpich, mvapich, openmpi, ssh-wrapper)"),
		),
		mcp.WithArgument("integration_type",
			mcp.ArgumentDescription("Integration approach (loose or tight)"),
		),
		mcp.WithArgument("queue_name",
			mcp.ArgumentDescription("Target queue to configure"),
		),
		mcp.WithArgument("network_type",
			mcp.ArgumentDescription("Network interconnect used (ethernet, infiniband, omni-path)"),
		),
		mcp.WithArgument("check_existing",
			mcp.ArgumentDescription("Whether to check for existing PE configurations first (true/false)"),
		),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		log.Printf("Handling MPI configuration prompt")
		return &mcp.GetPromptResult{
			Description: "MPI configuration",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleAssistant,
					Content: mcp.TextContent{Text: MPIConfigurationPrompt},
				},
			},
		}, nil
	})

	// Register Scheduling Optimization prompt
	s.server.AddPrompt(mcp.NewPrompt("scheduling_optimization",
		mcp.WithPromptDescription("Offers best practices for optimizing job scheduling, including resource request strategies, queue selection guidelines, advanced techniques like job dependencies and reservations, monitoring approaches, and information about scheduling policies."),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		log.Printf("Handling scheduling optimization prompt")
		return &mcp.GetPromptResult{
			Description: "Scheduling optimization",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleAssistant,
					Content: mcp.TextContent{Text: SchedulingOptimizationPrompt},
				},
			},
		}, nil
	})

	return nil
}
