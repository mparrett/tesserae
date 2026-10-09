#!/usr/bin/env bash
# gif-level.sh <level> <out.gif> [warmup-s] [frames] [interval] [wait-regex]
# Autoplay a level in tmux session $TESS_SESSION, wait, capture an animated GIF
# cropped to arena+panel and scaled to 60%. Runs under the heavy lock.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
lvl="$1"; out="$2"; warm="${3:-6}"; n="${4:-30}"; iv="${5:-0.07}"; re="${6:-}"
s="${TESS_SESSION:-tess}"
"$here/run-level.sh" "$lvl"
sleep "$warm"
[ -n "$re" ] && "$here/wait-for.sh" "$s" "$re" 60
tmp="$(mktemp -d)"
for i in $(seq -w 1 "$n"); do tmux capture-pane -t "$s" -e -p -N > "$tmp/f$i.ans"; sleep "$iv"; done
tmux kill-session -t "$s" 2>/dev/null || true
ms=$(python3 -c "print(int($iv*1000)+40)")
uv run -q --with pillow "$here/ansi_render.py" gif "$tmp/full.gif" "$tmp"/f*.ans --ms "$ms"
uv run -q --with pillow python3 - "$tmp/full.gif" "$out" <<'PY'
import sys
from PIL import Image, ImageSequence
src, dst = sys.argv[1], sys.argv[2]
im = Image.open(src)
frames = []
for f in ImageSequence.Iterator(im):
    f = f.convert("RGB").crop((140, 16, 1130, 816))
    frames.append(f.resize((int(f.width * 0.6), int(f.height * 0.6)), Image.LANCZOS))
frames[0].save(dst, save_all=True, append_images=frames[1:], duration=im.info.get("duration", 80), loop=0, optimize=True)
PY
rm -rf "$tmp"
ls -la "$out"
