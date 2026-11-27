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

package core

import (
	"bytes"
	"context"
	"io"
	"strconv"
)

// Reading the accounting file without reading all of it.
//
// qmaster appends one record per line and never rewrites a line, so the
// newest records are the last lines. A file grows without bound (qmaster
// does not rotate it), and qacct reads it from the first byte on every call.
// ReadTail and ReadForward read only what was asked for: their cost depends
// on the number of records, not on the size of the file.
//
// Three facts of the file decide how they work:
//
//   - The order of the lines is the order qmaster wrote them, which is NOT
//     the order of end_time (a job which never started has end_time 0). The
//     position in the file is the only measure of recency, so nothing here
//     searches by time.
//   - qmaster writes its buffer with one fwrite, which a reader can see half
//     done. A last line without its newline is not a record yet: it is left
//     for the next read, never parsed and never skipped.
//   - A line can be bad (too long, cut by a crash, not JSON). It is counted
//     in NotParsed and passed over; it never fails a read.
//
// The functions take an io.ReaderAt and a size instead of a path, so the
// caller decides which file is opened and how.

// Record is one accounting record and where it is in the file.
//
// It is made by the readers, not for JSON: it embeds JobDetail and with it
// JobDetail's UnmarshalJSON, so a marshalled Record does not read back its
// Offset and Length.
type Record struct {
	JobDetail
	// Offset is the offset of the first byte of the record's line.
	Offset int64
	// Length is the length of the line without its newline.
	Length int
}

// ReadOptions bound a read. The zero value reads without a limit, with
// DefaultMaxLineBytes.
type ReadOptions struct {
	// MaxRecords stops the read after this many records; 0 is no limit.
	MaxRecords int
	// MaxBytes stops the read once this many bytes were read from the
	// file; 0 is no limit. The read stops at a line boundary and never
	// before one line was examined, so more can be read: a read with a
	// limit always gets ahead.
	MaxBytes int64
	// MaxLineBytes is the longest line which is parsed; a longer one is
	// counted in NotParsed. 0 is DefaultMaxLineBytes.
	MaxLineBytes int
	// Lean reads only what a LIST of records shows: the job, task and PE
	// task, who, where, when, how it ended and the wallclock. Extra, the
	// custom usage and every other usage value stay empty. It is about six
	// times faster and allocates a tenth, which is what a window of a
	// hundred thousand records is made of. A line the lean decoder cannot
	// read is read by the full one, so the same lines are records either
	// way. ReadTail and ReadForward only.
	Lean bool
}

// DefaultMaxLineBytes is the longest line read when ReadOptions does not
// say. A record is about 1 KB; qmaster sets no limit on submit_cmd_line.
const DefaultMaxLineBytes = 1 << 20

// readChunkBytes is how much is read from the file at once.
const readChunkBytes = 256 << 10

// ReadResult is the outcome of a read.
type ReadResult struct {
	// Records are in file order (oldest first), whatever the direction of
	// the read.
	Records []Record
	// Start and End bound the part of the file which was examined in whole
	// lines: Start is the offset of the first byte of its oldest line, End
	// the offset after the newline of its newest. A read which continues
	// this one starts at End (forward) or ends at Start (backward).
	Start int64
	End   int64
	// NotParsed counts the lines in that part which are no record: longer
	// than MaxLineBytes or not an accounting record. Empty lines and lines
	// starting with '#' are not counted.
	NotParsed int
	// AtStart is true when the first line of the file was examined.
	AtStart bool
}

func (o ReadOptions) maxLine() int {
	if o.MaxLineBytes > 0 {
		return o.MaxLineBytes
	}
	return DefaultMaxLineBytes
}

// lineKind is what scanning passes to its callback for one line.
type lineKind int

const (
	lineComplete lineKind = iota // a whole line of at most MaxLineBytes
	lineTooLong                  // a whole line longer than that; line is nil
)

// ReadTail reads the last records of the accounting file: the complete
// lines before size, backwards, until MaxRecords records were found,
// MaxBytes were read or the start of the file was reached.
//
// When ctx ends the records found so far are returned together with the
// error of ctx; Start says how far back the read got.
func ReadTail(ctx context.Context, r io.ReaderAt, size int64, o ReadOptions) (ReadResult, error) {
	var res ReadResult
	err := scanBackward(ctx, r, size, o, &res, func(kind lineKind, line []byte, offset int64) bool {
		addRecord(&res, kind, line, offset, o.Lean)
		return o.MaxRecords > 0 && len(res.Records) >= o.MaxRecords
	})
	reverseRecords(res.Records)
	return res, err
}

