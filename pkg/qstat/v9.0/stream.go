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
// exist so a consumer (qtelemetry, or any other poller) can read that
// output ONE ELEMENT AT A TIME with a bounded memory footprint, under a
// context deadline, with the child environment it chooses.
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
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
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
	// ("-u *"); without -u qstat lists only the calling user's jobs. A name
	// containing "," or space, or starting with "-", is refused.
	Users []string
	// Extended adds "-ext", which carries project and department per job
	// but drops the start and submission times.
	Extended bool
	// ExtraArgs are appended verbatim, for example "-s", "p" for pending
	// jobs only. Control characters are refused. Switches that change the
	// document ("-f", "-j", "-r") or a second "-u" are passed on as well;
	// keeping the listing a job listing is up to the caller.
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
	// Name and Owner are kept exactly as printed, surrounding blanks
	// included; they are the user's text, not a column.
	Name  string
	Owner string
	// State holds the state letters as qstat prints them ("r", "qw",
	// "hqw", "Eqw", "dr", ...). The alphabet belongs to the scheduler;
	// nothing here interprets it.
	State string
	// StartTime is set for running tasks, SubmissionTime for pending ones;
	// exactly one of the two is non-empty. With Extended both are empty:
	// "qstat -ext" prints the ticket columns instead of the time.
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
	// The ticket and share columns of "qstat -ext", Extended only (zero
	// otherwise): NormalizedTickets is JAT_ntix (the "ntckts" column),
	// Tickets the total, OverrideTickets, FunctionalTickets and
	// ShareTreeTickets its parts (JB_override_tickets, otickets, ftickets,
	// stickets), Share JAT_share. qstat writes them for every job that is
	// not a zombie (ocs_qstat_xml.cc).
	NormalizedTickets float64
	Tickets           uint64
	OverrideTickets   uint64
	OTickets          uint64
	FunctionalTickets uint64
	ShareTreeTickets  uint64
	Share             float64
	// CPUUsage, MemUsage and IOUsage are the scaled usage of a running
	// task as printed (cpu_usage, mem_usage, io_usage), Extended only.
	// qstat writes each one only after the first load report, so the empty
	// string means "not reported yet", never zero.
	CPUUsage string
	MemUsage string
	IOUsage  string
	// Master is "MASTER" or "SLAVE" when qstat prints parallel task
	// distribution ("-g t").
	Master string
	// HardRequests are the job's hard resource requests, one
	// <hard_request name="gpu" ...>1</hard_request> each. They are
	// printed with "-r" only (JobStreamOptions.ExtraArgs or
	// QueueDetailOptions.Requests) and are nil otherwise.
	HardRequests []ResourceRequestXML
	// RequestedPE is the name attribute of <requested_pe>, the parallel
	// environment the job asked for; "-r" only, empty without one.
	RequestedPE string
	// HardRequestedQueues are the <hard_req_queue> entries, the cluster
	// queues or queue instances the job asked for with "-q"; "-r" only.
	HardRequestedQueues []string
	// MasterHardRequests and SlaveHardRequests are the hard requests a
	// parallel job scoped to its master task or to its slave tasks
	// ("qsub -scope master|slave -l ..."), printed as
	// <master_hard_request> and <slave_hard_request>; "-r" only, nil
	// without such a request. HardRequests holds the global ones only, so
	// a caller asking which jobs use a resource reads all three.
	MasterHardRequests []ResourceRequestXML
	SlaveHardRequests  []ResourceRequestXML
	// MasterHardRequestedQueues and SlaveHardRequestedQueues are the
	// scoped "-q" requests, <master_hard_req_queue> and
	// <slave_hard_req_queue>; "-r" only.
	MasterHardRequestedQueues []string
	SlaveHardRequestedQueues  []string
}

// ResourceRequestXML is one resource request of a job as "qstat -r -xml"
// prints it: the complex name and the requested value, both as text.
type ResourceRequestXML struct {
	Name  string
	Value string
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
	// LoadAvg is the <load_avg> text as printed. It is empty in practice:
	// the XML writer (ocs_qstat_xml.cc) prints the element only when the
	// load lookup failed, the inverse of the text column, so no release
	// from 9.0.12 to 9.1.7 (OCS and GCS) prints a real load. Read host load
	// from the qhost stream (pkg/qhost, HostXML.LoadAvg and Reporting).
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
// "Following jobs do not exist" for every requested id. It is
// helper.ErrNoSuchJob, shared with qalter, so errors.Is matches across
// packages.
var ErrNoSuchJob = helper.ErrNoSuchJob

// maxStderrBytes bounds the stderr a streamed command may leave behind. The
// messages that matter ("unable to send message to qmaster", "Following jobs
// do not exist") come first and are short.
const maxStderrBytes = 64 * 1024

// streamWaitDelay bounds how long Wait blocks after the context is done when
// a grandchild still holds the pipes.
const streamWaitDelay = time.Second

var _ Streamer = (*QStatImpl)(nil)

// commandError is the error of a failed qstat run, in the shape the other
// command wrappers of this module use: "failed to run command (<stderr>):
// <cause>".
func commandError(stderr fmt.Stringer, cause error) error {
	return fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), cause)
}

