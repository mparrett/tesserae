#!/usr/bin/env bash
# Run TESSERAE with the pinned let-go. LG overrides the binary.
# Progress is saved in let-go storage under the store id TESS_STORAGE_ID
# (default "tesserae": ~/.config/let-go/storage/k-dGVzc2VyYWU/ on Linux).
here="$(cd "$(dirname "$0")" && pwd)"
LG="${LG:-$here/tools/lg-pinned}"
cd "$here" && exec "$LG" -storage-id "${TESS_STORAGE_ID:-tesserae}" -source-paths . main.lg "$@"
