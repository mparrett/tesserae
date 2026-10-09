#!/usr/bin/env bash
# test.sh — run the test suites (test/*.lg) with the pinned let-go.
# Uses its own storage id so the meta tests never touch real progress.
here="$(cd "$(dirname "$0")/.." && pwd)"
LG="${LG:-$here/tools/lg-pinned}"
cd "$here" && exec "$LG" -storage-id tesserae-test -source-paths . test/run.lg "$@"