// RunContext implements Streamer.
func (q *QStatImpl) RunContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qstat %v", args), "", nil
	}
	return helper.RunContext(ctx, q.config.Executable, env, nil, args...)
}

// StreamJobsXML implements Streamer.
func (q *QStatImpl) StreamJobsXML(ctx context.Context, env []string, opts JobStreamOptions, fn func(JobXML) error) (Stats, error) {
	args := []string{"-xml"}
	if opts.Extended {
		args = append(args, "-ext")
	}
	users, err := userList(opts.Users)
	if err != nil {
		return Stats{}, err
	}
	args = append(args, "-u", users)
	args = append(args, opts.ExtraArgs...)
	return q.StreamXML(ctx, env, args, func(r io.Reader) (Stats, error) {
		return DecodeJobsXML(r, fn)
	})
}

// StreamQueueInstancesXML implements Streamer.
func (q *QStatImpl) StreamQueueInstancesXML(ctx context.Context, env []string, fn func(QueueInstanceXML) error) (Stats, error) {
	return q.StreamXML(ctx, env, []string{"-f", "-xml"}, func(r io.Reader) (Stats, error) {
		return DecodeQueueInstancesXML(r, fn)
	})
}

// userList is the -u value for users: "*" (every user) when empty, else the
// names comma-joined. Each name is validated first, so a name holding a ","
// or a leading "-" cannot add users ("alice,*") or forge an option.
func userList(users []string) (string, error) {
	if len(users) == 0 {
		return "*", nil
	}
	if err := validate.Enforce(validate.List("user", users)); err != nil {
		return "", err
	}
	return strings.Join(users, ","), nil
}

// StreamXML runs qstat with args and feeds its stdout to decode while the
// process runs, so any qstat XML document can be read with the same process
// handling as the typed streams of this package. The process is killed when
// decode stops early (callback error, malformed document) or when ctx ends.
// decode must read r to EOF on success (the typed decoders do so by reading
// the root close tag); otherwise a qstat with more to say than the pipe
// holds would block until ctx ends. Errors are ranked as the typed streams
// rank them: a complete document wins over a late deadline, the deadline
// over the exit status, the exit status over an incomplete document.
func (q *QStatImpl) StreamXML(ctx context.Context, env []string, args []string, decode func(io.Reader) (Stats, error)) (Stats, error) {
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
	stderr := helper.NewCappedBuffer(maxStderrBytes)
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Stats{}, fmt.Errorf("failed to start qstat: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return Stats{}, fmt.Errorf("failed to start qstat: %w", err)
	}

	// decode returns nil only after reading the root close tag and EOF, so
	// on success there is nothing left to drain. On failure the child is
	// killed and the pipe is NOT drained: a grandchild (a wrapper script's
	// qstat, ssh) may hold the write end long after the child is gone, and
	// Wait closes the read end as soon as the child has exited.
	stats, decodeErr := decode(stdout)
	if decodeErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()

	// Precedence: a complete document is the answer, even when the deadline
	// fired while qstat was exiting. Then the deadline, then the process (a
	// failed qstat prints no document, so the "incomplete document" it
	// leaves behind is a symptom, not the cause), then whatever the decoder
	// refused.
	switch {
	case decodeErr == nil && (waitErr == nil || errors.Is(waitErr, ctx.Err())):
		return stats, nil
	case ctx.Err() != nil:
		return stats, commandError(stderr, ctx.Err())
	case waitErr != nil && (decodeErr == nil || errors.Is(decodeErr, ErrIncompleteDocument)):
		return stats, commandError(stderr, waitErr)
	}
	return stats, decodeErr
}

// SchedulerJobInfoContext runs "qstat -j <ids>" under ctx and parses the
// result with this package's ParseSchedulerJobInfo, which reads the 9.0
// layout. For a 9.1 cluster use the v9.1 package's SchedulerJobInfoContext
// function instead; it parses the per-task block this parser has no field
// for. ErrNoSuchJob is returned when qstat answers "Following jobs do not
// exist" and prints no job.
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
					return stats, incompleteDocument(stats)
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
			return incompleteDocument(stats)
		}
		return fmt.Errorf("malformed qstat XML after %d elements: %w", stats.Decoded+stats.Skipped, err)
	}
	return err
}

