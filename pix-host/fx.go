package main

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Window effects, owned by the host and keyed on F-keys the game never sees:
//   F1  bloom strength: off, 0.5, 1.0, 1.6 (0.5 to start)
//   F2  posterize the light (on to start; off shows the raw sum, clamped)
//   F3  CRT: scanlines on arena rows, a slight RGB split, vignette (off)
// PIX_FX="bloom=1,post=0,crt=1" sets them at start, for headless shots.
//
// Bloom works from the frame's light alone (the acc buffers presentArena
// reads), so tiles and the static field don't glow, only what is lit:
// half-size and quarter-size copies, each blurred both ways, added back
// over the arena.

var bloomLevels = []float64{0, 0.5, 1.0, 1.6}

var (
	bloomIx  atomic.Int32
	posterOn atomic.Bool
	crtOn    atomic.Bool

	fxNote   string
	fxNoteAt time.Time

	blurShader, crtShader *ebiten.Shader
)

func init() {
	bloomIx.Store(1)
	posterOn.Store(true)
	for _, kv := range strings.Split(os.Getenv("PIX_FX"), ",") {
		k, v, _ := strings.Cut(kv, "=")
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		switch k {
		case "bloom":
			ix := 0
			for i, l := range bloomLevels {
				if f >= l {
					ix = i
				}
			}
			bloomIx.Store(int32(ix))
		case "post":
			posterOn.Store(f != 0)
		case "crt":
			crtOn.Store(f != 0)
		}
	}
}

const blurSrc = `//kage:unit pixels
package main

var Dir vec2

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src) * 0.227027
	c += (imageSrc0At(src+Dir) + imageSrc0At(src-Dir)) * 0.1945946
	c += (imageSrc0At(src+2*Dir) + imageSrc0At(src-2*Dir)) * 0.1216216
	c += (imageSrc0At(src+3*Dir) + imageSrc0At(src-3*Dir)) * 0.054054
	c += (imageSrc0At(src+4*Dir) + imageSrc0At(src-4*Dir)) * 0.016216
	return c
}
`

const crtSrc = `//kage:unit pixels
package main

var Row float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	o := imageSrc0Origin()
	sz := imageSrc0Size()
	p := src - o
	r := imageSrc0At(src + vec2(0.6, 0)).r
	g := imageSrc0At(src).g
	b := imageSrc0At(src - vec2(0.6, 0)).b
	c := vec3(r, g, b)
	// the last third of each arena pixel row darkens: a scanline per row
	if mod(p.y, Row) > Row*0.66 {
		c *= 0.72
	}
	q := p/sz*2 - 1
	c *= 1 - 0.28*dot(q, q)*0.5
	return vec4(c, 1)
}
`

func shaders() {
	if blurShader != nil {
		return
	}
	var err error
	if blurShader, err = ebiten.NewShader([]byte(blurSrc)); err != nil {
		panic(fmt.Sprint("pix: blur shader: ", err))
	}
	if crtShader, err = ebiten.NewShader([]byte(crtSrc)); err != nil {
		panic(fmt.Sprint("pix: crt shader: ", err))
	}
}

// fxKeys runs in Update.
func fxKeys() {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		ix := (bloomIx.Load() + 1) % int32(len(bloomLevels))
		bloomIx.Store(ix)
		note(fmt.Sprintf("bloom %.1f", bloomLevels[ix]))
	case inpututil.IsKeyJustPressed(ebiten.KeyF2):
		posterOn.Store(!posterOn.Load())
		note(map[bool]string{true: "posterize on", false: "posterize off"}[posterOn.Load()])
	case inpututil.IsKeyJustPressed(ebiten.KeyF3):
		crtOn.Store(!crtOn.Load())
		note(map[bool]string{true: "CRT on", false: "CRT off"}[crtOn.Load()])
	}
}

func note(s string) {
	textMu.Lock()
	fxNote, fxNoteAt = s, time.Now()
	textMu.Unlock()
}

// fx holds the offscreen images for one arena size.
type fx struct {
	w, h, k         int
	view            *ebiten.Image // the scaled arena, before CRT
	half, halfT     *ebiten.Image
	quarter, quartT *ebiten.Image
}

func sized(img *ebiten.Image, w, h int) *ebiten.Image {
	if img != nil && img.Bounds().Dx() == w && img.Bounds().Dy() == h {
		return img
	}
	if img != nil {
		img.Deallocate()
	}
	return ebiten.NewImage(w, h)
}

func (f *fx) resize(iw, ih, k int) {
	f.w, f.h, f.k = iw, ih, k
	f.view = sized(f.view, iw*k, ih*k)
	f.half, f.halfT = sized(f.half, iw/2, ih/2), sized(f.halfT, iw/2, ih/2)
	f.quarter, f.quartT = sized(f.quarter, iw/4, ih/4), sized(f.quartT, iw/4, ih/4)
}

func blur(dst, tmp *ebiten.Image) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	for _, d := range [][2]float32{{1, 0}, {0, 1}} {
		src, out := dst, tmp
		if d[1] == 1 {
			src, out = tmp, dst
		}
		out.Clear()
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = src
		op.Uniforms = map[string]any{"Dir": []float32{d[0], d[1]}}
		out.DrawRectShader(w, h, blurShader, op)
	}
}

// drawArena puts the arena, scaled by k, at (ox, oy) on screen, with bloom
// from light and the CRT pass when they are on.
func (f *fx) drawArena(screen, arena, light *ebiten.Image, ox, oy, k int) {
	shaders()
	iw, ih := arena.Bounds().Dx(), arena.Bounds().Dy()
	if f.w != iw || f.h != ih || f.k != k {
		f.resize(iw, ih, k)
	}
	f.view.Clear()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(k), float64(k))
	f.view.DrawImage(arena, op)

	if s := bloomLevels[bloomIx.Load()]; s > 0 && light != nil {
		down := func(dst *ebiten.Image, src *ebiten.Image, by float64) {
			dst.Clear()
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(by, by)
			dst.DrawImage(src, op)
		}
		down(f.half, light, 0.5)
		blur(f.half, f.halfT)
		down(f.quarter, f.half, 0.5)
		blur(f.quarter, f.quartT)
		for _, b := range []struct {
			img   *ebiten.Image
			scale float64
			gain  float32
		}{{f.half, 2, 0.6}, {f.quarter, 4, 1.0}} {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
			op.GeoM.Scale(b.scale*float64(k), b.scale*float64(k))
			g := float32(s) * b.gain
			op.ColorScale.Scale(g, g, g, 1)
			f.view.DrawImage(b.img, op)
		}
	}

	if crtOn.Load() {
		op := &ebiten.DrawRectShaderOptions{}
		op.GeoM.Translate(float64(ox), float64(oy))
		op.Images[0] = f.view
		op.Uniforms = map[string]any{"Row": float32(k)}
		screen.DrawRectShader(iw*k, ih*k, crtShader, op)
		return
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(ox), float64(oy))
	screen.DrawImage(f.view, op)
}

// drawNote shows the last effect change for a moment, bottom left.
func drawNote(screen *ebiten.Image) {
	textMu.Lock()
	s, at := fxNote, fxNoteAt
	textMu.Unlock()
	if s == "" || time.Since(at) > 1500*time.Millisecond {
		return
	}
	drawText(screen, s, 16, float64(screen.Bounds().Dy())-20, color.RGBA{0xc8, 0xd0, 0xe8, 255}, text.AlignStart)
}
