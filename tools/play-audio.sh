#!/usr/bin/env bash
# Spike: run TESSERAE with sound. Builds audio-host/lg (the pinned let-go CLI
# with undertone's `audio` namespace linked in) on first use, then runs
# play.sh with it and with undertone's lg sources (the `ambience` namespace)
# on the search path. All play.sh flags pass through; --mute silences it.
# The lg sources come from the module audio-host/go.mod pins, so the Go code
# and the `ambience` namespace are always the same undertone version.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
host="$here/audio-host/lg"
if [[ ! -x "$host" || -n "${REBUILD:-}" ]]; then
  echo "building audio-host/lg ..." >&2
  (cd "$here/audio-host" && go build -o lg .)
fi
undertone="$(cd "$here/audio-host" && go list -m -f '{{.Dir}}' github.com/mparrett/undertone)"
LG="$host" TESS_SOURCE_PATHS=".:$undertone/letgo/lg" exec "$here/play.sh" "$@"
