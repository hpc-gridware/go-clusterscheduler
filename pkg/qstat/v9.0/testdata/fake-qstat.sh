#!/bin/sh
# Stands in for qstat in the streaming tests: prints FAKE_QSTAT_FILE after an
# optional delay, writes FAKE_QSTAT_STDERR to stderr, exits FAKE_QSTAT_EXIT.
if [ -n "$FAKE_QSTAT_SLEEP" ]; then sleep "$FAKE_QSTAT_SLEEP"; fi
if [ -n "$FAKE_QSTAT_FILE" ]; then cat "$FAKE_QSTAT_FILE"; fi
if [ -n "$FAKE_QSTAT_STDERR" ]; then echo "$FAKE_QSTAT_STDERR" >&2; fi
exit "${FAKE_QSTAT_EXIT:-0}"
