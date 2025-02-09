package main

import (
	"fmt"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/accounting"
	"golang.org/x/exp/rand"
)

// This is an example epilog which adds arbitrary accounting records for
// jobs to the system. In order to use the example additional accounting records
// for the "test" namespace - a namespace is just a subsection in the JSON
// accounting file - (with usage values with the "tst prefix) needs to
// be enabled:
//
// qconf -mconf
// ..
// reporting_params ...  usage_patterns=test:tst*
//
// Compile this example (go build) and copy the binary to your cluster
// scheduler installation directory.
//
// Then configure your queue epilog script with sgeadmin@/path/to/flexibleaccounting
// Ensure that the binary is executable. The sgeadmin is the correct user
// to use for this (check the owner of $SGE_ROOT).
//
// When configured correctly the following command should output the
// accounting records:
//
// qacct -j <job_id>
//
// You should see the two tst_random* records in the qacct after your job
// has finished.
//
// tst_random1                        351.000
// tst_random2                        121.000
func main() {
	usageFilePath, err := accounting.GetUsageFilePath()
	if err != nil {
		fmt.Printf("Failed to get usage file path: %v\n", err)
		return
	}
	err = accounting.AppendToAccounting(usageFilePath, []accounting.Record{
		{
			AccountingKey:   "tst_random1",
			AccountingValue: rand.Intn(1000),
		},
		{
			AccountingKey:   "tst_random2",
			AccountingValue: rand.Intn(1000),
		},
	})
	if err != nil {
		fmt.Printf("Failed to append to accounting: %v\n", err)
	}
}
