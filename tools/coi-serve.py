# Adapted from the xsofy workspace's coi-serve.py (static server with COOP/COEP,
# so the page is crossOriginIsolated and lg's SharedArrayBuffer input ring works).
import http.server, sys, os

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8301
DIRECTORY = sys.argv[2] if len(sys.argv) > 2 else "."
# Third arg is the bind address. Localhost by default — a dev server holding a
# game build has no business on the network unless asked. Pass 0.0.0.0 to reach
# it from a phone over LAN/Tailscale.
HOST = sys.argv[3] if len(sys.argv) > 3 else "127.0.0.1"
os.chdir(DIRECTORY)

class COIHandler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        # cross-origin isolation → SharedArrayBuffer available (lg wasm input ring)
        self.send_header("Cross-Origin-Opener-Policy", "same-origin")
        self.send_header("Cross-Origin-Embedder-Policy", "require-corp")
        self.send_header("Cache-Control", "no-store")
        super().end_headers()

# Threaded: a browser parallelizes index.html + a multi-MB main.wasm + the
# service worker. A single-connection server stalls under that and the page dies
# with ERR_EMPTY_RESPONSE — ThreadingHTTPServer serves the concurrent requests.
http.server.ThreadingHTTPServer.allow_reuse_address = True
httpd = http.server.ThreadingHTTPServer((HOST, PORT), COIHandler)
print(f"COI server on http://{HOST}:{PORT}/ serving {DIRECTORY}", flush=True)
httpd.serve_forever()
