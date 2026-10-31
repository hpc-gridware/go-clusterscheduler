#!/bin/sh
# Stands in for qquota in the streaming tests: records its arguments one per
# line in FAKE_QQUOTA_ARGS_FILE, prints FAKE_QQUOTA_FILE after an optional
# delay (FAKE_QQUOTA_SLEEP), writes FAKE_QQUOTA_STDERR to stderr
# (FAKE_QQUOTA_STDERR_BYTES more bytes of "x" after it), then sleeps
# FAKE_QQUOTA_HANG seconds and exits FAKE_QQUOTA_EXIT. FAKE_QQUOTA_LINGER first
# starts a background child sleeping that long with stdout open, like the
# qquota of a wrapper script, which outlives a killed wrapper.
if [ -n "$FAKE_QQUOTA_ARGS_FILE" ]; then printf '%s\n' "$@" > "$FAKE_QQUOTA_ARGS_FILE"; fi
if [ -n "$FAKE_QQUOTA_LINGER" ]; then sleep "$FAKE_QQUOTA_LINGER" & fi
if [ -n "$FAKE_QQUOTA_SLEEP" ]; then sleep "$FAKE_QQUOTA_SLEEP"; fi
if [ -n "$FAKE_QQUOTA_FILE" ]; then cat "$FAKE_QQUOTA_FILE"; fi
if [ -n "$FAKE_QQUOTA_STDERR" ]; then printf '%s\n' "$FAKE_QQUOTA_STDERR" >&2; fi
if [ -n "$FAKE_QQUOTA_STDERR_BYTES" ]; then head -c "$FAKE_QQUOTA_STDERR_BYTES" /dev/zero | tr '\0' x >&2; fi
if [ -n "$FAKE_QQUOTA_HANG" ]; then exec sleep "$FAKE_QQUOTA_HANG"; fi
exit "${FAKE_QQUOTA_EXIT:-0}"
