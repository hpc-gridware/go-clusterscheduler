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

// Layer 2 argument-injection tests for the qsub option builder and the fluent
// JobBuilder. These are pure (no qsub binary): they assert that a tainted
// structured option is rejected before any submission is attempted, and that
// legitimate values pass.

package core_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
)

func TestQsubCoreValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qsub Core Validation Suite")
}

// fakeSubmitter records whether SubmitWithNativeSpecification was invoked, so a
// spec can assert a tainted build never reaches submission.
type fakeSubmitter struct{ called bool }

func (f *fakeSubmitter) SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error) {
	f.called = true
	return "1", nil
}

var _ = Describe("BuildQsubArgs Layer 2", func() {
	It("rejects a comma in an environment variable value", func() {
		_, err := qsub.BuildQsubArgs(qsub.JobOptions{
			EnvVariables: map[string]string{"FOO": "a,b"},
		})
		Expect(err).To(HaveOccurred())
	})

	It("rejects a leading-dash queue element", func() {
		_, err := qsub.BuildQsubArgs(qsub.JobOptions{Queue: []string{"-x"}})
		Expect(err).To(HaveOccurred())
	})

	It("accepts a legitimate env value that itself contains '='", func() {
		_, err := qsub.BuildQsubArgs(qsub.JobOptions{
			EnvVariables: map[string]string{"FOO": "a=b"},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("accepts a job name with a leading digit outside strict mode", func() {
		name := "1job"
		_, err := qsub.BuildQsubArgs(qsub.JobOptions{JobName: &name})
		Expect(err).ToNot(HaveOccurred())
	})

	It("rejects a job name with a leading digit in strict mode (QSUB_TABLE)", func() {
		GinkgoT().Setenv("GCS_VALIDATION", "strict")
		name := "1job"
		_, err := qsub.BuildQsubArgs(qsub.JobOptions{JobName: &name})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("JobBuilder Layer 2", func() {
	It("rejects a tainted resource value and never submits", func() {
		f := &fakeSubmitter{}
		_, _, err := qsub.NewJobBuilder(f, "sleep", "10").
			Resource("h_vmem", "1G,evil=x").
			Submit(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(f.called).To(BeFalse())
	})

	It("rejects a leading-dash queue and never submits", func() {
		f := &fakeSubmitter{}
		_, _, err := qsub.NewJobBuilder(f, "sleep", "10").
			Queue("-x").
			Submit(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(f.called).To(BeFalse())
	})

	It("submits when all builder inputs are clean", func() {
		f := &fakeSubmitter{}
		_, _, err := qsub.NewJobBuilder(f, "sleep", "10").
			Queue("all.q").
			Resource("h_vmem", "1G").
			Env("MYVAR", "value").
			Submit(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(f.called).To(BeTrue())
	})
})
