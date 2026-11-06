/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
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
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// incompleteQConf answers GetClusterConfiguration with a configuration that
// skipped a host; every other method is the nil embedded interface, so a
// call Apply must not make panics the spec.
type incompleteQConf struct {
	QConf
	cfg ClusterConfig
}

func (q incompleteQConf) GetClusterConfiguration() (ClusterConfig, error) {
	return q.cfg, nil
}

var _ = Describe("hosts that cannot be read while collecting the configuration", func() {
	It("are reported in SkippedHosts, never dropped silently", func() {
		read := func(name string) (HostExecConfig, error) {
			if name == "node5" {
				return HostExecConfig{}, fmt.Errorf("error: can't resolve hostname %q", name)
			}
			return HostExecConfig{Name: name}, nil
		}
		got, skipped := readHosts([]string{"node1", "node5", "node2"}, SectionExecHosts, read)
		Expect(got).To(HaveLen(2))
		Expect(got).To(HaveKey("node1"))
		Expect(got).To(HaveKey("node2"))
		Expect(skipped).To(Equal([]SkippedHost{{
			Name: "node5", Section: SectionExecHosts, Reason: `error: can't resolve hostname "node5"`,
		}}))
	})

	It("leave SkippedHosts empty for a complete read", func() {
		got, skipped := readHosts([]string{"a"}, SectionHostConfigurations,
			func(name string) (HostConfiguration, error) { return HostConfiguration{Name: name}, nil })
		Expect(got).To(HaveLen(1))
		Expect(skipped).To(BeEmpty())
	})

	It("make Apply refuse to plan against the incomplete configuration", func() {
		qc := incompleteQConf{cfg: ClusterConfig{SkippedHosts: []SkippedHost{
			{Name: "node5", Section: SectionExecHosts, Reason: "timeout"},
		}}}
		err := Apply(qc, ClusterConfig{}, true)
		Expect(errors.Is(err, ErrIncompleteConfiguration)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("node5"))
	})
})
