/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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

// Integration tests that exercise the share-tree operations against a
// real Open Cluster Scheduler. They mutate cluster state (share tree,
// temporary projects) and therefore must never run against a production
// cluster. Tests self-skip when:
//
//   - env GOCS_SKIP_CLUSTER_TESTS is set (explicit opt-out), OR
//   - qconf is not on PATH, OR
//   - qconf cannot answer a simple status probe (cluster not reachable).
//
// Use env GOCS_CLUSTER_TESTS=1 to document intent in CI output; the auto-
// detection does the real work so developers inside the dev container do
// not need to remember the flag.
//
// Every spec saves the pre-test share tree and any projects it creates
// in BeforeEach and restores them in AfterEach, so the test environment
// is left in the same state it was found.

package core_test

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// clusterAvailable reports whether we can talk to a live qconf without
// disturbing it. The probe runs "qconf -sconf global", which is read-only
// and always defined on a configured scheduler.
func clusterAvailable() bool {
	if v := os.Getenv("GOCS_SKIP_CLUSTER_TESTS"); v != "" && v != "0" {
		return false
	}
	if _, err := exec.LookPath("qconf"); err != nil {
		return false
	}
	cmd := exec.Command("qconf", "-sconf", "global")
	return cmd.Run() == nil
}

// skipIfNoCluster skips the current spec unless a live cluster is available.
func skipIfNoCluster() {
	if !clusterAvailable() {
		Skip("no reachable cluster; set GOCS_CLUSTER_TESTS=1 inside the dev container to run these integration tests")
	}
}

// newClusterQConf builds a CommandLineQConf pointed at the real qconf on
// PATH. A DelayAfter keeps qmaster happy if the cluster is heavily loaded.
func newClusterQConf() *core.CommandLineQConf {
	qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{
		Executable: "qconf",
		DelayAfter: 50 * time.Millisecond,
	})
	Expect(err).NotTo(HaveOccurred())
	return qc
}

// saveShareTree captures the current share tree as a text blob for later
// restoration. An empty string signals "no tree was configured"; callers
// restore that state by running DeleteShareTree.
func saveShareTree(qc *core.CommandLineQConf) string {
	txt, err := qc.ShowShareTree()
	if err != nil {
		if strings.Contains(err.Error(), "no sharetree") ||
			strings.Contains(err.Error(), "no share tree") {
			return ""
		}
		// Any other failure is unexpected at this point.
		Expect(err).NotTo(HaveOccurred())
	}
	return txt
}

// restoreShareTree writes the original share tree back; if the original
// was empty, the current tree is deleted instead.
func restoreShareTree(qc *core.CommandLineQConf, saved string) {
	if saved == "" {
		_ = qc.DeleteShareTree()
		return
	}
	_ = qc.ModifyShareTree(saved)
}

// ensureProjects idempotently creates the named projects and returns the
// set that the caller is responsible for removing in cleanup.
func ensureProjects(qc *core.CommandLineQConf, names ...string) []string {
	created := make([]string, 0, len(names))
	for _, n := range names {
		if _, err := qc.ShowProject(n); err == nil {
			continue
		}
		if err := qc.AddProject(core.ProjectConfig{Name: n}); err == nil {
			created = append(created, n)
		}
	}
	return created
}

// ensureUsers idempotently creates the named qconf users so share-tree
// leaf references resolve (qmaster refuses -Mstree when it encounters a
// leaf name that is neither a real user nor the reserved "default").
// Returns the names the caller must clean up.
func ensureUsers(qc *core.CommandLineQConf, names ...string) []string {
	created := make([]string, 0, len(names))
	for _, n := range names {
		if _, err := qc.ShowUser(n); err == nil {
			continue
		}
		if err := qc.AddUser(core.UserConfig{Name: n}); err == nil {
			created = append(created, n)
		}
	}
	return created
}

// cleanupKnownLeftovers removes users and projects that earlier
// crashed-test runs left behind. The share tree must already be
// dropped or restored; otherwise qconf refuses to delete referenced
// entries. All errors are ignored — this is a belt-and-braces step
// intended to keep the container reproducible across CI runs.
func cleanupKnownLeftovers(qc *core.CommandLineQConf) {
	for _, u := range []string{"alice", "devel", "kurt", "kurt_clone"} {
		_ = qc.DeleteUser([]string{u})
	}
	_ = qc.DeleteProject([]string{"P30"})
}