// FindJob reads the records of one job: like ReadTail, but only lines of
// that job are parsed. It reads the lines themselves first (a comparison of
// bytes) and decodes only the ones which name the job, which is about
// twenty times less work than decoding every record; it still reads the
// file back to its start unless MaxBytes, MaxRecords or ctx stop it.
//
// NotParsed counts lines which are too long (they could hold the job) and
// lines which name the job but are no record.
func FindJob(ctx context.Context, r io.ReaderAt, size int64, jobID int64, o ReadOptions) (ReadResult, error) {
	var res ReadResult
	id := []byte(strconv.FormatInt(jobID, 10))
	err := scanBackward(ctx, r, size, o, &res, func(kind lineKind, line []byte, offset int64) bool {
		if kind == lineTooLong {
			res.NotParsed++
			return false
		}
		if !namesJob(line, id) {
			return false
		}
		job, err := ParseAccountingRecord(line)
		switch {
		case err != nil:
			res.NotParsed++
		case job.JobNumber == jobID:
			res.Records = append(res.Records, Record{JobDetail: job, Offset: offset, Length: len(line)})
		}
		return o.MaxRecords > 0 && len(res.Records) >= o.MaxRecords
	})
	reverseRecords(res.Records)
	return res, err
}

// ReadForward reads the records from offset to size, in file order, until
// MaxRecords records were found or MaxBytes were read. offset has to be the
// start of a line: 0, or the End of an earlier read.
//
// A last line without its newline is not read: End stays in front of it, so
// the next call reads it once qmaster has written the rest.
func ReadForward(ctx context.Context, r io.ReaderAt, offset, size int64, o ReadOptions) (ReadResult, error) {
	res := ReadResult{Start: offset, End: offset, AtStart: offset == 0}
	maxLine := o.maxLine()
	var carry []byte // the head of a line whose newline was not read yet
	skipping := false
	lineStart := offset
	pos := offset
	var read int64
	// One buffer for every read: a line is parsed before the next read
	// overwrites it, and what is carried over is copied.
	buffer := make([]byte, readChunkBytes)
	for pos < size {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		n := int64(readChunkBytes)
		if size-pos < n {
			n = size - pos
		}
		chunk := buffer[:n]
		if got, err := r.ReadAt(chunk, pos); int64(got) != n {
			return res, readError(err)
		}
		read += n
		rest := chunk
		chunkOffset := pos
		pos += n
		for {
			i := bytes.IndexByte(rest, '\n')
			if i < 0 {
				break
			}
			line := rest[:i]
			next := chunkOffset + int64(i) + 1
			switch {
			case skipping:
				skipping = false
				addRecord(&res, lineTooLong, nil, lineStart, o.Lean)
			case len(carry)+len(line) > maxLine:
				addRecord(&res, lineTooLong, nil, lineStart, o.Lean)
			case len(carry) > 0:
				carry = append(carry, line...)
				addRecord(&res, lineComplete, carry, lineStart, o.Lean)
			default:
				addRecord(&res, lineComplete, line, lineStart, o.Lean)
			}
			carry = carry[:0]
			res.End = next
			lineStart = next
			rest = rest[i+1:]
			chunkOffset = next
			if o.MaxRecords > 0 && len(res.Records) >= o.MaxRecords {
				return res, nil
			}
		}
		// rest is the head of a line which continues in the next chunk.
		if !skipping {
			if len(carry)+len(rest) > maxLine {
				skipping = true
				carry = carry[:0]
			} else {
				carry = append(carry, rest...)
			}
		}
		// Only once a line was finished: a limit smaller than a line would
		// otherwise stop every read in front of that line.
		if o.MaxBytes > 0 && read >= o.MaxBytes && res.End > offset {
			break
		}
	}
	return res, nil
}

