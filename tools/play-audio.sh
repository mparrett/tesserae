#!/usr/bin/env bash
# Spike: run TESSERAE with sound. Builds audio-host/lg (the pinned let-go CLI
# with undertone's `audio` namespace linked in) on first use, then runs
# play.sh with it and with undertone's lg sources (the `ambience` namespace)
# on the search path. All play.sh flags pass through; --mute silences it.
#
# UNDERTONE is the undertone checkout to use; audio-host/go.mod's replace
# must point at the same one.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
UNDERTONE="${UNDERTONE:-$here/../let-go-undertone-tesserae-sfx}"
host="$here/audio-host/lg"
if [[ ! -x "$host" || -n "${REBUILD:-}" ]]; then
  echo "building audio-host/lg ..." >&2
  (cd "$here/audio-host" && go build -o lg .)
fi
LG="$host" TESS_SOURCE_PATHS=".:$UNDERTONE/letgo/lg" exec "$here/play.sh" "$@"
