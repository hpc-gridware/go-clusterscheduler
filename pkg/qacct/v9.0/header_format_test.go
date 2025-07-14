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

package qacct_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
)

var _ = Describe("Header format parsing", func() {

	Context("All discovered qacct header formats", func() {

		It("should parse all known header variations", func() {
			testCases := []struct {
				name   string
				input  string
				hasData bool
			}{
				{
					"Owner only format",
					`OWNER     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
======================================================================================================================
root            138         2.000         1.517         3.517              0.340              0.001              0.000`,
					true,
				},
				{
					"Group + Owner format", 
					`GROUP OWNER     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
============================================================================================================================
root  root            138         2.000         1.517         3.517              0.340              0.001              0.000`,
					true,
				},
				{
					"Owner + Department format",
					`OWNER DEPARTMENT            WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
========================================================================================================================================
root  defaultdepartment           138         2.000         1.517         3.517              0.340              0.001              0.000`,
					true,
				},
				{
					"Queue + Host format",
					`HOST   CLUSTER QUEUE     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
=====================================================================================================================================
master all.q                   138         2.000         1.517         3.517              0.340              0.001              0.000`,
					true,
				},
				{
					"Owner + Queue + Host format", 
					`HOST   CLUSTER QUEUE OWNER     WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
===========================================================================================================================================
master all.q         root            138         2.000         1.517         3.517              0.340              0.001              0.000`,
					true,
				},
				{
					"Owner + Slots format",
					`OWNER SLOTS    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
===========================================================================================================================
root      1          138         2.000         1.517         3.517              0.340              0.001              0.000`,
					true,
				},
				{
					"PE format (empty)",
					`PE              WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
============================================================================================================================`,
					false,
				},
				{
					"PE + Owner format (empty)",
					`OWNER PE       WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
===========================================================================================================================`,
					false,
				},
				{
					"Slots only format (empty)",
					`SLOTS    WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
=====================================================================================================================`,
					false,
				},
				{
					"Complex multi-column format (empty)",
					`HOST CLUSTER QUEUE OWNER PE              WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
=====================================================================================================================================================`,
					false,
				},
				{
					"Project + Department format (empty)",
					`PROJECT DEPARTMENT            WALLCLOCK         UTIME         STIME           CPU             MEMORY                 IO                IOW
==========================================================================================================================================`,
					false,
				},
			}

			for _, tc := range testCases {
				By("Testing " + tc.name)
				usage, err := qacct.ParseSummaryOutput(tc.input)
				Expect(err).NotTo(HaveOccurred())
				
				if tc.hasData {
					// Should have actual usage data
					Expect(usage.WallClock).To(Equal(138.0))
					Expect(usage.UserTime).To(Equal(2.000))
					Expect(usage.SystemTime).To(Equal(1.517))
					Expect(usage.CPU).To(Equal(3.517))
					Expect(usage.Memory).To(Equal(0.340))
					Expect(usage.IO).To(Equal(0.001))
					Expect(usage.IOWait).To(Equal(0.000))
				} else {
					// Should be empty (all zeros)
					Expect(usage.WallClock).To(Equal(0.0))
					Expect(usage.UserTime).To(Equal(0.0))
					Expect(usage.SystemTime).To(Equal(0.0))
					Expect(usage.CPU).To(Equal(0.0))
					Expect(usage.Memory).To(Equal(0.0))
					Expect(usage.IO).To(Equal(0.0))
					Expect(usage.IOWait).To(Equal(0.0))
				}
			}
		})
	})
})