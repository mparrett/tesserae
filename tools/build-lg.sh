#!/usr/bin/env bash
# build-lg.sh — point tools/lg-pinned at a stock lg built from the let-go
# version audio-host/go.mod pins, building it first if needed.
#
# The tests, sim and stock web build run tools/lg-pinned, and the Pages build
# compiles against audio-host/go.mod. This keeps them on the same let-go: go.mod
# is the only place the version is written. Each version builds once, into
# tools/.lg/lg-<version> (gitignored); a bump in go.mod builds the new one. A
# `replace` of let-go in go.mod (a local checkout, say) is rebuilt every run,
# since its source can change under the same version; go's build cache keeps
# that quick.
#
# lg reads its own version from the Go build info (pkg/cli/wasm.go), so lg -w
# fetches the same let-go source for the wasm it compiles.
#
# To run a different let-go, set LG=/path/to/lg instead; the scripts that use
# tools/lg-pinned take LG first.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
link="$here/tools/lg-pinned"

# Without a Go that can read go.mod, a checkout that already has an lg can
# still run it; only a missing one is an error.
if ! ver="$(go list -C "$here/audio-host" -m \
    -f '{{with .Replace}}replaced{{else}}{{.Version}}{{end}}' github.com/nooga/let-go 2>&1)" || [[ -z $ver ]]; then
  if [[ -x $link ]]; then
    echo "build-lg: can't read let-go's version from audio-host/go.mod; using the existing tools/lg-pinned ($ver)" >&2
    exit 0
  fi
  echo "build-lg: can't read let-go's version from audio-host/go.mod: $ver" >&2; exit 1
fi

bin="$here/tools/.lg/lg-$ver"
if [[ ! -x $bin || $ver == replaced ]]; then
  echo "build-lg: building lg at let-go $ver ..." >&2
  mkdir -p "$here/tools/.lg"
  # A temp name of its own, so two builds at once can't move a half-written
  # binary into place. Built in audio-host's module, so go.mod and go.sum fix
  # the version and its dependencies. -s -w as let-go's own Makefile builds it.
  tmp="$(mktemp "$bin.XXXXXX")"
  trap 'rm -f "$tmp"' EXIT
  go build -C "$here/audio-host" -ldflags="-s -w" -o "$tmp" github.com/nooga/let-go
  chmod +x "$tmp"
  mv "$tmp" "$bin"
fi
# Replaces a hand-made symlink from before this script, too.
ln -sfn ".lg/lg-$ver" "$link"
