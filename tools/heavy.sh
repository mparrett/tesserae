#!/usr/bin/env bash
# heavy.sh <cmd...> — run a CPU-heavy job (go build, AOT lowering, long captures)
# behind a machine-wide flock so only one runs at a time on this 4-vCPU box.
# Agents and the main session all go through this. Waits up to HEAVY_WAIT seconds.
lock="${HEAVY_LOCK:-$HOME/projects-new/.tesserae-heavy.lock}"
exec flock -w "${HEAVY_WAIT:-3600}" "$lock" nice -n 10 "$@"
