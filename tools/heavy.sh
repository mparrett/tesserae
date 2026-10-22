#!/usr/bin/env bash
# heavy.sh <cmd...> — run a CPU-heavy job (go build, AOT lowering, long captures)
# behind a machine-wide lock so only one runs at a time (useful on a small box
# shared by several jobs). Waits up to HEAVY_WAIT seconds.
lock="${HEAVY_LOCK:-${TMPDIR:-/tmp}/tesserae-heavy.lock}"
wait="${HEAVY_WAIT:-3600}"
if command -v flock >/dev/null; then
  exec flock -w "$wait" "$lock" nice -n 10 "$@"
fi
# macOS ships lockf(1) rather than flock(1); same lock-then-exec shape.
if command -v lockf >/dev/null; then
  exec lockf -t "$wait" "$lock" nice -n 10 "$@"
fi
echo "heavy.sh: no flock or lockf; running without the lock" >&2
exec nice -n 10 "$@"
