#!/usr/bin/env bash
# Run TESSERAE with the pinned let-go (audio-host/go.mod's; see
# tools/build-lg.sh). LG overrides the binary.
# Progress is saved in let-go storage under the store id TESS_STORAGE_ID
# (default "tesserae": ~/.config/let-go/storage/k-dGVzc2VyYWU/ on Linux).
here="$(cd "$(dirname "$0")" && pwd)"
[[ -n ${LG:-} ]] || "$here/tools/build-lg.sh" || exit 1  # lg-pinned at audio-host/go.mod's let-go
LG="${LG:-$here/tools/lg-pinned}"
cd "$here" && exec "$LG" -storage-id "${TESS_STORAGE_ID:-tesserae}" -source-paths . main.lg "$@"
