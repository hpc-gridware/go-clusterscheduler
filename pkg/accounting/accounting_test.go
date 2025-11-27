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

package accounting_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/accounting"
)

var _ = Describe("Accounting", func() {

	Context("GetUsageFilePath", func() {

		It("should return the usage file path", func() {
			os.Unsetenv("SGE_JOB_SPOOL_DIR")
			usageFilePath, err := accounting.GetUsageFilePath()
			Expect(err).To(HaveOccurred())
			Expect(usageFilePath).To(Equal(""))
		})

		It("should return the usage file path", func() {
			os.Setenv("SGE_JOB_SPOOL_DIR", "/var/spool/gridengine/job_spool")
			usageFilePath, err := accounting.GetUsageFilePath()
			Expect(err).NotTo(HaveOccurred())
			Expect(usageFilePath).To(Equal("/var/spool/gridengine/job_spool/usage"))
		})

	})

	Context("AppendToAccounting", func() {

		var tempDir string
		var usageFilePath string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "accounting-test-")
			Expect(err).NotTo(HaveOccurred())
			usageFilePath = filepath.Join(tempDir, "usage")
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("should create and write to a new usage file", func() {
			records := []accounting.Record{
				{AccountingKey: "wait_status", AccountingValue: int(1)},
				{AccountingKey: "exit_status", AccountingValue: int64(0)},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("wait_status=1\nexit_status=0\n"))
		})

		It("should append to an existing usage file", func() {
			// First write
			records1 := []accounting.Record{
				{AccountingKey: "wait_status", AccountingValue: int64(1)},
			}
			err := accounting.AppendToAccounting(usageFilePath, records1)
			Expect(err).NotTo(HaveOccurred())

			// Second write (append)
			records2 := []accounting.Record{
				{AccountingKey: "exit_status", AccountingValue: 0},
			}
			err = accounting.AppendToAccounting(usageFilePath, records2)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("wait_status=1\nexit_status=0\n"))
		})

		It("should format integer values as integers", func() {
			records := []accounting.Record{
				{AccountingKey: "wait_status", AccountingValue: int(1)},
				{AccountingKey: "exit_status", AccountingValue: int64(0)},
				{AccountingKey: "signal", AccountingValue: int(0)},
				{AccountingKey: "ru_wallclock", AccountingValue: int64(1)},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			expected := "wait_status=1\nexit_status=0\nsignal=0\nru_wallclock=1\n"
			Expect(string(content)).To(Equal(expected))
		})

		It("should format float values with proper precision", func() {
			records := []accounting.Record{
				{AccountingKey: "ru_utime", AccountingValue: 0.142177},
				{AccountingKey: "ru_stime", AccountingValue: 0.146392},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			expected := "ru_utime=0.142177\nru_stime=0.146392\n"
			Expect(string(content)).To(Equal(expected))
		})

		It("should handle mixed integer and float values", func() {
			records := []accounting.Record{
				{AccountingKey: "wait_status", AccountingValue: int(1)},
				{AccountingKey: "exit_status", AccountingValue: int64(0)},
				{AccountingKey: "signal", AccountingValue: int(0)},
				{AccountingKey: "start_time", AccountingValue: int64(1764257363873083)},
				{AccountingKey: "end_time", AccountingValue: int64(1764257365184384)},
				{AccountingKey: "ru_wallclock", AccountingValue: int(1)},
				{AccountingKey: "ru_utime", AccountingValue: 0.142177},
				{AccountingKey: "ru_stime", AccountingValue: 0.146392},
				{AccountingKey: "ru_maxrss", AccountingValue: int64(11156)},
				{AccountingKey: "ru_ixrss", AccountingValue: int(0)},
				{AccountingKey: "ru_idrss", AccountingValue: int64(0)},
				{AccountingKey: "ru_isrss", AccountingValue: int(0)},
				{AccountingKey: "ru_minflt", AccountingValue: int64(22916)},
				{AccountingKey: "ru_majflt", AccountingValue: int(0)},
				{AccountingKey: "ru_nswap", AccountingValue: int64(0)},
				{AccountingKey: "ru_inblock", AccountingValue: int(0)},
				{AccountingKey: "ru_oublock", AccountingValue: int64(3)},
				{AccountingKey: "ru_msgsnd", AccountingValue: int(0)},
				{AccountingKey: "ru_msgrcv", AccountingValue: int64(0)},
				{AccountingKey: "ru_nsignals", AccountingValue: int(0)},
				{AccountingKey: "ru_nvcsw", AccountingValue: int64(3175)},
				{AccountingKey: "ru_nivcsw", AccountingValue: int(19)},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())

			expected := `wait_status=1
exit_status=0
signal=0
start_time=1764257363873083
end_time=1764257365184384
ru_wallclock=1
ru_utime=0.142177
ru_stime=0.146392
ru_maxrss=11156
ru_ixrss=0
ru_idrss=0
ru_isrss=0
ru_minflt=22916
ru_majflt=0
ru_nswap=0
ru_inblock=0
ru_oublock=3
ru_msgsnd=0
ru_msgrcv=0
ru_nsignals=0
ru_nvcsw=3175
ru_nivcsw=19
`
			Expect(string(content)).To(Equal(expected))
		})

		It("should handle empty records", func() {
			records := []accounting.Record{}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(""))
		})

		It("should return an error for unsupported types", func() {
			records := []accounting.Record{
				{AccountingKey: "invalid", AccountingValue: "string_value"},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported type"))

			// File should be created but empty since no valid records
			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(""))
		})

		It("should write valid records even when some have unsupported types", func() {
			records := []accounting.Record{
				{AccountingKey: "valid1", AccountingValue: int(42)},
				{AccountingKey: "invalid", AccountingValue: "string_value"},
				{AccountingKey: "valid2", AccountingValue: 3.14},
				{AccountingKey: "invalid2", AccountingValue: true},
				{AccountingKey: "valid3", AccountingValue: int64(100)},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			// Should return error about unsupported types
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported type"))

			// But valid records should still be written
			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("valid1=42\nvalid2=3.14\nvalid3=100\n"))
		})

		It("should handle float64 values that are whole numbers", func() {
			records := []accounting.Record{
				{AccountingKey: "whole_float", AccountingValue: float64(42)},
				{AccountingKey: "actual_float", AccountingValue: float64(42.5)},
			}

			err := accounting.AppendToAccounting(usageFilePath, records)
			Expect(err).NotTo(HaveOccurred())

			content, err := os.ReadFile(usageFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("whole_float=42\nactual_float=42.5\n"))
		})

	})
})
