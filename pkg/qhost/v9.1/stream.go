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

package qhost

import (
	"io"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/core"
)

// The streaming XML surface is the core one; see core/stream.go.

// HostStreamer is the bounded-memory, context-aware part of the qhost client.
type HostStreamer = core.HostStreamer

// HostXML is one <host> element of "qhost -xml".
type HostXML = core.HostXML

// HostStats describes one streamed document.
type HostStats = core.HostStats

// GlobalHostName is the pseudo-host qhost lists first.
const GlobalHostName = core.GlobalHostName

// NoValue is what qhost prints for a value it does not have.
const NoValue = core.NoValue

// ErrIncompleteHostDocument is returned when qhost's output did not end with
// the root close tag.
var ErrIncompleteHostDocument = core.ErrIncompleteHostDocument

// DecodeHostsXML reads a "qhost -xml" document and calls fn once per host.
func DecodeHostsXML(r io.Reader, fn func(HostXML) error) (HostStats, error) {
	return core.DecodeHostsXML(r, fn)
}

// ResourceStreamer is the resource part of the qhost client
// ("qhost -F <resources> -xml"); see core/resource_stream.go.
type ResourceStreamer = core.ResourceStreamer

// HostResourcesXML is one <host> element with its resource values.
type HostResourcesXML = core.HostResourcesXML

// ResourceValueXML is one resource value with its dominance.
type ResourceValueXML = core.ResourceValueXML

// ErrNoResources is returned when no resource is asked for.
var ErrNoResources = core.ErrNoResources

// DecodeHostResourcesXML reads a "qhost -F ... -xml" document and calls fn
// once per host.
func DecodeHostResourcesXML(r io.Reader, fn func(HostResourcesXML) error) (HostStats, error) {
	return core.DecodeHostResourcesXML(r, fn)
}
