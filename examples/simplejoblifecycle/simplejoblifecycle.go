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
	"encoding/json"
	"fmt"
	"os"
	"time"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
)

func main() {

	ctx := context.Background()

	qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
	if err != nil {
		fmt.Printf("error creating qsub client: %v\n", err)
		os.Exit(1)
	}

	// watching the job with qstat
	qstat, err := qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
	if err != nil {
		fmt.Printf("error creating qstat client: %v\n", err)
		os.Exit(1)
	}

	qacct, err := qacct.NewCommandLineQAcct(qacct.CommandLineQAcctConfig{})
	if err != nil {
		fmt.Printf("error creating qacct client: %v\n", err)
		os.Exit(1)
	}

	jobId, _, err := qs.Submit(ctx, qsub.JobOptions{
		Command:     "sleep",
		CommandArgs: []string{"10"},
		Binary:      qsub.ToPtr(true),
	})
	if err != nil {
		fmt.Printf("error submitting job: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("submitted job with id %d\n", jobId)

	jobInfoCh, err := qstat.WatchJobs(ctx, []int64{jobId})
	if err != nil {
		fmt.Printf("error watching job: %v\n", err)
		os.Exit(1)
	}

	// watch with qstat until the job is done
	for jobInfo := range jobInfoCh {
		// nicely formatted JSON output
		json, err := json.MarshalIndent(jobInfo, "", "  ")
		if err != nil {
			fmt.Printf("error marshalling job info: %v\n", err)
		} else {
			fmt.Printf("job info: %s\n", string(json))
		}
	}

	// when the job is done, it is out-of-scope of the scheduler
	fmt.Printf("job left the system\n")

	// it may take a while until the accounting information is available
	<-time.After(2 * time.Second)

	// get accounting information with qacct
	jobAcct, err := qacct.ShowJobDetails([]int64{jobId})
	if err != nil {
		fmt.Printf("error getting accounting information: %v\n", err)
		os.Exit(1)
	}

	// nicely formatted JSON output
	json, err := json.MarshalIndent(jobAcct, "", "  ")
	if err != nil {
		fmt.Printf("error marshalling accounting information: %v\n", err)
	} else {
		fmt.Printf("job accounting information: %s\n", string(json))
	}
}
