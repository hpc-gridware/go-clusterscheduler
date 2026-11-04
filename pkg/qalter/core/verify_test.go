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

package core_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/helper"
	qalter "github.com/hpc-gridware/go-clusterscheduler/pkg/qalter/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// reasonCheck names one reason of a fixture and the fields it must carry.
// Text and JobID are compared only when set in want.
type reasonCheck struct {
	index int
	want  qalter.VerifyReason
}

func expectReason(got, want qalter.VerifyReason) {
	GinkgoHelper()
	if want.Text == "" {
		got.Text = ""
	}
	if want.JobID == 0 {
		got.JobID = 0
	}
	Expect(got).To(Equal(want))
}

func readFixture(name string) string {
	GinkgoHelper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

var _ = Describe("qalter -w verification", func() {

	Context("ParseVerifyOutput on captured GCS 9.1.7 output", func() {
		qi := func(name string) (qalter.VerifyTargetKind, string) { return qalter.VerifyTargetQueueInstance, name }
		notAvailable := func(name string) qalter.VerifyReason {
			kind, target := qi(name)
			return qalter.VerifyReason{Kind: qalter.VerifyReasonQueueInstanceNotAvailable, TargetKind: kind, Target: target}
		}

		DescribeTable("decides the outcome from the text and parses the reason lines",
			func(file string, mode qalter.VerifyMode, outcome qalter.VerifyOutcome, reasons int, wantErr error, checks []reasonCheck) {
				result, err := qalter.ParseVerifyOutput(mode, readFixture(file), "", qalter.VerifyOptions{})
				if wantErr != nil {
					Expect(err).To(MatchError(wantErr))
					Expect(result.Denied).NotTo(BeEmpty())
					Expect(err.Error()).To(ContainSubstring(result.Denied))
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(result.Denied).To(BeEmpty())
				}
				Expect(result.Mode).To(Equal(mode))
				Expect(result.Outcome).To(Equal(outcome))
				Expect(result.Reasons).To(HaveLen(reasons))
				Expect(result.Dropped).To(BeZero())
				for _, r := range result.Reasons {
					Expect(r.Text).NotTo(HavePrefix("Job "))
					Expect(r.Kind).NotTo(Equal(qalter.VerifyReasonUnknown), r.Text)
				}
				for _, c := range checks {
					expectReason(result.Reasons[c.index], c.want)
				}
			},
			Entry("-w p: job of an advance reservation that is gone", "qalter-w-p-ad-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyOutcomeUnknown, 0, qalter.ErrNoSuchJob, nil),
			Entry("-w p: queue in load alarm", "qalter-w-p-alarm-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w p: advance reservation job", "qalter-w-p-ar-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 10, nil, []reasonCheck{
					{0, qalter.VerifyReason{JobID: 3448, Kind: qalter.VerifyReasonNotReservedByAR,
						TargetKind: qalter.VerifyTargetClusterQueue, Target: "slurm.q",
						Text: `cannot run in queue "slurm.q" because it was not reserved by advance reservation 2`}},
					{2, notAvailable("gpu.q@ocs-worker1")},
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonNotReservedByAR,
						TargetKind: qalter.VerifyTargetClusterQueue, Target: "alarm.q"}},
				}),
			Entry("-w p: job in error state", "qalter-w-p-eqw-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w p: fitting job (running)", "qalter-w-p-fit-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyAlreadyRunning, 0, nil, nil),
			Entry("-w p: held job", "qalter-w-p-hold-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyAlreadyRunning, 0, nil, nil),
			Entry("-w p: global consumable exhausted", "qalter-w-p-matlab-one-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 10, nil, []reasonCheck{
					{1, qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission,
						TargetKind: qalter.VerifyTargetClusterQueue, Target: "gpu.q"}},
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient,
						TargetKind: qalter.VerifyTargetGlobal, Target: "global",
						Request: "matlab=1", Resource: "matlab", Dominance: "gc", Available: "0.000000"}},
				}),
			Entry("-w p: request that can never be satisfied", "qalter-w-p-never-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 20, nil, []reasonCheck{
					{0, notAvailable("slurm.q@ocs-master")},
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonForcedResourceNotRequested,
						TargetKind: qalter.VerifyTargetQueueInstance, Target: "slurm.q@ocs-worker2", Resource: "slurm_shim"}},
					{11, qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient,
						TargetKind: qalter.VerifyTargetQueueInstance, Target: "bigmem.q@ocs-worker2",
						Request: "gpu=99", Resource: "slots", Dominance: "qc", Available: "0.000000"}},
					{12, qalter.VerifyReason{Kind: qalter.VerifyReasonQueueType,
						TargetKind: qalter.VerifyTargetQueueInstance, Target: "interactive.q@ocs-worker2"}},
					{14, qalter.VerifyReason{Kind: qalter.VerifyReasonUnknownResource,
						TargetKind: qalter.VerifyTargetQueueInstance, Target: "long.q@ocs-worker2",
						Request: "gpu=99", Resource: "gpu"}},
					{15, qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient,
						TargetKind: qalter.VerifyTargetHost, Target: "ocs-worker1",
						Request: "gpu=99", Resource: "gpu", Dominance: "hc", Available: "4.000000"}},
				}),
			Entry("-w p: hard queue list excludes every queue", "qalter-w-p-neverq-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 10, nil, []reasonCheck{
					{0, qalter.VerifyReason{Kind: qalter.VerifyReasonNotInHardQueueList,
						TargetKind: qalter.VerifyTargetClusterQueue, Target: "slurm.q"}},
					{2, notAvailable("gpu.q@ocs-worker1")},
				}),
			Entry("-w p: not the owner", "qalter-w-p-notowner-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyOutcomeUnknown, 0, qalter.ErrNotPermitted, nil),
			Entry("-w p: PE slot range out of reach", "qalter-w-p-pe999-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 10, nil, []reasonCheck{
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonPESlots,
						TargetKind: qalter.VerifyTargetPE, Target: "smp", Slots: 0}},
				}),
			Entry("-w p: resource quota reached", "qalter-w-p-rqs-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 10, nil, []reasonCheck{
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonRQSLimit,
						TargetKind: qalter.VerifyTargetRQSRule, Target: "max_slots_per_user/1", Limit: "carol/////"}},
				}),
			Entry("-w p: running job", "qalter-w-p-running-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyAlreadyRunning, 0, nil, nil),
			Entry("-w p: scoped parallel request over the global limit", "qalter-w-p-scoped-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyNoSuitableQueues, 12, nil, []reasonCheck{
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient,
						TargetKind: qalter.VerifyTargetGlobal, Target: "global",
						Request: "matlab=61", Resource: "matlab", Dominance: "gc", Available: "60.000000"}},
					{11, qalter.VerifyReason{Kind: qalter.VerifyReasonPESlots,
						TargetKind: qalter.VerifyTargetPE, Target: "mpi", Slots: 0}},
				}),
			Entry("-w p: unknown job id", "qalter-w-p-unknown-9.1.7.txt", qalter.VerifyNow,
				qalter.VerifyOutcomeUnknown, 0, qalter.ErrNoSuchJob, nil),
			Entry("-w v: job of an advance reservation that is gone", "qalter-w-v-ad-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyOutcomeUnknown, 0, qalter.ErrNoSuchJob, nil),
			Entry("-w v: queue in load alarm", "qalter-w-v-alarm-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w v: advance reservation job", "qalter-w-v-ar-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w v: job in error state", "qalter-w-v-eqw-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w v: fitting job (running)", "qalter-w-v-fit-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyAlreadyRunning, 0, nil, nil),
			Entry("-w v: held job", "qalter-w-v-hold-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyAlreadyRunning, 0, nil, nil),
			Entry("-w v: global consumable exhausted", "qalter-w-v-matlab-one-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w v: request that can never be satisfied", "qalter-w-v-never-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyNoSuitableQueues, 18, nil, []reasonCheck{
					{0, qalter.VerifyReason{Kind: qalter.VerifyReasonForcedResourceNotRequested,
						TargetKind: qalter.VerifyTargetQueueInstance, Target: "slurm.q@ocs-worker2", Resource: "slurm_shim"}},
					{4, qalter.VerifyReason{Kind: qalter.VerifyReasonRQSLimit,
						TargetKind: qalter.VerifyTargetRQSRule, Target: "gpu_per_user/1", Limit: "alice/////"}},
					{9, qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient,
						TargetKind: qalter.VerifyTargetHost, Target: "ocs-worker1",
						Request: "gpu=99", Resource: "gpu", Dominance: "hc", Available: "4.000000"}},
					{17, qalter.VerifyReason{Kind: qalter.VerifyReasonUnknownResource,
						TargetKind: qalter.VerifyTargetQueueInstance, Target: "alarm.q@ocs-worker2",
						Request: "gpu=99", Resource: "gpu"}},
				}),
			Entry("-w v: hard queue list excludes every queue", "qalter-w-v-neverq-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w v: PE slot range out of reach", "qalter-w-v-pe999-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyNoSuitableQueues, 1, nil, []reasonCheck{
					{0, qalter.VerifyReason{JobID: 3451, Kind: qalter.VerifyReasonPESlots,
						TargetKind: qalter.VerifyTargetPE, Target: "smp", Slots: 0,
						Text: `cannot run in PE "smp" because it only offers 0 slots`}},
				}),
			Entry("-w v: resource quota reached", "qalter-w-v-rqs-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyFoundSuitableQueues, 0, nil, nil),
			Entry("-w v: running job", "qalter-w-v-running-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyAlreadyRunning, 0, nil, nil),
			Entry("-w v: scoped parallel request over the global limit", "qalter-w-v-scoped-9.1.7.txt", qalter.VerifyEmptyCluster,
				qalter.VerifyNoSuitableQueues, 3, nil, []reasonCheck{
					{0, qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient,
						TargetKind: qalter.VerifyTargetGlobal, Target: "global",
						Request: "matlab=61", Resource: "matlab", Dominance: "gc", Available: "60.000000"}},
					{2, qalter.VerifyReason{Kind: qalter.VerifyReasonPESlots,
						TargetKind: qalter.VerifyTargetPE, Target: "mpi", Slots: 0}},
				}),
		)

		It("shares its no-such-job sentinel with helper and qstat", func() {
			Expect(errors.Is(qalter.ErrNoSuchJob, helper.ErrNoSuchJob)).To(BeTrue())
			Expect(errors.Is(fmt.Errorf("wrapped: %w", helper.ErrNoSuchJob), qalter.ErrNoSuchJob)).To(BeTrue())
		})

		It("fills Denied and reports ErrNoSuchJob for a denial printed on stderr", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, "", readFixture("qalter-w-p-unknown-9.1.7.txt"), qalter.VerifyOptions{})
			Expect(err).To(MatchError(qalter.ErrNoSuchJob))
			Expect(result.Denied).To(Equal(`job "999999999" does not exist`))
			Expect(result.Outcome).To(Equal(qalter.VerifyOutcomeUnknown))
		})

		It("reports ErrNotPermitted with the scheduler's text for an ownership denial", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, readFixture("qalter-w-p-notowner-9.1.7.txt"), "", qalter.VerifyOptions{})
			Expect(err).To(MatchError(qalter.ErrNotPermitted))
			Expect(err.Error()).To(Equal(`not permitted: "alice" must be at least owner of job 3452 to change job attributes`))
			Expect(result.Denied).To(Equal(`"alice" must be at least owner of job 3452 to change job attributes`))
		})

		DescribeTable("reports ErrNotPermitted for the qmaster permission texts",
			func(denied string) {
				result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, "", "denied: "+denied, qalter.VerifyOptions{})
				Expect(err).To(MatchError(qalter.ErrNotPermitted))
				Expect(err).NotTo(MatchError(qalter.ErrNoSuchJob))
				Expect(result.Denied).To(Equal(denied))
			},
			Entry("not the owner", `"alice" must be at least owner of job 7 to change job attributes`),
			Entry("not a manager", `"alice" must be manager to modify job attributes`),
			Entry("not an operator", `"alice" must be operator to change priority`),
			Entry("manager for the operation", `"alice" must be manager for this operation`),
			Entry("no submit or admin host", `host "laptop" is neither submit nor admin host`),
			Entry("no admin host", `host "laptop" is no admin host`),
		)

		It("returns a plain error carrying the text for a denial unrelated to permissions", func() {
			const denied = `soft requests on consumables like "gpu" are not supported`
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, "", "denied: "+denied+"\n", qalter.VerifyOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(qalter.ErrNotPermitted))
			Expect(err).NotTo(MatchError(qalter.ErrNoSuchJob))
			Expect(err.Error()).To(Equal("denied: " + denied))
			Expect(result.Denied).To(Equal(denied))
			Expect(result.Outcome).To(Equal(qalter.VerifyOutcomeUnknown))
		})

		It("returns an unknown outcome for a result line it does not know", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, "verification: something new\n", "", qalter.VerifyOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Outcome).To(Equal(qalter.VerifyOutcomeUnknown))
			Expect(result.Slots).To(BeZero())
			Expect(result.Reasons).To(BeEmpty())
		})

		It("returns an unknown outcome without error when nothing was printed", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyEmptyCluster, "", "", qalter.VerifyOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Outcome).To(Equal(qalter.VerifyOutcomeUnknown))
			Expect(result.Reasons).To(BeEmpty())
		})
	})

	Context("ParseVerifyOutput on scheduler texts not in the fixtures", func() {
		parseOne := func(line string) qalter.VerifyReason {
			GinkgoHelper()
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, line+"\nverification: no suitable queues\n", "", qalter.VerifyOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Reasons).To(HaveLen(1))
			return result.Reasons[0]
		}

		DescribeTable("classifies the SCHEDD_INFO messages that name a target or a resource",
			func(line string, want qalter.VerifyReason) {
				got := parseOne("Job 7 " + line)
				Expect(got.JobID).To(Equal(uint64(7)))
				Expect(got.Text).To(Equal(line))
				expectReason(got, want)
			},
			Entry("forced resource of a host",
				`does not request 'forced' resource "gpu" of host node01`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonForcedResourceNotRequested, TargetKind: qalter.VerifyTargetHost, Target: "node01", Resource: "gpu"}),
			Entry("forced resource of the global host",
				`does not request 'forced' resource "lic" of host global`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonForcedResourceNotRequested, TargetKind: qalter.VerifyTargetGlobal, Target: "global", Resource: "lic"}),
			Entry("globally forced resource",
				`does not request globally 'forced' resource "lic"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonForcedResourceNotRequested, TargetKind: qalter.VerifyTargetGlobal, Target: "global", Resource: "lic"}),
			Entry("no permission for a host",
				`has no permission for host "node01"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetHost, Target: "node01"}),
			Entry("no permission for a queue instance (printed as queue)",
				`has no permission for queue "all.q@node01"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("no permission for a queue instance",
				`has no permission for queue instance "all.q@node01"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("no permission for a cluster queue",
				`has no permission for cluster queue "all.q"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetClusterQueue, Target: "all.q"}),
			Entry("wrong project for a queue instance",
				`(project p1) does not have the correct project to run in queue instance "all.q@node01"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonProjectAccess, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("no permission for a PE",
				`has no permission for PE "mpi"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetPE, Target: "mpi"}),
			Entry("no access to a PE",
				`cannot run because no access to pe "mpi"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetPE, Target: "mpi"}),
			Entry("no permission for an advance reservation",
				`has no permission for advance_reservation "12"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNoPermission, TargetKind: qalter.VerifyTargetAdvanceReservation, Target: "12"}),
			Entry("wrong project",
				`(project p1) does not have the correct project to run in cluster queue "all.q"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonProjectAccess, TargetKind: qalter.VerifyTargetClusterQueue, Target: "all.q"}),
			Entry("no project",
				`(no project) does not have the correct project to run in host "node01"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonProjectAccess, TargetKind: qalter.VerifyTargetHost, Target: "node01"}),
			Entry("excluded project",
				`(project p1) is not allowed to run in cluster queue "all.q" based on the excluded project list`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonProjectAccess, TargetKind: qalter.VerifyTargetClusterQueue, Target: "all.q"}),
			Entry("exclusive resource in use",
				`(-l excl=1) cannot run at host "node01" because exclusive resource (excl) is already in use`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonExclusiveInUse, TargetKind: qalter.VerifyTargetHost, Target: "node01", Request: "excl=1", Resource: "excl"}),
			Entry("default request fails",
				`(-l h_vmem=1G) cannot run in queue "all.q@node01" because for default request it offers only qc:h_vmem=512.000000M`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01",
					Request: "h_vmem=1G", Resource: "h_vmem", Dominance: "qc", Available: "512.000000M"}),
			Entry("insufficient load value due to a reservation",
				`(-l mem_free=8G) cannot run at host "node01" because it offers only hl:mem_free=4.000000G due to a reservation`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient, TargetKind: qalter.VerifyTargetHost, Target: "node01",
					Request: "mem_free=8G", Resource: "mem_free", Dominance: "hl", Available: "4.000000G"}),
			Entry("non requestable resource",
				`(-l arch=lx-amd64) cannot run globally because job requests non requestable resource arch"`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNonRequestableResource, TargetKind: qalter.VerifyTargetGlobal, Target: "global",
					Request: "arch=lx-amd64", Resource: "arch"}),
			Entry("boolean request without a value",
				`(-l excl) cannot run globally because job requests unknown resource (excl)`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonUnknownResource, TargetKind: qalter.VerifyTargetGlobal, Target: "global",
					Request: "excl", Resource: "excl"}),
			Entry("queue instance with zero slots",
				`cannot run in queue instance "all.q@node01" because it has "0" slots`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01",
					Resource: "slots", Available: "0"}),
			Entry("suspend alarm",
				`queue instance "all.q@node01" is in suspend alarm: np_load_avg=2.500000 (= 2.500000 + 0.50 * 0.000000 with nproc=1) >= 1.75`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonSuspendAlarm, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("overloaded",
				`queue instance "all.q@node01" dropped because it is overloaded: np_load_avg=2.500000 >= 1.75`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonLoadAlarm, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("would set in load alarm",
				`(4 slots) would set queue instance "all.q@node01" in load alarm state`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonLoadAlarm, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01", Slots: 4}),
			Entry("full",
				`queue instance "all.q@node01" dropped because it is full`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueFull, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("suspended",
				`queue instance "all.q@node01" dropped because it is suspended`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueSuspended, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("disabled",
				`queue instance "all.q@node01" dropped because it is disabled`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueDisabled, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("not of parallel type",
				`cannot run in queue instance "all.q@node01" because it is not of parallel type`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueType, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("not interactive",
				`cannot run in queue "all.q" because it is not of type interactive`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueType, TargetKind: qalter.VerifyTargetClusterQueue, Target: "all.q"}),
			Entry("not checkpointing",
				`cannot run in queue instance "all.q@node01" because it is not of type checkpointing`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueType, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("job not parallel",
				`cannot run in queue instance "mpi.q@node01" because the job is not parallel`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueType, TargetKind: qalter.VerifyTargetQueueInstance, Target: "mpi.q@node01"}),
			Entry("queues non requestable",
				`cannot run in queue instance "all.q@node01" because queues are non requestable`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonQueueType, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("PE not in pe list",
				`cannot run in queue "all.q" because PE "mpi" is not in pe list`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNotInQueueList, TargetKind: qalter.VerifyTargetClusterQueue, Target: "all.q", Resource: "mpi"}),
			Entry("ckpt not in ckpt list",
				`cannot run in queue instance "all.q@node01" because ckpt object "blcr" is not in ckpt list of queue`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNotInQueueList, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01", Resource: "blcr"}),
			Entry("queue instance not reserved by AR",
				`cannot run in queue instance "all.q@node01" because it was not reserved by advance reservation 3`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNotReservedByAR, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01"}),
			Entry("host not reserved by AR",
				`cannot run on host "node01" because it was not reserved by advance reservation 3`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonNotReservedByAR, TargetKind: qalter.VerifyTargetHost, Target: "node01"}),
			Entry("resource quota per queue instance (message carries its own Job prefix)",
				`Job cannot run in queue instance "all.q@node01" because exceeds limit in rule max_slots/1`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonRQSLimit, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01", Limit: "max_slots/1"}),
			Entry("invalid dynamical limit",
				`dropped because of invalid dynamical limit $num_proc*2 in rule max_slots/1`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonRQSLimit, TargetKind: qalter.VerifyTargetRQSRule, Target: "max_slots/1", Limit: "$num_proc*2"}),
			Entry("total PE slots not in range",
				`cannot run because total slots of pe "mpi" not in range of job`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonPESlots, TargetKind: qalter.VerifyTargetPE, Target: "mpi"}),
			Entry("PE allocation rule",
				`pe "mpi" dropped because allocation rule is not suitable`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonPESlots, TargetKind: qalter.VerifyTargetPE, Target: "mpi"}),
			Entry("calendar",
				`cannot run in queue instance "night.q@node01" because the job runtime of 7200 sec. is too long`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonCalendar, TargetKind: qalter.VerifyTargetQueueInstance, Target: "night.q@node01", Available: "7200"}),
			Entry("consumable used as load threshold without value",
				`cannot run in queue instance "all.q@node01" because consumable "mem_free" used as load threshold has no value at queue, host or global level`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient, TargetKind: qalter.VerifyTargetQueueInstance, Target: "all.q@node01", Resource: "mem_free"}),
			Entry("host cleanup",
				`cannot run on host "node01" until clean up of a previous run has finished`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonHostCleanup, TargetKind: qalter.VerifyTargetHost, Target: "node01"}),
			Entry("resource not granted on a selected host",
				`resource "gpu" could not be granted on host "node01" although the host was selected for the job`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonResourceInsufficient, TargetKind: qalter.VerifyTargetHost, Target: "node01", Resource: "gpu"}),
			Entry("job-wide condition without a target is kept as unknown",
				`cannot run because requested ckpt object not found`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonUnknown, TargetKind: qalter.VerifyTargetJob}),
			Entry("a wording never seen is kept as unknown",
				`cannot run because the moon is full`,
				qalter.VerifyReason{Kind: qalter.VerifyReasonUnknown, TargetKind: qalter.VerifyTargetJob}),
		)

		It("keeps a line without the job prefix as unknown with its full text", func() {
			got := parseOne("(Scheduler job information not available for every job)")
			Expect(got).To(Equal(qalter.VerifyReason{Kind: qalter.VerifyReasonUnknown, TargetKind: qalter.VerifyTargetJob,
				Text: "(Scheduler job information not available for every job)"}))
		})

		It("reads the slot count of a parallel assignment", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyEmptyCluster, "verification: found possible assignment with 16 slots\n", "", qalter.VerifyOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Outcome).To(Equal(qalter.VerifyFoundAssignment))
			Expect(result.Slots).To(Equal(16))
		})
	})

	Context("ParseVerifyOutput with MaxReasons and Keep", func() {
		never := func() string { return readFixture("qalter-w-p-never-9.1.7.txt") }
		// onHost keeps the lines about a host and its queue instances.
		onHost := func(host string) func(qalter.VerifyReason) bool {
			return func(r qalter.VerifyReason) bool {
				target := r.Target
				if r.TargetKind == qalter.VerifyTargetQueueInstance {
					_, target, _ = strings.Cut(target, "@")
				}
				return strings.EqualFold(target, host)
			}
		}

		It("caps the reasons and counts the rest in Dropped", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, never(), "", qalter.VerifyOptions{MaxReasons: 5})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Outcome).To(Equal(qalter.VerifyNoSuitableQueues))
			Expect(result.Reasons).To(HaveLen(5))
			Expect(result.Dropped).To(Equal(15))
			Expect(result.Reasons[0].Target).To(Equal("slurm.q@ocs-master"))
		})

		It("keeps every line Keep accepts without counting it against the cap", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, never(), "",
				qalter.VerifyOptions{MaxReasons: 1, Keep: onHost("OCS-Worker1")})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Reasons).To(HaveLen(7))
			Expect(result.Dropped).To(Equal(13))
			Expect(result.Reasons[0].Target).To(Equal("slurm.q@ocs-master"))
			for _, r := range result.Reasons[1:] {
				Expect(strings.ToLower(r.Target)).To(HaveSuffix("ocs-worker1"), r.Text)
			}
			Expect(result.Reasons[6]).To(HaveField("TargetKind", qalter.VerifyTargetHost))
		})

		It("hands Keep the parsed reason so it can select by queue and target kind", func() {
			keep := func(r qalter.VerifyReason) bool {
				return r.TargetKind == qalter.VerifyTargetGlobal || strings.HasPrefix(r.Target, "night.q@")
			}
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, readFixture("qalter-w-p-matlab-one-9.1.7.txt"), "",
				qalter.VerifyOptions{MaxReasons: 1, Keep: keep})
			Expect(err).NotTo(HaveOccurred())
			targets := make([]string, 0, len(result.Reasons))
			for _, r := range result.Reasons {
				targets = append(targets, r.Target)
			}
			Expect(targets).To(Equal([]string{"slurm.q@ocs-master", "night.q@ocs-worker1", "night.q@ocs-worker2", "night.q@ocs-master", "global"}))
			Expect(result.Dropped).To(Equal(5))
		})

		It("keeps nothing beyond the cap when Keep accepts no line", func() {
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, never(), "",
				qalter.VerifyOptions{MaxReasons: 2, Keep: func(qalter.VerifyReason) bool { return false }})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Reasons).To(HaveLen(2))
			Expect(result.Dropped).To(Equal(18))
		})

		It("still reads the result and denial lines past the cap, on stdout and stderr", func() {
			stdout := "Job 1 queue instance \"a.q@n1\" dropped because it is full\n" +
				"Job 1 queue instance \"a.q@n2\" dropped because it is full\n" +
				"Job 1 queue instance \"a.q@n3\" dropped because it is full\n" +
				"verification: no suitable queues"
			stderr := "Job 1 queue instance \"a.q@n4\" dropped because it is full\n" +
				`denied: "alice" must be at least owner of job 1 to change job attributes` + "\n"
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, stdout, stderr, qalter.VerifyOptions{MaxReasons: 1})
			Expect(err).To(MatchError(qalter.ErrNotPermitted))
			Expect(result.Reasons).To(HaveLen(1))
			Expect(result.Dropped).To(Equal(3))
			Expect(result.Denied).To(HavePrefix(`"alice"`))
		})

		It("applies the default cap of 2000 lines", func() {
			var sb strings.Builder
			for i := 0; i < 2500; i++ {
				fmt.Fprintf(&sb, "Job 1 queue instance \"all.q@node%04d\" dropped because it is full\n", i)
			}
			sb.WriteString("verification: no suitable queues\n")
			result, err := qalter.ParseVerifyOutput(qalter.VerifyNow, sb.String(), "", qalter.VerifyOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Reasons).To(HaveLen(qalter.DefaultMaxReasons))
			Expect(result.Dropped).To(Equal(500))
		})
	})

	Context("against a qalter process", func() {
		var impl *qalter.CommandLineQAlter
		fixture := func(name string) string {
			abs, err := filepath.Abs(filepath.Join("testdata", name))
			Expect(err).NotTo(HaveOccurred())
			return abs
		}
		env := func(kv ...string) []string {
			return append([]string{"PATH=" + os.Getenv("PATH")}, kv...)
		}
		readLines := func(path string) []string {
			data, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			return strings.Split(strings.TrimSpace(string(data)), "\n")
		}

		BeforeEach(func() {
			var err error
			impl, err = qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
				Executable: fixture("fake-qalter.sh"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("treats exit code 1 with a result line as a regular result and passes -w <mode> <job>", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			result, err := impl.VerifyContext(context.Background(),
				env("FAKE_QALTER_EXIT=1", "FAKE_QALTER_FILE="+fixture("qalter-w-v-pe999-9.1.7.txt"), "FAKE_QALTER_ARGS_FILE="+argsFile),
				"3451", qalter.VerifyEmptyCluster, qalter.VerifyOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Outcome).To(Equal(qalter.VerifyNoSuitableQueues))
			Expect(result.Reasons).To(HaveLen(1))
			Expect(readLines(argsFile)).To(Equal([]string{"-w", "v", "3451"}))
		})

		It("maps a denial on stderr with exit code 1 to the sentinel", func() {
			result, err := impl.VerifyContext(context.Background(),
				env("FAKE_QALTER_EXIT=1", `FAKE_QALTER_STDERR=denied: job "999999999" does not exist`),
				"999999999", qalter.VerifyNow, qalter.VerifyOptions{})
			Expect(err).To(MatchError(qalter.ErrNoSuchJob))
			Expect(result.Denied).To(Equal(`job "999999999" does not exist`))
		})

		It("reports a failed run without a result line in the command-error shape", func() {
			_, err := impl.VerifyContext(context.Background(),
				env("FAKE_QALTER_EXIT=1", `FAKE_QALTER_STDERR=error: unable to send message to qmaster using port 6444 on host "master": got send error`),
				"1", qalter.VerifyNow, qalter.VerifyOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix("failed to run command (error: unable to send message to qmaster"))
			Expect(err.Error()).To(HaveSuffix("exit status 1"))
		})

		It("ends with the context deadline and reports the context error", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			start := time.Now()
			_, err := impl.VerifyContext(ctx,
				env("FAKE_QALTER_SLEEP=5", "FAKE_QALTER_FILE="+fixture("qalter-w-v-pe999-9.1.7.txt")),
				"3451", qalter.VerifyEmptyCluster, qalter.VerifyOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), err.Error())
			Expect(time.Since(start)).To(BeNumerically("<", 3*time.Second))
		})

		Context("with DelayAfter longer than the deadline", func() {
			BeforeEach(func() {
				var err error
				impl, err = qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
					Executable: fixture("fake-qalter.sh"),
					DelayAfter: 5 * time.Second,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			DescribeTable("returns the complete answer when the context ends after qalter exited",
				func(exitCode string, want qalter.VerifyOutcome) {
					ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
					defer cancel()
					start := time.Now()
					result, err := impl.VerifyContext(ctx,
						env("FAKE_QALTER_EXIT="+exitCode, "FAKE_QALTER_FILE="+fixture("qalter-w-v-pe999-9.1.7.txt")),
						"3451", qalter.VerifyEmptyCluster, qalter.VerifyOptions{})
					Expect(time.Since(start)).To(BeNumerically("<", 3*time.Second))
					Expect(ctx.Err()).To(HaveOccurred())
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Outcome).To(Equal(want))
				},
				Entry("exit code 0", "0", qalter.VerifyNoSuitableQueues),
				Entry("exit code 1 for no suitable queues", "1", qalter.VerifyNoSuitableQueues),
			)

			It("never returns a nil result without an error", func() {
				for _, timeout := range []time.Duration{time.Nanosecond, 50 * time.Millisecond, 300 * time.Millisecond} {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					result, err := impl.VerifyContext(ctx,
						env("FAKE_QALTER_FILE="+fixture("qalter-w-p-running-9.1.7.txt")),
						"1", qalter.VerifyNow, qalter.VerifyOptions{})
					cancel()
					if err == nil {
						Expect(result).NotTo(BeNil(), "timeout %v", timeout)
					} else {
						Expect(result).To(BeNil(), "timeout %v", timeout)
						Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), err.Error())
					}
				}
			})
		})

		It("replaces the child environment when env is given and inherits it when nil", func() {
			Expect(os.Setenv("QALTER_VERIFY_TEST_PARENT_ONLY", "1")).To(Succeed())
			DeferCleanup(os.Unsetenv, "QALTER_VERIFY_TEST_PARENT_ONLY")

			envFile := filepath.Join(GinkgoT().TempDir(), "env")
			_, _, err := impl.RunContext(context.Background(), env("FAKE_QALTER_ENV_FILE="+envFile, "FAKE_QALTER_MARKER=yes"), "-w", "v", "1")
			Expect(err).NotTo(HaveOccurred())
			replaced := readLines(envFile)
			Expect(replaced).To(ContainElement("FAKE_QALTER_MARKER=yes"))
			Expect(replaced).To(ContainElement("SGE_SINGLE_LINE=true"))
			Expect(replaced).NotTo(ContainElement("QALTER_VERIFY_TEST_PARENT_ONLY=1"))

			Expect(os.Setenv("FAKE_QALTER_ENV_FILE", envFile)).To(Succeed())
			DeferCleanup(os.Unsetenv, "FAKE_QALTER_ENV_FILE")
			_, _, err = impl.RunContext(context.Background(), nil, "-w", "v", "1")
			Expect(err).NotTo(HaveOccurred())
			inherited := readLines(envFile)
			Expect(inherited).To(ContainElement("QALTER_VERIFY_TEST_PARENT_ONLY=1"))
			Expect(inherited).To(ContainElement("SGE_SINGLE_LINE=true"))
		})

		It("returns stdout and stderr separately", func() {
			stdout, stderr, err := impl.RunContext(context.Background(),
				env("FAKE_QALTER_FILE="+fixture("qalter-w-p-running-9.1.7.txt"), "FAKE_QALTER_STDERR=warning: something"),
				"-w", "p", "1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(Equal("verification: job is already running\n"))
			Expect(stderr).To(Equal("warning: something\n"))
		})

		It("refuses a mode other than v or p without spawning", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			_, err := impl.VerifyContext(context.Background(), env("FAKE_QALTER_ARGS_FILE="+argsFile), "1", qalter.VerifyMode("e"), qalter.VerifyOptions{})
			Expect(err).To(MatchError(ContainSubstring(`verify mode "e"`)))
			Expect(argsFile).NotTo(BeAnExistingFile())
		})

		DescribeTable("refuses anything but one job without spawning",
			func(job string) {
				argsFile := filepath.Join(GinkgoT().TempDir(), "args")
				result, err := impl.VerifyContext(context.Background(), env("FAKE_QALTER_ARGS_FILE="+argsFile), job, qalter.VerifyNow, qalter.VerifyOptions{})
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				Expect(argsFile).NotTo(BeAnExistingFile())
			},
			Entry("a flag", "-u"),
			Entry("a comma list", "1,2"),
			Entry("a space list", "1 2"),
			Entry("a wildcard", "*"),
			Entry("a name pattern", "job*"),
			Entry("a job name", "myjob"),
			Entry("an empty string", ""),
		)

		DescribeTable("passes one job id, with or without a task range",
			func(job string) {
				argsFile := filepath.Join(GinkgoT().TempDir(), "args")
				result, err := impl.VerifyContext(context.Background(),
					env("FAKE_QALTER_FILE="+fixture("qalter-w-p-running-9.1.7.txt"), "FAKE_QALTER_ARGS_FILE="+argsFile),
					job, qalter.VerifyNow, qalter.VerifyOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Outcome).To(Equal(qalter.VerifyAlreadyRunning))
				Expect(readLines(argsFile)).To(Equal([]string{"-w", "p", job}))
			},
			Entry("a job id", "123"),
			Entry("a job id with a task range", "123.4-10"),
		)

		It("refuses an argument with a control character without spawning", func() {
			argsFile := filepath.Join(GinkgoT().TempDir(), "args")
			_, _, err := impl.RunContext(context.Background(), env("FAKE_QALTER_ARGS_FILE="+argsFile), "-w", "v", "1\n2")
			Expect(err).To(HaveOccurred())
			Expect(argsFile).NotTo(BeAnExistingFile())
		})

		Context("in dry run mode", func() {
			BeforeEach(func() {
				var err error
				impl, err = qalter.NewCommandLineQAlter(qalter.CommandLineQAlterConfig{
					Executable: fixture("fake-qalter.sh"),
					DryRun:     true,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("prints the command instead of running it", func() {
				argsFile := filepath.Join(GinkgoT().TempDir(), "args")
				stdout, stderr, err := impl.RunContext(context.Background(), env("FAKE_QALTER_ARGS_FILE="+argsFile), "-w", "v", "1")
				Expect(err).NotTo(HaveOccurred())
				Expect(stdout).To(BeEmpty())
				Expect(stderr).To(BeEmpty())
				Expect(argsFile).NotTo(BeAnExistingFile())
			})

			It("returns an unknown outcome from VerifyContext", func() {
				result, err := impl.VerifyContext(context.Background(), nil, "1", qalter.VerifyNow, qalter.VerifyOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Outcome).To(Equal(qalter.VerifyOutcomeUnknown))
				Expect(result.Reasons).To(BeEmpty())
			})
		})
	})
})
