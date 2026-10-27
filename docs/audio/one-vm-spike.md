# One-VM browser audio: spike notes

The shipped spike (`tools/build-web-audio.sh`) uses two VMs: the game in a
worker sends EDN commands, and a second wasm on the page plays them
([two-vm-bridge.html](two-vm-bridge.html)). This note records how the one-VM
version was built and what it measured, so it can be rebuilt once its pieces
are upstream.

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
