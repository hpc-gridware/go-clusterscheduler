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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GetDefaultQacctFile returns the path to the default accounting file based
// on the SGE_ROOT and SGE_CELL environment variables.
func GetDefaultQacctFile() string {
	sgeRoot := os.Getenv("SGE_ROOT")
	sgeCell := os.Getenv("SGE_CELL")
	return filepath.Join(sgeRoot, sgeCell, "common", "accounting.jsonl")
}

// WatchOptions say how WatchFileFrom follows the accounting file.
type WatchOptions struct {
	// BufferSize is the size of the channel's buffer.
	BufferSize int
	// FromEnd starts with the records written after the call. Without it
	// every record already in the file is emitted first, which reads the
	// whole file.
	FromEnd bool
	// PollInterval is how often the file is looked at; 0 is one second.
	PollInterval time.Duration
	// MaxLineBytes is the longest line which is parsed; 0 is
	// DefaultMaxLineBytes.
	MaxLineBytes int
	// OnError, when set, is called for what the watch passes over or
	// retries: lines which are no record (as a count) and read errors.
	OnError func(error)
}

// watchBatchRecords bounds how many records are read from the file before
// they are sent, so a watch from the start of a large file does not hold
// the whole file in memory.
const watchBatchRecords = 1000

// WatchFile returns a channel that emits all JobDetail objects from the accounting
// file. It continues to emit JobDetail objects as new lines are added to the file.
// The channel is buffered with the given buffer size.
//
// It is WatchFileFrom from the start of the file; see there. A line which
// is no record is passed over without a word (earlier versions logged it);
// use WatchFileFrom with OnError to be told.
func WatchFile(ctx context.Context, path string, bufferSize int) (<-chan JobDetail, error) {
	return WatchFileFrom(ctx, path, WatchOptions{BufferSize: bufferSize})
}

// WatchFileFrom returns a channel which emits the records of the accounting
// file as qmaster writes them, until ctx ends; the channel is closed then.
//
//   - A record is emitted once its line is complete. A line qmaster has
//     written half of is waited for, not skipped.
//   - A line which is no record (too long, not JSON) is passed over and the
//     watch goes on.
//   - When the file is replaced (rotation) the rest of the old file is
//     emitted, then the new file from its start. A file which got shorter
//     is read from its start as well.
//   - Sending stops when ctx ends, also while the consumer does not read.
func WatchFileFrom(ctx context.Context, path string, o WatchOptions) (<-chan JobDetail, error) {
	if path == "" {
		path = GetDefaultQacctFile()
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	var offset int64
	if o.FromEnd {
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to stat file: %v", err)
		}
		// The end of the last complete line: a line qmaster is writing
		// right now belongs to the watch.
		tail, err := ReadTail(ctx, file, info.Size(), ReadOptions{MaxRecords: 1, MaxLineBytes: o.MaxLineBytes})
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		offset = tail.End
	}
	interval := o.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	report := func(err error) {
		if o.OnError != nil && err != nil {
			o.OnError(err)
		}
	}

	out := make(chan JobDetail, o.BufferSize)
	go func() {
		defer close(out)
		defer func() { file.Close() }()

		// drain emits what the open file holds after offset. It returns
		// false when ctx ended.
		drain := func() bool {
			for {
				info, err := file.Stat()
				if err != nil {
					report(err)
					return true
				}
				if info.Size() < offset {
					offset = 0 // the file was truncated
				}
				if info.Size() == offset {
					return true
				}
				res, err := ReadForward(ctx, file, offset, info.Size(),
					ReadOptions{MaxRecords: watchBatchRecords, MaxLineBytes: o.MaxLineBytes})
				if res.NotParsed > 0 {
					report(fmt.Errorf("%d lines of %s are no accounting record", res.NotParsed, path))
				}
				for _, record := range res.Records {
					select {
					case out <- record.JobDetail:
					case <-ctx.Done():
						return false
					}
				}
				if err != nil {
					// What was read is behind us: without this the next
					// look would emit the same records again.
					offset = res.End
					report(err)
					return ctx.Err() == nil
				}
				if res.End == offset {
					return true // only an unfinished line is left
				}
				offset = res.End
			}
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if !drain() {
				return
			}
			// Rotation: the path names another file now. What the old
			// one still got was emitted by the drain above, or is by the
			// one below.
			if current, err := os.Stat(path); err == nil {
				if opened, err := file.Stat(); err == nil && !os.SameFile(current, opened) {
					if !drain() {
						return
					}
					if next, err := os.Open(path); err == nil {
						file.Close()
						file, offset = next, 0
						continue
					} else {
						report(err)
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return out, nil
}
