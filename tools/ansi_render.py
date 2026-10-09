#!/usr/bin/env python3
"""Render `tmux capture-pane -e -p` dumps to PNG / animated GIF / HTML.

    uv run --with pillow tools/ansi_render.py png  out.png  frame.ans
    uv run --with pillow tools/ansi_render.py gif  out.gif  f1.ans f2.ans ... [--ms 80]
    uv run --with pillow tools/ansi_render.py html out.html frame.ans [--title T]

Half blocks, quadrants (U+2596-259F) and sextants (U+1FB00-1FB3B) are drawn
as exact filled sub-rectangles, so the PNG shows the game's square-pixel
raster and sub-cell sprites the way a good terminal font would.
"""
import html
import re
import sys

CW, CH = 8, 16
DEFAULT_FG = (200, 200, 200)
DEFAULT_BG = (0, 0, 0)
SGR = re.compile(r"\x1b\[([0-9;:]*)m")


def parse(text):
    """-> list of rows, each a list of (char, fg, bg)."""
    rows = []
    fg, bg = DEFAULT_FG, DEFAULT_BG
    for line in text.split("\n"):
        row = []
        i = 0
        while i < len(line):
            m = SGR.match(line, i)
            if m:
                params = [int(p) if p else 0 for p in re.split("[;:]", m.group(1) or "0")]
                j = 0
                while j < len(params):
                    p = params[j]
                    if p == 0:
                        fg, bg = DEFAULT_FG, DEFAULT_BG
                    elif p == 39:
                        fg = DEFAULT_FG
                    elif p == 49:
                        bg = DEFAULT_BG
                    elif p in (38, 48) and j + 1 < len(params) and params[j + 1] == 2:
                        c = tuple(params[j + 2 : j + 5])
                        if p == 38:
                            fg = c
                        else:
                            bg = c
                        j += 4
                    elif p in (38, 48) and j + 1 < len(params) and params[j + 1] == 5:
                        j += 2
                    j += 1
                i = m.end()
                continue
            if line[i] == "\x1b":  # other escape: skip to its final byte
                k = i + 1
                while k < len(line) and not ("@" <= line[k] <= "~" and line[k] != "["):
                    k += 1
                i = k + 1
                continue
            row.append((line[i], fg, bg))
            i += 1
        rows.append(row)
    while rows and not rows[-1]:
        rows.pop()
    return rows


# Sub-cell block glyphs -> (columns, rows, mask). Bit 2*row+col, as in gfx.lg.
QUAD = "\u0020\u2598\u259d\u2580\u2596\u258c\u259e\u259b\u2597\u259a\u2590\u259c\u2584\u2599\u259f\u2588"
BLOCKS = {ch: (2, 2, m) for m, ch in enumerate(QUAD) if ch != " "}
for _m in range(1, 63):
    if _m in (21, 42):
        continue
    BLOCKS[chr(0x1FB00 + _m - 1 - (_m > 21) - (_m > 42))] = (2, 3, _m)


def draw_block(d, x0, y0, spec, fg):
    """Fill the set subcells exactly (sextant rows split 16px as 5/6/5)."""
    nc, nr, m = spec
    xs = [x0 + CW * i // nc for i in range(nc + 1)]
    ys = [y0 + round(CH * j / nr) for j in range(nr + 1)]
    for j in range(nr):
        for i in range(nc):
            if m >> (2 * j + i) & 1:
                d.rectangle([xs[i], ys[j], xs[i + 1] - 1, ys[j + 1] - 1], fill=fg)


def to_image(rows, font):
    from PIL import Image, ImageDraw

    w = max((len(r) for r in rows), default=1)
    img = Image.new("RGB", (w * CW, len(rows) * CH), DEFAULT_BG)
    d = ImageDraw.Draw(img)
    for y, row in enumerate(rows):
        for x, (ch, fg, bg) in enumerate(row):
            x0, y0 = x * CW, y * CH
            d.rectangle([x0, y0, x0 + CW - 1, y0 + CH - 1], fill=bg)
            spec = BLOCKS.get(ch)
            if spec:
                draw_block(d, x0, y0, spec, fg)
            elif ch != " ":
                d.text((x0, y0 + 1), ch, fill=fg, font=font)
    return img


def load_font():
    from PIL import ImageFont

    for p in ("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",):
        try:
            return ImageFont.truetype(p, 13)
        except OSError:
            pass
    return ImageFont.load_default()


def to_html(rows, title):
    out = [
        "<!doctype html><html><head><meta charset='utf-8'>",
        f"<title>{html.escape(title)}</title>",
        "<style>body{background:#000;margin:0;padding:16px}"
        "pre{font:13px/1.0 'DejaVu Sans Mono',Menlo,monospace;margin:0;letter-spacing:0}"
        "span{display:inline-block;width:1ch;height:1em}</style></head><body><pre>",
    ]
    for row in rows:
        line = []
        for ch, fg, bg in row:
            line.append(
                f"<span style='color:rgb{fg};background:rgb{bg}'>{html.escape(ch)}</span>"
            )
        out.append("".join(line))
    out.append("</pre></body></html>")
    return "\n".join(out)


def main(argv):
    mode, out, *rest = argv
    ms = 80
    title = "capture"
    files = []
    it = iter(rest)
    for a in it:
        if a == "--ms":
            ms = int(next(it))
        elif a == "--title":
            title = next(it)
        else:
            files.append(a)
    frames = [parse(open(f, encoding="utf-8", errors="replace").read()) for f in files]
    if mode == "html":
        open(out, "w").write(to_html(frames[0], title))
        return
    font = load_font()
    imgs = [to_image(r, font) for r in frames]
    if mode == "png":
        imgs[0].save(out)
    elif mode == "gif":
        imgs[0].save(out, save_all=True, append_images=imgs[1:], duration=ms, loop=0, optimize=True)


if __name__ == "__main__":
    main(sys.argv[1:])
