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

package qalter_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qalter "github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/v9.0"
)

var _ = Describe("QalterImpl", func() {

	var q *qalter.CommandLineQAlter

	BeforeEach(func() {
		var err error
		q, err = qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
			DryRun: true,
		})
		Expect(err).To(BeNil())
		Expect(q).NotTo(BeNil())
	})

	Context("Basic functionality", func() {

		It("should be able to create a new qalter instance", func() {
			q, err := qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{})
			Expect(q).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should implement the QAlter interface", func() {
			var _ qalter.QAlter = q
		})
	})

	Context("Common methods (dry-run)", func() {

		It("should set start time with time.Time", func() {
			t := time.Date(2025, 12, 25, 12, 0, 0, 0, time.UTC)
			out, err := q.SetStartTime("1234", t)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set job name", func() {
			out, err := q.SetJobName("1234", "my_job")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set hard resource list with slice", func() {
			out, err := q.SetHardResourceList("1234", []string{"mem_free=4G", "h_rt=3600"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set priority with int", func() {
			out, err := q.SetPriority("1234", 100)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set mail recipients with slice", func() {
			out, err := q.SetMailRecipients("1234", []string{"alice@example.com", "bob@example.com"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set max running tasks with int", func() {
			out, err := q.SetMaxRunningTasks("1234", 10)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set verify mode", func() {
			out, err := q.SetVerifyMode("1234", "p")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should run native specification", func() {
			out, err := q.NativeSpecification([]string{"-w", "p", "1234"})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})

	Context("v9.0 binding (dry-run)", func() {

		It("should set binding with linear spec", func() {
			out, err := q.SetBinding("1234", "env", "linear:4")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding with striding spec", func() {
			out, err := q.SetBinding("1234", "pe", "striding:2:4")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})

		It("should set binding with explicit spec", func() {
			out, err := q.SetBinding("1234", "set", "explicit:0,0:1,0")
			Expect(err).To(BeNil())
			Expect(out).To(Equal(""))
		})
	})
})
