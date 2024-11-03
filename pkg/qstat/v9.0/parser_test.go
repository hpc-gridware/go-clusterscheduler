package qstat_test

import (
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
			Expect(jobs[0].SubmitStartAt).To(Equal("2024-10-28 07:21:41"))
			Expect(jobs[0].Queue).To(Equal("all.q@master"))
			Expect(jobs[0].Master).To(Equal("MASTER"))
			Expect(jobs[0].TaskID).To(Equal(""))
			Expect(jobs[1].JobID).To(Equal(15))
			Expect(jobs[1].Name).To(Equal("sleep"))
			Expect(jobs[1].User).To(Equal("root"))
			Expect(jobs[1].State).To(Equal("r"))
			Expect(jobs[1].SubmitStartAt).To(Equal("2024-10-28 07:26:14"))
			Expect(jobs[1].TaskID).To(Equal("1"))
			Expect(jobs[2].JobID).To(Equal(15))
			Expect(jobs[2].Name).To(Equal("sleep"))
			Expect(jobs[2].User).To(Equal("root"))
			Expect(jobs[2].State).To(Equal("r"))
			Expect(jobs[2].SubmitStartAt).To(Equal("2024-10-28 07:26:14"))
			Expect(jobs[2].TaskID).To(Equal("3"))
			Expect(jobs[3].JobID).To(Equal(15))
			Expect(jobs[3].Name).To(Equal("sleep"))
			Expect(jobs[3].User).To(Equal("root"))
			Expect(jobs[3].State).To(Equal("r"))
			Expect(jobs[3].SubmitStartAt).To(Equal("2024-10-28 07:26:14"))
			Expect(jobs[3].TaskID).To(Equal("5"))
			Expect(jobs[4].JobID).To(Equal(17))
			Expect(jobs[4].Name).To(Equal("sleep"))
			Expect(jobs[4].User).To(Equal("root"))
			Expect(jobs[4].State).To(Equal("qw"))
			Expect(jobs[4].SubmitStartAt).To(Equal("2024-10-28 07:27:50"))
			Expect(jobs[4].TaskID).To(Equal(""))
			Expect(jobs[5].JobID).To(Equal(12))
			Expect(jobs[5].Name).To(Equal("sleep"))
			Expect(jobs[5].User).To(Equal("root"))
			Expect(jobs[5].State).To(Equal("qw"))
			Expect(jobs[5].SubmitStartAt).To(Equal("2024-10-28 07:17:34"))
			Expect(jobs[5].TaskID).To(Equal(""))
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
			Expect(jobs[40].SubmitStartAt).To(Equal("2024-10-28 07:17:34"))

			// job before last
			Expect(jobs[39].JobID).To(Equal(20))
			Expect(jobs[39].TaskID).To(Equal("2"))
			Expect(jobs[39].Queue).To(Equal("all.q@sim9"))
			Expect(jobs[39].Master).To(Equal("SLAVE"))
			Expect(jobs[39].SubmitStartAt).To(Equal("2024-10-28 08:34:36"))
		})

	})

})
