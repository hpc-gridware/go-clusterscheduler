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

// Streaming access to "qstat -F <resources> -q <queues> -xml": what is left
// of each listed resource in every queue instance of the listed cluster
// queues, with its dominance (qc: capacity set in the queue's
// complex_values, qf: a fixed value; a host or global value that is
// smaller wins and is printed as hc:/gc:). Only qstat reports queue-level
// values; qhost -F never does. Without -q, qstat prints every queue
// instance of the cluster, so callers restrict it to the queues that set a
// resource.

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// QueueResourceStreamer is the queue-level resource part of the qstat
// client. *QStatImpl implements it.
type QueueResourceStreamer interface {
	// StreamQueueResourcesXML runs "qstat -F <resources> -q <queues> -xml"
	// and calls fn once per <Queue-List> element (one queue instance).
	// resources and queues must not be empty. Lists longer than
	// MaxListArgument are handled as that constant says.
	StreamQueueResourcesXML(ctx context.Context, env []string, resources, queues []string,
		fn func(QueueResourcesXML) error) (Stats, error)
}

var _ QueueResourceStreamer = (*QStatImpl)(nil)

// ErrNoQueueResources is returned when no resource or no queue is asked for.
var ErrNoQueueResources = errors.New("no resource or no queue to ask qstat for")

// QueueResourcesXML is one queue instance with its state letters and the
// listed resources' values.
type QueueResourcesXML struct {
	// Name is the queue instance, "all.q@node01"; State its state letters
	// ("d", "au", ...), empty for a healthy instance.
	Name      string
	State     string
	Resources []QueueResourceValueXML
}

// QueueResourceValueXML is one <resource name type>value</resource>: what
// is left, as printed.
type QueueResourceValueXML struct {
	Name string
	// Dominance is the type attribute, two letters as in qhost: the scope
	// ('g' global, 'h' host, 'q' queue) and the source ('c' consumable
	// capacity left, 'l' load report, 'f' fixed value), for example "qc".
	Dominance string
	Value     string
}

// StreamQueueResourcesXML implements QueueResourceStreamer.
func (q *QStatImpl) StreamQueueResourcesXML(ctx context.Context, env []string, resources, queues []string,
	fn func(QueueResourcesXML) error) (Stats, error) {
	if len(resources) == 0 || len(queues) == 0 {
		return Stats{}, ErrNoQueueResources
	}
	if err := validate.Enforce(validate.List("resource", resources), validate.List("queue", queues)); err != nil {
		return Stats{}, err
	}
	list := strings.Join(resources, ",")
	var total Stats
	for _, chunk := range chunkList(queues, MaxListArgument) {
		args := []string{"-q", strings.Join(chunk, ","), "-xml", "-F"}
		if len(list) <= MaxListArgument {
			args = []string{"-F", list, "-q", strings.Join(chunk, ","), "-xml"}
		}
		st, err := q.stream(ctx, env, args, func(r io.Reader) (Stats, error) {
			return DecodeQueueResourcesXML(r, fn)
		})
		total.Decoded += st.Decoded
		total.Skipped += st.Skipped
		total.Complete = st.Complete
		if err != nil {
			total.Complete = false
			return total, err
		}
	}
	return total, nil
}

// MaxListArgument is the longest comma-joined list passed as one argument.
// Linux refuses a single argument above 128 KiB (MAX_ARG_STRLEN) with E2BIG
// before the command runs. A longer resource list is left out (qstat -F then
// prints every value, the caller keeps the names it asked for); a longer
// queue list is split into several calls -- the queue instances of
// different queues never overlap, so nothing is read twice.
const MaxListArgument = 96 * 1024

// chunkList splits list into runs whose comma-joined length stays within
// max (an element longer than max is a run of its own).
func chunkList(list []string, max int) [][]string {
	var out [][]string
	start, size := 0, 0
	for i, s := range list {
		add := len(s)
		if i > start {
			add++
		}
		if i > start && size+add > max {
			out = append(out, list[start:i])
			start, size, add = i, 0, len(s)
		}
		size += add
	}
	return append(out, list[start:])
}

// DecodeQueueResourcesXML reads a "qstat -F ... -xml" document from r and
// calls fn once per <Queue-List> element. Running jobs inside an instance
// and the pending job list are not read.
func DecodeQueueResourcesXML(r io.Reader, fn func(QueueResourcesXML) error) (Stats, error) {
	return decodeElements(r, "Queue-List", func(dec *xml.Decoder, start xml.StartElement) (bool, error) {
		var raw queueResourcesXML
		if err := dec.DecodeElement(&raw, &start); err != nil {
			return false, err
		}
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			return false, nil
		}
		out := QueueResourcesXML{Name: name, State: strings.TrimSpace(raw.State),
			Resources: make([]QueueResourceValueXML, 0, len(raw.Values))}
		for _, v := range raw.Values {
			rn := strings.TrimSpace(v.Name)
			if rn == "" {
				continue
			}
			out.Resources = append(out.Resources, QueueResourceValueXML{Name: rn,
				Dominance: strings.TrimSpace(v.Type), Value: strings.TrimSpace(v.Value)})
		}
		return true, fn(out)
	})
}

// queueResourcesXML is the raw shape of a <Queue-List> element with -F;
// only its direct name, state and resource children are read.
type queueResourcesXML struct {
	Name   string `xml:"name"`
	State  string `xml:"state"`
	Values []struct {
		Name  string `xml:"name,attr"`
		Type  string `xml:"type,attr"`
		Value string `xml:",chardata"`
	} `xml:"resource"`
}
