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
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Record is a key-value pair representing an accounting record.
// AccountingValue can be int, int64, or float64.
type Record struct {
	AccountingKey   string
	AccountingValue any
}

// AppendToAccounting appends accounting records to the usage file so
// that it gets send to the execution daemon. Typically sgeadmin user
// (Cluster Scheduler install user).
// Hence you need to prefix your epilog script with sgeadmin@/path/to/epilog.
// Valid records are written even if some records have unsupported types.
func AppendToAccounting(usageFilePath string, records []Record) error {
	f, err := os.OpenFile(usageFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var nl string
	var errs []error

	for _, record := range records {
		var line string
		// Use type switch to format based on the actual type
		switch v := record.AccountingValue.(type) {
		case int:
			line = fmt.Sprintf("%s=%d\n", record.AccountingKey, v)
		case int64:
			line = fmt.Sprintf("%s=%d\n", record.AccountingKey, v)
		case float64:
			// Format as integer if the value is a whole number
			if v == float64(int64(v)) {
				line = fmt.Sprintf("%s=%d\n", record.AccountingKey, int64(v))
			} else {
				// Format as float with appropriate precision
				line = fmt.Sprintf("%s=%g\n", record.AccountingKey, v)
			}
		default:
			errs = append(errs, fmt.Errorf("unsupported type %T for key %s", v, record.AccountingKey))
			continue // Skip this record but continue processing others
		}
		nl += line
	}

	// Write all valid records
	if nl != "" {
		_, err = f.WriteString(nl)
		if err != nil {
			return err
		}
	}

	// Return error if any unsupported types were encountered
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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
