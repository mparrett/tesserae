#!/usr/bin/env bash
# build-lg.sh — point tools/lg-pinned at a stock lg built from the let-go
# version audio-host/go.mod pins, building it first if needed.
#
# The tests, sim and stock web build run tools/lg-pinned, and the Pages build
# compiles against audio-host/go.mod. This keeps them on the same let-go: go.mod
# is the only place the version is written. Each version builds once, into
# tools/.lg/lg-<version> (gitignored); a bump in go.mod builds the new one.
#
# lg reads its own version from the Go build info (pkg/cli/wasm.go), so lg -w
# fetches the same let-go source for the wasm it compiles.
#
# To run a different let-go (a local checkout, say), set LG=/path/to/lg instead;
# the scripts that use tools/lg-pinned take LG first.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
ver="$(go list -C "$here/audio-host" -m -f '{{.Version}}' github.com/nooga/let-go)"
[[ -n $ver ]] || { echo "build-lg: no let-go version in audio-host/go.mod" >&2; exit 1; }
bin="$here/tools/.lg/lg-$ver"
if [[ ! -x $bin ]]; then
  echo "build-lg: building lg at let-go $ver ..." >&2
  mkdir -p "$here/tools/.lg"
  # Built in audio-host's module, so go.mod and go.sum fix the version and its
  # dependencies. -s -w as let-go's own Makefile builds it.
  go build -C "$here/audio-host" -ldflags="-s -w" -o "$bin.tmp" github.com/nooga/let-go
  mv "$bin.tmp" "$bin"
fi
# Replaces a hand-made symlink from before this script, too.
ln -sfn ".lg/lg-$ver" "$here/tools/lg-pinned"
