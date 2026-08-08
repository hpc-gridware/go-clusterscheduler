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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeBinary writes an executable shell script and returns its path. Used to
// stand in for a client binary so the tests exercise the real exec path --
// timeout, exit code, stream selection -- without needing a cluster.
func fakeBinary(t *testing.T, name, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fixtures are POSIX-only")
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// The banner is on the first line of -help and is what every supported
// product emits. Captured verbatim from a live OCS 9.0.10 container.
func TestGetVersionReadsTheBanner(t *testing.T) {
	bin := fakeBinary(t, "qconf", `echo "OCS 9.0.10 (131225-1739)"; exit 0`)
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: bin, Timeout: 5 * time.Second}}

	v, err := q.GetVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Version != "9.0.10" || v.Major != 9 || v.Minor != 0 {
		t.Fatalf("got %+v", v)
	}
}

// "-help" exits non-zero on some products and versions. The banner is still
// on stdout, so the exit code must not be consulted.
func TestGetVersionIgnoresANonZeroExit(t *testing.T) {
	bin := fakeBinary(t, "qconf", `echo "GCS 9.1.0 (130126-1240)"; exit 1`)
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: bin, Timeout: 5 * time.Second}}

	v, err := q.GetVersion()
	if err != nil {
		t.Fatalf("a non-zero exit must not hide the banner: %v", err)
	}
	if v.Major != 9 || v.Minor != 1 {
		t.Fatalf("got %+v", v)
	}
}

// The CONFIGURED binary is authoritative. Before this, GetVersion always ran
// qhost -- derived from the qconf path by string substitution -- so it
// reported the wrong program, and failed outright on hosts where qhost is
// not installed beside qconf.
func TestGetVersionPrefersTheConfiguredBinary(t *testing.T) {
	dir := t.TempDir()
	qconfPath := filepath.Join(dir, "qconf")
	qhostPath := filepath.Join(dir, "qhost")
	for path, out := range map[string]string{
		qconfPath: "OCS 9.0.10 (aaa)",
		qhostPath: "OCS 9.1.4 (bbb)",
	} {
		if err := os.WriteFile(path,
			[]byte("#!/bin/sh\necho \""+out+"\"\n"), 0o755); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: qconfPath, Timeout: 5 * time.Second}}

	v, err := q.GetVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Version != "9.0.10" {
		t.Fatalf("want the configured binary's version 9.0.10, got %q "+
			"(that is qhost's -- the configured executable must win)", v.Version)
	}
}

// ...but qhost remains a fallback, so a deployment where only qhost answers
// keeps working.
func TestGetVersionFallsBackToQhost(t *testing.T) {
	dir := t.TempDir()
	qconfPath := filepath.Join(dir, "qconf")
	qhostPath := filepath.Join(dir, "qhost")
	// qconf produces nothing at all (e.g. not installed on this host).
	if err := os.WriteFile(qconfPath, []byte("#!/bin/sh\nexit 127\n"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.WriteFile(qhostPath,
		[]byte("#!/bin/sh\necho \"OCS 9.1.4 (bbb)\"\n"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: qconfPath, Timeout: 5 * time.Second}}

	v, err := q.GetVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Version != "9.1.4" {
		t.Fatalf("want qhost's 9.1.4 as the fallback, got %q", v.Version)
	}
}

// A client binary that hangs -- on an unreachable shared filesystem, say --
// must not block the caller forever. This is the command most likely to run
// at startup, before anything else has had a chance to fail fast.
func TestGetVersionHonoursTheTimeout(t *testing.T) {
	bin := fakeBinary(t, "qconf", `sleep 30`)
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: bin, Timeout: 300 * time.Millisecond}}

	start := time.Now()
	_, err := q.GetVersion()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a hanging binary must produce an error, not block")
	}
	// Both candidates are tried, so allow for two timeouts plus slack.
	if elapsed > 5*time.Second {
		t.Fatalf("GetVersion blocked for %s; the configured Timeout was ignored", elapsed)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("the error should name the timeout, got %v", err)
	}
}

func TestGetVersionReportsNoOutput(t *testing.T) {
	bin := fakeBinary(t, "qconf", `exit 0`)
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: bin, Timeout: 5 * time.Second}}

	if _, err := q.GetVersion(); err == nil {
		t.Fatal("a binary producing no output must be an error")
	}
}

// DryRun must not spawn anything.
func TestGetVersionDryRun(t *testing.T) {
	bin := fakeBinary(t, "qconf", `echo "OCS 9.0.10"; touch `+
		filepath.Join(t.TempDir(), "ran"))
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: bin, DryRun: true, Timeout: 5 * time.Second}}

	v, err := q.GetVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Version != "" {
		t.Fatalf("dry run must not report a version, got %+v", v)
	}
}

func TestParseVersionInfoAcrossProducts(t *testing.T) {
	tests := []struct {
		in           string
		major, minor int
		version      string
		wantErr      bool
	}{
		{in: "OCS 9.0.10 (131225-1739)", major: 9, minor: 0, version: "9.0.10"},
		{in: "GCS 9.1.0beta1 (130126-1240)", major: 9, minor: 1, version: "9.1.0beta1"},
		{in: "OCS 9.0.7", major: 9, minor: 0, version: "9.0.7"},
		{in: "SGE 8.1.9 (12345-6789)", major: 8, minor: 1, version: "8.1.9"},
		// The banner is the FIRST line; usage text follows it.
		{in: "OCS 9.0.10 (x)\nusage: qconf [options]\n", major: 9, minor: 0, version: "9.0.10"},
		{in: "", wantErr: true},
		{in: "\n\n", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(strings.SplitN(tc.in, "\n", 2)[0], func(t *testing.T) {
			v, err := ParseVersionInfo(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if v.Major != tc.major || v.Minor != tc.minor || v.Version != tc.version {
				t.Fatalf("got %+v, want %d.%d / %q", v, tc.major, tc.minor, tc.version)
			}
		})
	}
}

// TestRunCommandHonoursTheTimeout guards the same trap as
// TestGetVersionHonoursTheTimeout, on the path every other qconf call uses.
//
// CommandContext alone is not enough: it kills the direct child, but Run
// keeps waiting on the stdout pipe, which a grandchild inherits. Before
// WaitDelay, a client that spawned a helper blocked for the helper's whole
// lifetime regardless of the configured Timeout -- which silently defeats
// any caller that sets one to stay inside its own request deadline.
func TestRunCommandHonoursTheTimeout(t *testing.T) {
	bin := fakeBinary(t, "qconf", `sleep 30 & wait`)
	q := &CommandLineQConf{config: CommandLineQConfConfig{
		Executable: bin, Timeout: 300 * time.Millisecond}}

	start := time.Now()
	_, err := q.RunCommand("-sql")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a hanging command must produce an error")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("RunCommand blocked for %s; the configured Timeout was ignored", elapsed)
	}
}
