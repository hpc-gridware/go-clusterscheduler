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

// Package fakeqconf provides a reusable bash-script-backed stub for the
// qconf binary. Tests can point CommandLineQConf at the stub's Path(),
// seed a canned stdout body and exit code, then inspect the captured
// argv to assert how the wrapper built its command line.
//
// The stub lives in a per-instance temp directory and is cleaned up via
// Cleanup(). Because the stub is a bash script, Available() reports
// false on Windows; callers must skip accordingly.
package fakeqconf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// TestingT is the subset of testing.TB the helper needs. Any type that
// implements Fatalf and Helper works, including *testing.T, *testing.B,
// and ginkgo.GinkgoT(). A narrow interface keeps the helper decoupled
// from any single testing framework.
type TestingT interface {
	Fatalf(format string, args ...any)
	Helper()
}

// Fake is a test-owned, bash-script-backed stand-in for the qconf
// binary. One Fake == one temp dir containing the script plus an argv
// log file that records every invocation (one line per call, space
// separated).
type Fake struct {
	scriptPath string
	logPath    string
	tmpDir     string
}

// Available reports whether the fake can be constructed on the current
// platform. The stub relies on bash, so Windows is unsupported. Call
// this before New() and skip the test when it returns false.
func Available() bool {
	return runtime.GOOS != "windows"
}

// New creates a fresh Fake whose script emits stdout and exits with
// rc. Setup errors call t.Fatalf; callers do not need to check a
// return value. The caller is responsible for invoking Cleanup
// (typically via defer or t.Cleanup on the *testing.T path).
func New(t TestingT, stdout string, rc int) *Fake {
	t.Helper()
	if !Available() {
		t.Fatalf("fakeqconf.New called on unsupported platform %s", runtime.GOOS)
	}

	dir, err := os.MkdirTemp("", "fake-qconf-*")
	if err != nil {
		t.Fatalf("fakeqconf: MkdirTemp: %v", err)
	}

	logPath := filepath.Join(dir, "argv.log")
	bodyPath := filepath.Join(dir, "stdout.txt")
	if err := os.WriteFile(bodyPath, []byte(stdout), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("fakeqconf: write stdout fixture: %v", err)
	}

	script := fmt.Sprintf(`#!/usr/bin/env bash
printf "%%s\n" "$*" >> %q
cat %q
exit %d
`, logPath, bodyPath, rc)

	scriptPath := filepath.Join(dir, "qconf")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("fakeqconf: write script: %v", err)
	}
	return &Fake{scriptPath: scriptPath, logPath: logPath, tmpDir: dir}
}

// Path returns the absolute path to the fake qconf script. Pass it as
// Executable in core.CommandLineQConfConfig.
func (f *Fake) Path() string {
	if f == nil {
		return ""
	}
	return f.scriptPath
}

// Argv returns the first invocation's argv tokens (space-split). Use
// this when a spec only makes a single call against the fake.
func (f *Fake) Argv() []string {
	lines := f.AllArgvLines()
	if len(lines) == 0 {
		return nil
	}
	return strings.Split(lines[0], " ")
}

// AllArgvLines returns every invocation line captured so far, in call
// order. Each line is the raw "$*" of one call. Returns an empty slice
// when the script has not been invoked yet.
func (f *Fake) AllArgvLines() []string {
	raw, err := os.ReadFile(f.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		// Non-NotExist read errors are genuinely unexpected; surfacing
		// them as an empty slice would hide real bugs, but this
		// package can't fail the test from here. Panic with the error
		// so the failure is unambiguous.
		panic(fmt.Sprintf("fakeqconf: read argv log: %v", err))
	}
	trimmed := strings.TrimRight(string(raw), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// Cleanup removes the fake's temp directory. Safe to call on a nil
// receiver.
func (f *Fake) Cleanup() {
	if f == nil {
		return
	}
	_ = os.RemoveAll(f.tmpDir)
}
