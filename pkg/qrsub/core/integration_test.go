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

// Integration specs for the AR lifecycle across qrsub, qrstat and qrdel.
// They need a live cluster (the dev container, see the Makefile) and skip
// otherwise. Every change they make is undone: the ARs and jobs they
// create, the arusers membership they add, max_advance_reservations, and
// the throwaway second user (created only when running as root).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	qdel "github.com/hpc-gridware/go-clusterscheduler/pkg/qdel/core"
	qrdel "github.com/hpc-gridware/go-clusterscheduler/pkg/qrdel/core"
	qrstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qrstat/core"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qrsub/core"
	qsub "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
)

// otherUser owns the ARs of a second user in the multi-user specs.
const otherUser = "gotestar"

// clusterAvailable reports whether a live cluster with the AR clients is
// reachable. The probe "qconf -sconf global" is read-only.
func clusterAvailable() bool {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return false
	}
	for _, binary := range []string{"qconf", "qrsub", "qrstat", "qrdel"} {
		if _, err := exec.LookPath(binary); err != nil {
			return false
		}
	}
	return exec.Command("qconf", "-sconf", "global").Run() == nil
}

// runAs runs an AR client command as another user via su, with the cluster
// settings sourced. Arguments must not need shell quoting.
func runAs(username string, args ...string) (string, error) {
	cell := os.Getenv("SGE_CELL")
	if cell == "" {
		cell = "default"
	}
	settings := filepath.Join(os.Getenv("SGE_ROOT"), cell, "common", "settings.sh")
	script := ". " + settings + "; " + strings.Join(args, " ")
	out, err := exec.Command("su", "-s", "/bin/bash", username, "-c", script).CombinedOutput()
	return string(out), err
}

func one(details []qrstat.Reservation, err error) qrstat.Reservation {
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, details).To(HaveLen(1))
	return details[0]
}

