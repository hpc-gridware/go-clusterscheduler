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

package qstat

// Streaming, context-aware access to qstat for large clusters.
//
// NativeSpecification buffers the whole qstat output in memory and cannot be
// cancelled; on a cluster with 200,000 queued or running jobs "qstat -xml
// -u '*'" is tens of megabytes and takes seconds. The functions in this file
// exist so a consumer (Qontrol's workloads indexer, qtelemetry, or any other
// poller) can read that output ONE ELEMENT AT A TIME with a bounded memory
// footprint, under a context deadline, with the child environment it chooses.
//
// Why XML and not the fixed-width text output: plain qstat truncates job
// names to 10 and owners to 12 characters (ocs_qstat.cc), so aggregates by
// owner built from text output merge "administrator1" and "administrator2";
// -xml carries the full JB_name and JB_owner. Job names may also contain
// spaces, so whitespace-split parsing is unsafe. The XML is produced by the
// qstat CLIENT, so the cost on qmaster is identical to the text output.
//
// Two documents are understood:
//
//   - the job listing ("qstat -xml [-u ...]"): one <job_list> element per
//     running task and per pending task range, grouped under <queue_info>
//     (running) and <job_info> (pending), see StreamJobsXML;
//   - the full queue listing ("qstat -f -xml"): one <Queue-List> element per
//     queue instance, each possibly carrying <job_list> children for the jobs
//     of the calling user, see StreamQueueInstancesXML.
//
// Both report in Stats whether the root close tag was seen. A document cut
// short (process killed by the deadline, qmaster going away mid-answer) is
// reported, never silently accepted as a complete listing.

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// Streamer is the bounded-memory, context-aware part of the qstat client.
// *QStatImpl implements it; it is a separate small interface so consumers
// can depend on exactly these three methods and fake them in tests.
type Streamer interface {
	// RunContext runs qstat with args under ctx and returns stdout and
	// stderr. When env is non-nil it becomes the child's whole environment
	// (the caller decides what the child may see); a nil env inherits the
	// parent's. A non-zero exit returns an error of the form
	// "failed to run command (<stderr>): <exit error>" together with the
	// captured output, so callers can still inspect what qstat printed.
	RunContext(ctx context.Context, env []string, args ...string) (stdout, stderr string, err error)
	// StreamJobsXML runs "qstat -xml" with the given options and calls fn
	// once per <job_list> element, in document order (running first, then
	// pending). The document is never held in memory.
	StreamJobsXML(ctx context.Context, env []string, opts JobStreamOptions, fn func(JobXML) error) (Stats, error)
	// StreamQueueInstancesXML runs "qstat -f -xml" and calls fn once per
	// <Queue-List> element (one per queue instance). Job elements nested
	// in a queue are skipped.
	StreamQueueInstancesXML(ctx context.Context, env []string, fn func(QueueInstanceXML) error) (Stats, error)
}

// JobStreamOptions selects what StreamJobsXML asks qstat for.
type JobStreamOptions struct {
	// Users restricts the listing to these owners. Empty means EVERY user
	// ("-u *"); without -u qstat lists only the calling user's jobs.
	Users []string
	// Extended adds "-ext", which carries project and department per job.
	Extended bool
	// ExtraArgs are appended verbatim, for example "-s", "p" for pending
	// jobs only. Control characters are refused.
	ExtraArgs []string
}

