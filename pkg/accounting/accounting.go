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

package accounting

import (
	"fmt"
	"os"
	"path/filepath"
)

// Record is a key-value pair representing an accounting record.
type Record struct {
	AccountingKey   string
	AccountingValue int
}

// AppendToAccounting appends accounting records to the usage file so
// that it gets send to the execution daemon. Typically sgeadmin user
// (Cluster Scheduler install user).
// Hence you need to prefix your epilog script with sgeadmin@/path/to/epilog.
func AppendToAccounting(usageFilePath string, records []Record) error {
	f, err := os.OpenFile(usageFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var nl string

	for _, record := range records {
		nl += fmt.Sprintf("%s=%d\n",
			record.AccountingKey,
			record.AccountingValue)
	}

	_, err = f.WriteString(nl)
	return err
}

// GetUsageFilePath returns the path to the job usage file in the
// job spool directory.
func GetUsageFilePath() (string, error) {
	jobSpoolDir := os.Getenv("SGE_JOB_SPOOL_DIR")
	if jobSpoolDir == "" {
		return "", fmt.Errorf("SGE_JOB_SPOOL_DIR is not set")
	}
	return filepath.Join(jobSpoolDir, "usage"), nil
}