// incompleteDocument says how far the document got before it ended.
func incompleteDocument(stats Stats) error {
	return fmt.Errorf("after %d elements: %w", stats.Decoded+stats.Skipped, ErrIncompleteDocument)
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
	// The -ext elements; absent without -ext.
	NormalizedTickets string `xml:"JAT_ntix"`
	Tickets           string `xml:"tickets"`
	OverrideTickets   string `xml:"JB_override_tickets"`
	OTickets          string `xml:"otickets"`
	FunctionalTickets string `xml:"ftickets"`
	ShareTreeTickets  string `xml:"stickets"`
	Share             string `xml:"JAT_share"`
	CPUUsage          string `xml:"cpu_usage"`
	MemUsage          string `xml:"mem_usage"`
	IOUsage           string `xml:"io_usage"`
	// The -r elements; absent without -r.
	// The scoped ones (qsub -scope) are written under their own element
	// names (ocs_qstat_xml.cc, qstat_xml_job_report_hard_resource and
	// qstat_xml_job_hard_requested_queue).
	HardRequests       []namedValueXML `xml:"hard_request"`
	MasterHardRequests []namedValueXML `xml:"master_hard_request"`
	SlaveHardRequests  []namedValueXML `xml:"slave_hard_request"`
	RequestedPE        namedValueXML   `xml:"requested_pe"`
	HardQueues         []string        `xml:"hard_req_queue"`
	MasterHardQueues   []string        `xml:"master_hard_req_queue"`
	SlaveHardQueues    []string        `xml:"slave_hard_req_queue"`
}

// namedValueXML is an element with a name attribute and a text value, the
// shape of <hard_request name="gpu">1</hard_request> and <requested_pe
// name="smp">999</requested_pe>.
type namedValueXML struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
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
	job := JobXML{
		ListState:           r.ListState,
		JobNumber:           id,
		Priority:            prio,
		Name:                r.Name,
		Owner:               r.Owner,
		State:               strings.TrimSpace(r.State),
		StartTime:           strings.TrimSpace(r.StartTime),
		SubmissionTime:      strings.TrimSpace(r.SubmissionTime),
		QueueName:           strings.TrimSpace(r.QueueName),
		Slots:               slots,
		Tasks:               strings.TrimSpace(r.Tasks),
		Project:             r.Project,
		Department:          r.Department,
		Master:              strings.TrimSpace(r.Master),
		RequestedPE:         strings.TrimSpace(r.RequestedPE.Name),
		HardRequestedQueues: trimAll(r.HardQueues),
		HardRequests:        resourceRequests(r.HardRequests),
		MasterHardRequests:  resourceRequests(r.MasterHardRequests),
		SlaveHardRequests:   resourceRequests(r.SlaveHardRequests),

		MasterHardRequestedQueues: trimAll(r.MasterHardQueues),
		SlaveHardRequestedQueues:  trimAll(r.SlaveHardQueues),

		NormalizedTickets: parseFloatOrZero(r.NormalizedTickets),
		Tickets:           parseUintOrZero(r.Tickets),
		OverrideTickets:   parseUintOrZero(r.OverrideTickets),
		OTickets:          parseUintOrZero(r.OTickets),
		FunctionalTickets: parseUintOrZero(r.FunctionalTickets),
		ShareTreeTickets:  parseUintOrZero(r.ShareTreeTickets),
		Share:             parseFloatOrZero(r.Share),
		CPUUsage:          strings.TrimSpace(r.CPUUsage),
		MemUsage:          strings.TrimSpace(r.MemUsage),
		IOUsage:           strings.TrimSpace(r.IOUsage),
	}
	return job, true
}

// parseFloatOrZero and parseUintOrZero read an informational -ext column:
// a value that does not parse is 0 rather than a skipped job, like the
// priority. The job's identity and slots stay strict.
func parseFloatOrZero(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func parseUintOrZero(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v
}

// resourceRequests turns the raw request elements into typed ones,
// dropping those without a name; nil when there is none.
func resourceRequests(in []namedValueXML) []ResourceRequestXML {
	var out []ResourceRequestXML
	for _, h := range in {
		name := strings.TrimSpace(h.Name)
		if name == "" {
			continue
		}
		out = append(out, ResourceRequestXML{Name: name, Value: strings.TrimSpace(h.Value)})
	}
	return out
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
// JAT_submission_time: ISO 8601 without a zone, with microseconds (as
// captured from 9.0.12, 9.1.0 and 9.1.7); the whole-second form is accepted
// for older releases. The value is in the process zone, exactly like
// qstat's text output.
var xmlTimeLayouts = []string{
	"2006-01-02T15:04:05.999999",
	"2006-01-02T15:04:05",
}

// ParseXMLTime parses a JAT_start_time or JAT_submission_time value in the
// local zone. The empty string and "0" (how "qstat -j -xml" prints an unset
// JB_execution_time or JB_deadline) are a zero time without error.
func ParseXMLTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
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
