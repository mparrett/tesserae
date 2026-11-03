"""Frame pacing of a browser build: how many frames the game produces, and how
many the page gets to paint.

Hooks the page's xterm write. A frame is a write carrying the DEC 2026 sync
begin (as in tools/web-shot.mjs). Frames that arrive within 4 ms of each other
land in one paint, so only the last of them shows: "painted" counts those
clumps, "produced" counts every frame. Produced near the 50 fps cap with
painted far below it means the page's main thread (xterm parsing, the WebGL
renderer) is the bottleneck, not the game in its worker. ?stats on the page
shows only the worker's side.

  uv run --with=playwright python tools/frame-probe.py URL [--secs 30] [--headed] [--channel chrome]

URL is a served build, for example
http://127.0.0.1:8301/?level=1&skip-card&autoplay&grid=large&mute
(autoplay keeps runs repeatable; grid=large makes a shortfall obvious).
Headless Chromium paints WebGL in software, which overstates the page's cost;
--headed --channel chrome runs installed Chrome on the GPU. As of 2026-11-02,
level 1 autoplay at grid=large painted about 8 of 46 fps headless and all 46
in headed Chrome, so read a headless shortfall as a hint, not a result. Load swings move
these numbers more than most builds differ, so check `uptime` and alternate
the builds you compare.
"""
import argparse
import json

from playwright.sync_api import sync_playwright

ap = argparse.ArgumentParser()
ap.add_argument("url")
ap.add_argument("--secs", type=int, default=30)
ap.add_argument("--headed", action="store_true")
ap.add_argument("--channel", help="installed browser to drive, such as chrome")
a = ap.parse_args()

HOOK = """() => {
  const t = window._tessTerm, w = t.write.bind(t);
  window._ts = [];
  t.write = (s, cb) => { if (typeof s === 'string' && s.includes('\\x1b[?2026h')) window._ts.push(performance.now()); return w(s, cb); };
}"""

with sync_playwright() as pw:
    b = pw.chromium.launch(headless=not a.headed, channel=a.channel,
                           args=["--autoplay-policy=no-user-gesture-required"])
    page = b.new_page(viewport={"width": 1400, "height": 900})
    page.goto(a.url, wait_until="commit")
    page.wait_for_function("() => window._tessTerm", timeout=60000)
    page.wait_for_timeout(4000)  # the COI service worker may reload once
    page.wait_for_function("() => window._tessTerm", timeout=60000)
    page.mouse.click(700, 450)  # the gesture undertone waits for
    page.wait_for_timeout(2000)
    page.evaluate(HOOK)
    page.wait_for_timeout(a.secs * 1000)
    ts = page.evaluate("() => window._ts")
    grid = page.evaluate("() => [window._tessTerm.cols, window._tessTerm.rows, window._tessTerm.options.fontSize]")
    b.close()

gaps = sorted(y - x for x, y in zip(ts, ts[1:]))
n = len(gaps)
pct = lambda p: round(gaps[min(n - 1, int(p * n))], 1) if n else None
print(json.dumps({
    "url": a.url,
    "grid": grid,
    "produced_fps": round(len(ts) / a.secs, 1),
    "painted_fps": round((1 + sum(g > 4 for g in gaps)) / a.secs, 1) if ts else 0.0,
    "gap_ms": {"p50": pct(0.5), "p95": pct(0.95), "p99": pct(0.99), "max": round(gaps[-1], 1) if n else None},
    "gaps_over_50ms": sum(g > 50 for g in gaps),
    "gaps_over_100ms": sum(g > 100 for g in gaps),
}))