// JobXML is one <job_list> element of "qstat -xml": one running task, or
// one pending task range of an array job, or one plain job. Names are the
// FULL names (not the truncated text columns). Time fields are the text as
// printed; ParseXMLTime turns them into a time.Time.
type JobXML struct {
	// ListState is the state attribute of the element: "running",
	// "pending", "zombie", ...
	ListState string
	JobNumber uint64
	Priority  float64
	Name      string
	Owner     string
	// State holds the state letters as qstat prints them ("r", "qw",
	// "hqw", "Eqw", "dr", ...). The alphabet belongs to the scheduler;
	// nothing here interprets it.
	State string
	// StartTime is set for running tasks, SubmissionTime for pending ones;
	// exactly one of the two is non-empty.
	StartTime      string
	SubmissionTime string
	// QueueName is the queue instance ("all.q@node01") for running tasks
	// and empty for pending ones.
	QueueName string
	Slots     uint32
	// Tasks is the task id of a running array task ("361") or the range of
	// pending array tasks ("6001-100000:3", "1,3,5-9"); empty for a plain
	// job. The range grammar is sge_range(5): n[-m[:s]] joined by commas.
	Tasks string
	// Project and Department are present with Extended only.
	Project    string
	Department string
	// Master is "MASTER" or "SLAVE" when qstat prints parallel task
	// distribution ("-g t").
	Master string
}

// QueueInstanceXML is one <Queue-List> element of "qstat -f -xml".
type QueueInstanceXML struct {
	// Name is the queue instance, "all.q@node01".
	Name  string
	QType string
	// Slot counts as printed in the "resv/used/tot." column.
	SlotsUsed     uint32
	SlotsReserved uint32
	SlotsTotal    uint32
	// LoadAvg is the text as printed, empty when the host reports no load
	// (unknown host, or a simulated one).
	LoadAvg string
	Arch    string
	// State holds the queue instance state letters ("au", "E", "d", "D",
	// "c", "o", "s", "S", "u", ...), empty for a healthy instance.
	State string
}

// Stats describes one streamed document.
type Stats struct {
	// Decoded counts elements handed to the callback.
	Decoded int
	// Skipped counts elements that were well-formed XML but did not parse
	// into the typed struct (a non-numeric job number, for example). They
	// are reported, not fatal: one odd element must not abort a listing of
	// 200,000.
	Skipped int
	// Complete is true when the root close tag was seen. False means the
	// document was cut short and the elements decoded so far are a prefix
	// of the listing, not the listing.
	Complete bool
}

// ErrIncompleteDocument is returned when qstat exited successfully but its
// output did not end with the root close tag.
var ErrIncompleteDocument = errors.New("qstat XML document ended before its root close tag")

// ErrNoSuchJob is returned by SchedulerJobInfoContext when qstat reports
// "Following jobs do not exist" for every requested id.
var ErrNoSuchJob = errors.New("job does not exist")

// maxStderrBytes bounds the stderr a streamed command may leave behind. The
// messages that matter ("unable to send message to qmaster", "Following jobs
// do not exist") come first and are short.
const maxStderrBytes = 64 * 1024

// streamWaitDelay bounds how long Wait blocks after the context is done when
// a grandchild still holds the pipes.
const streamWaitDelay = time.Second

// RunContext implements Streamer.
func (q *QStatImpl) RunContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qstat %v", args), "", nil
	}
	cmd := exec.CommandContext(ctx, q.config.Executable, args...)
	if env != nil {
		cmd.Env = env
	}
	cmd.WaitDelay = streamWaitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = newCappedWriter(&stderr, maxStderrBytes)
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return stdout.String(), stderr.String(), fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), ctx.Err())
		}
		return stdout.String(), stderr.String(), fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), err)
	}
	return stdout.String(), stderr.String(), nil
}

// StreamJobsXML implements Streamer.
func (q *QStatImpl) StreamJobsXML(ctx context.Context, env []string, opts JobStreamOptions, fn func(JobXML) error) (Stats, error) {
	args := []string{"-xml"}
	if opts.Extended {
		args = append(args, "-ext")
	}
	users := "*"
	if len(opts.Users) > 0 {
		users = strings.Join(opts.Users, ",")
	}
	args = append(args, "-u", users)
	args = append(args, opts.ExtraArgs...)
	return q.stream(ctx, env, args, func(r io.Reader) (Stats, error) {
		return DecodeJobsXML(r, fn)
	})
}

