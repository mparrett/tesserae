package main

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/nooga/let-go/pkg/vm"
	"golang.org/x/image/font/gofont/gomono"
)

// Text in the window, drawn as type rather than terminal cells:
//   (pix/text! cx cy s rgb)  arena text from gfx/text! (banners, floats,
//               "SPACE · launch", the tally), centred where its cells were
//   (pix/panel! row segs)  a panel row, segs [[text rgb] ...], beside the arena
//   (pix/card! title accent lines prompt-rgb)  a level card over the arena
//               until (pix/card! nil nil nil nil)
//
// The font is Iosevka when it can be found ($PIX_FONT, else the user's
// fonts), else Go Mono, which ships in x/image; monospace either way, so runs
// padded to a width (the tally's columns) stay aligned.

const (
	textSize  = 18.0
	panelPx   = 300 // window width kept for the panel, right of the arena
	panelRows = 40
)

type textRun struct {
	x, y, n int
	s       string
	c       int64
}

type seg struct {
	s string
	c int64
}

type cardText struct {
	title, prompt string
	accent        int64
	promptC       int64
	lines         []string
}

var (
	face     text.Face
	runs     []textRun // under mu, with frame
	panel    [panelRows][]seg
	card     *cardText
	textMu   sync.Mutex
	runsNext []textRun
)

func loadFace() text.Face {
	var ttf []byte
	for _, p := range []string{os.Getenv("PIX_FONT"),
		filepath.Join(os.Getenv("HOME"), "Library/Fonts/iosevka-medium.ttf")} {
		if p == "" {
			continue
		}
		if b, err := os.ReadFile(p); err == nil {
			ttf = b
			break
		}
	}
	if ttf == nil {
		ttf = gomono.TTF
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
	if err != nil {
		fmt.Println("pix: font:", err)
		src, _ = text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	}
	return &text.GoTextFace{Source: src, Size: textSize}
}

func rgb(c int64) color.Color {
	return color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255}
}

// addText is (pix/text! cx cy s rgb): gfx/text!'s string, kept whole so
// texts in neighbouring cells don't run together. Shown from the next
// present-arena! until the one after.
func addText(a []vm.Value) (vm.Value, error) {
	s := str(a[2])
	n := len([]rune(s))
	runsNext = append(runsNext, textRun{toInt(a[0]), toInt(a[1]), n, s, int64(toInt(a[3]))})
	return vm.NIL, nil
}

// flushText makes this frame's texts the ones Draw shows.
func flushText() {
	textMu.Lock()
	runs = append(runs[:0], runsNext...)
	textMu.Unlock()
	runsNext = runsNext[:0]
}

func drawText(dst *ebiten.Image, s string, x, y float64, c color.Color, align text.Align) {
	op := &text.DrawOptions{}
	op.LayoutOptions.PrimaryAlign = align
	op.LayoutOptions.SecondaryAlign = text.AlignCenter
	// a one-pixel shadow keeps type legible over bright tiles and light
	op.GeoM.Translate(x+1, y+1)
	op.ColorScale.ScaleWithColor(color.Black)
	text.Draw(dst, s, face, op)
	op.GeoM.Translate(-1, -1)
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

// drawOverlay draws arena text, the panel and any card. ox, oy, k place the
// arena (iw px wide) on screen; the panel starts at px.
func drawOverlay(screen *ebiten.Image, ox, oy, k, iw, ih, px int) {
	if face == nil {
		face = loadFace()
	}
	textMu.Lock()
	defer textMu.Unlock()
	lineH := textSize * 1.25
	if card != nil {
		vector.DrawFilledRect(screen, float32(ox), float32(oy), float32(iw*k), float32(ih*k),
			color.RGBA{5, 5, 10, 235}, false)
		x, y := float64(ox)+float64(iw*k)*0.08, float64(oy)+float64(ih*k)*0.18
		drawText(screen, card.title, x, y, rgb(card.accent), text.AlignStart)
		vector.StrokeLine(screen, float32(x), float32(y+lineH*0.7), float32(x+text.Advance(card.title, face)),
			float32(y+lineH*0.7), 1, rgb(dim(card.accent, 0.4)), false)
		for i, ln := range card.lines {
			drawText(screen, ln, x, y+lineH*float64(2+i), rgb(0xd8dcea), text.AlignStart)
		}
		drawText(screen, card.prompt, x, y+lineH*float64(3+len(card.lines)), rgb(card.promptC), text.AlignStart)
	} else {
		// Texts that would overlap (floats spawned together) stack a line
		// apart, where the terminal would let the later one overwrite cells.
		var placed [][4]float64
		for _, r := range runs {
			cx := float64(ox) + (float64(r.x)+float64(r.n)/2)*float64(k)
			cy := float64(oy) + float64(2*r.y+1)*float64(k)
			hw := text.Advance(r.s, face) / 2
			for clash := true; clash; {
				clash = false
				for _, p := range placed {
					if cx-hw < p[1] && cx+hw > p[0] && cy-lineH/2 < p[3] && cy+lineH/2 > p[2] {
						cy, clash = p[2]-lineH/2, true
					}
				}
			}
			placed = append(placed, [4]float64{cx - hw, cx + hw, cy - lineH/2, cy + lineH/2})
			drawText(screen, r.s, cx, cy, rgb(r.c), text.AlignCenter)
		}
	}
	if px < 0 {
		return
	}
	for i, row := range panel {
		x := float64(px)
		y := float64(oy) + lineH*(float64(i)+0.5)
		for _, sg := range row {
			drawText(screen, sg.s, x, y, rgb(sg.c), text.AlignStart)
			x += text.Advance(sg.s, face)
		}
	}
}

func dim(c int64, f float64) int64 {
	ch := func(s uint) int64 { return int64(float64((c>>s)&255)*f) << s }
	return ch(16) | ch(8) | ch(0)
}

func str(v vm.Value) string {
	if v == nil || v == vm.NIL {
		return ""
	}
	if s, ok := v.Unbox().(string); ok {
		return s
	}
	return fmt.Sprint(v.Unbox())
}

// seqOf is a vector's (or list's) items.
func seqOf(v vm.Value) []vm.Value {
	sq, ok := v.(vm.Sequable)
	if !ok {
		return nil
	}
	var out []vm.Value
	for s := sq.Seq(); s != nil; s = s.Next() {
		out = append(out, s.First())
	}
	return out
}

func setPanel(a []vm.Value) (vm.Value, error) {
	row := toInt(a[0])
	if row < 0 || row >= panelRows {
		return vm.NIL, nil
	}
	var segs []seg
	for _, sv := range seqOf(a[1]) {
		p := seqOf(sv)
		if len(p) == 2 {
			segs = append(segs, seg{str(p[0]), int64(toInt(p[1]))})
		}
	}
	textMu.Lock()
	panel[row] = segs
	textMu.Unlock()
	return vm.NIL, nil
}

func setCard(a []vm.Value) (vm.Value, error) {
	var c *cardText
	if a[0] != vm.NIL {
		c = &cardText{title: str(a[0]), accent: int64(toInt(a[1])),
			prompt: "SPACE to continue", promptC: int64(toInt(a[3]))}
		for _, l := range seqOf(a[2]) {
			c.lines = append(c.lines, str(l))
		}
	}
	textMu.Lock()
	card = c
	textMu.Unlock()
	return vm.NIL, nil
}
