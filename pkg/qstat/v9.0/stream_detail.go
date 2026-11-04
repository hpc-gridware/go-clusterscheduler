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

// The queue instance DETAIL listing: "qstat -f -xml -explain aAcE -s r",
// usually scoped to a few queue instances (every instance of one host,
// "-q '*@node01'") and, when the caller opts in with AllQueues, the whole
// cluster. Unlike the plain
// queue pass it keeps what the scheduler says about each instance -- why it
// is in alarm, why it is in error or configuration ambiguous -- and the
// running jobs nested in it, with their requests when asked ("-r"). It is
// meant for a request, not a poller.
//
// The reason elements are written by the qstat client
// (source/clients/qstat/ocs_qstat_xml.cc): load-alarm-reason for an a
// state, suspend-alarm-reason for A, and message for the E and c
// explanations.

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// QueueDetailStreamer reads the queue instance detail listing. *QStatImpl
// implements it. It is separate from Streamer so existing fakes of Streamer
// keep compiling.
type QueueDetailStreamer interface {
	StreamQueueInstanceDetailsXML(ctx context.Context, env []string, opts QueueDetailOptions, fn func(QueueInstanceDetailXML) error) (Stats, error)
}

// QueueDetailOptions selects the queue instances of the detail listing.
type QueueDetailOptions struct {
	// Queues is the -q list: cluster queues, queue instances or queue
	// domains ("*@node01"). Required unless AllQueues is set: an empty
	// list returns ErrNoQueues, so a forgotten list never turns into a
	// whole-cluster query, and an empty entry is an error. A list longer
	// than MaxListArgument is asked for in several qstat runs.
	Queues []string
	// AllQueues lists every queue instance of the cluster (no -q). It is
	// the explicit opt-in to the heavy query; Queues must then be empty.
	AllQueues bool
	// Users restricts the nested jobs to these owners; empty means every
	// user ("-u *"). Validated like JobStreamOptions.Users.
	Users []string
	// Requests adds "-r", so the nested jobs carry their hard requests,
	// requested parallel environment and requested queues (JobXML.
	// HardRequests, RequestedPE, HardRequestedQueues and their master and
	// slave scoped counterparts).
	Requests bool
}

// QueueInstanceDetailXML is one <Queue-List> element of the detail listing.
type QueueInstanceDetailXML struct {
	QueueInstanceXML
	// LoadAlarmReasons explain an a state, one line per exceeded load
	// threshold ("alarm hl:np_load_avg=1.2 load-threshold=1.0").
	LoadAlarmReasons []string
	// SuspendAlarmReasons explain an A state.
	SuspendAlarmReasons []string
	// Messages explain an E or c state. They come from qmaster and may name
	// paths and user ids; a consumer that shows them to other users should
	// redact them. They may be empty even for an E state: qmaster does not
	// keep them across a restart (seen on GCS 9.1.4).
	Messages []string
	// Jobs are the running tasks in this instance, QueueName set to it.
	Jobs []JobXML
}

var _ QueueDetailStreamer = (*QStatImpl)(nil)

// ErrNoQueues is returned by StreamQueueInstanceDetailsXML when
// QueueDetailOptions.Queues is empty and AllQueues is not set.
var ErrNoQueues = errors.New("no queue given for the queue instance detail listing")

// StreamQueueInstanceDetailsXML implements QueueDetailStreamer.
//
// A -q list that matches no queue instance makes qstat exit 1 without
// printing anything (9.0.12 to 9.1.7), so the error of a failed run names
// the -q list. A -q list longer than MaxListArgument is split into several
// qstat runs, one after the other; fn sees their instances in order and the
// returned Stats add up all runs. An instance matched by entries that land
// in different runs ("all.q" and "*@node01") is then reported once per run.
func (q *QStatImpl) StreamQueueInstanceDetailsXML(ctx context.Context, env []string, opts QueueDetailOptions, fn func(QueueInstanceDetailXML) error) (Stats, error) {
	switch {
	case opts.AllQueues && len(opts.Queues) > 0:
		return Stats{}, errors.New("queue instance detail: AllQueues and Queues are mutually exclusive")
	case !opts.AllQueues && len(opts.Queues) == 0:
		return Stats{}, ErrNoQueues
	}
	// An empty entry is checked here, not through Enforce: under a relaxed
	// GCS_VALIDATION it would otherwise join to an empty -q list and turn
	// into the whole-cluster query that AllQueues guards.
	for _, queue := range opts.Queues {
		if strings.TrimSpace(queue) == "" {
			return Stats{}, fmt.Errorf("queue instance detail: empty queue entry: %w", ErrNoQueues)
		}
	}
	if err := validate.Enforce(validate.List("queue", opts.Queues)); err != nil {
		return Stats{}, err
	}
	users, err := userList(opts.Users)
	if err != nil {
		return Stats{}, err
	}
	args := []string{"-f", "-xml", "-explain", "aAcE", "-s", "r"}
	if opts.Requests {
		args = append(args, "-r")
	}
	args = append(args, "-u", users)
	decode := func(r io.Reader) (Stats, error) {
		return DecodeQueueInstanceDetailsXML(r, fn)
	}
	if opts.AllQueues {
		return q.StreamXML(ctx, env, args, decode)
	}
	var total Stats
	for _, chunk := range chunkList(opts.Queues, MaxListArgument) {
		queues := strings.Join(chunk, ",")
		st, err := q.StreamXML(ctx, env, append(args[:len(args):len(args)], "-q", queues), decode)
		total.Decoded += st.Decoded
		total.Skipped += st.Skipped
		total.Complete = st.Complete
		if err != nil {
			total.Complete = false
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				err = fmt.Errorf("queue instance detail for %q (an unknown queue fails without a message): %w", queues, err)
			}
			return total, err
		}
	}
	return total, nil
}

// DecodeQueueInstanceDetailsXML reads a "qstat -f -xml -explain" document and
// calls fn once per <Queue-List> element, with its reasons and nested jobs.
func DecodeQueueInstanceDetailsXML(r io.Reader, fn func(QueueInstanceDetailXML) error) (Stats, error) {
	return decodeElements(r, "Queue-List", func(dec *xml.Decoder, start xml.StartElement) (bool, error) {
		var raw queueDetailXML
		if err := dec.DecodeElement(&raw, &start); err != nil {
			return false, err
		}
		qi, ok := raw.queueListXML.typed()
		if !ok {
			return false, nil
		}
		d := QueueInstanceDetailXML{QueueInstanceXML: qi,
			LoadAlarmReasons:    trimAll(raw.LoadAlarm),
			SuspendAlarmReasons: trimAll(raw.SuspendAlarm),
			Messages:            trimAll(raw.Messages),
		}
		for _, j := range raw.Jobs {
			if job, ok := j.typed(); ok {
				// A nested job_list names no queue: it is the enclosing one.
				if job.QueueName == "" {
					job.QueueName = qi.Name
				}
				d.Jobs = append(d.Jobs, job)
			}
		}
		return true, fn(d)
	})
}

// queueDetailXML is the raw shape of a detail <Queue-List> element.
type queueDetailXML struct {
	queueListXML
	LoadAlarm    []string     `xml:"load-alarm-reason"`
	SuspendAlarm []string     `xml:"suspend-alarm-reason"`
	Messages     []string     `xml:"message"`
	Jobs         []jobListXML `xml:"job_list"`
}

func trimAll(in []string) []string {
	var out []string
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