// StreamQueueInstancesXML implements Streamer.
func (q *QStatImpl) StreamQueueInstancesXML(ctx context.Context, env []string, fn func(QueueInstanceXML) error) (Stats, error) {
	return q.stream(ctx, env, []string{"-f", "-xml"}, func(r io.Reader) (Stats, error) {
		return DecodeQueueInstancesXML(r, fn)
	})
}

// stream runs qstat with args and feeds its stdout to decode while the
// process runs. The process is killed when decode stops early (callback
// error, malformed document) or when ctx ends.
func (q *QStatImpl) stream(ctx context.Context, env []string, args []string, decode func(io.Reader) (Stats, error)) (Stats, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return Stats{}, err
	}
	if q.config.DryRun {
		return Stats{Complete: true}, nil
	}
	cmd := exec.CommandContext(ctx, q.config.Executable, args...)
	if env != nil {
		cmd.Env = env
	}
	cmd.WaitDelay = streamWaitDelay
	var stderr bytes.Buffer
	cmd.Stderr = newCappedWriter(&stderr, maxStderrBytes)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Stats{}, fmt.Errorf("failed to run command (): %w", err)
	}
	if err := cmd.Start(); err != nil {
		return Stats{}, fmt.Errorf("failed to run command (): %w", err)
	}

	stats, decodeErr := decode(stdout)
	if decodeErr != nil {
		// Stop producing: the reader is done with the stream, whatever is
		// left would only block the child on a full pipe.
		_ = cmd.Process.Kill()
	}
	// Drain so a child blocked on write can exit; nothing after a decode
	// error is of interest.
	_, _ = io.Copy(io.Discard, stdout)
	waitErr := cmd.Wait()

	// Precedence: the deadline, then the process (a failed qstat prints no
	// document, so the "incomplete document" it leaves behind is a symptom,
	// not the cause), then whatever the decoder refused.
	switch {
	case ctx.Err() != nil:
		return stats, fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), ctx.Err())
	case waitErr != nil && (decodeErr == nil || errors.Is(decodeErr, ErrIncompleteDocument)):
		return stats, fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), waitErr)
	case decodeErr != nil:
		return stats, decodeErr
	case !stats.Complete:
		return stats, ErrIncompleteDocument
	}
	return stats, nil
}

// SchedulerJobInfoContext runs "qstat -j <ids>" under ctx and parses the
// result with this package's ParseSchedulerJobInfo. ErrNoSuchJob is returned
// when qstat answers "Following jobs do not exist" and prints no job.
func (q *QStatImpl) SchedulerJobInfoContext(ctx context.Context, env []string, ids ...string) ([]SchedulerJobInfo, error) {
	out, err := RunSchedulerJobInfo(ctx, env, q, ids...)
	if err != nil {
		return nil, err
	}
	return ParseSchedulerJobInfo(out)
}

// RunSchedulerJobInfo runs "qstat -j <ids>" and returns the raw text, so a
// version package can apply its own parser. It maps the scheduler's
// "Following jobs do not exist" answer to ErrNoSuchJob when no job block was
// printed at all.
func RunSchedulerJobInfo(ctx context.Context, env []string, s Streamer, ids ...string) (string, error) {
	if len(ids) == 0 {
		return "", errors.New("no job id given")
	}
	stdout, stderr, err := s.RunContext(ctx, env, "-j", strings.Join(ids, ","))
	if err != nil {
		if IsNoSuchJobOutput(stderr) && !strings.Contains(stdout, "job_number:") {
			return "", fmt.Errorf("%w: %s", ErrNoSuchJob, strings.Join(ids, ","))
		}
		return "", err
	}
	return stdout, nil
}

// IsNoSuchJobOutput reports whether text carries qstat's "Following jobs do
// not exist" answer (printed to stderr with exit status 1 under LC_ALL=C).
func IsNoSuchJobOutput(text string) bool {
	return strings.Contains(text, "Following jobs do not exist")
}

