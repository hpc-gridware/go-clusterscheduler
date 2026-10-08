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

package qrsub_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	qrdel "github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/v9.1"
	qrstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/v9.1"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
	qrsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/v9.1"
)

// clusterVersion returns the version of a reachable live cluster, or false
// if there is none.
func clusterVersion() (qconf.ClusterSchedulerVersion, bool) {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return qconf.ClusterSchedulerVersion{}, false
	}
	if _, err := exec.LookPath("qrsub"); err != nil {
		return qconf.ClusterSchedulerVersion{}, false
	}
	qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{})
	if err != nil {
		return qconf.ClusterSchedulerVersion{}, false
	}
	version, err := qc.GetVersion()
	return version, err == nil
}

var _ = Describe("QRSub v9.1 on a live cluster", Label("integration"), func() {
	It("creates an AR with core binding", func() {
		version, ok := clusterVersion()
		if !ok {
			Skip("no reachable cluster; run inside the dev container (make run)")
		}
		if version.Major < 9 || (version.Major == 9 && version.Minor < 1) {
			Skip("qrsub binding switches need 9.1, cluster runs " + version.Version)
		}
		ctx := context.Background()
		c, err := qrsub.NewCommandLineQRSub(qrsub.CommandLineQRSubConfig{})
		Expect(err).NotTo(HaveOccurred())
		id, out, err := c.Submit(ctx, qrsub.ReservationOptions{
			ReservationOptions: core.ReservationOptions{
				Name:     qrsub.ToPtr(fmt.Sprintf("gotestbind%d", time.Now().UnixNano()%1000000)),
				Duration: qrsub.ToPtr(5 * time.Minute),
			},
			BindingAmount:   qrsub.ToPtr(1),
			BindingUnit:     qrsub.ToPtr("C"),
			BindingStrategy: qrsub.ToPtr("packed"),
		})
		Expect(err).NotTo(HaveOccurred(), out)
		del, err := qrdel.NewCommandLineQRDel(qrdel.CommandLineQRDelConfig{Force: true})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _, _ = del.DeleteReservations(ctx, id) })

		stat, err := qrstat.NewCommandLineQRStat(qrstat.CommandLineQRStatConfig{})
		Expect(err).NotTo(HaveOccurred())
		details, err := stat.GetReservations(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(details).To(HaveLen(1))
		Expect(details[0].Binding).To(ContainSubstring("bamount=1"))
		Expect(details[0].Binding).To(ContainSubstring("bunit=C"))
		Expect(details[0].ExecBindingList).NotTo(BeEmpty())
	})
})
