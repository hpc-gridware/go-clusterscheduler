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

// Streaming, context-aware access to "qhost -xml" for large clusters.
//
// NativeSpecification buffers the output, cannot be cancelled, inherits the
// whole parent environment and drops stderr. A poller that reads every host
// of a 2000-host cluster on a cycle needs the opposite: one element at a
// time, under a context deadline, with the child environment it chooses,
// and the qmaster error text when it fails.
// The shape follows pkg/qstat/v9.0/stream.go on purpose.
//
// Without -q, -j or -F, qhost asks qmaster for the execution host list with
// its load values only: no queue instances and no jobs, which is what makes
// a whole-cluster host pass cheap.

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// GlobalHostName is the pseudo-host qhost lists first. It carries the
// cluster-wide consumables, never load values; it is not an execution host.
const GlobalHostName = "global"

// NoValue is what qhost prints for a value it does not have: every value of
// the global pseudo-host, and the load and memory values of a host whose
// execution daemon has not reported within max_unheard.
const NoValue = "-"

// HostStreamer is the bounded-memory, context-aware part of the qhost
// client. *QHostImpl implements it; it is a separate small interface so
// consumers can depend on exactly these two methods and fake them in tests.
type HostStreamer interface {
	// RunContext runs qhost with args under ctx and returns stdout and
	// stderr. When env is non-nil it becomes the child's whole environment;
	// a nil env inherits the parent's. A non-zero exit returns an error of
	// the form "failed to run command (<stderr>): <exit error>" together
	// with the captured output.
	RunContext(ctx context.Context, env []string, args ...string) (stdout, stderr string, err error)
	// StreamHostsXML runs "qhost -xml" and calls fn once per <host>
	// element, in document order, the global pseudo-host included. Nested
	// queue, job and resource elements are skipped.
	StreamHostsXML(ctx context.Context, env []string, fn func(HostXML) error) (HostStats, error)
}

// HostXML is one <host> element of "qhost -xml". Every value is the text as
// printed, so NoValue ("-") stays distinguishable from "0": a host that
// reports a load of 0.00 is up, a host that reports "-" is not reporting.
// Memory values keep their unit suffix ("11.7G").
type HostXML struct {
	Name      string
	Arch      string
	NumProc   string
	Sockets   string
	Cores     string
	Threads   string
	LoadAvg   string
	MemTotal  string
	MemUsed   string
	SwapTotal string
	SwapUsed  string
}

// IsGlobal reports whether the element is the global pseudo-host.
func (h HostXML) IsGlobal() bool {
	return h.Name == GlobalHostName
}

// Reporting reports whether qhost has a load value for the host. qmaster
// keeps a host's last load report until max_unheard has passed without a
// new one (sge_conf(5)); after that qhost prints NoValue. So false means the
// execution daemon has been silent for at least max_unheard, and true means
// it was heard from within it -- not that it is up this second.
func (h HostXML) Reporting() bool {
	v := strings.TrimSpace(h.LoadAvg)
	return v != "" && v != NoValue
}

// HostStats describes one streamed document.
type HostStats struct {
	// Decoded counts host elements handed to the callback.
	Decoded int
	// Skipped counts host elements without a name.
	Skipped int
	// Complete is true when the root close tag was seen. False means the
	// document was cut short and the hosts decoded so far are a prefix of
	// the list, not the list.
	Complete bool
}

// ErrIncompleteHostDocument is returned when qhost's output did not end with
// the root close tag.
var ErrIncompleteHostDocument = errors.New("qhost XML document ended before its root close tag")

// maxStderrBytes bounds the stderr a streamed command may leave behind.
const maxStderrBytes = 64 * 1024

// streamWaitDelay bounds how long Wait blocks after the context is done when
// a grandchild still holds the pipes.
const streamWaitDelay = time.Second

var _ HostStreamer = (*QHostImpl)(nil)

func commandError(stderr fmt.Stringer, cause error) error {
	return fmt.Errorf("failed to run command (%s): %w", strings.TrimSpace(stderr.String()), cause)
}

