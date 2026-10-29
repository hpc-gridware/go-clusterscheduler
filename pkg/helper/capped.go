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

package helper

import "bytes"

// CappedBuffer keeps the first bytes written to it, up to a limit, and
// discards the rest. Every write reports full success, so a child process
// writing to it never blocks on a full pipe. It bounds what a misbehaving
// command may leave in memory (the stderr of the qstat, qhost and qmod
// wrappers).
type CappedBuffer struct {
	buf  bytes.Buffer
	left int
}

// NewCappedBuffer returns a CappedBuffer that keeps at most limit bytes.
func NewCappedBuffer(limit int) *CappedBuffer {
	return &CappedBuffer{left: limit}
}

// Write implements io.Writer.
func (c *CappedBuffer) Write(p []byte) (int, error) {
	if c.left > 0 {
		n := min(len(p), c.left)
		c.buf.Write(p[:n])
		c.left -= n
	}
	return len(p), nil
}

// String returns the bytes kept so far.
func (c *CappedBuffer) String() string {
	return c.buf.String()
}
