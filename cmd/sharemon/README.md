# sharemon

Small example utility that wires the high-level share-tree API
(`pkg/qconf/core`) together with `qsub` and `sge_share_mon`:

1. Ensures a share tree exists. If the cluster has none, a two-project
   demo tree (`default`, `P1`, `P2`) is installed via
   `ModifyShareTreeStructured`.
2. Ensures both projects exist (`AddProject` is idempotent).
3. Submits a configurable number of `/bin/sleep` jobs under each
   project.
4. Polls `ShowShareTreeMonitoring` at a fixed interval and prints a
   compact per-node table.

Use `-clean` to tear down the demo state on exit.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-jobs` | 3 | jobs submitted per project |
| `-sleep` | 30 | seconds each job sleeps |
| `-interval` | 5s | polling interval for `sge_share_mon` |
| `-duration` | 90s | how long to monitor before exiting |
| `-projects` | `P1,P2` | comma-separated project names |
| `-clean` | false | delete demo projects + share tree on exit |
| `-setup-only` | false | install the tree + projects, then exit |
| `-monitor-only` | false | do not modify the cluster; just monitor |

## Reading the output

```
node               owner      shares   jobs     level%     actual     usage
----------------------------------------------------------------------------
/                  -          1        0        100.0%     0.0000     0.00
/P1                P1         100      0        47.6%      0.0000     0.00
/P2                P2         100      0        47.6%      0.0000     0.00
/default           -          10       0        4.8%       0.0000     0.00
/default/root      root       10       0        100.0%     0.0000     0.00
```

- **node**: path form exactly as emitted by `sge_share_mon` (`/`, `/P1`,
  `/default/root`). Root is always `/`.
- **owner**: `ProjectName` for project leaves, `UserName` for user
  leaves, `-` for interior nodes.
- **shares**: the `shares` attribute configured on the node.
- **jobs**: `job_count` reported by `sge_share_mon`. **Not the same as
  `qstat -s r` count** — see "Gotchas" below.
- **level%**: `level%` from the tool multiplied by 100. The raw field
  is a fraction in [0, 1] despite the `%` suffix; sharemon normalises
  at the UI layer so `0.476190` shows as `47.6%`.
- **actual**: `actual_share`, raw fraction in [0, 1].
- **usage**: cumulative decayed usage in the scheduler's units.

## Gotchas

### `sge_share_mon` delimiters

`-l` and `-r` take delimiters as **literal strings** (no backslash
escape interpretation). Passing `-l "\n\n"` inserts the four-character
sequence `\n\n` between records, not two newlines. `runShareMon` in
`pkg/qconf/core/share_tree_mon.go` invokes the tool with `-n` only;
the defaults (TAB between fields, `\n` between records) are already
the format `ParseShareMonOutput` expects.

### `level%` / `total%` are fractions

Despite the `%` suffix in the raw field names, `sge_share_mon` emits
values in `[0, 1]`. sharemon multiplies by 100 at display time so the
column header's `%` matches the printed value.

### `job_count` is not "running jobs"

`job_count` from `sge_share_mon` counts jobs the scheduler has *billed*
to a share-tree node, not jobs currently in the `r` state. On a
scheduler with `weight_tickets_share = 0` the share tree is passive
and `job_count` stays zero regardless of load. Turning on share
ticketing also requires other scheduler settings (`halflife`,
`usage_weight_*`) before `actual_share` and `usage` become
meaningful.

**Empirical observation (OCS 9.0.12 dev container):** even with
`weight_tickets_share=100`, a CPU-burning `-P P1` job running
continuously, and `usage`/`actual_share` populating correctly on
`/P1`, the `job_count` column stays at 0. Raw `sge_share_mon` reports
the same value — sharemon is faithful. The `actual`, `usage`, and
`cpu` columns are the reliable signals that a project or user leaf
is consuming resources.

```
# With a CPU-burning job running under P1:
node               owner      shares   jobs     level%     actual     usage
----------------------------------------------------------------------------
/                  -          1        0        100.0%     1.0000     3.01
/P1                P1         100      0        47.6%      0.9968     3.00
/P2                P2         100      0        47.6%      0.0032     0.01
/default           -          10       0        4.8%       0.0000     0.00
/default/root      root       10       0        100.0%     0.0000     0.00
```

`/P1` `actual=0.9968` means P1 is currently consuming 99.68 % of the
share-tree's tickets. `usage=3.00` is the decayed CPU-weighted
accumulator (units are scheduler-internal).

## Verifying parsing correctness

Compare sharemon against the raw tool side-by-side:

```bash
docker exec go-clusterscheduler bash -lc '
  source /opt/ocs/default/common/settings.sh
  $SGE_ROOT/utilbin/lx-amd64/sge_share_mon -c 1 -n
  cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/sharemon
  ./sharemon -monitor-only -duration 3s -interval 3s
'
```

Every numeric field in the sharemon table matches the raw tool's
value (modulo the `* 100` for `level%`).

## Building

```bash
cd cmd/sharemon
GOFLAGS=-buildvcs=false go build .
./sharemon -h
```
