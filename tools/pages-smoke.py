# pages-smoke.py <site-dir> — boot the Pages build the way GitHub Pages serves
# it: no COOP/COEP headers, under a project subpath (/tesserae/). Passes when
# the COI service worker has made the page cross-origin isolated, the game is
# drawing frames, and undertone's page.js has built its worklet node from the
# ring the game's worker posted, with no page errors.
#   uv run --with=playwright python tools/pages-smoke.py _site
import functools, http.server, os, sys, tempfile, threading, time
from playwright.sync_api import sync_playwright, Error as PWError

site = os.path.abspath(sys.argv[1])
root = tempfile.mkdtemp()
os.symlink(site, os.path.join(root, "tesserae"))


class Quiet(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *a):
        pass


httpd = http.server.ThreadingHTTPServer(("127.0.0.1", 0), functools.partial(Quiet, directory=root))
threading.Thread(target=httpd.serve_forever, daemon=True).start()
url = f"http://127.0.0.1:{httpd.server_port}/tesserae/"


def poll(page, expr, secs):
    # The first visit registers the worker and reloads, which destroys the
    # context an evaluate runs in; retry through that.
    end = time.time() + secs
    while time.time() < end:
        try:
            if page.evaluate(expr):
                return True
        except PWError:
            pass
        page.wait_for_timeout(250)
    return False


fails, errors = [], []
with sync_playwright() as p:
    # PW_CHROMIUM=<binary> uses an existing browser when this playwright's own isn't installed.
    browser = p.chromium.launch(executable_path=os.environ.get("PW_CHROMIUM") or None, args=["--use-angle=swiftshader", "--enable-unsafe-swiftshader"])
    page = browser.new_page(viewport={"width": 1400, "height": 900})
    page.on("pageerror", lambda e: errors.append(str(e).split("\n")[0]))
    page.on("console", lambda m: m.type == "error" and errors.append(m.text[:200]))
    # Count the worklet nodes page.js builds. Without a gesture the context
    # stays suspended, but the node is built as soon as the ring arrives.
    page.add_init_script("""(() => { const N = window.AudioWorkletNode; if (!N) return; window._utNodes = 0;
        window.AudioWorkletNode = function (c, name, o) { if (name === 'undertone-shared') window._utNodes++; return new N(c, name, o); }; })();""")
    page.goto(url, wait_until="domcontentloaded")
    if not poll(page, "self.crossOriginIsolated", 30):
        fails.append("page never became cross-origin isolated")
    if not poll(page, "!!window._tessTerm", 90):
        fails.append("terminal never appeared")
    else:
        page.evaluate("""() => { const t = window._tessTerm, w = t.write.bind(t); window._frames = 0;
            t.write = (s, cb) => { if (typeof s === 'string' && s.includes('\\x1b[?2026h')) window._frames++; return w(s, cb); }; }""")
        if not poll(page, "window._frames > 5", 60):
            fails.append("game drew no frames")
    if not poll(page, "window._utNodes > 0", 60):
        fails.append("undertone's audio ring never reached the page")
    browser.close()
httpd.shutdown()

fails += [f"page error: {e}" for e in dict.fromkeys(errors)]
for f in fails:
    print("FAIL", f)
print("ok" if not fails else f"{len(fails)} failure(s)", url)
sys.exit(1 if fails else 0)
