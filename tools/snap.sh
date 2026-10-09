#!/usr/bin/env bash
# snap.sh <tmux-target> <out-prefix> [frames] [interval-s]
# Capture one PNG (frames=1) or an animated GIF from a running tmux pane.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
t="$1"; out="$2"; n="${3:-1}"; iv="${4:-0.1}"
tmp="$(mktemp -d)"
for i in $(seq -w 1 "$n"); do
  tmux capture-pane -t "$t" -e -p -N > "$tmp/f$i.ans"
  [ "$n" -gt 1 ] && sleep "$iv"
done
if [ "$n" -eq 1 ]; then
  uv run -q --with pillow "$here/ansi_render.py" png "$out.png" "$tmp"/f*.ans
  cp "$tmp"/f*.ans "$out.ans"
else
  ms=$(python3 -c "print(int($iv*1000))")
  uv run -q --with pillow "$here/ansi_render.py" gif "$out.gif" "$tmp"/f*.ans --ms "$ms"
fi
rm -rf "$tmp"
