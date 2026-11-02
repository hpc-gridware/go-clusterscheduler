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
	// must not be empty. A resource list longer than MaxListArgument is not
	// passed at all: qhost -F then prints every value of every host, and
	// the caller must keep only the names it asked for.
	StreamHostResourcesXML(ctx context.Context, env []string, resources, hosts []string,
		fn func(HostResourcesXML) error) (HostStats, error)
}

var _ ResourceStreamer = (*QHostImpl)(nil)

// ErrNoResources is returned when StreamHostResourcesXML is asked for no
// resource.
var ErrNoResources = errors.New("no resource to ask qhost for")

// MaxListArgument is the longest comma-joined list passed as one argument.
// Linux refuses a single argument above 128 KiB (MAX_ARG_STRLEN) with E2BIG
// before the command runs; 2,000 complex names of 64 characters are past it.
const MaxListArgument = 96 * 1024

// ErrListTooLong is returned for a host list longer than MaxListArgument.
var ErrListTooLong = errors.New("the list is too long for one command argument")

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
	hostList := strings.Join(hosts, ",")
	if len(hostList) > MaxListArgument {
		return HostStats{}, ErrListTooLong
	}
	args := []string{"-xml"}
	list := strings.Join(resources, ",")
	if len(list) <= MaxListArgument {
		args = append(args, "-F", list)
	}
	if len(hosts) > 0 {
		args = append(args, "-h", hostList)
	}
	if len(list) > MaxListArgument {
		// -F takes an optional list, so without one it comes last
		// (verified on 9.1.7: "qhost -xml -h <host> -F" prints every value).
		args = append(args, "-F")
	}
	return q.streamXML(ctx, env, args, func(r io.Reader) (HostStats, error) {
		return DecodeHostResourcesXML(r, fn)
	})
}

// DecodeHostResourcesXML reads a "qhost -F ... -xml" document from r and
// calls fn once per <host> element. It is the pure half of
// StreamHostResourcesXML and works on captured output as well.
//
// The design size is 2,000 hosts x 2,000 resources, 4,000,000 values in
// about 250 MB of XML, so the decoder walks tokens instead of reflecting
// per element, grows one scratch slice across hosts and copies it once per
// host, and hands out one string per distinct name, dominance and (up to
// maxCachedValues) value text. Measured per value: DecodeElement 20
// allocations, this loop 14, which is what encoding/xml's Token alone costs --
// going below that needs a hand-written scanner for this document.
func DecodeHostResourcesXML(r io.Reader, fn func(HostResourcesXML) error) (HostStats, error) {
	d := resourceDecoder{strs: map[string]string{}}
	return decodeHostElements(r, func(dec *xml.Decoder, start xml.StartElement) (bool, error) {
		h, err := d.host(dec, start)
		if err != nil || h.Name == "" {
			return false, err
		}
		return true, fn(h)
	})
}

// maxCachedValues bounds the shared value texts: names and dominances
// repeat on every host, values often do, but a cluster whose every host
// prints a different value must not grow the cache without end.
const maxCachedValues = 1 << 16

type resourceDecoder struct {
	buf    []ResourceValueXML
	strs   map[string]string
	values int
}

// str returns one string per distinct text; bounded counts value texts
// against maxCachedValues.
func (d *resourceDecoder) str(b []byte, bounded bool) string {
	b = trimSpace(b)
	if s, ok := d.strs[string(b)]; ok {
		return s
	}
	return d.keep(string(b), bounded)
}

// attr is str for an attribute value, already a string.
func (d *resourceDecoder) attr(v string) string {
	v = strings.TrimSpace(v)
	if s, ok := d.strs[v]; ok {
		return s
	}
	return d.keep(v, false)
}

func (d *resourceDecoder) keep(s string, bounded bool) string {
	if bounded {
		if d.values >= maxCachedValues {
			return s
		}
		d.values++
	}
	d.strs[s] = s
	return s
}

// host reads one <host> element through its end tag. A host without a name
// returns an empty Name (skipped); only direct <resourcevalue> children are
// read.
func (d *resourceDecoder) host(dec *xml.Decoder, start xml.StartElement) (HostResourcesXML, error) {
	h := HostResourcesXML{}
	for _, a := range start.Attr {
		if a.Name.Local == "name" {
			h.Name = strings.TrimSpace(a.Value)
		}
	}
	d.buf = d.buf[:0]
	depth := 1
	var cur *ResourceValueXML
	var text []byte
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return HostResourcesXML{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 && t.Name.Local == "resourcevalue" {
				d.buf = append(d.buf, ResourceValueXML{})
				cur = &d.buf[len(d.buf)-1]
				text = text[:0]
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "name":
						cur.Name = d.attr(a.Value)
					case "dominance":
						cur.Dominance = d.attr(a.Value)
					}
				}
			}
		case xml.CharData:
			if cur != nil && depth == 2 {
				text = append(text, t...)
			}
		case xml.EndElement:
			if depth == 2 && cur != nil {
				cur.Value = d.str(text, true)
				if cur.Name == "" {
					d.buf = d.buf[:len(d.buf)-1]
				}
				cur = nil
			}
			depth--
		}
	}
	h.Resources = append(make([]ResourceValueXML, 0, len(d.buf)), d.buf...)
	return h, nil
}

// trimSpace is bytes.TrimSpace for the ASCII blanks XML allows.
func trimSpace(b []byte) []byte {
	for len(b) > 0 && isXMLSpace(b[0]) {
		b = b[1:]
	}
	for len(b) > 0 && isXMLSpace(b[len(b)-1]) {
		b = b[:len(b)-1]
	}
	return b
}

func isXMLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
