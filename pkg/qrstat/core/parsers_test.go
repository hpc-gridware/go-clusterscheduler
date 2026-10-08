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

package core_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/core"
)

// fixture returns a captured qrstat output from testdata/<version>/.
func fixture(version, name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", version, name))
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

func localTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05.000000", s, time.Local)
	Expect(err).NotTo(HaveOccurred())
	return t
}

func byID[T any](items []T, id func(T) int64) map[int64]T {
	m := map[int64]T{}
	for _, item := range items {
		m[id(item)] = item
	}
	return m
}

var _ = Describe("ParseSummary", func() {

	It("returns an empty slice for empty output", func() {
		ars, err := core.ParseSummary("")
		Expect(err).NotTo(HaveOccurred())
		Expect(ars).NotTo(BeNil())
		Expect(ars).To(BeEmpty())
	})

	DescribeTable("parses captured summaries",
		func(version string, count int) {
			ars, err := core.ParseSummary(fixture(version, "qrstat_summary.stdout"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ars).To(HaveLen(count))
			for _, ar := range ars {
				Expect(ar.ID).To(BeNumerically(">", 0))
				Expect(ar.Owner).To(Equal("root"))
				Expect(ar.EndTime.Sub(ar.StartTime)).To(Equal(ar.Duration))
			}
		},
		Entry("9.0", "9.0", 6),
		Entry("9.1", "9.1", 7),
	)

	It("keeps names with spaces and leading dashes and the truncated long name", func() {
		ars, err := core.ParseSummary(fixture("9.0", "qrstat_summary.stdout"))
		Expect(err).NotTo(HaveOccurred())
		m := byID(ars, func(s core.ReservationSummary) int64 { return s.ID })
		Expect(m[1].Name).To(Equal("plain"))
		Expect(m[1].State).To(Equal(core.StateRunning))
		Expect(m[1].StartTime).To(Equal(localTime("2026-10-08 10:10:05.000000")))
		Expect(m[1].Duration).To(Equal(time.Hour))
		Expect(m[2].Name).To(Equal("has space"))
		Expect(m[2].State).To(Equal(core.StateWaiting))
		Expect(m[2].Duration).To(Equal(time.Hour + 4*time.Second))
		Expect(m[5].Name).To(Equal("averyveryv"))
		Expect(m[6].Name).To(Equal("-foo"))
	})

	DescribeTable("attaches -explain messages to their AR",
		func(version string) {
			ars, err := core.ParseSummary(fixture(version, "qrstat_explain.stdout"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ars).NotTo(BeEmpty())
			for _, ar := range ars {
				Expect(ar.State).To(BeElementOf(core.StateError, core.StateWarning))
				Expect(ar.Messages).To(Equal([]string{"reserved queue all.q@master is disabled"}))
			}
		},
		Entry("9.0", "9.0"),
		Entry("9.1", "9.1"),
	)

	It("handles ids wider than the 7 digit column", func() {
		out := "ar-id   name       owner        state start at             end at               duration\n" +
			"------------------------------------------------------------------------------------------\n" +
			"4294967295 wide name  someverylong r     2026-10-08 10:09:00  2026-10-09 11:09:00  25:00:00\n" +
			"      7 x          root         w     2026-10-08 10:09:00  2026-10-08 10:19:00  00:10:00\n"
		ars, err := core.ParseSummary(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(ars).To(HaveLen(2))
		Expect(ars[0].ID).To(Equal(int64(4294967295)))
		Expect(ars[0].Name).To(Equal("wide name"))
		Expect(ars[0].Owner).To(Equal("someverylong"))
		Expect(ars[0].State).To(Equal(core.StateRunning))
		Expect(ars[0].Duration).To(Equal(25 * time.Hour))
		Expect(ars[1].ID).To(Equal(int64(7)))
	})

	It("accepts the 9.2 D:HH:MM:SS duration", func() {
		out := "      1 x          root         w     2026-10-08 10:09:00  2026-10-10 11:09:00  2:01:00:00\n"
		ars, err := core.ParseSummary(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(ars[0].Duration).To(Equal(49 * time.Hour))
	})

	DescribeTable("rejects garbage instead of returning partial data",
		func(out string) {
			_, err := core.ParseSummary(out)
			Expect(err).To(HaveOccurred())
		},
		Entry("message without row", "       reserved queue all.q@master is disabled\n"),
		Entry("unknown line", "something else\n"),
		Entry("bad date", "      1 x          root         w     2026-13-08 10:09:00  2026-10-08 10:19:00  00:10:00\n"),
		Entry("bad duration", "      1 x          root         w     2026-10-08 10:09:00  2026-10-08 10:19:00  10 min\n"),
		Entry("truncated row after a valid one",
			"      1 x          root         w     2026-10-08 10:09:00  2026-10-08 10:19:00  00:10:00\n"+
				"      9 trunc      root  r  2026-10-08 10:09:41\n"),
	)

	It("keeps name and owner valid UTF-8 when qrstat cuts inside a character", func() {
		// "ab" followed by 2.67 three-byte characters fills the 10 byte column.
		name := "ab" + "\u4e2d\u4e2d" + "\xe4\xb8"
		out := "      1 " + name + " root         w     2026-10-08 10:09:00  2026-10-08 10:19:00  00:10:00\n"
		ars, err := core.ParseSummary(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(ars[0].Name).To(Equal("ab\u4e2d\u4e2d"))
	})
})

var _ = Describe("ParseDetail", func() {

	It("parses the 9.0 detail format with granted_slots_list", func() {
		ars, err := core.ParseDetail(fixture("9.0", "qrstat_detail.stdout"))
		Expect(err).NotTo(HaveOccurred())
		Expect(ars).To(HaveLen(6))
		m := byID(ars, func(r core.Reservation) int64 { return r.ID })

		plain := m[1]
		Expect(plain.Name).To(Equal("plain"))
		Expect(plain.State).To(Equal(core.StateRunning))
		Expect(plain.StartTime).To(Equal(localTime("2026-10-08 10:10:05.005149")))
		Expect(plain.SubmissionTime).To(Equal(localTime("2026-10-08 10:10:05.005643")))
		Expect(plain.Duration).To(Equal(time.Hour))
		Expect(plain.Group).To(Equal("root"))
		Expect(plain.Account).To(Equal("sge"))
		Expect(plain.ExecQueueList).To(Equal(map[string]int{"all.q@master": 1}))
		Expect(plain.HardErrorHandling).To(BeFalse())
		Expect(plain.ExtraFields).To(BeEmpty())

		full := m[2]
		Expect(full.Name).To(Equal("has space name"))
		Expect(full.Account).To(Equal("acct1"))
		Expect(full.ResourceList).To(Equal(map[string]string{"h_rt": "600", "arch": "lx-amd64"}))
		Expect(full.HardErrorHandling).To(BeTrue())
		Expect(full.CheckpointName).To(Equal("ck"))
		Expect(full.MailOptions).To(Equal("be"))
		Expect(full.MailList).To(Equal([]string{"root@master", "other@NONE"}))
		Expect(full.ACLList).To(Equal([]string{"root", "@myacl"}))
		Expect(full.XACLList).To(Equal([]string{"nobody"}))

		pe := m[3]
		Expect(pe.GrantedParallelEnvironment).To(Equal(&core.GrantedPE{Name: "mpi", Range: "2"}))
		Expect(pe.MasterQueueList).To(Equal([]string{"all.q@master"}))
		Expect(pe.ExecQueueList).To(Equal(map[string]int{"all.q@master": 2}))
		Expect(m[4].GrantedParallelEnvironment.Range).To(Equal("1,2"))
		Expect(m[5].Name).To(Equal("averyveryverylongreservationname"))
		Expect(m[6].Name).To(Equal("-foo"))
	})

	It("parses the 9.1 detail format with exec_queue_list and binding", func() {
		ars, err := core.ParseDetail(fixture("9.1", "qrstat_detail.stdout"))
		Expect(err).NotTo(HaveOccurred())
		Expect(ars).To(HaveLen(7))
		m := byID(ars, func(r core.Reservation) int64 { return r.ID })
		Expect(m[8].ExecQueueList).To(Equal(map[string]int{"all.q@master": 1}))
		Expect(m[9].ResourceList).To(HaveKeyWithValue("arch", "lx-amd64"))
		Expect(m[10].GrantedParallelEnvironment).To(Equal(&core.GrantedPE{Name: "mpi", Range: "2"}))
		bind := m[18]
		Expect(bind.Binding).To(Equal("bamount=1,binstance=set,bstrategy=packed,btype=slot,bunit=C"))
		Expect(bind.ExecBindingList).To(Equal(map[string]string{"master": "ScCCCCCCCCCCCCC"}))
		for _, r := range ars {
			Expect(r.ExtraFields).To(BeEmpty())
		}
	})

	DescribeTable("collects error messages and the deleted state",
		func(version, file string, state core.ReservationState, messages int) {
			ars, err := core.ParseDetail(fixture(version, file))
			Expect(err).NotTo(HaveOccurred())
			Expect(ars).NotTo(BeEmpty())
			Expect(ars[0].State).To(Equal(state))
			Expect(ars[0].Messages).To(HaveLen(messages))
		},
		Entry("9.0 error", "9.0", "qrstat_error_detail.stdout", core.StateError, 1),
		Entry("9.1 error", "9.1", "qrstat_error_detail.stdout", core.StateError, 1),
		Entry("9.0 deleted", "9.0", "qrstat_deleted.stdout", core.StateDeleted, 0),
		Entry("9.1 deleted", "9.1", "qrstat_deleted.stdout", core.StateDeleted, 0),
	)

	It("parses multi-line granted_resources_list (9.1.6+) and keeps unknown keys", func() {
		out := "--------------------------------------------------------------------------------\n" +
			"id                             17\n" +
			"message                        reserved queue all.q@node01 is disabled\n" +
			"message                        host node02 is unknown\n" +
			"granted_resources_list         node01: gpu=2(gpu0 gpu1),fpga=1\n" +
			"                               node02: gpu=1(gpu3)\n" +
			"master hard queue_list         gpu.q@node01,gpu.q@node02\n" +
			"future_attribute               some value\n"
		ars, err := core.ParseDetail(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(ars).To(HaveLen(1))
		r := ars[0]
		Expect(r.Messages).To(Equal([]string{"reserved queue all.q@node01 is disabled", "host node02 is unknown"}))
		Expect(r.GrantedResourcesList).To(Equal(map[string][]core.GrantedResource{
			"node01": {{Name: "gpu", Amount: "2", IDs: []string{"gpu0", "gpu1"}}, {Name: "fpga", Amount: "1"}},
			"node02": {{Name: "gpu", Amount: "1", IDs: []string{"gpu3"}}},
		}))
		Expect(r.MasterQueueList).To(Equal([]string{"gpu.q@node01", "gpu.q@node02"}))
		Expect(r.ExtraFields).To(Equal(map[string]string{"future_attribute": "some value"}))
	})

	It("accepts the 9.2 comma separator in resource_list", func() {
		out := "--------------------------------------------------------------------------------\n" +
			"id                             1\n" +
			"resource_list                  h_rt=600,mem=1G\n"
		ars, err := core.ParseDetail(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(ars[0].ResourceList).To(Equal(map[string]string{"h_rt": "600", "mem": "1G"}))
	})

	It("returns an empty slice for empty output", func() {
		ars, err := core.ParseDetail("")
		Expect(err).NotTo(HaveOccurred())
		Expect(ars).To(BeEmpty())
	})

	DescribeTable("rejects malformed blocks",
		func(out string) {
			_, err := core.ParseDetail(out)
			Expect(err).To(HaveOccurred())
		},
		Entry("attribute before separator", "id                             1\n"),
		Entry("block without id", detailBlock("name                           x\n")),
		Entry("bad id", detailBlock("id                             abc\n")),
		Entry("bad queue slots", detailBlock("id                             1\nexec_queue_list                all.q@master=x\n")),
		Entry("bad pe", detailBlock("id                             1\ngranted_parallel_environment   mpi\n")),
		Entry("leading continuation", detailBlock("                               node01: gpu=1\n")),
		Entry("repeated owner", detailBlock("id                             1\nowner                          a\nowner                          b\n")),
	)
})

func detailBlock(body string) string {
	return "--------------------------------------------------------------------------------\n" + body
}
