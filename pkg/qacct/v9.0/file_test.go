package qacct_test

import (
	"context"
	"fmt"
	"log"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/v9.0"
)

var _ = Describe("File", func() {

	Context("WatchFile", func() {

		It("returns an error when the file does not exist", func() {
			_, err := qacct.WatchFile(context.Background(),
				"nonexistentfile.txt", 10)
			Expect(err).To(HaveOccurred())
		})

		It("returns a channel that emits JobDetail objects for 10 jobs", func() {

			jobDetailsChan, err := qacct.WatchFile(context.Background(),
				qacct.GetDefaultQacctFile(), 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(jobDetailsChan).NotTo(BeNil())

			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())

			jobIDs := make([]int, 10)
			for i := 0; i < 10; i++ {
				jobID, _, err := qs.Submit(context.Background(),
					qsub.JobOptions{
						Command:     "/bin/bash",
						CommandArgs: []string{"-c", fmt.Sprintf("echo job %d; sleep 0", i+1)},
						Binary:      qsub.ToPtr(true),
					})
				Expect(err).NotTo(HaveOccurred())
				log.Printf("jobID: %d", jobID)
				jobIDs[i] = int(jobID)
			}

			receivedJobs := make(map[int]bool)
			Eventually(func() bool {
				select {
				case jd := <-jobDetailsChan:
					log.Printf("job: %+v", jd.JobNumber)
					// check if jobID is in the jobIDs list
					if slices.Contains(jobIDs, int(jd.JobNumber)) {
						Expect(jd.SubmitCommandLine).To(ContainSubstring("bash"))
						Expect(jd.JobUsage.Usage.Memory).To(BeNumerically(">=", 0))
						receivedJobs[int(jd.JobNumber)] = true
					}
				default:
					return len(receivedJobs) == 10
				}
				return false
			}, "10s").Should(BeTrue())
		})
	})
})
