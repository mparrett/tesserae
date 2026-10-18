#!/usr/bin/env bash
# Spike: run TESSERAE with sound. Builds audio-host/lg (the pinned let-go CLI
# with undertone's `audio` namespace linked in) on first use, then runs
# play.sh with it. undertone registers its `undertone.ambience` source at init,
# so the Go code and the namespace are always the same undertone version and no
# source path is needed. All play.sh flags pass through; --mute silences it.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
host="$here/audio-host/lg"
if [[ ! -x "$host" || -n "${REBUILD:-}" ]]; then
  echo "building audio-host/lg ..." >&2
  (cd "$here/audio-host" && go build -o lg .)
fi
LG="$host" exec "$here/play.sh" "$@"
