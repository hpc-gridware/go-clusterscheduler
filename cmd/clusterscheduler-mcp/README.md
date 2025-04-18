# Open Cluster Scheduler & Gridware Cluster Scheduler MCP Server Integration

This repository contains a sample Model Context Protocol (MCP) server integration for both the Open Cluster Scheduler (OCS) and Gridware Cluster Scheduler (GCS). This README provides detailed instructions on building, running, and using the example integration. It also includes guidance on how you can integrate this setup with tools such as Claude and Cursor for cluster configuration, job status analysis, and more.

**This is primary for research and customization for your needs.**

## Overview

This example MCP server (“describe-mcp”) demonstrates how to integrate with the Open Cluster Scheduler (OCS) and Gridware Cluster Scheduler (GCS). By leveraging MCP, you can easily fetch configuration data, view job status, retrieve job accounting details, and perform write functions such as job submission or cluster configuration updates.

When properly configured, tools like Claude, Cursor, or other MCP clients can seamlessly query the cluster configuration, run commands like `qstat` or `qacct`, and view or modify cluster objects through this MCP integration.  

## Build and Installation

1. Clone or download this repository.  
2. Open a terminal in the repository directory.  
3. Build the binary:
   ```bash
   go build
   ```
   
4. Run the binary with all tools enabled (dangerous, only for testing,
   like when running a container with a simulated cluster - "make simulate"):
   ```bash
   export WITH_JOB_SUBMISSION_ACCESS="true"
   export WITH_WRITE_ACCESS="true"
   ./clusterscheduler-mcp
   ```
   
   To run in a restricted (read-only) mode that disables any “write” functionality 
   (such as job submission or config changes):
   ```bash
   ./clusterscheduler-mcp
   ```

## Available MCP Tools

Within this repository, you will find multiple MCP tools that can be called by your MCP clients (like Claude or Cursor). Each tool communicates with `clusterscheduler-mcp` over SSE (Server-Sent Events) and invokes the respective OCS/GCS commands:

1. **get_cluster_configuration**  
   • Fetches the complete cluster configuration in JSON format, including hosts, queues, users, projects, and resource settings.  
   • Useful for quickly retrieving or backing up the entire configuration.

2. **job_details**  
   • Retrieves detailed accounting information about finished jobs in a structured format.  
   • Specify job IDs or leave it blank to fetch data about all finished jobs.

3. **diagnose_pending_job**
   • Retrieves detailed job information with qstat -j <jobid>  
   • Checks the output of qalter -w p to get more details.

4. **qacct**  
   • Queries historical job accounting data, including resource usage, execution details, and job outcomes.

5. **qstat**  
   • Fetches real-time information about running and pending jobs, as well as queue states and scheduling details.  
   • You can use options like `-j <job_id>` to see additional granularity.

6. **qsub_help**  
   • Retrieves a thorough reference for the `qsub` command, listing parameters, examples, and usage notes.  
   • Helpful for crafting precise job submission calls.

7. **set_cluster_configuration**  
   • Applies a new cluster configuration to the system, supplied as JSON.  
   • !DANGEROUS! Only for container based test clusters, for testing - disable in code if not needed or by env variable.

8. **submit_job**  
   • Submits a job to the cluster using SGE-compatible command line parameters.  
   • Allows direct control over resource requests, scheduling policies, environment settings, and job array usage.

## Example Usage

Below are some illustrative queries you might pose to your MCP-based tools (e.g., Claude) to interact with the cluster:

1. **Show me a summary of all running jobs as a table.**  
   • Internally, your client might call the `qstat` tool and parse the results into a table.

2. **Provide a high-level overview of the cluster configuration. How many jobs can run concurrently in the cluster?**  
   • Calls `get_cluster_configuration` and aggregates relevant capacity or slot data to inform concurrency limits.

3. **Submit a job array with 100 tasks executing “sleep 100”.**  
   • Leverages `submit_job` with an SGE array argument like `-t 1-100 -b y sleep 100`.

4. **Why is my job X not running?**
   • Leverages different commands.

## MCP Integration Configuration

This repository shows how you might configure Claude (or a similar service) to connect to the MCP server. Below is a sample JSON snippet for using `npx mcp-remote`, adapting it to your environment:

```json
{
    "mcpServers": {
        "gridware": {
            "command": "npx",
            "args": [
                "mcp-remote",
                "http://localhost:8888/sse"
            ]
        }
    }
}
```

When configured correctly, Claude (or another client) will be able to send queries to the `gridware` MCP server (i.e., this `describe-mcp` process) via the SSE endpoint. Make sure the container or process is exposing the relevant port (e.g., `8888`) and that it has the required privileges to run OCS/GCS commands (`qconf`, `qstat`, `qacct`, etc.).

## Security and Privileges

Be mindful of the following:  
• The MCP server requires sufficient privileges to run the cluster management
commands. Test this only in a temporary test installation,
like using "make simulate" or "make run" in go-clusterscheduler project
to generate a local test cluster which runs in a container.

---

## Contributing

We welcome PRs and contributions for improvements or new features. Feel free to open an issue for questions, feedback, or discussion.

---

## License

You can use or modify this integration for your unique setup. Check the
repository’s LICENSE file for specific terms.

---

## Contact

If you have any questions or require further assistance, reach out via the
issues section in this repository. We’re happy to help you get started or
troubleshoot any issues along the way.

---

**Thank you for exploring this MCP integration for OCS & GCS.**