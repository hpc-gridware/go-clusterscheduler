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

package qstat_test

import (
	"time"

	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {

	Context("ParseGroupByTask", func() {

		It("should parse the output of qstat -g t", func() {
			input := `job-ID  prior   name       user         state submit/start at     queue                          master ja-task-ID
------------------------------------------------------------------------------------------------------------------
     14 0.50500 sleep      root         r     2024-10-28 07:21:41 all.q@master                   MASTER
     15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 1
     15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 3
     15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 5
     17 0.60500 sleep      root         qw    2024-10-28 07:27:50
     12 0.50500 sleep      root         qw    2024-10-28 07:17:34
     15 0.50500 sleep      root         qw    2024-10-28 07:26:14                                       7-99:2`
			jobs, err := qstat.ParseGroupByTask(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(jobs)).To(Equal(7))
			Expect(jobs[0].JobID).To(Equal(14))
			Expect(jobs[0].Name).To(Equal("sleep"))
			Expect(jobs[0].User).To(Equal("root"))
			Expect(jobs[0].State).To(Equal("r"))
			// "2024-10-28 07:21:41"
			Expect(jobs[0].StartTime.Year()).To(Equal(2024))
			Expect(jobs[0].StartTime.Month()).To(Equal(time.October))
			Expect(jobs[0].StartTime.Day()).To(Equal(28))
			Expect(jobs[0].StartTime.Hour()).To(Equal(7))
			Expect(jobs[0].StartTime.Minute()).To(Equal(21))
			Expect(jobs[0].StartTime.Second()).To(Equal(41))
			Expect(jobs[0].Queue).To(Equal("all.q@master"))
			Expect(jobs[0].Master).To(Equal("MASTER"))
			Expect(len(jobs[0].JaTaskIDs)).To(Equal(0))
			Expect(jobs[1].JobID).To(Equal(15))
			Expect(jobs[1].Name).To(Equal("sleep"))
			Expect(jobs[1].User).To(Equal("root"))
			Expect(jobs[1].State).To(Equal("r"))
			// "2024-10-28 07:26:14"
			Expect(jobs[1].StartTime.Year()).To(Equal(2024))
			Expect(jobs[1].StartTime.Month()).To(Equal(time.October))
			Expect(jobs[1].StartTime.Day()).To(Equal(28))
			Expect(jobs[1].StartTime.Hour()).To(Equal(7))
			Expect(jobs[1].StartTime.Minute()).To(Equal(26))
			Expect(jobs[1].StartTime.Second()).To(Equal(14))
			Expect(jobs[1].JaTaskIDs).To(Equal([]int64{1}))
			Expect(jobs[2].JobID).To(Equal(15))
			Expect(jobs[2].Name).To(Equal("sleep"))
			Expect(jobs[2].User).To(Equal("root"))
			Expect(jobs[2].State).To(Equal("r"))
			// "2024-10-28 07:26:14"
			Expect(jobs[2].StartTime.Year()).To(Equal(2024))
			Expect(jobs[2].StartTime.Month()).To(Equal(time.October))
			Expect(jobs[2].StartTime.Day()).To(Equal(28))
			Expect(jobs[2].StartTime.Hour()).To(Equal(7))
			Expect(jobs[2].StartTime.Minute()).To(Equal(26))
			Expect(jobs[2].StartTime.Second()).To(Equal(14))
			Expect(jobs[2].JaTaskIDs).To(Equal([]int64{3}))
			Expect(jobs[3].JobID).To(Equal(15))
			Expect(jobs[3].Name).To(Equal("sleep"))
			Expect(jobs[3].User).To(Equal("root"))
			Expect(jobs[3].State).To(Equal("r"))
			// "2024-10-28 07:26:14"
			Expect(jobs[3].StartTime.Year()).To(Equal(2024))
			Expect(jobs[3].StartTime.Month()).To(Equal(time.October))
			Expect(jobs[3].StartTime.Day()).To(Equal(28))
			Expect(jobs[3].StartTime.Hour()).To(Equal(7))
			Expect(jobs[3].StartTime.Minute()).To(Equal(26))
			Expect(jobs[3].StartTime.Second()).To(Equal(14))
			Expect(jobs[3].JaTaskIDs).To(Equal([]int64{5}))
			Expect(jobs[4].JobID).To(Equal(17))
			Expect(jobs[4].Name).To(Equal("sleep"))
			Expect(jobs[4].User).To(Equal("root"))
			Expect(jobs[4].State).To(Equal("qw"))
			// "2024-10-28 07:27:50"
			Expect(jobs[4].SubmitTime.Year()).To(Equal(2024))
			Expect(jobs[4].SubmitTime.Month()).To(Equal(time.October))
			Expect(jobs[4].SubmitTime.Day()).To(Equal(28))
			Expect(jobs[4].SubmitTime.Hour()).To(Equal(7))
			Expect(jobs[4].SubmitTime.Minute()).To(Equal(27))
			Expect(jobs[4].SubmitTime.Second()).To(Equal(50))
			Expect(jobs[5].JobID).To(Equal(12))
			Expect(jobs[5].Name).To(Equal("sleep"))
			Expect(jobs[5].User).To(Equal("root"))
			Expect(jobs[5].State).To(Equal("qw"))
			// "2024-10-28 07:17:34"
			Expect(jobs[5].SubmitTime.Year()).To(Equal(2024))
			Expect(jobs[5].SubmitTime.Month()).To(Equal(time.October))
			Expect(jobs[5].SubmitTime.Day()).To(Equal(28))
			Expect(jobs[5].SubmitTime.Hour()).To(Equal(7))
			Expect(jobs[5].SubmitTime.Minute()).To(Equal(17))
			Expect(jobs[5].SubmitTime.Second()).To(Equal(34))
		})

		It("should parse the output of qstat -g t", func() {

			output := `job-ID  prior   name       user         state submit/start at     queue                          master ja-task-ID
------------------------------------------------------------------------------------------------------------------
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@master                   SLAVE
                                                                  all.q@master                   SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@master                   MASTER 1
                                                                  all.q@master                   SLAVE  1
                                                                  all.q@master                   SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@master                   SLAVE  2
                                                                  all.q@master                   SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim1                     SLAVE
                                                                  all.q@sim1                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim1                     SLAVE  1
                                                                  all.q@sim1                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim1                     SLAVE  2
                                                                  all.q@sim1                     SLAVE  2
                                                                  all.q@sim1                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim10                    SLAVE
                                                                  all.q@sim10                    SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim10                    SLAVE  1
                                                                  all.q@sim10                    SLAVE  1
                                                                  all.q@sim10                    SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim10                    SLAVE  2
                                                                  all.q@sim10                    SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim11                    SLAVE
                                                                  all.q@sim11                    SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim11                    SLAVE  1
                                                                  all.q@sim11                    SLAVE  1
                                                                  all.q@sim11                    SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim11                    SLAVE  2
                                                                  all.q@sim11                    SLAVE  2
     18 0.50500 sleep      root         r     2024-10-28 08:33:57 all.q@sim12                    MASTER
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim12                    SLAVE
                                                                  all.q@sim12                    SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim12                    SLAVE  1
                                                                  all.q@sim12                    SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim12                    SLAVE  2
                                                                  all.q@sim12                    SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim2                     SLAVE
                                                                  all.q@sim2                     SLAVE
                                                                  all.q@sim2                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim2                     SLAVE  1
                                                                  all.q@sim2                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim2                     SLAVE  2
                                                                  all.q@sim2                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim3                     MASTER
                                                                  all.q@sim3                     SLAVE
                                                                  all.q@sim3                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim3                     SLAVE  1
                                                                  all.q@sim3                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim3                     SLAVE  2
                                                                  all.q@sim3                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim4                     SLAVE
                                                                  all.q@sim4                     SLAVE
                                                                  all.q@sim4                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim4                     SLAVE  1
                                                                  all.q@sim4                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim4                     SLAVE  2
                                                                  all.q@sim4                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim5                     SLAVE
                                                                  all.q@sim5                     SLAVE
                                                                  all.q@sim5                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim5                     SLAVE  1
                                                                  all.q@sim5                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim5                     SLAVE  2
                                                                  all.q@sim5                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim6                     SLAVE
                                                                  all.q@sim6                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim6                     SLAVE  1
                                                                  all.q@sim6                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim6                     SLAVE  2
                                                                  all.q@sim6                     SLAVE  2
                                                                  all.q@sim6                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim7                     SLAVE
                                                                  all.q@sim7                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim7                     SLAVE  1
                                                                  all.q@sim7                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim7                     MASTER 2
                                                                  all.q@sim7                     SLAVE  2
                                                                  all.q@sim7                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim8                     SLAVE
                                                                  all.q@sim8                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim8                     SLAVE  1
                                                                  all.q@sim8                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim8                     SLAVE  2
                                                                  all.q@sim8                     SLAVE  2
                                                                  all.q@sim8                     SLAVE  2
     19 0.60500 sleep      root         r     2024-10-28 08:34:25 all.q@sim9                     SLAVE
                                                                  all.q@sim9                     SLAVE
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim9                     SLAVE  1
                                                                  all.q@sim9                     SLAVE  1
                                                                  all.q@sim9                     SLAVE  1
     20 0.60500 sleep      root         r     2024-10-28 08:34:36 all.q@sim9                     SLAVE  2
                                                                  all.q@sim9                     SLAVE  2
     12 0.50500 sleep      root         qw    2024-10-28 07:17:34`

			jobs, err := qstat.ParseGroupByTask(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(jobs)).To(Equal(41))

			// last job
			Expect(jobs[40].JobID).To(Equal(12))
			Expect(jobs[40].Name).To(Equal("sleep"))
			Expect(jobs[40].User).To(Equal("root"))
			Expect(jobs[40].State).To(Equal("qw"))
			Expect(jobs[40].SubmitTime.Format(qstat.QstatDateFormat)).To(Equal("2024-10-28 07:17:34"))

			// job before last
			Expect(jobs[39].JobID).To(Equal(20))
			Expect(jobs[39].Queue).To(Equal("all.q@sim9"))
			Expect(jobs[39].Master).To(Equal("SLAVE"))
			Expect(jobs[39].StartTime.Format(qstat.QstatDateFormat)).To(Equal("2024-10-28 08:34:36"))
		})

	})

	/*

	   job-ID  prior   name       user         state submit/start at     queue                          master ja-task-ID
	   ------------------------------------------------------------------------------------------------------------------
	        14 0.50500 sleep      root         r     2024-10-28 07:21:41 all.q@master                   MASTER
	        15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 1
	        15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 3
	        15 0.50500 sleep      root         r     2024-10-28 07:26:14 all.q@master                   MASTER 5
	        17 0.60500 sleep      root         qw    2024-10-28 07:27:50
	        12 0.50500 sleep      root         qw    2024-10-28 07:17:34
	        15 0.50500 sleep      root         qw    2024-10-28 07:26:14                                       7-99:2
	*/

	Context("ParseQstatExt", func() {

		It("should parse the output of qstat -ext", func() {

			input := `job-ID  prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
     36 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1
     37 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1
`

			jobs, err := qstat.ParseExtendedJobInfo(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(jobs)).To(Equal(2))
			Expect(jobs[0].JobID).To(Equal(36))
			Expect(jobs[0].Name).To(Equal("sleep"))
			Expect(jobs[0].User).To(Equal("root"))
			Expect(jobs[0].State).To(Equal("r"))
			Expect(jobs[0].Queue).To(Equal("all.q@sim10"))
			Expect(jobs[0].Slots).To(Equal(1))

			input2 := `job-ID  prior   ntckts  name       user         project          department state cpu        mem     io      tckts ovrts otckt ftckt stckt share queue                          slots ja-task-ID
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 1
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 2
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 3
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 4
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 5
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 6
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 7
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 8
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 9
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 10
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 11
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 12
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 13
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 14
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 15
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 16
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 17
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 18
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 19
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 20
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 21
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 22
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 23
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 24
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 25
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 26
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 27
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 28
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 29
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 30
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 31
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 32
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 33
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 34
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 35
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 36
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 37
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 38
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 39
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 40
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 41
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 42
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 43
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 44
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 45
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 46
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 47
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 48
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 49
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 50
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 51
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 52
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 53
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 54
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 55
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 56
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 57
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 58
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 59
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 60
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 61
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 62
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 63
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 64
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 65
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 66
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 67
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 68
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 69
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 70
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 71
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 72
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 73
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 74
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 75
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 76
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 77
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 78
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 79
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 80
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 81
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 82
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 83
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 84
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 85
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 86
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 87
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 88
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 89
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 90
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 91
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 92
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 93
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 94
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 95
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 96
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 97
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 98
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 99
     33 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 100
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 1
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 2
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 3
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 4
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 5
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 6
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 7
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 8
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 9
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 10
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 11
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 12
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 13
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 14
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 15
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 16
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 17
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim1                         1 18
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim6                         1 19
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim8                         1 20
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim12                        1 21
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim3                         1 22
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim4                         1 23
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim5                         1 24
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim2                         1 25
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@master                       1 26
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim11                        1 27
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim10                        1 28
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim9                         1 29
     34 0.55500 0.50000 sleep      root         NA               defaultdep r     NA         NA      NA          0     0     0     0     0 0.00  all.q@sim7                         1 30
     34 0.55500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1 31-100:1
     35 0.55500 0.50000 sleep      root         NA               defaultdep qw                                   0     0     0     0     0 0.00                                     1
`

			jobs, err = qstat.ParseExtendedJobInfo(input2)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(jobs)).To(Equal(132))
			Expect(jobs[130].JATaskID).To(Equal("31-100:1"))
			Expect(jobs[131].JobID).To(Equal(35))
			Expect(jobs[131].Name).To(Equal("sleep"))
			Expect(jobs[131].User).To(Equal("root"))
			Expect(jobs[131].State).To(Equal("qw"))
			Expect(jobs[131].Priority).To(Equal(0.555))
			Expect(jobs[131].Ntckts).To(Equal(0.5))
			Expect(jobs[131].User).To(Equal("root"))
			Expect(jobs[131].Slots).To(Equal(1))
		})

	})

	Describe("ClusterQueueSummary", func() {

		It("should parse the output of qstat -g c", func() {
			input := `CLUSTER QUEUE                   CQLOAD   USED    RES  AVAIL  TOTAL aoACDS  cdsuE
--------------------------------------------------------------------------------
all.q                             0.08      0      0      4      4      0      0
test.q                            0.08      0      0      2      2      0      0`
			summary, err := qstat.ParseClusterQueueSummary(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(summary)).To(Equal(2))

			Expect(summary[0].ClusterQueue).To(Equal("all.q"))
			Expect(summary[0].CQLoad).To(Equal(0.08))
			Expect(summary[0].Used).To(Equal(0))
			Expect(summary[0].Reserved).To(Equal(0))
			Expect(summary[0].Available).To(Equal(4))
			Expect(summary[0].Total).To(Equal(4))
			Expect(summary[0].AoACDS).To(Equal(0))
			Expect(summary[0].CdsuE).To(Equal(0))

			Expect(summary[1].ClusterQueue).To(Equal("test.q"))
			Expect(summary[1].CQLoad).To(Equal(0.08))
			Expect(summary[1].Used).To(Equal(0))
			Expect(summary[1].Reserved).To(Equal(0))
			Expect(summary[1].Available).To(Equal(2))
			Expect(summary[1].Total).To(Equal(2))
			Expect(summary[1].AoACDS).To(Equal(0))
			Expect(summary[1].CdsuE).To(Equal(0))
		})

	})

})
