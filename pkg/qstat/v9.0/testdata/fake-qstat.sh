#!/bin/sh
# Stands in for qstat in the streaming tests: records its arguments one per
# line in FAKE_QSTAT_ARGS_FILE, prints FAKE_QSTAT_FILE after an optional
# delay, writes FAKE_QSTAT_STDERR to stderr (FAKE_QSTAT_STDERR_BYTES more
# bytes of "x" after it), then sleeps FAKE_QSTAT_HANG seconds (a qstat that
# has more to say) and exits FAKE_QSTAT_EXIT. FAKE_QSTAT_LINGER first starts
# a background child sleeping that long with stdout open, like the qstat of
# a wrapper script, which outlives a killed wrapper.
if [ -n "$FAKE_QSTAT_ARGS_FILE" ]; then printf '%s\n' "$@" > "$FAKE_QSTAT_ARGS_FILE"; fi
if [ -n "$FAKE_QSTAT_LINGER" ]; then sleep "$FAKE_QSTAT_LINGER" & fi
if [ -n "$FAKE_QSTAT_SLEEP" ]; then sleep "$FAKE_QSTAT_SLEEP"; fi
if [ -n "$FAKE_QSTAT_FILE" ]; then cat "$FAKE_QSTAT_FILE"; fi
if [ -n "$FAKE_QSTAT_STDERR" ]; then printf '%s\n' "$FAKE_QSTAT_STDERR" >&2; fi
if [ -n "$FAKE_QSTAT_STDERR_BYTES" ]; then head -c "$FAKE_QSTAT_STDERR_BYTES" /dev/zero | tr '\0' x >&2; fi
if [ -n "$FAKE_QSTAT_HANG" ]; then exec sleep "$FAKE_QSTAT_HANG"; fi
exit "${FAKE_QSTAT_EXIT:-0}"
