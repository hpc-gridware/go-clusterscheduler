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

// Example utility that wires the high-level share-tree API together with
// qsub and sge_share_mon:
//
//  1. Ensures a share tree exists. If the cluster has none, a small demo
//     tree with two project leaves (P1, P2) plus "default" is installed.
//  2. Ensures both projects exist (AddProject is idempotent here).
//  3. Submits a configurable number of sleep jobs per project so
//     sge_share_mon has real usage to report.
//  4. Polls ShowShareTreeMonitoring at a fixed interval and prints a
//     compact per-node table (shares, running jobs, usage, actual
//     share, level%).
//
// The utility is intentionally read-heavy: it restores nothing at the
// end because long-running share tree effects (usage decay) are the
// interesting signal. Use -clean to tear down the demo state.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
	qsubcore "github.com/hpc-gridware/go-clusterscheduler/pkg/qsub/core"
)

type config struct {
	jobsPerProject int
	perJobSleep    int
	interval       time.Duration
	duration       time.Duration
	projects       []string
	clean          bool
	setupOnly      bool
	monitorOnly    bool
}

func main() {
	cfg := parseFlags()

	qc, err := core.NewCommandLineQConf(core.CommandLineQConfConfig{
		Executable: "qconf",
	})
	if err != nil {
		fatal("failed to construct qconf client: %v", err)
	}

	if !cfg.monitorOnly {
		if err := ensureShareTree(qc, cfg.projects); err != nil {
			fatal("share-tree setup: %v", err)
		}
	}

	if cfg.setupOnly {
		fmt.Println("share tree ready; exiting (-setup-only)")
		return
	}

	// SIGINT / SIGTERM stop monitoring gracefully. Submitted jobs keep
	// running in qmaster; operators can qdel them by name.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if !cfg.monitorOnly {
		if err := submitDemoJobs(ctx, cfg); err != nil {
			fatal("job submission: %v", err)
		}
	}

	if err := monitor(ctx, qc, cfg); err != nil && !errors.Is(err, context.Canceled) {
		fatal("monitor loop: %v", err)
	}

	if cfg.clean {
		cleanup(qc, cfg.projects)
	}
}

// parseFlags reads CLI flags and returns a populated config. Defaults
// are tuned to produce visible output within ~1 minute against a small
// test cluster.
// sharemon is a small single-file example; stdlib flag is sufficient and cobra overhead is not warranted.
func parseFlags() config {
	cfg := config{}
	projects := ""

	flag.IntVar(&cfg.jobsPerProject, "jobs", 3, "jobs to submit per project")
	flag.IntVar(&cfg.perJobSleep, "sleep", 30, "seconds each job sleeps")
	flag.DurationVar(&cfg.interval, "interval", 5*time.Second, "share_mon polling interval")
	flag.DurationVar(&cfg.duration, "duration", 90*time.Second, "how long to monitor before exiting")
	flag.StringVar(&projects, "projects", "P1,P2", "comma-separated project names to demo")
	flag.BoolVar(&cfg.clean, "clean", false, "delete the demo projects and drop the share tree at the end")
	flag.BoolVar(&cfg.setupOnly, "setup-only", false, "install the demo share tree and exit")
	flag.BoolVar(&cfg.monitorOnly, "monitor-only", false, "do not modify the cluster; just monitor the current tree")
	flag.Parse()

	for _, p := range strings.Split(projects, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			cfg.projects = append(cfg.projects, p)
		}
	}
	if len(cfg.projects) == 0 {
		fatal("at least one project must be given via -projects")
	}
	return cfg
}

// ensureShareTree installs a two-project demo tree when the cluster has
// no share tree configured. If a tree already exists, this function
// leaves it alone so the utility can run safely against a live setup.
func ensureShareTree(qc *core.CommandLineQConf, projects []string) error {
	for _, p := range projects {
		if _, err := qc.ShowProject(p); err != nil {
			if addErr := qc.AddProject(core.ProjectConfig{Name: p}); addErr != nil {
				return fmt.Errorf("AddProject %s: %w", p, addErr)
			}
			fmt.Printf("created project %s\n", p)
		}
	}

	current, err := qc.ShowShareTreeStructured()
	if err == nil {
		fmt.Printf("share tree already configured (%d top-level nodes); leaving it in place\n",
			len(current.Root.Children))
		return nil
	}
	if !errors.Is(err, core.ErrNoShareTree) {
		return fmt.Errorf("ShowShareTreeStructured: %w", err)
	}

	children := []*core.StructuredShareTreeNode{
		{Name: "default", Type: core.ShareTreeNodeUser, Shares: 10},
	}
	// Give each project an equal share so the weight_tickets effects
	// produced by the sample jobs are easy to interpret.
	for _, p := range projects {
		children = append(children, &core.StructuredShareTreeNode{
			Name: p, Type: core.ShareTreeNodeProject, Shares: 100,
		})
	}
	tree := &core.StructuredShareTree{
		Root: &core.StructuredShareTreeNode{
			Name: "Root", Type: core.ShareTreeNodeUser, Shares: 1,
			Children: children,
		},
	}
	if err := qc.ModifyShareTreeStructured(tree); err != nil {
		return fmt.Errorf("ModifyShareTreeStructured: %w", err)
	}
	fmt.Printf("installed demo share tree with projects: %s\n", strings.Join(projects, ", "))
	return nil
}

