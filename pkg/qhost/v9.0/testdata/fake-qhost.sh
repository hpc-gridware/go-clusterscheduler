#!/bin/sh
# Stands in for qhost in the streaming tests: records its arguments one per
# line in FAKE_QHOST_ARGS_FILE, prints FAKE_QHOST_FILE after an optional
# delay (FAKE_QHOST_SLEEP), writes FAKE_QHOST_STDERR to stderr
# (FAKE_QHOST_STDERR_BYTES more bytes of "x" after it), then sleeps
# FAKE_QHOST_HANG seconds and exits FAKE_QHOST_EXIT. FAKE_QHOST_LINGER first
# starts a background child sleeping that long with stdout open, like the
# qhost of a wrapper script, which outlives a killed wrapper.
if [ -n "$FAKE_QHOST_ARGS_FILE" ]; then printf '%s\n' "$@" > "$FAKE_QHOST_ARGS_FILE"; fi
if [ -n "$FAKE_QHOST_LINGER" ]; then sleep "$FAKE_QHOST_LINGER" & fi
if [ -n "$FAKE_QHOST_SLEEP" ]; then sleep "$FAKE_QHOST_SLEEP"; fi
if [ -n "$FAKE_QHOST_FILE" ]; then cat "$FAKE_QHOST_FILE"; fi
if [ -n "$FAKE_QHOST_STDERR" ]; then printf '%s\n' "$FAKE_QHOST_STDERR" >&2; fi
if [ -n "$FAKE_QHOST_STDERR_BYTES" ]; then head -c "$FAKE_QHOST_STDERR_BYTES" /dev/zero | tr '\0' x >&2; fi
if [ -n "$FAKE_QHOST_HANG" ]; then exec sleep "$FAKE_QHOST_HANG"; fi
exit "${FAKE_QHOST_EXIT:-0}"
