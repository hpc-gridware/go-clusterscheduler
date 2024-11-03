package qacct

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

// DefaultQacctFile returns the path to the default accounting file based
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

	// offset points to the last processed line
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
				// TODO parsing can be done in parallel
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

			// store processed offset
			offset, err = file.Seek(0, io.SeekCurrent)
			if err != nil {
				log.Printf("failed to get current offset: %v", err)
				return
			}

			// wait a little before re-scanning for new data and reset scanner
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
