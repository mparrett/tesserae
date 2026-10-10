# TESSERAE — phases

Started 2026-10-09 on let-go upstream `a13e042`.

## v0: a playable slice across the whole brief (done)

Goal: every headline idea is playable in at least one level, inside a story frame, on a
renderer good enough to show off the rotation.

- [x] Square-pixel raster engine: light accumulation, overlay, text layer, rotation-aware
      sampling, posterize-before-diff, SGR-tracked diff emitter, synchronized frames
- [x] World-pixel colour cache, snapped rest angles (full redraws ~2× cheaper)
- [x] Input: kitty press/repeat/release, repeat-hold fallback, SGR any-motion mouse
- [x] Tetromino-packed brick walls (seeded), piece-joined bevels, SET bonus per piece
- [x] Multiple balls, mitosis tiles, hard tiles, combo multiplier
- [x] 90° field rotation during play (warning, eased 1.2 s turn, open edge follows the turn)
- [x] Twin trowels (two open edges)
- [x] Reverse mode: ghost blueprint tiles set on hit; Grout Creep decay variant
- [x] Tilt labyrinth: no paddle, gravity from tilt, pits, lamps, multiple marbles
- [x] Gravity, lodestone tiles, magnet trowel (catch / release), rubber and lead balls
- [x] The Heart: ball starts walled in, no paddle, a borrowed tile the game picks (TAB to choose another)
- [x] Story: prologue, chapter cards (typewriter), epilogue; title attract mode
- [x] Capture tooling: tmux → PNG/GIF, level runner, autopilot
- [x] AOT-lowered `lg` for the raster hot paths (optional binary, `tools/build-aot.sh`)

## v1: depth and feel (done)

- [x] **Yars-style boss**: 6.2 The Grout Core, with a scrolling, regenerating shield, a
      neutral zone, a homing Destroyer and the Swirl. No field rotation (deferred: the band
      would need a screen-frame overlay).
- [x] **Shape-aware sprite glyphs**: `--glyphs quadrant|sextant|off`; the coverage bitmask is
      the glyph index.
- [x] **Falling mortar piece with line clears**: 3.3 Mortar Well. Blueprint pieces snap whole
      when struck.
- [x] **Ball materials**: glass (shatters into three) and ghost (passes three tiles). Sticky
      is deferred; the magnet trowel already covers catching.
- [x] **Saved progress**: best scores, unlocks and per-level records via let-go `storage`.
- [x] **Difficulty pass**: par times and gold/silver/bronze medals, a flawless bonus, sim
      sweeps (`tools/sim.lg`), "singing" last tiles, +20% spark speed.
- [ ] Sound: deferred. A terminal has no good channel, and the browser build would need the
      xsofy audio module.

## v2: platforms and reach (done, with follow-ups)

- [x] **Browser build** (`lg -w`, xterm.js): `tools/serve-web.sh`. Follow-ups: saving
      progress (let-go's storage needs a bridge into the worker), and frame rate (the
      Go-wasm VM is 3–5× slower).
- [x] **Level editor**: `--edit [file]`, EDN custom maps, `--map file`, in-process play-test.
      Follow-ups: in-editor naming, boss and Orrery parts in the palette, mouse painting.
- [x] **Orrery**: bonus chapter VII · THE ORRERY (7.1 Escapement, 7.2 Gearwork, 7.3 Counting
      House), opened by clearing 6.2. Follow-ups: more parts (an interrupt, a trigger), a
      puzzle that needs both lanes and a gear, editor support.

## Next (v3 candidates)

- Rotation on the boss level (a screen-frame neutral-zone overlay).
- A mortar fall that speeds up with progress, and a landing shadow for the piece.
- An adaptive raster size for the browser, and IndexedDB storage once let-go supports it.
- Re-lowering `compose!`: unblocked now that let-go#1051 has merged. It needs `wt/tesserae-aot`
  rebased onto let-go tip, dropping the half of `a8c24ea` that #1044 covers (the earlier full AOT build
  measured 71 → 32 ms on a rotated frame).

## Performance notes

- VM cost is about 0.3 µs per simple op. Steady frames: 1–4 ms compose for 60–150 dirty
  cells. Full-field compose, rotated: 107 → 71 → 54 ms (VM), 36 ms coarse, ~20 ms coarse
  on the AOT binary. The DEVLOG has the full arc.
- Remaining levers: emitting runs of equal-bg cells as `ECH`, and rebuilding the AOT binary
  after gfx changes.
- let-go gaps found: see `docs/letgo-upstream-candidates.md`.
