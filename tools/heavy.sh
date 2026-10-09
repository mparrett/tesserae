#!/usr/bin/env bash
# heavy.sh <cmd...> — run a CPU-heavy job (go build, AOT lowering, long captures)
# behind a machine-wide flock so only one runs at a time (useful on a small box
# shared by several jobs). Waits up to HEAVY_WAIT seconds.
lock="${HEAVY_LOCK:-${TMPDIR:-/tmp}/tesserae-heavy.lock}"
exec flock -w "${HEAVY_WAIT:-3600}" "$lock" nice -n 10 "$@"