var _ = Describe("Share Tree integration (live cluster)", func() {

	var (
		qc         *core.CommandLineQConf
		savedTree  string
		projectsWE []string
	)

	BeforeEach(func() {
		skipIfNoCluster()
		qc = newClusterQConf()
		savedTree = saveShareTree(qc)
		// Most specs assume projects P10 and P20 exist so the share-tree
		// under test references real objects.
		projectsWE = ensureProjects(qc, "P10", "P20")
	})

	AfterEach(func() {
		if qc == nil {
			return
		}
		restoreShareTree(qc, savedTree)
		if len(projectsWE) > 0 {
			_ = qc.DeleteProject(projectsWE)
		}
		// Drop the restored tree and recreate it so leftover
		// orphan-cleanup runs against an empty tree; otherwise qconf
		// refuses to remove users/projects still referenced by the
		// restored tree.
		if savedTree != "" {
			_ = qc.DeleteShareTree()
			cleanupKnownLeftovers(qc)
			_ = qc.ModifyShareTree(savedTree)
		} else {
			cleanupKnownLeftovers(qc)
		}
	})

	// A helper tree used across multiple specs. Matches the exact format
	// qconf emits so ShowShareTree/ModifyShareTree round-trips cleanly.
	shareTreeConfig := `id=0
name=Root
type=0
shares=1
childnodes=1,2,3
id=1
name=default
type=0
shares=10
childnodes=NONE
id=2
name=P20
type=1
shares=11
childnodes=NONE
id=3
name=P10
type=1
shares=11
childnodes=NONE
`

	Describe("ShowShareTree / ModifyShareTree text format", func() {
		It("round-trips a simple tree through qconf -Mstree / -sstree", func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())

			got, err := qc.ShowShareTree()
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(shareTreeConfig))
		})

		It("reports a missing tree after DeleteShareTree", func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())
			Expect(qc.DeleteShareTree()).To(Succeed())

			_, err := qc.ShowShareTree()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no sharetree"))
		})
	})

	Describe("ShowShareTreeNodes / Add / Modify / Delete node-level ops", func() {
		BeforeEach(func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())
		})

		It("lists all nodes with their shares", func() {
			nodes, err := qc.ShowShareTreeNodes(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P10", Share: 11}))
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P20", Share: 11}))
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/default", Share: 10}))
		})

		It("strips a /Root prefix in path arguments (qconf -sstnode rejects it)", func() {
			nodes, err := qc.ShowShareTreeNodes([]string{"/Root/P10"})
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P10", Share: 11}))
		})

		It("modifies existing node shares via qconf -mstnode", func() {
			Expect(qc.ModifyShareTreeNodes([]core.ShareTreeNode{
				{Node: "/P10", Share: 77},
			})).To(Succeed())

			nodes, err := qc.ShowShareTreeNodes([]string{"/P10"})
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P10", Share: 77}))
		})

		It("adds a new node via qconf -astnode and removes it via -dstnode", func() {
			// Add a fresh project node. P10/P20 already exist so reuse
			// one as the new node's target is not required; we add a new
			// project first to avoid colliding with existing entries.
			created := ensureProjects(qc, "P30")
			defer func() {
				if len(created) > 0 {
					_ = qc.DeleteProject(created)
				}
			}()

			Expect(qc.AddShareTreeNode(core.ShareTreeNode{
				Node: "/P30", Share: 5,
			})).To(Succeed())
			nodes, err := qc.ShowShareTreeNodes(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P30", Share: 5}))

			Expect(qc.DeleteShareTreeNodes([]string{"/P30"})).To(Succeed())
			nodes, err = qc.ShowShareTreeNodes(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).NotTo(ContainElement(core.ShareTreeNode{Node: "/P30", Share: 5}))
		})
	})

	Describe("ShowShareTreeStructured / ModifyShareTreeStructured", func() {
		It("writes a structured tree and reads it back with matching shape", func() {
			src := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root", Type: core.ShareTreeNodeUser, Shares: 1,
					Children: []*core.StructuredShareTreeNode{
						{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
						{Name: "P10", Type: core.ShareTreeNodeProject, Shares: 42},
						{Name: "P20", Type: core.ShareTreeNodeProject, Shares: 21},
					},
				},
			}
			Expect(qc.ModifyShareTreeStructured(src)).To(Succeed())

			got, err := qc.ShowShareTreeStructured()
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Root.Name).To(Equal("Root"))
			Expect(got.Root.Children).To(HaveLen(3))

			shares := map[string]int{}
			types := map[string]core.ShareTreeNodeType{}
			for _, c := range got.Root.Children {
				shares[c.Name] = c.Shares
				types[c.Name] = c.Type
			}
			Expect(shares).To(HaveKeyWithValue("P10", 42))
			Expect(shares).To(HaveKeyWithValue("P20", 21))
			Expect(shares).To(HaveKeyWithValue("default", 10))
			Expect(types).To(HaveKeyWithValue("P10", core.ShareTreeNodeProject))
			Expect(types).To(HaveKeyWithValue("default", core.ShareTreeNodeUser))
		})

		It("returns ErrNoShareTree when no tree is configured", func() {
			Expect(qc.DeleteShareTree()).To(Succeed())
			_, err := qc.ShowShareTreeStructured()
			Expect(errors.Is(err, core.ErrNoShareTree)).To(BeTrue())
		})
	})

	Describe("Subtree operations (Add/Modify/Delete/Copy/Move)", func() {
		BeforeEach(func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())
		})

		It("returns a copy of the subtree at a path", func() {
			sub, err := qc.ShowShareTreeSubtree("/Root/P10")
			Expect(err).NotTo(HaveOccurred())
			Expect(sub).NotTo(BeNil())
			Expect(sub.Name).To(Equal("P10"))
			Expect(sub.Type).To(Equal(core.ShareTreeNodeProject))
			Expect(sub.Shares).To(Equal(11))
		})

		It("returns SHARE_PATH_NOT_FOUND for a bogus path", func() {
			_, err := qc.ShowShareTreeSubtree("/Root/NoSuchThing")
			Expect(err).To(HaveOccurred())
			var ve *core.ShareTreeValidationErrors
			ok := errors.As(err, &ve)
			Expect(ok).To(BeTrue())
			Expect(ve.Errs[0].Code).To(Equal(core.ShareCodePathNotFound))
		})

		It("adds a subtree under Root via read-mutate-write", func() {
			created := ensureProjects(qc, "P30")
			defer func() {
				if len(created) > 0 {
					_ = qc.DeleteProject(created)
				}
			}()
			addition := &core.StructuredShareTreeNode{
				Name: "P30", Type: core.ShareTreeNodeProject, Shares: 33,
			}
			Expect(qc.AddShareTreeSubtree("/Root", addition)).To(Succeed())

			nodes, err := qc.ShowShareTreeNodes(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P30", Share: 33}))
		})

		It("replaces a subtree at a path", func() {
			replacement := &core.StructuredShareTreeNode{
				Name: "P10", Type: core.ShareTreeNodeProject, Shares: 88,
			}
			Expect(qc.ModifyShareTreeSubtree("/Root/P10", replacement)).To(Succeed())

			nodes, err := qc.ShowShareTreeNodes([]string{"/P10"})
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P10", Share: 88}))
		})

		It("deletes a subtree and removes all its descendants", func() {
			Expect(qc.DeleteShareTreeSubtree("/Root/P10")).To(Succeed())
			nodes, err := qc.ShowShareTreeNodes(nil)
			Expect(err).NotTo(HaveOccurred())
			for _, n := range nodes {
				Expect(n.Node).NotTo(Equal("/P10"))
			}
		})

		It("moves a subtree under a new parent", func() {
			// qmaster has two seemingly-contradictory rules that together
			// force interior grouping nodes to be *projects*, not users:
			//   1. Every non-"default" name in the tree must resolve to
			//      a known user or project.
			//   2. A known user cannot appear as a non-leaf node.
			// The nested testdata file follows the same pattern
			// (projects with user-leaves under them). We therefore use
			// P10/P20 (created in the outer BeforeEach) as the interior
			// parents and alice as the moved leaf.
			users := ensureUsers(qc, "alice")
			defer func() {
				if len(users) > 0 {
					_ = qc.DeleteUser(users)
				}
			}()

			src := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root", Shares: 1,
					Children: []*core.StructuredShareTreeNode{
						{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
						{Name: "P10", Type: core.ShareTreeNodeProject, Shares: 100,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser, Shares: 5},
							},
						},
						{Name: "P20", Type: core.ShareTreeNodeProject, Shares: 11},
					},
				},
			}
			Expect(qc.ModifyShareTreeStructured(src)).To(Succeed())

			Expect(qc.MoveShareTreeSubtree("/Root/P10/alice", "/Root/P20")).To(Succeed())
			moved, err := qc.ShowShareTreeSubtree("/Root/P20/alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(moved.Shares).To(Equal(5))

			_, err = qc.ShowShareTreeSubtree("/Root/P10/alice")
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("ClearShareTreeUsage", func() {
		It("succeeds against a real qmaster", func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())
			Expect(qc.ClearShareTreeUsage()).To(Succeed())
		})
	})

	Describe("ShowShareTreeMonitoring (sge_share_mon)", func() {
		It("returns a snapshot whose Root entry matches the tree shares", func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())

			mon, err := qc.ShowShareTreeMonitoring()
			if err != nil && errors.Is(err, core.ErrShareTreeMonNotAvail) {
				Skip("sge_share_mon not available on PATH / utilbin: " + err.Error())
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(mon).NotTo(BeNil())
			Expect(mon.CollectedAt.IsZero()).To(BeFalse())
			// sge_share_mon emits node_name in path form ("/", "/P1",
			// "/default/alice", ...); the root is the literal "/".
			Expect(mon.Nodes).To(HaveKey("/"))
			Expect(mon.Nodes["/"].Shares).To(Equal(1))
			// Each project node carries its ProjectName; user leaves
			// carry their UserName.
			if n, ok := mon.Nodes["/P10"]; ok {
				Expect(n.ProjectName).To(Equal("P10"))
			}
		})

		It("maps 'No share tree' (sge_share_mon rc=2) to ErrNoShareTree", func() {
			// Tear down the tree before invoking share_mon so we hit
			// the empty-tree code path that the OCS 9.0.x scheduler
			// signals with exit code 2 and stdout body "No share tree".
			Expect(qc.DeleteShareTree()).To(Succeed())

			_, err := qc.ShowShareTreeMonitoring()
			Expect(err).To(HaveOccurred())
			if errors.Is(err, core.ErrShareTreeMonNotAvail) {
				Skip("sge_share_mon binary not reachable: " + err.Error())
			}
			Expect(errors.Is(err, core.ErrNoShareTree)).To(BeTrue(),
				"expected ErrNoShareTree, got %v", err)
		})
	})

	Describe("Edge cases surfaced by ground-truth capture", func() {
		It("qconf -astnode on an existing node overwrites (upsert) rather than failing", func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())

			// /P10 already has share=11 in the fixture. Re-adding with
			// a different share must succeed and replace the value.
			Expect(qc.AddShareTreeNode(core.ShareTreeNode{
				Node: "/P10", Share: 123,
			})).To(Succeed())
			nodes, err := qc.ShowShareTreeNodes([]string{"/P10"})
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P10", Share: 123}))
		})

		It("ModifyShareTreeNodes surfaces partial-success 'Unable to locate' as an error", func() {
			Expect(qc.ModifyShareTree(shareTreeConfig)).To(Succeed())

			// qconf -mstnode applies /P10=1 and warns on /bogus=2, then
			// exits 0. Our wrapper must detect the warning and return
			// a non-nil error so callers cannot miss the miss.
			err := qc.ModifyShareTreeNodes([]core.ShareTreeNode{
				{Node: "/P10", Share: 1},
				{Node: "/bogus", Share: 2},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("share tree node not found"))

			// The valid half of the batch IS applied before the error
			// is returned. This is a documented partial-success
			// compromise: the underlying tool silently writes /P10.
			nodes, _ := qc.ShowShareTreeNodes([]string{"/P10"})
			Expect(nodes).To(ContainElement(core.ShareTreeNode{Node: "/P10", Share: 1}))
		})

		It("ShowShareTree on an empty cluster returns an error whose text contains 'no sharetree'", func() {
			Expect(qc.DeleteShareTree()).To(Succeed())
			_, err := qc.ShowShareTree()
			Expect(err).To(HaveOccurred())
			Expect(strings.ToLower(err.Error())).To(ContainSubstring("no sharetree"))
		})

		It("qmaster rejects a tree that references an unknown interior name", func() {
			// "NotAProject" does not exist as user or project; qmaster
			// answers: denied: share tree contains reference to unknown
			// user/project "NotAProject".
			src := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root", Shares: 1,
					Children: []*core.StructuredShareTreeNode{
						{Name: "NotAProject", Type: core.ShareTreeNodeProject, Shares: 100},
					},
				},
			}
			err := qc.ModifyShareTreeStructured(src)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown user/project"))
		})

		It("qmaster rejects a registered user used as a non-leaf node", func() {
			// Register alice, then place her as an interior node with a
			// child. qmaster answers: found user "alice" in share tree
			// as a non-leaf node.
			users := ensureUsers(qc, "alice")
			defer func() {
				if len(users) > 0 {
					_ = qc.DeleteUser(users)
				}
			}()

			src := &core.StructuredShareTree{
				Root: &core.StructuredShareTreeNode{
					Name: "Root", Shares: 1,
					Children: []*core.StructuredShareTreeNode{
						{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
						{Name: "P10", Type: core.ShareTreeNodeProject, Shares: 100,
							Children: []*core.StructuredShareTreeNode{
								{Name: "alice", Type: core.ShareTreeNodeUser, Shares: 5,
									Children: []*core.StructuredShareTreeNode{
										{Name: "default", Type: core.ShareTreeNodeUser, Shares: 1},
									},
								},
							},
						},
					},
				},
			}
			err := qc.ModifyShareTreeStructured(src)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("non-leaf"))
		})
	})
})
