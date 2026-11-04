#!/bin/sh
# Stands in for qalter in the verify tests: records its arguments one per
# line in FAKE_QALTER_ARGS_FILE and its environment in FAKE_QALTER_ENV_FILE,
# sleeps FAKE_QALTER_SLEEP seconds, prints FAKE_QALTER_FILE on stdout, writes
# FAKE_QALTER_STDERR to stderr and exits FAKE_QALTER_EXIT (default 0).
if [ -n "$FAKE_QALTER_ARGS_FILE" ]; then printf '%s\n' "$@" > "$FAKE_QALTER_ARGS_FILE"; fi
if [ -n "$FAKE_QALTER_ENV_FILE" ]; then env > "$FAKE_QALTER_ENV_FILE"; fi
if [ -n "$FAKE_QALTER_SLEEP" ]; then sleep "$FAKE_QALTER_SLEEP"; fi
if [ -n "$FAKE_QALTER_FILE" ]; then cat "$FAKE_QALTER_FILE"; fi
if [ -n "$FAKE_QALTER_STDERR" ]; then printf '%s\n' "$FAKE_QALTER_STDERR" >&2; fi
exit "${FAKE_QALTER_EXIT:-0}"