// scanBackward calls fn for every complete line before size, newest first,
// until fn returns true, the limits of o are reached or the file starts. It
// fills Start, End and AtStart of res; fn fills the rest.
func scanBackward(ctx context.Context, r io.ReaderAt, size int64, o ReadOptions, res *ReadResult,
	fn func(kind lineKind, line []byte, offset int64) (stop bool)) error {

	maxLine := o.maxLine()
	res.Start, res.End = size, size
	// terminated: the newline which ends the line being collected was seen.
	// It is false only for what follows the last newline of the file, which
	// is a line qmaster has not finished: it is dropped.
	terminated := false
	skipping := false // inside a line which is too long
	var carry []byte  // the tail of a line whose start was not read yet
	pos := size
	var read int64
	// One buffer for every read: fn is done with a line before the next
	// read overwrites it, and carry is a copy.
	buffer := make([]byte, readChunkBytes)
	var newlines []int
	for pos > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := int64(readChunkBytes)
		if pos < n {
			n = pos
		}
		start := pos - n
		if need := int(n) + len(carry); cap(buffer) < need {
			buffer = make([]byte, need)
		}
		buf := buffer[:n]
		if got, err := r.ReadAt(buf, start); int64(got) != n {
			return readError(err)
		}
		read += n
		pos = start
		buf = append(buf, carry...)
		carry = carry[:0]

		// The newlines are searched forwards (bytes.IndexByte is several
		// times faster than LastIndexByte) and then walked backwards.
		newlines = newlines[:0]
		for from := 0; ; {
			i := bytes.IndexByte(buf[from:], '\n')
			if i < 0 {
				break
			}
			newlines = append(newlines, from+i)
			from += i + 1
		}
		hi := len(buf)
		for k := len(newlines) - 1; k >= 0; k-- {
			i := newlines[k]
			lineOffset := start + int64(i) + 1
			line := buf[i+1 : hi]
			switch {
			case !terminated:
				// The unfinished last line; the examined part ends here.
				res.End, res.Start = lineOffset, lineOffset
			case skipping, len(line) > maxLine:
				skipping = false
				res.Start = lineOffset
				if fn(lineTooLong, nil, lineOffset) {
					return nil
				}
			default:
				res.Start = lineOffset
				if fn(lineComplete, line, lineOffset) {
					return nil
				}
			}
			terminated = true
			hi = i
		}

		// buf[:hi] is the tail of a line which starts in an earlier chunk,
		// or at the start of the file.
		if !terminated && start > 0 {
			// Still inside the unfinished last line: nothing after start
			// is known to be complete.
			res.Start, res.End = start, start
		}
		if start == 0 {
			if terminated {
				res.Start = 0
				res.AtStart = true
				kind, line := lineComplete, buf[:hi]
				if skipping || hi > maxLine {
					kind, line = lineTooLong, nil
				}
				fn(kind, line, 0)
			} else {
				// No newline in the whole file: nothing is complete.
				res.Start, res.End = 0, 0
				res.AtStart = true
			}
			return nil
		}
		if terminated && !skipping {
			if hi > maxLine {
				skipping = true
			} else {
				carry = append(carry[:0], buf[:hi]...)
			}
		}
		// Only once a line was examined, as in ReadForward.
		if o.MaxBytes > 0 && read >= o.MaxBytes && res.Start < res.End {
			return nil
		}
	}
	if size == 0 {
		res.AtStart = true
	}
	return nil
}

// addRecord parses one line into res. Empty lines and comment lines are
// passed over; every other line which is no record is counted.
func addRecord(res *ReadResult, kind lineKind, line []byte, offset int64, lean bool) {
	if kind == lineTooLong {
		res.NotParsed++
		return
	}
	text := bytes.TrimSpace(line)
	if len(text) == 0 || text[0] == '#' {
		return
	}
	var job JobDetail
	var err error
	if !lean || !parseLean(text, &job) {
		job, err = ParseAccountingRecord(text)
	}
	if err != nil {
		res.NotParsed++
		return
	}
	res.Records = append(res.Records, Record{JobDetail: job, Offset: offset, Length: len(line)})
}

// namesJob tells whether a line has a "job_number" member with this value.
// It is a filter in front of the decoder, so it may say yes too often (the
// text can be inside a string) but never no for a record of the job. It
// reads the number to its end: 47 is not found in 4711 or 147.
func namesJob(line, id []byte) bool {
	// qmaster writes job_number first and without spaces. A line which
	// starts with it and a digit is decided by that member alone: searching
	// the rest of every line for a second one is what a scan of the whole
	// file would spend most of its time on. (A line which names the member
	// twice is not something qmaster writes; it is judged by the first.)
	// Any other shape, a blank after the colon for one, takes the general
	// scan below.
	if v, ok := bytes.CutPrefix(line, firstMemberJobNumber); ok && len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
		return isNumber(v, id)
	}
	key := []byte(`"job_number"`)
	for rest := line; ; {
		i := bytes.Index(rest, key)
		if i < 0 {
			return false
		}
		rest = rest[i+len(key):]
		v := bytes.TrimLeft(rest, " \t")
		if len(v) == 0 || v[0] != ':' {
			continue
		}
		if isNumber(bytes.TrimLeft(v[1:], " \t"), id) {
			return true
		}
	}
}

var firstMemberJobNumber = []byte(`{"job_number":`)

// isNumber tells whether v starts with the number id and not with a longer
// one.
func isNumber(v, id []byte) bool {
	return bytes.HasPrefix(v, id) && (len(v) == len(id) || v[len(id)] < '0' || v[len(id)] > '9')
}

func reverseRecords(records []Record) {
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
}

// readError is the error of a ReadAt which returned fewer bytes than the
// size the caller gave: the file is shorter than that.
func readError(err error) error {
	if err == nil || err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}
