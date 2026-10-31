# One-VM browser audio: spike notes

Shipped on 2026-10-31: `tools/build-web-onevm.sh` is the Pages build, and the
two-VM bridge is gone. What follows is how it got there.

Until then the Pages build used two VMs (the since-removed
`tools/build-web-audio.sh`): the game in a worker sent EDN commands, and a
second wasm on the page played them ([two-vm-bridge.html](two-vm-bridge.html)).
This note records how the one-VM version was built and what it measured.
How the bridge came to ship first is in the DEVLOG's 2026-10-31 entry.

## Shape

The game's own wasm links undertone. Undertone renders in the worker into a
`SharedArrayBuffer` ring and posts it to the page as `undertone:open`; a small
`page.js` on the main thread owns the `AudioContext` and the worklet that
reads the ring. No second runtime, no EDN hop.

## Pieces and where they stand (as of 2026-10-17)

- **let-go `-import`** (#1037, merged): `lg -w -w-module DIR -import PKG` links
  a Go package into the bundle.
- **let-go `RegisterEmbeddedSource`** (#1064, merged as 8673d88a): undertone
  can ship `ambience` inside the Go package, so it resolves with no source path.
- **undertone registers `ambience`** as `undertone.ambience` (undertone#29, in
  review). Tesserae's `sfx.lg` must then require `undertone.ambience`.
- **undertone worker mode** (not upstream): `webaudio/worker.go` and
  `webaudio/page.js` in the `let-go-undertone-register-ambience` worktree,
  branch `spike/register-ambience`. `Open` detects a worker and posts the ring
  instead of opening a context. `page.js` wraps `window.Worker` to hear those
  messages, so it must load before let-go's runtime script. Needs the module
  rename and the handle caching from undertone#13 before it can be a PR.

Since then (2026-10-20): undertone#29 and worker mode (undertone#45) are on
undertone main, and undertone#57 renamed its natives from `audio/` to
`undertone/` (`undertone/trigger`, `undertone/set-parameters`, and so on), leaving
`audio` for let-go's seam. `sfx.lg` and `audio-host` follow undertone `7a58a1a`.

## Build recipe used

```sh
# H: a Go module whose main is audio-host's (cli.Main + undertone/letgo),
#    with go.mod pointing at let-go >= 8673d88a and the worker-mode undertone.
sed 's|<head>|<head>\n  <script src="undertone-page.js"></script>|' web/shell.html > /tmp/shell.html
"$H/lg" -w OUT -w-shell /tmp/shell.html -w-module "$H" \
  -import github.com/mparrett/undertone/letgo \
  -storage-id tesserae-onevm -source-paths . main.lg
cp "$UNDERTONE/webaudio/page.js" OUT/undertone-page.js
```

Serve OUT with `tools/coi-serve.py` (the ring needs cross-origin isolation).

## Measured (2026-10-15, headless Chromium, `tools/audio-probe.py`)

- During autoplay the queue held 264–311 ms of its 300 ms target with no new
  underruns, and the stats overlay showed the 50 fps cap with sound and with
  `&mute`.
- At startup, 148–189 underrun quanta (about half a second) before the ring
  filled. Cause not investigated.
- A `BroadcastChannel` can't carry the ring: Chrome delivers a
  `SharedArrayBuffer` sent that way as `null`. Worker `postMessage` shares it.
- Not yet heard by ear, nor tried in Safari or on iOS.

## Rebuilt on let-go 18ce3194 and undertone 3fd695a (2026-10-31)

`tools/build-web-onevm.sh` is the recipe above as a script (TinyGo by
default, 2.7 MB). Undertone's browser queue is now hardcoded at 60 ms
(`letgo/output_js.go`, undertone#79), tuned for the bridge on the page's
main thread. Here the producer shares the worker with the game loop. The
other queue sizes were built from a scratch copy of undertone with only that
constant changed.

Headless Chromium, `audio-probe.py` on `?level=3&skip-card&autoplay`, load
average 4–5:

| Queue | Run | Underrun quanta | Lowest queued |
| --- | --- | --- | --- |
| 60 ms | 40 s | 103, rising steadily | 0 ms |
| 120 ms | 40 s / 90 s | 0 / 3 | 64 ms (90 s) |
| 160 ms | 90 s, twice | 5280 / 0 | 0 / 96 ms |
| 300 ms | 40 s | 0 | 258 ms |

The startup underruns from 10-15 didn't recur at any size. The bad 160 ms
run lined up with the machine's load climbing to about 9, and a main-thread
evaluate took 484 ms. But 5280 quanta is about 15 s of silence, far longer
than a GC pause, and the cause is unknown.

By ear on an iPhone 13 Pro (Safari, over tailnet HTTPS, 2026-10-31), relayed
through the undertone session: no crackles at 60, 120 or 160 ms. The wave of
white noise heard in all three is undertone's breeze patch, by design. The
pages were slow to load on the phone; not yet diagnosed. Hit latency at each
size wasn't reported, and desktop Safari is untried.

So undertone main's 60 ms held up on an idle phone but not under load in
headless Chromium, where 120–160 ms did. 120 ms became undertone's default in
a worker (undertone#80), with `?queue=MS` to override it per device.

## TinyGo, for comparison

As of 2026-10-15 TinyGo couldn't build this path: let-go's `-w-module`
refused `LETGO_USE_TINYGO`. It did build the two-VM bridge (1.8 MB gzipped against
6.5 MB for Go, 2026-10-15), which plays after undertone#13 but still showed GC
pauses of 200–300 ms in a 20 s run, against under 100 ms for Go.

With nooga/let-go#1077 (TinyGo under `-w-module`), #1079/#1080 (storage in
the worker) and undertone#45 (worker mode), TinyGo builds the one-VM path at
2.6 MB. A first phone ear test on 2026-10-18, one short listen per build,
compared three builds:

| Build | Result |
| --- | --- |
| Go one-VM | hitches |
| TinyGo two-VM bridge | hitches |
| TinyGo one-VM | no hitches |

That is a single listen, not a measurement. The headless underrun probe from
2026-10-17 ran under a load average of about 17, so it still needs a rerun on
a quiet machine.