// submitDemoJobs sprays simple sleep jobs across the configured projects
// via qsub -P <project>. Each submission uses -b y so no wrapper script
// is required; the job name is stamped so operators can find/clean them.
func submitDemoJobs(ctx context.Context, cfg config) error {
	qs, err := qsubcore.NewCommandLineQSub(qsubcore.CommandLineQSubConfig{
		QsubPath: "qsub",
	})
	if err != nil {
		return fmt.Errorf("NewCommandLineQSub: %w", err)
	}

	for _, project := range cfg.projects {
		for i := 0; i < cfg.jobsPerProject; i++ {
			name := fmt.Sprintf("sharemon-%s-%d", project, i)
			opts := &qsubcore.JobOptions{
				JobName: qsubcore.ToPtr(name),
				Project: qsubcore.ToPtr(project),
				Binary:  qsubcore.ToPtr(true),
				StdOut:  []string{"/dev/null"},
				StdErr:  []string{"/dev/null"},
			}
			_, out, err := qs.SubmitSimple(ctx, opts, "/bin/sleep",
				fmt.Sprintf("%d", cfg.perJobSleep))
			if err != nil {
				return fmt.Errorf("submit %s: %w", name, err)
			}
			fmt.Printf("submitted %s -> %s\n", name, strings.TrimSpace(out))
		}
	}
	return nil
}

// monitor polls ShowShareTreeMonitoring on a fixed interval and prints a
// compact per-node table until the configured duration elapses or the
// context is cancelled (e.g. Ctrl+C).
func monitor(ctx context.Context, qc *core.CommandLineQConf, cfg config) error {
	deadline, cancel := context.WithTimeout(ctx, cfg.duration)
	defer cancel()

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	// Print one snapshot right away so users see data without waiting
	// a full interval.
	printSnapshot(qc)

	for {
		select {
		case <-deadline.Done():
			if err := deadline.Err(); errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return deadline.Err()
		case <-ticker.C:
			printSnapshot(qc)
		}
	}
}

// printSnapshot pulls a single sge_share_mon sample and prints it as a
// fixed-width table. sge_share_mon is typically in $SGE_ROOT/utilbin,
// not on PATH; the core package already handles that fallback.
func printSnapshot(qc *core.CommandLineQConf) {
	mon, err := qc.ShowShareTreeMonitoring()
	if err != nil {
		now := time.Now().Format(time.RFC3339)
		switch {
		case errors.Is(err, core.ErrNoShareTree):
			fmt.Printf("[%s] no share tree configured on this cluster\n", now)
		case errors.Is(err, core.ErrShareTreeMonNotAvail):
			fmt.Printf("[%s] sge_share_mon unavailable: %v\n", now, err)
		default:
			fmt.Printf("[%s] share_mon error: %v\n", now, err)
		}
		return
	}

	fmt.Printf("\n[%s] share tree snapshot (%d nodes)\n",
		mon.CollectedAt.Format(time.RFC3339), len(mon.Nodes))
	// "level" (and "total") are fractions in [0, 1] as emitted by
	// sge_share_mon even though the raw field name is "level%".
	// printRow multiplies by 100 so the column header's percent sign
	// matches the displayed value.
	fmt.Printf("%-18s %-10s %-8s %-8s %-10s %-10s %-10s\n",
		"node", "owner", "shares", "jobs", "level%", "actual", "usage")
	fmt.Println(strings.Repeat("-", 76))

	// Iterate map in a stable, human-friendly order: root ("/") first,
	// then alphabetical by node_name (which is also in path form).
	printed := map[string]bool{}
	if root, ok := mon.Nodes["/"]; ok {
		printRow(root)
		printed["/"] = true
	}
	names := make([]string, 0, len(mon.Nodes))
	for n := range mon.Nodes {
		if !printed[n] {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		printRow(mon.Nodes[n])
	}
}

func printRow(n core.ShareTreeNodeStats) {
	// Owner: project leaves carry a ProjectName; user leaves carry a
	// UserName; interior nodes carry neither and print "-".
	owner := n.ProjectName
	if owner == "" {
		owner = n.UserName
	}
	if owner == "" {
		owner = "-"
	}
	// sge_share_mon's level% field is a fraction; multiply to display
	// a true percentage matching the column header.
	levelPct := fmt.Sprintf("%.1f%%", n.LevelPercent*100)
	fmt.Printf("%-18s %-10s %-8d %-8d %-10s %-10.4f %-10.2f\n",
		truncate(n.NodeName, 18), truncate(owner, 10),
		n.Shares, n.JobCount, levelPct, n.ActualShare, n.Usage)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// cleanup removes the demo projects and drops the share tree. It logs
// errors but does not abort: the tool is best-effort on teardown.
func cleanup(qc *core.CommandLineQConf, projects []string) {
	if err := qc.DeleteShareTree(); err != nil {
		fmt.Fprintf(os.Stderr, "DeleteShareTree: %v\n", err)
	}
	if err := qc.DeleteProject(projects); err != nil {
		fmt.Fprintf(os.Stderr, "DeleteProject %v: %v\n", projects, err)
	}
	fmt.Println("cleanup done")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "sharemon: "+format+"\n", args...)
	os.Exit(1)
}
