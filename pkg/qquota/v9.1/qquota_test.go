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

package qquota_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qquota "github.com/hpc-gridware/go-clusterscheduler/pkg/qquota/v9.1"
)

// v9.1 aliases the core surface; this pins that the version package keeps
// exposing it, so a version switch in a caller compiles and decodes the same.
var _ = Describe("qquota v9.1", func() {
	It("decodes a 9.1.7 capture through the version package", func() {
		f, err := os.Open(filepath.Join("..", "v9.0", "testdata", "qquota-xml-memory-named-9.1.7.xml"))
		Expect(err).NotTo(HaveOccurred())
		defer f.Close()
		var names []string
		stats, err := qquota.DecodeQuotasXML(f, func(r qquota.QuotaRuleXML) error {
			names = append(names, r.Name)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.Complete).To(BeTrue())
		Expect(names).To(ContainElement("zz_probe_named/probe_x"))
	})

	It("constructs a dry-run client", func() {
		_, err := qquota.NewCommandLineQQuota(qquota.CommandLineQQuotaConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
	})
})
