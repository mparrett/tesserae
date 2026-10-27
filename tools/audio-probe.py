"""Audio health of a browser build: is the worklet's ring kept full, and does
the page stay responsive while it plays?

Wraps AudioContext and AudioWorkletNode before the page loads, so it works for
any build whose worklet takes a shared ring as processorOptions {cap, ctrl}
(undertone's webaudio, on the page or fed from the game's worker). After an
optional click it samples once a second: context state, ms of audio queued,
underrun quanta, the page bridge's stats if present, and how long a trivial
evaluate took (a stalled main thread shows up there first).

  uv run --with=playwright python tools/audio-probe.py URL [SECS] [--autoplay] [--no-click]

--autoplay launches Chrome with the autoplay policy off, so audio may start
without a gesture; by default the probe clicks the page after load, which is
the gesture undertone waits for. Headless Chrome renders audio but plays
nothing, so this measures the pipeline, not what it sounds like.
"""
import json
import sys
import time

from playwright.sync_api import sync_playwright

args = [a for a in sys.argv[1:] if not a.startswith("--")]
flags = {a for a in sys.argv[1:] if a.startswith("--")}
URL = args[0]
SECS = int(args[1]) if len(args) > 1 else 15

INIT = """
(() => {
  window.__probe = { acs: [], nodes: [] };
  const AC = window.AudioContext;
  window.AudioContext = function (o) { const c = new AC(o); window.__probe.acs.push(c); return c; };
  window.AudioContext.prototype = AC.prototype;
  const N = window.AudioWorkletNode;
  window.AudioWorkletNode = function (ctx, name, o) {
    const n = new N(ctx, name, o);
    const po = (o && o.processorOptions) || {};
    if (po.ctrl) window.__probe.nodes.push({ name, cap: po.cap, ctrl: new Int32Array(po.ctrl) });
    return n;
  };
})();
"""

SAMPLE = """
() => {
  const p = window.__probe, n = p.nodes[p.nodes.length - 1], ac = p.acs[p.acs.length - 1];
  const out = { state: ac ? ac.state : null };
  if (window.undertoneBridge) out.bridge = undertoneBridge.stats();
  if (n) {
    const w = Atomics.load(n.ctrl, 0), r = Atomics.load(n.ctrl, 1);
    out.queued_ms = Math.round(((w - r + n.cap) % n.cap) / ac.sampleRate * 1000);
    out.under = Atomics.load(n.ctrl, 2);
  }
  return out;
}
"""

with sync_playwright() as pw:
    launch = ["--autoplay-policy=no-user-gesture-required"] if "--autoplay" in flags else []
    b = pw.chromium.launch(headless=True, args=launch)
    page = b.new_page(viewport={"width": 1280, "height": 860})
    page.set_default_timeout(5000)
    page.on("pageerror", lambda e: print("pageerror:", str(e)[:200], flush=True))
    page.add_init_script(INIT)
    page.goto(URL, wait_until="commit")
    page.wait_for_timeout(5000)  # the COI service worker may reload once
    if "--no-click" not in flags:
        page.mouse.click(640, 430)
    try:
        for i in range(SECS):
            page.wait_for_timeout(1000)
            t = time.time()
            s = page.evaluate(SAMPLE)
            print(i + 1, f"eval {round((time.time() - t) * 1000)}ms", json.dumps(s), flush=True)
    except Exception as e:  # a stalled page times out the evaluate
        print("stalled:", type(e).__name__, str(e)[:120], flush=True)
    b.close()
