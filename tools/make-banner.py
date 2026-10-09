#!/usr/bin/env python3
"""Render the README banner: the TESSERAE wordmark as bevelled tesserae on the
indigo pegboard, set like a blueprint (ghost -> gold flash -> solid), then a
Spark streaks across and rings each letter. Drawn on the game's square-pixel
grid so it reads as terminal pixels.

    uv run --with pillow tools/make-banner.py docs/banner.gif [docs/banner.png]
"""
import math
import sys

from PIL import Image

# --- look (mirrors tesserae/levels.lg + world.lg) ---------------------------
BG0, BG1 = (0x0D, 0x0F, 0x24), (0x1A, 0x12, 0x40)
PIECE = {"I": (0x3F, 0xD0, 0xE0), "O": (0xF2, 0xD1, 0x4B), "T": (0xB0, 0x5C, 0xE0),
         "S": (0x5A, 0xD3, 0x5A), "Z": (0xE8, 0x53, 0x4A), "J": (0x4A, 0x6E, 0xF0),
         "L": (0xF0, 0x9A, 0x3A)}
COLORS = "IOTSZJLI"
GOLD = (0xFF, 0xE0, 0x8A)
SPARK = (0xFF, 0xD2, 0x7A)
FONT = {"T": ["###", ".#.", ".#.", ".#.", ".#."],
        "E": ["###", "#..", "##.", "#..", "###"],
        "S": ["###", "#..", "###", "..#", "###"],
        "R": ["##.", "#.#", "##.", "#.#", "#.#"],
        "A": [".#.", "#.#", "###", "#.#", "#.#"]}
WORD = "TESSERAE"

TP = 4          # pixels per tile (the game's k=2: a tile is 4x4 square pixels)
SCALE = 5       # screen pixels per game pixel
MX, MY = 3, 2   # tile margins
GW = len(WORD) * 4 - 1 + 2 * MX
GH = 5 + 2 * MY


def mix(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def scale(c, f):
    return tuple(max(0, min(255, int(v * f))) for v in c)


def add(c, d):
    return tuple(max(0, min(255, int(c[i] + d[i]))) for i in range(3))


# tiles: (tx, ty) -> letter index
tiles = {}
for n, ch in enumerate(WORD):
    for dy, row in enumerate(FONT[ch]):
        for dx, c in enumerate(row):
            if c == "#":
                tiles[(MX + n * 4 + dx, MY + dy)] = n


def background(px, py):
    tx, ty = px // TP, py // TP
    b = mix(BG0, BG1, ty / max(1, GH - 1))
    if (tx + ty) % 2:
        b = scale(b, 1.12)
    if px % TP == 0 and py % TP == 0:  # pegboard dot
        b = add(b, (13, 13, 18))
    return b


def tile_px(base, u, v, joined):
    """Edge bevel honouring joins to same-letter neighbours (like world/sample)."""
    l, t, r, b = joined
    if (u == 0 and not l) or (v == 0 and not t):
        return mix(base, (255, 255, 255), 0.38)
    if (u == TP - 1 and not r) or (v == TP - 1 and not b):
        return scale(base, 0.55)
    return base


def render(frame_state):
    """frame_state: per-letter state dict -> list of RGB pixels (game grid)."""
    W, H = GW * TP, GH * TP
    img = [[background(x, y) for x in range(W)] for y in range(H)]
    for (tx, ty), n in tiles.items():
        st = frame_state["letters"][n]  # (phase, t): ghost / flash / solid
        base = PIECE[COLORS[n]]
        joined = tuple((tx + dx, ty + dy) in tiles and tiles[(tx + dx, ty + dy)] == n
                       for dx, dy in ((-1, 0), (0, -1), (1, 0), (0, 1)))
        ring = frame_state["ring"].get(n, 0.0)
        for v in range(TP):
            for u in range(TP):
                x, y = tx * TP + u, ty * TP + v
                bg = img[y][x]
                edge = (u == 0 and not joined[0]) or (v == 0 and not joined[1]) or \
                       (u == TP - 1 and not joined[2]) or (v == TP - 1 and not joined[3])
                if st[0] == "ghost":
                    c = mix(bg, base, 0.5 if edge else 0.1)
                else:
                    c = tile_px(base, u, v, joined)
                    if st[0] == "flash":
                        c = mix(c, GOLD, st[1])
                    if ring:
                        c = mix(c, (255, 255, 255), ring * 0.55)
                img[y][x] = c
    # additive light: spark + trail
    for (sx, sy, inten, rad) in frame_state["lights"]:
        for y in range(max(0, int(sy - rad)), min(H, int(sy + rad) + 1)):
            for x in range(max(0, int(sx - rad)), min(W, int(sx + rad) + 1)):
                d2 = (x + 0.5 - sx) ** 2 + (y + 0.5 - sy) ** 2
                core = (rad * 0.28) ** 2
                if d2 < rad * rad:
                    f = inten if d2 < core else inten * core / d2 * (1 - d2 / (rad * rad))
                    img[y][x] = add(img[y][x], tuple(c * f for c in SPARK))
    return img


def to_image(px):
    H, W = len(px), len(px[0])
    im = Image.new("RGB", (W, H))
    im.putdata([c for row in px for c in row])
    return im.resize((W * SCALE, H * SCALE), Image.NEAREST)


def main(out_gif, out_png=None):
    frames, durs = [], []
    n_letters = len(WORD)
    letters = [("ghost", 0.0)] * n_letters

    def snap(ring=None, lights=(), ms=70):
        frames.append(to_image(render({"letters": letters, "ring": ring or {}, "lights": list(lights)})))
        durs.append(ms)

    # 1. blueprint glimmers
    snap(ms=500)
    # 2. letters set one by one: gold flash fading to solid
    for n in range(n_letters):
        for t in (1.0, 0.6, 0.25):
            letters[n] = ("flash", t)
            snap(ms=60)
        letters[n] = ("solid", 0.0)
    snap(ms=500)
    # 3. a Spark streaks across on a shallow bounce, ringing letters as it passes
    W, H = GW * TP, GH * TP
    trail = []
    rings = {}
    steps = 34
    for i in range(steps + 1):
        f = i / steps
        x = -6 + f * (W + 12)
        y = H * 0.5 + math.sin(f * math.pi * 3) * H * 0.33
        trail.append((x, y))
        trail = trail[-6:]
        n_hit = int((x / TP - MX) // 4)
        if 0 <= n_hit < n_letters and abs((x / TP - MX) % 4 - 1.5) < 1.6:
            rings[n_hit] = 1.0
        lights = [(tx, ty, 0.12 * (k + 1) / len(trail), 3.0) for k, (tx, ty) in enumerate(trail[:-1])]
        lights.append((x, y, 1.0, 6.5))
        snap(ring=dict(rings), lights=lights, ms=45)
        rings = {k: v * 0.72 for k, v in rings.items() if v > 0.08}
    while rings:
        snap(ring=dict(rings), ms=45)
        rings = {k: v * 0.72 for k, v in rings.items() if v > 0.08}
    # 4. hold
    snap(ms=2600)

    frames[0].save(out_gif, save_all=True, append_images=frames[1:], duration=durs,
                   loop=0, optimize=True, disposal=1)
    if out_png:
        to_image(render({"letters": [("solid", 0.0)] * n_letters, "ring": {}, "lights": []})).save(out_png)
    print(out_gif, frames[0].size, len(frames), "frames")


if __name__ == "__main__":
    main(*sys.argv[1:])