// RunContext implements HostStreamer.
func (q *QHostImpl) RunContext(ctx context.Context, env []string, args ...string) (string, string, error) {
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return "", "", err
	}
	if q.config.DryRun {
		return fmt.Sprintf("Dry run: qhost %v", args), "", nil
	}
	cmd := exec.CommandContext(ctx, q.config.Executable, args...)
	if env != nil {
		cmd.Env = env
	}
	cmd.WaitDelay = streamWaitDelay
	var stdout bytes.Buffer
	stderr := helper.NewCappedBuffer(maxStderrBytes)
	cmd.Stdout = &stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return stdout.String(), stderr.String(), commandError(stderr, err)
	}
	return stdout.String(), stderr.String(), nil
}

// StreamHostsXML implements HostStreamer.
func (q *QHostImpl) StreamHostsXML(ctx context.Context, env []string, fn func(HostXML) error) (HostStats, error) {
	args := []string{"-xml"}
	if err := validate.Enforce(validate.Args(args...)); err != nil {
		return HostStats{}, err
	}
	if q.config.DryRun {
		return HostStats{Complete: true}, nil
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
		return HostStats{}, fmt.Errorf("failed to start qhost: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return HostStats{}, fmt.Errorf("failed to start qhost: %w", err)
	}
	stats, decodeErr := DecodeHostsXML(stdout, fn)
	if decodeErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()

	// Same precedence as the qstat streams: a complete document is the
	// answer, then the deadline, then the process, then the decoder.
	switch {
	case decodeErr == nil && (waitErr == nil || errors.Is(waitErr, ctx.Err())):
		return stats, nil
	case ctx.Err() != nil:
		return stats, commandError(stderr, ctx.Err())
	case waitErr != nil && (decodeErr == nil || errors.Is(decodeErr, ErrIncompleteHostDocument)):
		return stats, commandError(stderr, waitErr)
	}
	return stats, decodeErr
}

// DecodeHostsXML reads a "qhost -xml" document from r and calls fn once per
// <host> element. It is the pure half of StreamHostsXML and works on
// captured output as well.
func DecodeHostsXML(r io.Reader, fn func(HostXML) error) (HostStats, error) {
	var stats HostStats
	dec := xml.NewDecoder(r)
	// Host names are what the admin configured; treat the bytes as UTF-8
	// and let invalid sequences through rather than fail the whole list.
	dec.Strict = false
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if !stats.Complete {
					return stats, incompleteHostDocument(stats)
				}
				return stats, nil
			}
			return stats, hostDecodeFailure(stats, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "host" {
				var raw hostXML
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return stats, hostDecodeFailure(stats, err)
				}
				h, ok := raw.typed()
				if !ok {
					stats.Skipped++
					continue
				}
				if err := fn(h); err != nil {
					return stats, err
				}
				stats.Decoded++
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

func hostDecodeFailure(stats HostStats, err error) error {
	var syntax *xml.SyntaxError
	if errors.As(err, &syntax) {
		if syntax.Msg == "unexpected EOF" {
			return incompleteHostDocument(stats)
		}
		return fmt.Errorf("malformed qhost XML after %d hosts: %w", stats.Decoded+stats.Skipped, err)
	}
	return err
}

func incompleteHostDocument(stats HostStats) error {
	return fmt.Errorf("after %d hosts: %w", stats.Decoded+stats.Skipped, ErrIncompleteHostDocument)
}

// hostXML is the raw shape of a <host> element. Only direct <hostvalue>
// children are read; <queue>, <job> and <resourcevalue> children are not.
type hostXML struct {
	Name   string `xml:"name,attr"`
	Values []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",chardata"`
	} `xml:"hostvalue"`
}

func (r hostXML) typed() (HostXML, bool) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return HostXML{}, false
	}
	h := HostXML{Name: name}
	for _, v := range r.Values {
		value := strings.TrimSpace(v.Value)
		switch v.Name {
		case "arch_string":
			h.Arch = value
		case "num_proc":
			h.NumProc = value
		case "m_socket":
			h.Sockets = value
		case "m_core":
			h.Cores = value
		case "m_thread":
			h.Threads = value
		case "load_avg":
			h.LoadAvg = value
		case "mem_total":
			h.MemTotal = value
		case "mem_used":
			h.MemUsed = value
		case "swap_total":
			h.SwapTotal = value
		case "swap_used":
			h.SwapUsed = value
		}
	}
	return h, true
}