var _ = Describe("Advance reservation lifecycle", Ordered, Label("integration"), func() {
	ctx := context.Background()
	var (
		qc           *qconf.CommandLineQConf
		sub          *core.CommandLineQRSub
		stat         *qrstat.CommandLineQRStat
		del          *qrdel.CommandLineQRDel
		created      []int64
		prefix       string
		me           string
		addedAruser  bool
		haveOther    bool
		addedOtherAR bool
	)

	BeforeAll(func() {
		if !clusterAvailable() {
			Skip("no reachable cluster; run inside the dev container (make run)")
		}
		var err error
		qc, err = qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{})
		Expect(err).NotTo(HaveOccurred())
		current, err := user.Current()
		Expect(err).NotTo(HaveOccurred())
		me = current.Username

		arusers, err := qc.ShowUserSetList("arusers")
		Expect(err).NotTo(HaveOccurred())
		isMember := false
		for _, entry := range arusers.Entries {
			if entry == me {
				isMember = true
			}
		}
		if !isMember {
			Expect(qc.AddUserToUserSetList(me, "arusers")).To(Succeed())
			addedAruser = true
		}

		// A second user for the ownership specs. Creating users needs root.
		if current.Uid == "0" {
			if _, err := user.Lookup(otherUser); err != nil {
				if exec.Command("useradd", "-m", otherUser).Run() == nil {
					haveOther = true
				}
			}
			if haveOther {
				Expect(qc.AddUserToUserSetList(otherUser, "arusers")).To(Succeed())
				addedOtherAR = true
			}
		}

		sub, err = core.NewCommandLineQRSub(core.CommandLineQRSubConfig{})
		Expect(err).NotTo(HaveOccurred())
		stat, err = qrstat.NewCommandLineQRStat(qrstat.CommandLineQRStatConfig{})
		Expect(err).NotTo(HaveOccurred())
		del, err = qrdel.NewCommandLineQRDel(qrdel.CommandLineQRDelConfig{Force: true})
		Expect(err).NotTo(HaveOccurred())
		prefix = fmt.Sprintf("gotest%d", time.Now().UnixNano()%1000000)
	})

	AfterAll(func() {
		for _, id := range created {
			_, _ = del.DeleteReservations(ctx, id)
		}
		if haveOther {
			_, _ = del.DeleteReservationsByUser(ctx, otherUser)
		}
		if addedOtherAR {
			_ = qc.DeleteUserFromUserSetList(otherUser, "arusers")
		}
		if haveOther {
			_ = exec.Command("userdel", "-r", otherUser).Run()
		}
		if addedAruser {
			_ = qc.DeleteUserFromUserSetList(me, "arusers")
		}
	})

	submit := func(opts core.ReservationOptions) int64 {
		id, out, err := sub.Submit(ctx, opts)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), out)
		ExpectWithOffset(1, id).To(BeNumerically(">", 0))
		created = append(created, id)
		return id
	}

	// submitAsOther creates an AR owned by the second user.
	submitAsOther := func(name string) int64 {
		out, err := runAs(otherUser, "qrsub", "-N", name, "-d", "300")
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), out)
		id, err := core.ParseQrsubOutput(out)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		return id
	}

	It("verifies, creates, lists, inspects, binds a job to and deletes an AR", func() {
		start := time.Now().Add(time.Hour).Truncate(time.Minute)
		opts := core.ReservationOptions{
			StartTime: &start,
			Duration:  core.ToPtr(10 * time.Minute),
			Name:      core.ToPtr(prefix + "life"),
			Resources: map[string]string{"h_rt": "600"},
		}

		found, out, err := sub.Verify(ctx, opts)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(found).To(BeTrue())

		id := submit(opts)

		summaries, err := stat.ListReservations(ctx, qrstat.ListOptions{Users: []string{"*"}})
		Expect(err).NotTo(HaveOccurred())
		var summary *qrstat.ReservationSummary
		for i := range summaries {
			if summaries[i].ID == id {
				summary = &summaries[i]
			}
		}
		Expect(summary).NotTo(BeNil())
		Expect(summary.State).To(Equal(qrstat.StateWaiting))
		Expect(summary.StartTime.Equal(start)).To(BeTrue(), "start %s, want %s", summary.StartTime, start)
		Expect(summary.Duration).To(Equal(10 * time.Minute))

		detail := one(stat.GetReservations(ctx, id))
		Expect(detail.Name).To(Equal(prefix + "life"))
		Expect(detail.Owner).To(Equal(me))
		Expect(detail.ResourceList).To(HaveKeyWithValue("h_rt", "600"))
		Expect(detail.EndTime.Sub(detail.StartTime)).To(Equal(10 * time.Minute))
		Expect(detail.ExecQueueList).NotTo(BeEmpty())

		qs, err := qsub.NewCommandLineQSub(qsub.CommandLineQSubConfig{})
		Expect(err).NotTo(HaveOccurred())
		jobID, out, err := qsub.NewJobBuilder(qs, "sleep", "1").
			Binary().
			AdvanceReservation(fmt.Sprint(id)).
			Resource("h_rt", "60").
			Submit(ctx)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(jobID).To(BeNumerically(">", 0))
		DeferCleanup(func() {
			// Normally gone with the AR; make sure it does not stay behind.
			if qd, err := qdel.NewCommandLineQDel(qdel.CommandLineQDelConfig{Force: true}); err == nil {
				_, _ = qd.DeleteJobs([]string{fmt.Sprint(jobID)})
			}
		})

		out, err = del.DeleteReservations(ctx, id)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(qrdel.ParseDeleteOutput(out).Deleted).To(ContainElement(id))

		Eventually(func() error {
			_, err := stat.GetReservations(ctx, id)
			return err
		}, 30*time.Second, time.Second).Should(MatchError(qrstat.ErrNotFound))
	})

	It("reserves a parallel environment and the requested queue", func() {
		id := submit(core.ReservationOptions{
			Duration: core.ToPtr(5 * time.Minute),
			Name:     core.ToPtr(prefix + "pe"),
			PEName:   core.ToPtr("make"),
			PESlots:  core.ToPtr("1"),
			Queues:   []string{"all.q"},
		})
		detail := one(stat.GetReservations(ctx, id))
		Expect(detail.GrantedParallelEnvironment).To(Equal(&qrstat.GrantedPE{Name: "make", Range: "1"}))
		Expect(detail.ExecQueueList).To(HaveLen(1))
		for queue, slots := range detail.ExecQueueList {
			Expect(queue).To(HavePrefix("all.q@"))
			Expect(slots).To(Equal(1))
		}
	})

	It("reports requests that cannot be granted", func() {
		opts := core.ReservationOptions{
			Duration: core.ToPtr(5 * time.Minute),
			PEName:   core.ToPtr("make"),
			PESlots:  core.ToPtr("100000"),
		}
		found, out, err := sub.Verify(ctx, opts)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(found).To(BeFalse())

		id, out, err := sub.Submit(ctx, opts)
		if err == nil {
			created = append(created, id)
		}
		var cmdErr *core.CommandError
		Expect(errors.As(err, &cmdErr)).To(BeTrue())
		Expect(out).To(ContainSubstring("no suitable queues"))
		Expect(core.IsLimitReached(err)).To(BeFalse())
	})

	It("detects the cluster wide AR limit", func() {
		global, err := qc.ShowGlobalConfiguration()
		Expect(err).NotTo(HaveOccurred())
		original := global.MaxAdvanceReservations
		DeferCleanup(func() {
			g, err := qc.ShowGlobalConfiguration()
			Expect(err).NotTo(HaveOccurred())
			g.MaxAdvanceReservations = original
			Expect(qc.ModifyGlobalConfig(*g)).To(Succeed())
		})

		submit(core.ReservationOptions{Duration: core.ToPtr(5 * time.Minute), Name: core.ToPtr(prefix + "lim")})
		all, err := stat.ListReservations(ctx, qrstat.ListOptions{Users: []string{"*"}})
		Expect(err).NotTo(HaveOccurred())
		global.MaxAdvanceReservations = len(all)
		Expect(qc.ModifyGlobalConfig(*global)).To(Succeed())

		id, _, err := sub.Submit(ctx, core.ReservationOptions{Duration: core.ToPtr(5 * time.Minute), Name: core.ToPtr(prefix + "lim2")})
		if err == nil {
			created = append(created, id)
		}
		Expect(core.IsLimitReached(err)).To(BeTrue(), "error: %v", err)
	})

	It("keeps the output of a partially failed delete", func() {
		const missing = 2147483000
		_, err := stat.GetReservations(ctx, missing)
		Expect(err).To(MatchError(qrstat.ErrNotFound), "AR %d must not exist for this spec", missing)

		id := submit(core.ReservationOptions{Duration: core.ToPtr(5 * time.Minute), Name: core.ToPtr(prefix + "part")})
		out, err := del.DeleteReservations(ctx, id, missing)
		var cmdErr *qrdel.CommandError
		Expect(errors.As(err, &cmdErr)).To(BeTrue())
		Expect(errors.Is(err, qrdel.ErrNotFound)).To(BeFalse())
		result := qrdel.ParseDeleteOutput(out)
		Expect(result.Deleted).To(Equal([]int64{id}))
		Expect(result.Missing).To(Equal([]string{fmt.Sprint(missing)}))
		_, err = stat.GetReservations(ctx, id)
		Expect(err).To(MatchError(qrstat.ErrNotFound))

		_, err = del.DeleteReservations(ctx, missing)
		Expect(err).To(MatchError(qrdel.ErrNotFound))
	})

	It("truncates long names with spaces in the summary but not in the details", func() {
		name := prefix + " with a long name"
		id := submit(core.ReservationOptions{Duration: core.ToPtr(5 * time.Minute), Name: &name})
		summaries, err := stat.ListReservations(ctx, qrstat.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		var summaryName string
		for _, s := range summaries {
			if s.ID == id {
				summaryName = s.Name
			}
		}
		Expect(summaryName).To(Equal(strings.TrimSpace(name[:10])))
		Expect(one(stat.GetReservations(ctx, id)).Name).To(Equal(name))
	})

	It("creates with the builder and deletes by owner and name", func() {
		name := prefix + "bld"
		id, out, err := core.NewReservationBuilder(sub).
			Name(name).
			Start(time.Now().Add(2 * time.Hour)).
			Duration(5 * time.Minute).
			Users(me).
			MailOptions("n").
			Submit(ctx)
		Expect(err).NotTo(HaveOccurred(), out)
		created = append(created, id)
		Expect(one(stat.GetReservations(ctx, id)).ACLList).To(Equal([]string{me}))

		out, err = del.DeleteReservationsByName(ctx, me, name)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(qrdel.ParseDeleteOutput(out).Deleted).To(Equal([]int64{id}))
		_, err = stat.GetReservations(ctx, id)
		Expect(err).To(MatchError(qrstat.ErrNotFound))
	})

	Context("with a second user", func() {
		BeforeEach(func() {
			if !haveOther {
				Skip("needs root to create the user " + otherUser)
			}
		})

		It("deletes by name only the ARs of the given owner (CS-2863)", func() {
			name := prefix + "same"
			mine := submit(core.ReservationOptions{Duration: core.ToPtr(5 * time.Minute), Name: &name})
			theirs := submitAsOther(name)

			out, err := del.DeleteReservationsByName(ctx, me, name)
			Expect(err).NotTo(HaveOccurred(), out)
			Expect(qrdel.ParseDeleteOutput(out).Deleted).To(Equal([]int64{mine}))
			Expect(one(stat.GetReservations(ctx, theirs)).Owner).To(Equal(otherUser))
		})

		It("lists the ARs of other users with *", func() {
			theirs := submitAsOther(prefix + "lst")
			summaries, err := stat.ListReservations(ctx, qrstat.ListOptions{Users: []string{"*"}})
			Expect(err).NotTo(HaveOccurred())
			owners := map[int64]string{}
			for _, s := range summaries {
				owners[s.ID] = s.Owner
			}
			Expect(owners).To(HaveKeyWithValue(theirs, otherUser))
		})

		It("deletes all ARs of a user, and only those", func() {
			mine := submit(core.ReservationOptions{Duration: core.ToPtr(5 * time.Minute), Name: core.ToPtr(prefix + "keep")})
			theirs := submitAsOther(prefix + "usr")

			out, err := del.DeleteReservationsByUser(ctx, otherUser)
			Expect(err).NotTo(HaveOccurred(), out)
			Expect(qrdel.ParseDeleteOutput(out).Deleted).To(ContainElement(theirs))
			_, err = stat.GetReservations(ctx, theirs)
			Expect(err).To(MatchError(qrstat.ErrNotFound))
			Expect(one(stat.GetReservations(ctx, mine)).Owner).To(Equal(me))
		})
	})
})
