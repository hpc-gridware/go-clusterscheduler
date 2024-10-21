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

package qsub_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qsub"
)

var _ = Describe("QsubImpl", func() {

	Context("NativeSpecification", func() {

		It("should return the native qsub command line specification for the given options", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())

			output, err := qs.SubmitWithNativeSpecification(ctx,
				[]string{"-help"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("usage: qsub"))
		})

		It("should return the job ID", func() {
			ctx := context.Background()
			qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
			Expect(err).NotTo(HaveOccurred())

			output, err := qs.SubmitWithNativeSpecification(ctx,
				[]string{"-b", "y", "sleep", "10"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("Your job"))

			jobId, output, err := qs.SubmitSimpleBinary(ctx, "sleep", "0")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobId).To(BeNumerically(">", 0))
			Expect(output).NotTo(Equal(""))

			//submit again
			jobId2, output, err := qs.SubmitSimpleBinary(ctx, "sleep", "0")
			Expect(err).NotTo(HaveOccurred())
			Expect(jobId2).To(BeNumerically(">", 0))
			Expect(output).NotTo(Equal(""))
			// jobId2 should be higher than jobId
			Expect(jobId2).To(BeNumerically(">", jobId))
		})

	})

})