// DecodeJobsXML reads a "qstat -xml" document from r and calls fn once per
// <job_list> element, in document order. It is the pure half of
// StreamJobsXML and works on captured output as well.
func DecodeJobsXML(r io.Reader, fn func(JobXML) error) (Stats, error) {
	return decodeElements(r, "job_list", func(dec *xml.Decoder, start xml.StartElement) (bool, error) {
		var raw jobListXML
		if err := dec.DecodeElement(&raw, &start); err != nil {
			return false, err
		}
		job, ok := raw.typed()
		if !ok {
			return false, nil
		}
		return true, fn(job)
	})
}

// DecodeQueueInstancesXML reads a "qstat -f -xml" document from r and calls
// fn once per <Queue-List> element. Nested <job_list> elements (the jobs of
// the calling user) are skipped without being decoded.
func DecodeQueueInstancesXML(r io.Reader, fn func(QueueInstanceXML) error) (Stats, error) {
	return decodeElements(r, "Queue-List", func(dec *xml.Decoder, start xml.StartElement) (bool, error) {
		var raw queueListXML
		if err := dec.DecodeElement(&raw, &start); err != nil {
			return false, err
		}
		qi, ok := raw.typed()
		if !ok {
			return false, nil
		}
		return true, fn(qi)
	})
}

// decodeElements walks the token stream and hands every element named
// name to handle. handle reports whether the element was accepted (counted
// as Decoded) or skipped. The document is complete when the root element's
// close tag has been read.
func decodeElements(r io.Reader, name string, handle func(*xml.Decoder, xml.StartElement) (bool, error)) (Stats, error) {
	var stats Stats
	dec := xml.NewDecoder(r)
	// qstat writes the document in the process locale; it declares no
	// encoding and the names are what the admin typed. Treat the bytes as
	// UTF-8 and let invalid sequences through as text rather than fail.
	dec.Strict = false
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if !stats.Complete {
					return stats, fmt.Errorf("after %d elements: %w", stats.Decoded+stats.Skipped, ErrIncompleteDocument)
				}
				return stats, nil
			}
			return stats, decodeFailure(stats, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == name {
				// DecodeElement consumes through the matching end tag, so
				// the depth is unchanged afterwards.
				ok, err := handle(dec, t)
				if err != nil {
					return stats, decodeFailure(stats, err)
				}
				if ok {
					stats.Decoded++
				} else {
					stats.Skipped++
				}
				continue
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				stats.Complete = true
			}
		}
	}
}

// decodeFailure wraps a token-level error. An unexpected EOF is the document
// ending before its root close tag (the process was killed or qmaster went
// away mid-answer) and is reported as ErrIncompleteDocument; anything else
// is a malformed document. Callback errors pass through unchanged.
func decodeFailure(stats Stats, err error) error {
	var syntax *xml.SyntaxError
	if errors.As(err, &syntax) {
		if syntax.Msg == "unexpected EOF" {
			return fmt.Errorf("after %d elements: %w", stats.Decoded+stats.Skipped, ErrIncompleteDocument)
		}
		return fmt.Errorf("malformed qstat XML after %d elements: %w", stats.Decoded+stats.Skipped, err)
	}
	return err
}

// jobListXML is the raw shape of a <job_list> element. Every field is text
// so that one unparseable value is a skipped element, not a failed stream
// (encoding/xml would otherwise return a conversion error mid-document).
type jobListXML struct {
	ListState      string `xml:"state,attr"`
	JobNumber      string `xml:"JB_job_number"`
	Priority       string `xml:"JAT_prio"`
	Name           string `xml:"JB_name"`
	Owner          string `xml:"JB_owner"`
	State          string `xml:"state"`
	StartTime      string `xml:"JAT_start_time"`
	SubmissionTime string `xml:"JAT_submission_time"`
	QueueName      string `xml:"queue_name"`
	Slots          string `xml:"slots"`
	Tasks          string `xml:"tasks"`
	Project        string `xml:"JB_project"`
	Department     string `xml:"JB_department"`
	Master         string `xml:"master"`
}

