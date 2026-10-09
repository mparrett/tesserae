#!/usr/bin/env bash
# Run TESSERAE with the pinned let-go. LG overrides the binary.
here="$(cd "$(dirname "$0")" && pwd)"
LG="${LG:-$here/tools/lg-pinned}"
cd "$here" && exec "$LG" -source-paths . main.lg "$@"
