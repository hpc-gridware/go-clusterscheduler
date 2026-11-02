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
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qhost "github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/v9.1"
)

var _ = Describe("Streaming qhost XML through the 9.1 aliases", func() {
	It("decodes a capture and reports a silent host", func() {
		f, err := os.Open(filepath.Join("..", "v9.0", "testdata", "qhost-xml-unreachable-9.1.7.xml"))
		Expect(err).NotTo(HaveOccurred())
		defer f.Close()
		silent := map[string]bool{}
		stats, err := qhost.DecodeHostsXML(f, func(h qhost.HostXML) error {
			if !h.IsGlobal() {
				silent[h.Name] = !h.Reporting()
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats).To(Equal(qhost.HostStats{Decoded: 4, Complete: true}))
		Expect(silent).To(Equal(map[string]bool{"ocs-master": false, "ocs-worker1": false, "ocs-worker2": true}))
	})

	It("reports a truncated document with the shared sentinel", func() {
		_, err := qhost.DecodeHostsXML(strings.NewReader("<qhost><host name='a'>"), func(qhost.HostXML) error { return nil })
		Expect(errors.Is(err, qhost.ErrIncompleteHostDocument)).To(BeTrue())
	})

	It("exposes the streamer on the 9.1 client", func() {
		impl, err := qhost.NewCommandLineQhost(qhost.CommandLineQHostConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		var streamer qhost.HostStreamer = impl
		stats, err := streamer.StreamHostsXML(context.Background(), nil, func(qhost.HostXML) error { return nil })
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
	})
})

var _ = Describe("v9.1 qhost resource stream", func() {
	It("is the core decoder", func() {
		var names []string
		stats, err := qhost.DecodeHostResourcesXML(strings.NewReader(
			`<qhost><host name='global'><resourcevalue name='matlab' dominance='gc'>60</resourcevalue></host></qhost>`),
			func(h qhost.HostResourcesXML) error {
				names = append(names, h.Name+"="+h.Resources[0].Dominance+":"+h.Resources[0].Value)
				return nil
			})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		Expect(names).To(Equal([]string{"global=gc:60"}))
		var _ qhost.ResourceStreamer = (*qhost.QHostImpl)(nil)
	})
})
