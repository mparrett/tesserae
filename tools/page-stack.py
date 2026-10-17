"""Where is a stalled page spending its main thread? Loads URL, clicks after
--click-at seconds (0 for no click), waits, then pauses the page through the
DevTools protocol a few times and prints the top of each JS/wasm stack.

  uv run --with=playwright python tools/page-stack.py URL [--click-at 5] [--wait 4] [--samples 3]

A pause interrupts even a busy loop, so this works when page.evaluate no
longer returns. Wasm frames are named from the module's name section, which
a TinyGo -no-debug build still kept. This is how the TinyGo bridge stall was
traced to finalizer-driven GC (2026-10-15).
"""
import argparse

from playwright.sync_api import sync_playwright

ap = argparse.ArgumentParser()
ap.add_argument("url")
ap.add_argument("--click-at", type=float, default=5)
ap.add_argument("--wait", type=float, default=4)
ap.add_argument("--samples", type=int, default=3)
ap.add_argument("--frames", type=int, default=14)
a = ap.parse_args()

with sync_playwright() as pw:
    b = pw.chromium.launch(headless=True)
    page = b.new_page()
    cdp = page.context.new_cdp_session(page)
    paused = []
    cdp.on("Debugger.paused", lambda e: paused.append(e))
    cdp.send("Debugger.enable")
    page.goto(a.url, wait_until="commit")
    if a.click_at:
        page.wait_for_timeout(a.click_at * 1000)
        page.mouse.click(300, 300)
    page.wait_for_timeout(a.wait * 1000)
    for k in range(a.samples):
        cdp.send("Debugger.pause")
        # sync Playwright only delivers CDP events inside its own calls
        for _ in range(100):
            if paused:
                break
            page.wait_for_timeout(50)
        if not paused:
            print("no pause event")
            break
        e = paused.pop()
        print(f"--- sample {k}: {e['reason']}")
        for f in e["callFrames"][: a.frames]:
            print("  ", f["functionName"] or "(anon)", f["url"].split("/")[-1][:40], f["location"]["lineNumber"])
        cdp.send("Debugger.resume")
        page.wait_for_timeout(400)
    b.close()
