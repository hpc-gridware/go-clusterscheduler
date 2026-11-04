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

import (
	"context"
	"io"
	"time"

	v90 "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

// The streaming XML surface is identical in 9.0 and 9.1: same elements,
// same semantics, same time format (see the 9.0.12, 9.1.0 and 9.1.7
// captures in the v9.0 testdata). The names below are the v9.0 ones.

// Streamer is the bounded-memory, context-aware part of the qstat client.
type Streamer = v90.Streamer

// JobStreamOptions selects what StreamJobsXML asks qstat for.
type JobStreamOptions = v90.JobStreamOptions

// JobXML is one <job_list> element of "qstat -xml".
type JobXML = v90.JobXML

// ResourceRequestXML is one "-r" resource request of a JobXML.
type ResourceRequestXML = v90.ResourceRequestXML

// QueueInstanceXML is one <Queue-List> element of "qstat -f -xml".
type QueueInstanceXML = v90.QueueInstanceXML

// Stats describes one streamed document.
type Stats = v90.Stats

// ErrIncompleteDocument is returned when qstat exited successfully but its
// output did not end with the root close tag.
var ErrIncompleteDocument = v90.ErrIncompleteDocument

// ErrNoSuchJob is returned by SchedulerJobInfoContext and
// JobDetailXMLContext when qstat reports that no requested job exists. It
// is helper.ErrNoSuchJob, shared with v9.0 and qalter.
var ErrNoSuchJob = v90.ErrNoSuchJob

// DecodeJobsXML is the v9.0 decoder; the document shape is unchanged.
func DecodeJobsXML(r io.Reader, fn func(JobXML) error) (Stats, error) {
	return v90.DecodeJobsXML(r, fn)
}

// DecodeQueueInstancesXML is the v9.0 decoder; the document shape is unchanged.
func DecodeQueueInstancesXML(r io.Reader, fn func(QueueInstanceXML) error) (Stats, error) {
	return v90.DecodeQueueInstancesXML(r, fn)
}

// ParseXMLTime parses a JAT_start_time or JAT_submission_time value.
func ParseXMLTime(s string) (time.Time, error) {
	return v90.ParseXMLTime(s)
}

// SchedulerJobInfoContext runs "qstat -j <ids>" through s under ctx and
// parses the answer with THIS version's parser, which understands the 9.1
// per-task block (job_state, exec_host_list, usage per task) and the
// category, department and job-array fields. The v9.0 method of the same
// name on QStatImpl parses with the v9.0 layout; a 9.1 cluster must use this
// function. ErrNoSuchJob is returned when every id is unknown.
func SchedulerJobInfoContext(ctx context.Context, env []string, s Streamer, ids ...string) ([]SchedulerJobInfo, error) {
	out, err := v90.RunSchedulerJobInfo(ctx, env, s, ids...)
	if err != nil {
		return nil, err
	}
	return ParseSchedulerJobInfo(out)
}

// QueueDetailStreamer reads the queue instance detail listing
// ("qstat -f -xml -explain aAcE -s r -q ..."); the shape is unchanged.
type QueueDetailStreamer = v90.QueueDetailStreamer

// QueueDetailOptions selects the queue instances of the detail listing.
// Queues is required unless AllQueues opts in to the whole cluster; both
// set is an error.
type QueueDetailOptions = v90.QueueDetailOptions

// ErrNoQueues is returned by StreamQueueInstanceDetailsXML when
// QueueDetailOptions.Queues is empty and AllQueues is not set.
var ErrNoQueues = v90.ErrNoQueues

// QueueInstanceDetailXML is one <Queue-List> element of the detail listing.
type QueueInstanceDetailXML = v90.QueueInstanceDetailXML

// DecodeQueueInstanceDetailsXML is the v9.0 decoder; the document shape is
// unchanged.
func DecodeQueueInstanceDetailsXML(r io.Reader, fn func(QueueInstanceDetailXML) error) (Stats, error) {
	return v90.DecodeQueueInstanceDetailsXML(r, fn)
}

// QueueResourceStreamer is the queue-level resource part of the qstat
// client ("qstat -F <resources> -q <queues> -xml"); see v9.0.
type QueueResourceStreamer = v90.QueueResourceStreamer

// QueueResourcesXML is one queue instance with its resource values.
type QueueResourcesXML = v90.QueueResourcesXML

// QueueResourceValueXML is one resource value with its dominance.
type QueueResourceValueXML = v90.QueueResourceValueXML

// ErrNoQueueResources is returned when no resource or queue is asked for.
var ErrNoQueueResources = v90.ErrNoQueueResources

// MaxListArgument is the longest list passed to qstat as one argument.
const MaxListArgument = v90.MaxListArgument

// DecodeQueueResourcesXML is the v9.0 decoder; the document shape is unchanged.
func DecodeQueueResourcesXML(r io.Reader, fn func(QueueResourcesXML) error) (Stats, error) {
	return v90.DecodeQueueResourcesXML(r, fn)
}
