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

// The streaming XML surface is identical in 9.1: same elements, same
// semantics, microseconds in the time fields (ParseXMLTime reads both).
type Streamer = v90.Streamer
type JobStreamOptions = v90.JobStreamOptions
type JobXML = v90.JobXML
type QueueInstanceXML = v90.QueueInstanceXML
type Stats = v90.Stats

var (
	ErrIncompleteDocument = v90.ErrIncompleteDocument
	ErrNoSuchJob          = v90.ErrNoSuchJob
)

// DecodeJobsXML is the v9.0 decoder; the document shape is unchanged.
func DecodeJobsXML(r io.Reader, fn func(JobXML) error) (Stats, error) {
	return v90.DecodeJobsXML(r, fn)
}

// DecodeQueueInstancesXML is the v9.0 decoder; the document shape is unchanged.
func DecodeQueueInstancesXML(r io.Reader, fn func(QueueInstanceXML) error) (Stats, error) {
	return v90.DecodeQueueInstancesXML(r, fn)
}

// ParseXMLTime parses a JAT_start_time or JAT_submission_time value.
func ParseXMLTime(s string) (t time.Time, err error) {
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
