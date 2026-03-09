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
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-json"
)

// GetDefaultQacctFile returns the path to the default accounting file based
// on the SGE_ROOT and SGE_CELL environment variables.
func GetDefaultQacctFile() string {
	sgeRoot := os.Getenv("SGE_ROOT")
	sgeCell := os.Getenv("SGE_CELL")
	return filepath.Join(sgeRoot, sgeCell, "common", "accounting.jsonl")
}

// WatchFile returns a channel that emits all JobDetail objects from the accounting
// file. It continues to emit JobDetail objects as new lines are added to the file.
// The channel is buffered with the given buffer size.
func WatchFile(ctx context.Context, path string, bufferSize int) (<-chan JobDetail, error) {
	if path == "" {
		path = GetDefaultQacctFile()
	}

	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}

	jobDetailsChan := make(chan JobDetail, bufferSize)

	var offset int64 = 0

	go func() {
		defer file.Close()
		defer close(jobDetailsChan)

		scanner := bufio.NewScanner(file)

		for {
			if _, err := file.Seek(offset, io.SeekStart); err != nil {
				log.Printf("failed to seek to file end: %v", err)
				return
			}

			for scanner.Scan() {
				var job JobDetail
				line := scanner.Text()
				err := json.Unmarshal([]byte(line), &job)
				if err != nil {
					log.Printf("failed to unmarshal line: %v", err)
					continue
				}
				jobDetailsChan <- job
			}

			if err := scanner.Err(); err != nil {
				log.Printf("JSONL parsing error: %v", err)
				return
			}

			offset, err = file.Seek(0, io.SeekCurrent)
			if err != nil {
				log.Printf("failed to get current offset: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				<-time.After(1 * time.Second)
				scanner = bufio.NewScanner(file)
			}
		}
	}()

	return jobDetailsChan, nil
}
