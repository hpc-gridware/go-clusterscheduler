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
	"flag"
	"fmt"
	"log"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
)

func main() {
	var (
		user    = flag.String("user", "", "Filter by user/owner")
		group   = flag.String("group", "", "Filter by group")
		days    = flag.Int("days", -1, "Filter jobs from last N days")
		summary = flag.Bool("summary", false, "Show summary usage instead of job details")
		jobID   = flag.Int("job", 0, "Show details for specific job ID")
	)
	flag.Parse()

	qa, err := qacct.NewCommandLineQAcct(qacct.CommandLineQAcctConfig{})
	if err != nil {
		log.Fatalf("Failed to create qacct client: %v", err)
	}

	if *summary {
		// Use new builder API for summary
		builder := qa.Summary()
		if *user != "" {
			builder = builder.Owner(*user)
		}
		if *group != "" {
			builder = builder.Group(*group)
		}
		if *days >= 0 {
			builder = builder.LastDays(*days)
		}

		usage, err := builder.Execute()
		if err != nil {
			log.Fatalf("Failed to get summary: %v", err)
		}

		fmt.Printf("Summary Usage:\n")
		fmt.Printf("Wallclock: %.2f, CPU: %.2f, Memory: %.3f\n",
			usage.WallClock, usage.CPU, usage.Memory)
	} else if *jobID > 0 {
		// Use existing API for specific job
		jobs, err := qa.ShowJobDetails([]int64{int64(*jobID)})
		if err != nil {
			log.Fatalf("Failed to get job details: %v", err)
		}

		for _, job := range jobs {
			fmt.Printf("Job %d: %s (owner: %s, queue: %s)\n",
				job.JobNumber, job.JobName, job.Owner, job.QName)
		}
	} else {
		// Use new builder API for filtered job details
		builder := qa.Jobs()
		if *user != "" {
			builder = builder.Owner(*user)
		}
		if *group != "" {
			builder = builder.Group(*group)
		}
		if *days >= 0 {
			builder = builder.LastDays(*days)
		}

		jobs, err := builder.Execute()
		if err != nil {
			log.Fatalf("Failed to get jobs: %v", err)
		}

		fmt.Printf("Found %d jobs:\n", len(jobs))
		for _, job := range jobs {
			fmt.Printf("Job %d: %s (owner: %s)\n",
				job.JobNumber, job.JobName, job.Owner)
		}
	}
}
