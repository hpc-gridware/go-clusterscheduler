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
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/core"
)

// acctLine is a record of job id whose line is padded to about size bytes.
func acctLine(id int, pad int) string {
	return fmt.Sprintf(`{"job_number":%d,"task_number":0,"owner":"u%d","job_name":"n","submit_cmd_line":"%s","failed":0,"exit_status":0,"usage":{"rusage":{"ru_wallclock":1},"eusage":{"wallclock":1.5}}}`,
		id, id%7, strings.Repeat("x", pad))
}

// acctFile is n records, job numbers 1 to n, each ended by a newline.
func acctFile(n int) []byte {
	var b strings.Builder
	for id := 1; id <= n; id++ {
		b.WriteString(acctLine(id, 0))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func jobNumbers(records []core.Record) []int64 {
	ids := make([]int64, len(records))
	for i, r := range records {
		ids[i] = r.JobNumber
	}
	return ids
}

func ints(from, to int) []int64 {
	var ids []int64
	for id := from; id <= to; id++ {
		ids = append(ids, int64(id))
	}
	return ids
}

func tailOf(data []byte, o core.ReadOptions) core.ReadResult {
	res, err := core.ReadTail(context.Background(), bytes.NewReader(data), int64(len(data)), o)
	Expect(err).NotTo(HaveOccurred())
	return res
}

func forwardOf(data []byte, offset int64, o core.ReadOptions) core.ReadResult {
	res, err := core.ReadForward(context.Background(), bytes.NewReader(data), offset, int64(len(data)), o)
	Expect(err).NotTo(HaveOccurred())
	return res
}

// expectLines checks that every record's Offset and Length name its own
// line in data.
func expectLines(data []byte, records []core.Record) {
	for _, r := range records {
		line := data[r.Offset : r.Offset+int64(r.Length)]
		Expect(string(line)).To(HavePrefix(fmt.Sprintf(`{"job_number":%d,`, r.JobNumber)))
		Expect(r.Offset == 0 || data[r.Offset-1] == '\n').To(BeTrue(), "record %d starts a line", r.JobNumber)
		end := r.Offset + int64(r.Length)
		Expect(int(end) < len(data) && data[end] == '\n').To(BeTrue(), "record %d ends at a newline", r.JobNumber)
	}
}

var _ = Describe("Reading the accounting file from its end", func() {

	Describe("ReadTail", func() {
		It("returns the last records in file order, with where they are", func() {
			data := acctFile(20)
			res := tailOf(data, core.ReadOptions{MaxRecords: 5})
			Expect(jobNumbers(res.Records)).To(Equal(ints(16, 20)))
			expectLines(data, res.Records)
			Expect(res.End).To(Equal(int64(len(data))))
			Expect(res.Start).To(Equal(res.Records[0].Offset))
			Expect(res.AtStart).To(BeFalse())
			Expect(res.NotParsed).To(BeZero())
		})

		It("returns the whole file when it is asked for more than there is", func() {
			data := acctFile(20)
			for _, max := range []int{20, 21, 1000, 0} {
				res := tailOf(data, core.ReadOptions{MaxRecords: max})
				Expect(jobNumbers(res.Records)).To(Equal(ints(1, 20)), "MaxRecords %d", max)
				Expect(res.Start).To(BeZero())
				Expect(res.End).To(Equal(int64(len(data))))
			}
			Expect(tailOf(data, core.ReadOptions{}).AtStart).To(BeTrue())
		})

		It("does not read a last line qmaster has not finished", func() {
			data := acctFile(5)
			half := append(append([]byte(nil), data...), []byte(acctLine(6, 0)[:40])...)
			res := tailOf(half, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal(ints(1, 5)))
			Expect(res.End).To(Equal(int64(len(data))))
			Expect(res.NotParsed).To(BeZero())

			// Also when the unfinished line is a whole record but for its
			// newline.
			whole := append(append([]byte(nil), data...), []byte(acctLine(6, 0))...)
			res = tailOf(whole, core.ReadOptions{MaxRecords: 2})
			Expect(jobNumbers(res.Records)).To(Equal(ints(4, 5)))
			Expect(res.End).To(Equal(int64(len(data))))
		})

		It("reads an empty file and a file without a complete line", func() {
			res := tailOf(nil, core.ReadOptions{})
			Expect(res.Records).To(BeEmpty())
			Expect(res.AtStart).To(BeTrue())
			Expect(res.Start).To(BeZero())
			Expect(res.End).To(BeZero())

			res = tailOf([]byte(acctLine(1, 0)), core.ReadOptions{})
			Expect(res.Records).To(BeEmpty())
			Expect(res.End).To(BeZero())
			Expect(res.AtStart).To(BeTrue())

			// An unfinished line longer than what is read at once.
			res = tailOf([]byte(acctLine(1, 700<<10)), core.ReadOptions{})
			Expect(res.Records).To(BeEmpty())
			Expect(res.End).To(BeZero())
		})

		It("passes over empty and comment lines without counting them", func() {
			data := []byte("# written by a tool\n" + acctLine(1, 0) + "\n\n   \n" + acctLine(2, 0) + "\r\n")
			res := tailOf(data, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal(ints(1, 2)))
			Expect(res.NotParsed).To(BeZero())
		})

		It("counts a line which is no record and reads on", func() {
			data := []byte(acctLine(1, 0) + "\nnot json at all\n" + `{"job_number":2,"usage":{"eusage":` + "\n" +
				`{"owner":"nobody"}` + "\n" + acctLine(3, 0) + "\n")
			res := tailOf(data, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal([]int64{1, 3}))
			Expect(res.NotParsed).To(Equal(3))
		})

		DescribeTable("counts a line which is too long and reads on",
			func(build func() []byte, want []int64) {
				data := build()
				res := tailOf(data, core.ReadOptions{MaxLineBytes: 2000})
				Expect(jobNumbers(res.Records)).To(Equal(want))
				Expect(res.NotParsed).To(Equal(1))
				Expect(res.AtStart).To(BeTrue())
				expectLines(data, res.Records)

				forward := forwardOf(data, 0, core.ReadOptions{MaxLineBytes: 2000})
				Expect(jobNumbers(forward.Records)).To(Equal(want))
				Expect(forward.NotParsed).To(Equal(1))
				Expect(forward.End).To(Equal(int64(len(data))))
			},
			Entry("in the middle", func() []byte {
				return []byte(acctLine(1, 0) + "\n" + acctLine(2, 5000) + "\n" + acctLine(3, 0) + "\n")
			}, []int64{1, 3}),
			Entry("longer than what is read at once", func() []byte {
				return []byte(acctLine(1, 0) + "\n" + acctLine(2, 900<<10) + "\n" + acctLine(3, 0) + "\n")
			}, []int64{1, 3}),
			Entry("as the first line", func() []byte {
				return []byte(acctLine(1, 900<<10) + "\n" + acctLine(2, 0) + "\n" + acctLine(3, 0) + "\n")
			}, []int64{2, 3}),
			Entry("as the last line", func() []byte {
				return []byte(acctLine(1, 0) + "\n" + acctLine(2, 0) + "\n" + acctLine(3, 900<<10) + "\n")
			}, []int64{1, 2}),
		)

		It("reads a line as long as the limit allows", func() {
			long := acctLine(2, 600<<10)
			data := []byte(acctLine(1, 0) + "\n" + long + "\n" + acctLine(3, 0) + "\n")
			res := tailOf(data, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal(ints(1, 3)))
			Expect(res.Records[1].Length).To(Equal(len(long)))
			Expect(res.Records[1].SubmitCommandLine).To(HaveLen(600 << 10))
			expectLines(data, res.Records)
		})

		It("stops at a line boundary once MaxBytes were read", func() {
			var b strings.Builder
			for id := 1; id <= 3000; id++ {
				b.WriteString(acctLine(id, 800))
				b.WriteByte('\n')
			}
			data := []byte(b.String())
			res := tailOf(data, core.ReadOptions{MaxBytes: 300 << 10})
			Expect(res.AtStart).To(BeFalse())
			Expect(len(res.Records)).To(BeNumerically(">", 100))
			Expect(len(res.Records)).To(BeNumerically("<", 1000))
			first := res.Records[0]
			Expect(jobNumbers(res.Records)).To(Equal(ints(int(first.JobNumber), 3000)))
			Expect(res.Start).To(Equal(first.Offset))
			expectLines(data, res.Records)
		})

		It("examines at least one line, however small MaxBytes is", func() {
			data := []byte(acctLine(1, 0) + "\n" + acctLine(2, 700<<10) + "\n")
			// The limit is checked after each read of the file, so the
			// read which finishes the long line may bring more with it.
			res := tailOf(data, core.ReadOptions{MaxBytes: 1})
			Expect(res.Records).NotTo(BeEmpty())
			Expect(res.Records[len(res.Records)-1].JobNumber).To(Equal(int64(2)))

			forward := forwardOf(data, int64(len(acctLine(1, 0))+1), core.ReadOptions{MaxBytes: 1})
			Expect(jobNumbers(forward.Records)).To(Equal([]int64{2}))
			Expect(forward.End).To(Equal(int64(len(data))))
		})

		It("returns what it has when the context ends", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			data := acctFile(10)
			res, err := core.ReadTail(ctx, bytes.NewReader(data), int64(len(data)), core.ReadOptions{})
			Expect(err).To(MatchError(context.Canceled))
			Expect(res.Records).To(BeEmpty())
		})

		It("fails when the file is shorter than the size it was given", func() {
			data := acctFile(3)
			_, err := core.ReadTail(context.Background(), bytes.NewReader(data), int64(len(data))+100, core.ReadOptions{})
			Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		})
	})

	// The file of the property specs: lines of very different lengths, some
	// with multi-byte characters, so that lines cross the chunks the readers
	// read in at every possible place.
	randomFile := func(seed int64, n int) []byte {
		rng := rand.New(rand.NewSource(seed))
		var b strings.Builder
		for id := 1; id <= n; id++ {
			pad := rng.Intn(3000)
			if rng.Intn(40) == 0 {
				pad = 200<<10 + rng.Intn(200<<10)
			}
			line := acctLine(id, pad)
			if rng.Intn(5) == 0 {
				line = strings.Replace(line, `"job_name":"n"`, "\"job_name\":\"gr\u00fc\u00dfe \u65e5\u672c\"", 1)
				line = strings.Replace(line, `"owner":"u`, `"owner":"`+"\u00fc\u65e5", 1)
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		return []byte(b.String())
	}

	Describe("the readers agree", func() {
		It("ReadTail returns the last N of what ReadForward reads, for any N", func() {
			for seed := int64(1); seed <= 3; seed++ {
				data := randomFile(seed, 1200)
				all := forwardOf(data, 0, core.ReadOptions{})
				Expect(jobNumbers(all.Records)).To(Equal(ints(1, 1200)))
				Expect(all.End).To(Equal(int64(len(data))))
				expectLines(data, all.Records)

				rng := rand.New(rand.NewSource(seed))
				for _, n := range []int{1, 2, 3, 7, 100, 1199, 1200, 1 + rng.Intn(1200), 1 + rng.Intn(1200)} {
					res := tailOf(data, core.ReadOptions{MaxRecords: n})
					Expect(res.Records).To(Equal(all.Records[len(all.Records)-n:]), "seed %d, last %d", seed, n)
					Expect(res.Start).To(Equal(res.Records[0].Offset))
				}
			}
		})

		It("ReadForward in batches reads every record once", func() {
			data := randomFile(7, 800)
			var got []core.Record
			offset := int64(0)
			for {
				res := forwardOf(data, offset, core.ReadOptions{MaxRecords: 37})
				got = append(got, res.Records...)
				if res.End == offset {
					break
				}
				offset = res.End
			}
			Expect(jobNumbers(got)).To(Equal(ints(1, 800)))
			expectLines(data, got)
		})

		It("ReadForward limited by bytes reads every record once", func() {
			data := randomFile(9, 800)
			var got []core.Record
			offset := int64(0)
			for offset < int64(len(data)) {
				res := forwardOf(data, offset, core.ReadOptions{MaxBytes: 100 << 10})
				Expect(res.End).To(BeNumerically(">", offset))
				got = append(got, res.Records...)
				offset = res.End
			}
			Expect(jobNumbers(got)).To(Equal(ints(1, 800)))
		})

		It("a tail read and the forward read which continues it lose and repeat nothing", func() {
			data := randomFile(11, 600)
			cut := len(data) / 2
			// The file as it was at an arbitrary moment: cut inside a line.
			tail := tailOf(data[:cut], core.ReadOptions{MaxRecords: 50})
			last := tail.Records[len(tail.Records)-1].JobNumber
			rest := forwardOf(data, tail.End, core.ReadOptions{})
			Expect(jobNumbers(rest.Records)).To(Equal(ints(int(last)+1, 600)))
		})
	})

	Describe("the lean decoder", func() {
		// lean is what a list keeps of a record.
		lean := func(r core.Record) []any {
			wall := r.JobUsage.Usage.WallClock
			return []any{r.Offset, r.Length, r.JobNumber, r.TaskID, r.PETaskID, r.JobName, r.Owner, r.Group, r.Project,
				r.Department, r.QName, r.HostName, r.GrantedPE, r.Slots, r.Priority, r.SubmitTime, r.StartTime, r.EndTime,
				r.Failed, r.ExitStatus, r.DeletedBy, wall, r.JobUsage.RUsage.RuWallclock}
		}
		same := func(data []byte) {
			full := tailOf(data, core.ReadOptions{})
			fast := tailOf(data, core.ReadOptions{Lean: true})
			Expect(fast.NotParsed).To(Equal(full.NotParsed))
			Expect(fast.Records).To(HaveLen(len(full.Records)))
			for i := range full.Records {
				Expect(lean(fast.Records[i])).To(Equal(lean(full.Records[i])), "record %d", i)
			}
			forward := forwardOf(data, 0, core.ReadOptions{Lean: true})
			Expect(forward.Records).To(Equal(fast.Records))
		}

		It("reads the same records with the same values from the real files", func() {
			same(fixture("accounting-9.0.10.jsonl"))
			same(fixture("accounting-9.0.10-ar.jsonl"))
			same(fixture("accounting-9.1.7.jsonl"))
			same([]byte(record900 + "\n" + recordAR + "\n"))
		})

		It("agrees with the full decoder about which lines are records", func() {
			same([]byte(acctLine(1, 0) + "\nnot json\n" + `{"owner":"nobody"}` + "\n" +
				`{"job_number":1e3,"owner":"x"}` + "\n" + `{"job_number":0}` + "\n" +
				// A member of another type: the lean decoder gives up and the
				// full one keeps the record.
				`{"job_number":5,"priority":"high","owner":"bob","slots":2}` + "\n" +
				`{"job_number":6,"end_time":1.5e15,"owner":"carol"}` + "\n" +
				`{"job_number":7,"usage":{"rusage":{"ru_wallclock":"many"}}}` + "\n" +
				acctLine(2, 0) + "\n"))
		})

		It("leaves out what a list does not show", func() {
			line := `{"job_number":9,"submit_cmd_line":"qsub x","category":"-q a","binding":"0,1","usage":{"eusage":{"wallclock":2.5,"cpu":1},"gpu":{"count":2}}}`
			res := tailOf([]byte(line+"\n"), core.ReadOptions{Lean: true})
			job := res.Records[0]
			Expect(job.JobUsage.Usage.WallClock).To(Equal(2.5))
			Expect(job.SubmitCommandLine).To(BeEmpty())
			Expect(job.Extra).To(BeNil())
			Expect(job.JobUsage.Custom).To(BeNil())
			Expect(job.JobUsage.Usage.CPU).To(BeZero())
		})
	})

	Describe("ReadForward", func() {
		It("leaves an unfinished last line for the next read", func() {
			first := acctLine(1, 0) + "\n"
			second := acctLine(2, 0) + "\n"
			third := acctLine(3, 0) + "\n"

			// qmaster has written the first record and half of the second.
			data := []byte(first + second[:60])
			res := forwardOf(data, 0, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal(ints(1, 1)))
			Expect(res.End).To(Equal(int64(len(first))))
			Expect(res.NotParsed).To(BeZero())

			// Nothing new was written: nothing is read, nothing is lost.
			res = forwardOf(data, res.End, core.ReadOptions{})
			Expect(res.Records).To(BeEmpty())
			Expect(res.End).To(Equal(int64(len(first))))

			// qmaster has written the rest.
			data = []byte(first + second + third)
			res = forwardOf(data, res.End, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal(ints(2, 3)))
			Expect(res.End).To(Equal(int64(len(data))))
		})

		It("reads nothing at the end of the file", func() {
			data := acctFile(3)
			res := forwardOf(data, int64(len(data)), core.ReadOptions{})
			Expect(res.Records).To(BeEmpty())
			Expect(res.Start).To(Equal(int64(len(data))))
			Expect(res.End).To(Equal(int64(len(data))))
			Expect(res.AtStart).To(BeFalse())
		})

		It("stops after MaxRecords, at the end of that record", func() {
			data := acctFile(10)
			res := forwardOf(data, 0, core.ReadOptions{MaxRecords: 4})
			Expect(jobNumbers(res.Records)).To(Equal(ints(1, 4)))
			Expect(res.End).To(Equal(res.Records[3].Offset + int64(res.Records[3].Length) + 1))
			Expect(res.AtStart).To(BeTrue())
		})

		It("returns the error of the context", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			data := acctFile(10)
			_, err := core.ReadForward(ctx, bytes.NewReader(data), 0, int64(len(data)), core.ReadOptions{})
			Expect(err).To(MatchError(context.Canceled))
		})
	})

	Describe("FindJob", func() {
		find := func(data []byte, id int64, o core.ReadOptions) core.ReadResult {
			res, err := core.FindJob(context.Background(), bytes.NewReader(data), int64(len(data)), id, o)
			Expect(err).NotTo(HaveOccurred())
			return res
		}

		It("returns every record of the job, and of no other", func() {
			var b strings.Builder
			for _, id := range []int{147, 47, 4711, 47, 470, 7, 47, 4} {
				b.WriteString(acctLine(id, 0) + "\n")
			}
			data := []byte(b.String())
			res := find(data, 47, core.ReadOptions{})
			Expect(jobNumbers(res.Records)).To(Equal([]int64{47, 47, 47}))
			expectLines(data, res.Records)
			Expect(res.Records[0].Offset).To(BeNumerically("<", res.Records[1].Offset))
			Expect(res.AtStart).To(BeTrue())
			Expect(res.NotParsed).To(BeZero())

			Expect(find(data, 4, core.ReadOptions{}).Records).To(HaveLen(1))
			Expect(find(data, 471, core.ReadOptions{}).Records).To(BeEmpty())
		})

		It("is not fooled by the job number inside another record's text", func() {
			other := strings.Replace(acctLine(9, 0), `"submit_cmd_line":""`,
				`"submit_cmd_line":"qalter \"job_number\":47, -hold_jid 47"`, 1)
			Expect(other).To(ContainSubstring(`\"job_number\":47`))
			data := []byte(other + "\n" + acctLine(47, 0) + "\n")
			res := find(data, 47, core.ReadOptions{})
			Expect(res.Records).To(HaveLen(1))
			Expect(res.Records[0].Offset).To(Equal(int64(len(other) + 1)))
		})

		It("finds a record whose members are written with spaces or in another order", func() {
			data := []byte(`{"owner":"alice", "job_number" : 47 , "task_number":3}` + "\n" + acctLine(48, 0) + "\n")
			res := find(data, 47, core.ReadOptions{})
			Expect(res.Records).To(HaveLen(1))
			Expect(res.Records[0].TaskID).To(Equal(int64(3)))
		})

		It("finds every record ReadTail reads, however its job_number is written", func() {
			// The shapes another writer than qmaster produces (Python's
			// json.dumps puts a blank after the colon).
			data := []byte(`{"job_number": 1,"owner":"a"}` + "\n" +
				"{\"job_number\":\t1,\"owner\":\"b\"}\n" +
				`{"job_number" : 1 ,"owner":"c"}` + "\n" +
				`{ "job_number":1,"owner":"d"}` + "\n" +
				`{"job_number": 10,"owner":"x"}` + "\n" +
				acctLine(1, 0) + "\n")
			all := tailOf(data, core.ReadOptions{})
			var want []int64
			for _, r := range all.Records {
				if r.JobNumber == 1 {
					want = append(want, r.Offset)
				}
			}
			Expect(want).To(HaveLen(5))
			res := find(data, 1, core.ReadOptions{})
			var got []int64
			for _, r := range res.Records {
				got = append(got, r.Offset)
			}
			Expect(got).To(Equal(want))
			Expect(find(data, 10, core.ReadOptions{}).Records).To(HaveLen(1))
		})

		It("finds records anywhere in a large file", func() {
			data := randomFile(5, 1500)
			for _, id := range []int64{1, 2, 750, 1499, 1500} {
				res := find(data, id, core.ReadOptions{})
				Expect(jobNumbers(res.Records)).To(Equal([]int64{id}), "job %d", id)
				expectLines(data, res.Records)
			}
		})

		It("returns the newest records first to fill MaxRecords", func() {
			var b strings.Builder
			for i := 0; i < 10; i++ {
				b.WriteString(acctLine(47, i) + "\n" + acctLine(100+i, 0) + "\n")
			}
			data := []byte(b.String())
			res := find(data, 47, core.ReadOptions{MaxRecords: 3})
			Expect(res.Records).To(HaveLen(3))
			Expect(res.AtStart).To(BeFalse())
			// The three newest: their padding is 7, 8 and 9 bytes.
			Expect(res.Records[2].SubmitCommandLine).To(HaveLen(9))
			Expect(res.Records[0].SubmitCommandLine).To(HaveLen(7))
		})

		It("says how far back it got when MaxBytes stops it", func() {
			data := randomFile(6, 1500)
			res := find(data, 1, core.ReadOptions{MaxBytes: 256 << 10})
			Expect(res.Records).To(BeEmpty())
			Expect(res.AtStart).To(BeFalse())
			Expect(res.Start).To(BeNumerically(">", 0))
			// Start is the start of a line: the record there can be read.
			at := forwardOf(data, res.Start, core.ReadOptions{MaxRecords: 1})
			Expect(at.Records).To(HaveLen(1))
			Expect(at.Records[0].Offset).To(Equal(res.Start))
		})

		It("counts a line which is too long: the job could be in it", func() {
			data := []byte(acctLine(47, 5000) + "\n" + acctLine(48, 0) + "\n")
			res := find(data, 47, core.ReadOptions{MaxLineBytes: 2000})
			Expect(res.Records).To(BeEmpty())
			Expect(res.NotParsed).To(Equal(1))
		})

		It("returns what it found when the context ends", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			data := acctFile(10)
			_, err := core.FindJob(ctx, bytes.NewReader(data), int64(len(data)), 3, core.ReadOptions{})
			Expect(err).To(MatchError(context.Canceled))
		})
	})
})
