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

package helper_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
)

var _ = Describe("CappedBuffer", func() {
	It("keeps everything below the limit", func() {
		b := helper.NewCappedBuffer(10)
		n, err := b.Write([]byte("abc"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(3))
		Expect(b.String()).To(Equal("abc"))
	})

	It("keeps the first bytes up to the limit and reports every write as complete", func() {
		b := helper.NewCappedBuffer(5)
		for _, chunk := range []string{"abc", "defg", strings.Repeat("x", 100)} {
			n, err := b.Write([]byte(chunk))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(len(chunk)))
		}
		Expect(b.String()).To(Equal("abcde"))
	})

	It("keeps nothing with a zero limit", func() {
		b := helper.NewCappedBuffer(0)
		n, err := b.Write([]byte("abc"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(3))
		Expect(b.String()).To(BeEmpty())
	})
})
