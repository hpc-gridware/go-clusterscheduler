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

// Streaming access to "qhost -F <resources> -xml": what is left of each
// listed resource, per host, with its dominance.
//
// qhost prints the global pseudo-host first; its values are the cluster-wide
// pools. Every host then repeats those cluster-wide values, so a caller that
// lists cluster-wide resources should restrict the hosts (any one host is
// enough: the global block is printed whatever -h selects). The values are
// what the scheduler can still grant, never capacity: a consumable shows
// capacity minus bookings, and a load report wins when it is smaller.

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper/validate"
)

// ResourceStreamer is the resource part of the qhost client. *QHostImpl
// implements it.
type ResourceStreamer interface {
	// StreamHostResourcesXML runs "qhost -F <resources> -xml" (with
	// "-h <hosts>" when hosts is not empty) and calls fn once per <host>
	// element, in document order, the global pseudo-host first. resources
	// must not be empty: qhost -F without a list prints every value of every
	// host.
	StreamHostResourcesXML(ctx context.Context, env []string, resources, hosts []string,
		fn func(HostResourcesXML) error) (HostStats, error)
}

var _ ResourceStreamer = (*QHostImpl)(nil)

// ErrNoResources is returned when StreamHostResourcesXML is asked for no
// resource.
var ErrNoResources = errors.New("no resource to ask qhost for")

// HostResourcesXML is one <host> element with its <resourcevalue> children.
type HostResourcesXML struct {
	Name      string
	Resources []ResourceValueXML
}

// IsGlobal reports whether the element is the global pseudo-host.
func (h HostResourcesXML) IsGlobal() bool {
	return h.Name == GlobalHostName
}

// ResourceValueXML is one <resourcevalue name dominance>value</resourcevalue>:
// what is left of the resource, as printed (memory keeps its K/M/G suffix,
// a fixed boolean reads "1.000000").
type ResourceValueXML struct {
	Name string
	// Dominance is two letters: the scope ('g' global, 'h' host, 'q'
	// queue) and the source ('c' consumable capacity left, 'l' load
	// report, 'f' fixed value), for example "hc" or "gl".
	Dominance string
	Value     string
}

// StreamHostResourcesXML implements ResourceStreamer.
func (q *QHostImpl) StreamHostResourcesXML(ctx context.Context, env []string, resources, hosts []string,
	fn func(HostResourcesXML) error) (HostStats, error) {
	if len(resources) == 0 {
		return HostStats{}, ErrNoResources
	}
	// Each name and host is one list element: no leading "-" (it would read
	// as an option), no "," or whitespace (it would add elements).
	if err := validate.Enforce(validate.List("resource", resources), validate.List("host", hosts)); err != nil {
		return HostStats{}, err
	}
	args := []string{"-xml", "-F", strings.Join(resources, ",")}
	if len(hosts) > 0 {
		args = append(args, "-h", strings.Join(hosts, ","))
	}
	return q.streamXML(ctx, env, args, func(r io.Reader) (HostStats, error) {
		return DecodeHostResourcesXML(r, fn)
	})
}

// DecodeHostResourcesXML reads a "qhost -F ... -xml" document from r and
// calls fn once per <host> element. It is the pure half of
// StreamHostResourcesXML and works on captured output as well.
func DecodeHostResourcesXML(r io.Reader, fn func(HostResourcesXML) error) (HostStats, error) {
	return decodeHostElements(r, func(dec *xml.Decoder, start xml.StartElement) (bool, error) {
		var raw hostResourcesXML
		if err := dec.DecodeElement(&raw, &start); err != nil {
			return false, err
		}
		h, ok := raw.typed()
		if !ok {
			return false, nil
		}
		return true, fn(h)
	})
}

// hostResourcesXML is the raw shape of a <host> element; only direct
// <resourcevalue> children are read.
type hostResourcesXML struct {
	Name   string `xml:"name,attr"`
	Values []struct {
		Name      string `xml:"name,attr"`
		Dominance string `xml:"dominance,attr"`
		Value     string `xml:",chardata"`
	} `xml:"resourcevalue"`
}

func (r hostResourcesXML) typed() (HostResourcesXML, bool) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return HostResourcesXML{}, false
	}
	h := HostResourcesXML{Name: name, Resources: make([]ResourceValueXML, 0, len(r.Values))}
	for _, v := range r.Values {
		rn := strings.TrimSpace(v.Name)
		if rn == "" {
			continue
		}
		h.Resources = append(h.Resources, ResourceValueXML{Name: rn, Dominance: strings.TrimSpace(v.Dominance),
			Value: strings.TrimSpace(v.Value)})
	}
	return h, true
}
