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

package qmod_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qmod "github.com/hpc-gridware/go-clusterscheduler/pkg/qmod/v9.1"
)

var _ = Describe("qmod under a context through the 9.1 alias", func() {
	It("exposes the context runner on the 9.1 client", func() {
		q, err := qmod.NewCommandLineQMod(qmod.CommandLineQModConfig{DryRun: true})
		Expect(err).NotTo(HaveOccurred())
		var runner qmod.ContextRunner = q
		out, _, err := runner.ActionContext(context.Background(), nil, "-d", []string{"all.q@n1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("Dry run: qmod [-d all.q@n1]"))
	})
})
