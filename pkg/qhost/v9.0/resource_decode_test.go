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

package qhost_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qhost "github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/v9.0"
)

// syntheticResources is a qhost -F document of hosts x resources, every
// value distinct when distinct is set (the decoder's cache bound).
func syntheticResources(hosts, resources int, distinct bool) string {
	var b strings.Builder
	b.WriteString("<?xml version='1.0'?>\n<qhost xmlns:xsd=\"x\">\n <host name='global'>\n")
	b.WriteString("   <resourcevalue name='lic' dominance='gc'>60</resourcevalue>\n </host>\n")
	for h := 0; h < hosts; h++ {
		fmt.Fprintf(&b, " <host name='node%04d'>\n   <hostvalue name='arch_string'>lx-amd64</hostvalue>\n", h)
		for r := 0; r < resources; r++ {
			v := r % 7
			if distinct {
				v = h*resources + r
			}
			fmt.Fprintf(&b, "   <resourcevalue name='res%04d' dominance='hc'>%d</resourcevalue>\n", r, v)
		}
		b.WriteString(" </host>\n")
	}
	b.WriteString("</qhost>\n")
	return b.String()
}

var _ = Describe("Decoding qhost -F without reflection", func() {
	decodeAll := func(doc string) []qhost.HostResourcesXML {
		var out []qhost.HostResourcesXML
		stats, err := qhost.DecodeHostResourcesXML(strings.NewReader(doc), func(h qhost.HostResourcesXML) error {
			out = append(out, h)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		return out
	}

	It("keeps every host's values whole when the caller keeps them", func() {
		hosts := decodeAll(syntheticResources(3, 4, false))
		Expect(hosts).To(HaveLen(4))
		Expect(hosts[0].IsGlobal()).To(BeTrue())
		Expect(hosts[0].Resources).To(Equal([]qhost.ResourceValueXML{{Name: "lic", Dominance: "gc", Value: "60"}}))
		Expect(hosts[1].Name).To(Equal("node0000"))
		Expect(hosts[1].Resources).To(HaveLen(4), "a <hostvalue> is not a resource value")
		Expect(hosts[3].Resources[3]).To(Equal(qhost.ResourceValueXML{Name: "res0003", Dominance: "hc", Value: "3"}))
	})

	It("trims blanks, skips a value without a name and a host without one, and reads only direct children", func() {
		hosts := decodeAll(`<qhost><host name=' n1 '>
			<resourcevalue name=' a ' dominance=' hc '>
				4
			</resourcevalue>
			<resourcevalue name='' dominance='hc'>9</resourcevalue>
			<wrapper><resourcevalue name='nested' dominance='hc'>1</resourcevalue></wrapper>
		</host><host name=''><resourcevalue name='b' dominance='hc'>1</resourcevalue></host></qhost>`)
		Expect(hosts).To(Equal([]qhost.HostResourcesXML{{Name: "n1",
			Resources: []qhost.ResourceValueXML{{Name: "a", Dominance: "hc", Value: "4"}}}}))
	})

	It("reads every value right past the value cache's bound", func() {
		// 300 x 300 distinct values is above the 65,536 shared texts.
		hosts := decodeAll(syntheticResources(300, 300, true))
		last := hosts[len(hosts)-1]
		Expect(last.Resources[299].Value).To(Equal(fmt.Sprint(299*300 + 299)))
		Expect(hosts[1].Resources[0].Value).To(Equal("0"))
	})

	It("allocates less than the reflection decoder did", func() {
		doc := syntheticResources(100, 200, false)
		allocs := testing.AllocsPerRun(3, func() {
			_, err := qhost.DecodeHostResourcesXML(strings.NewReader(doc), func(qhost.HostResourcesXML) error { return nil })
			Expect(err).NotTo(HaveOccurred())
		})
		perValue := allocs / (100 * 200)
		GinkgoWriter.Printf("%.0f allocations for 20,000 values: %.2f per value\n", allocs, perValue)
		// Measured on this document: DecodeElement 20 per value,
		// encoding/xml's Token alone 14; this decoder must add nothing to
		// the tokenizer.
		Expect(perValue).To(BeNumerically("<", 15))
	})
})
