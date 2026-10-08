/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
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

// advancereservation shows the life cycle of an advance reservation (AR):
// verify it, create it, inspect it, run a job in it and delete it.
// The calling user must be a manager or in the "arusers" access list.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	qrdel "github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/v9.0"
	qrstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/v9.0"
	qrsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
)

func exitOnError(what string, err error) {
	if err != nil {
		fmt.Printf("error %s: %v\n", what, err)
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()

	sub, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{})
	exitOnError("creating qrsub client", err)
	stat, err := qrstat.NewCommandLineQRStat(qrstat.CommandLineQRStatConfig{})
	exitOnError("creating qrstat client", err)
	del, err := qrdel.NewCommandLineQRDel(qrdel.CommandLineQRDelConfig{})
	exitOnError("creating qrdel client", err)

	// The same AR request, first verified, then created.
	request := qrsub.NewReservationBuilder(sub).
		Name("example").
		Start(time.Now().Add(time.Minute)).
		Duration(10*time.Minute).
		Resource("h_rt", "600").
		MailOptions("n")

	found, report, err := request.Verify(ctx)
	exitOnError("verifying the advance reservation", err)
	if !found {
		fmt.Printf("the advance reservation cannot be granted:\n%s", report)
		os.Exit(1)
	}

	arID, _, err := request.Submit(ctx)
	exitOnError("creating the advance reservation", err)
	fmt.Printf("created advance reservation %d\n", arID)

	reservations, err := stat.GetReservations(ctx, arID)
	exitOnError("getting the advance reservation", err)
	if len(reservations) != 1 {
		exitOnError("getting the advance reservation", fmt.Errorf("expected 1 advance reservation, got %d", len(reservations)))
	}
	details, err := json.MarshalIndent(reservations[0], "", "  ")
	exitOnError("formatting the advance reservation", err)
	fmt.Printf("advance reservation details: %s\n", details)

	// Jobs request the AR with qsub -ar. The job starts with the AR.
	qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
	exitOnError("creating qsub client", err)
	jobID, _, err := qsub.NewJobBuilder(qs, "sleep", "10").
		Binary().
		AdvanceReservation(strconv.FormatInt(arID, 10)).
		Resource("h_rt", "60").
		Submit(ctx)
	exitOnError("submitting a job into the advance reservation", err)
	fmt.Printf("submitted job %d into advance reservation %d\n", jobID, arID)

	// Deleting the AR also deletes the jobs bound to it.
	output, err := del.DeleteReservations(ctx, arID)
	exitOnError("deleting the advance reservation", err)
	fmt.Print(output)
}