func (r jobListXML) typed() (JobXML, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(r.JobNumber), 10, 64)
	if err != nil || id == 0 {
		return JobXML{}, false
	}
	slots, err := parseUint32(r.Slots)
	if err != nil {
		return JobXML{}, false
	}
	prio, _ := strconv.ParseFloat(strings.TrimSpace(r.Priority), 64)
	return JobXML{
		ListState:      r.ListState,
		JobNumber:      id,
		Priority:       prio,
		Name:           r.Name,
		Owner:          r.Owner,
		State:          strings.TrimSpace(r.State),
		StartTime:      strings.TrimSpace(r.StartTime),
		SubmissionTime: strings.TrimSpace(r.SubmissionTime),
		QueueName:      strings.TrimSpace(r.QueueName),
		Slots:          slots,
		Tasks:          strings.TrimSpace(r.Tasks),
		Project:        r.Project,
		Department:     r.Department,
		Master:         strings.TrimSpace(r.Master),
	}, true
}

// queueListXML is the raw shape of a <Queue-List> element. The nested
// job_list children have no field here and are skipped by the decoder.
type queueListXML struct {
	Name       string `xml:"name"`
	QType      string `xml:"qtype"`
	SlotsUsed  string `xml:"slots_used"`
	SlotsResv  string `xml:"slots_resv"`
	SlotsTotal string `xml:"slots_total"`
	LoadAvg    string `xml:"load_avg"`
	Arch       string `xml:"arch"`
	State      string `xml:"state"`
}

func (r queueListXML) typed() (QueueInstanceXML, bool) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return QueueInstanceXML{}, false
	}
	used, err := parseUint32(r.SlotsUsed)
	if err != nil {
		return QueueInstanceXML{}, false
	}
	resv, err := parseUint32(r.SlotsResv)
	if err != nil {
		return QueueInstanceXML{}, false
	}
	total, err := parseUint32(r.SlotsTotal)
	if err != nil {
		return QueueInstanceXML{}, false
	}
	return QueueInstanceXML{
		Name:          name,
		QType:         strings.TrimSpace(r.QType),
		SlotsUsed:     used,
		SlotsReserved: resv,
		SlotsTotal:    total,
		LoadAvg:       strings.TrimSpace(r.LoadAvg),
		Arch:          strings.TrimSpace(r.Arch),
		State:         strings.TrimSpace(r.State),
	}, true
}

// parseUint32 reads a slot count; an empty value is 0.
func parseUint32(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(s, 10, 32)
	return uint32(v), err
}

// xmlTimeLayouts are the forms qstat -xml prints for JAT_start_time and
// JAT_submission_time: ISO 8601 without a zone, with microseconds since
// 9.1 and without them on older releases. The value is in the process
// zone, exactly like qstat's text output.
var xmlTimeLayouts = []string{
	"2006-01-02T15:04:05.999999",
	"2006-01-02T15:04:05",
}

// ParseXMLTime parses a JAT_start_time or JAT_submission_time value in the
// local zone. The empty string is a zero time without error.
func ParseXMLTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	var lastErr error
	for _, layout := range xmlTimeLayouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

// cappedWriter keeps the first n bytes written and drops the rest.
type cappedWriter struct {
	w    *bytes.Buffer
	left int
}

func newCappedWriter(w *bytes.Buffer, n int) *cappedWriter {
	return &cappedWriter{w: w, left: n}
}

func (c *cappedWriter) Write(p []byte) (int, error) {
	if c.left > 0 {
		n := len(p)
		if n > c.left {
			n = c.left
		}
		c.w.Write(p[:n])
		c.left -= n
	}
	return len(p), nil
}
