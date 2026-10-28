package main

import (
	"fmt"
	"math"
	"os"
	"slices"
	"time"

	"github.com/nooga/let-go/pkg/vm"
)

// presentArena is gfx/compose!'s per-pixel work done in Go over the whole
// arena (gfx.lg's compose-pix! passes the screen's buffers): each pixel is the
// overlay if one is set, else the cached static layer seen through the field's
// rotation, plus the frame's light, posterized as compose! does. Then it
// clears what compose! clears as it goes (light, overlay, text and sprite
// cells, dirty marks), so the next frame starts clean.
var arenaArgs = []string{"S", "k", "cos", "sin", "ident?", "outside",
	"ovr", "wcache", "acc-r", "acc-g", "acc-b", "tglyph", "smask", "mark", "rmark"}

var (
	rgba     []byte
	postLUT  = makePostLUT()
	stats    []time.Duration // Go time per frame
	gaps     []time.Duration // between frames
	lastCall time.Time
)

// makePostLUT is gfx.lg's post-lut: 6 bits a channel, endpoints kept.
func makePostLUT() [256]byte {
	const levels = 63
	var a [256]byte
	for c := range 256 {
		inner := (c*levels + 127) / 255
		a[c] = byte((inner*255 + levels/2) / levels)
	}
	return a
}

func post8(v float64) byte {
	switch {
	case v > 255:
		return postLUT[255]
	case v < 0:
		return postLUT[0]
	}
	return postLUT[int(v)]
}

func presentArena(a []vm.Value) (vm.Value, error) {
	t0 := time.Now()
	if !lastCall.IsZero() {
		gaps = append(gaps, t0.Sub(lastCall))
	}
	lastCall = t0
	S, k := toInt(a[0]), toInt(a[1])
	c, s := toFloat(a[2]), toFloat(a[3])
	ident := a[4] == vm.TRUE
	outside := int64(toInt(a[5]))
	ints := func(i int) ([]int64, error) {
		if ta, ok := a[i].(*vm.TypedArray); ok && ta.Kind() == vm.ArrayInt {
			return ta.Unbox().([]int64), nil
		}
		return nil, fmt.Errorf("pix/present-arena!: %s wants an int-array", arenaArgs[i])
	}
	floats := func(i int) ([]float64, error) {
		if ta, ok := a[i].(*vm.TypedArray); ok && ta.Kind() == vm.ArrayFloat {
			return ta.Unbox().([]float64), nil
		}
		return nil, fmt.Errorf("pix/present-arena!: %s wants a double-array", arenaArgs[i])
	}
	ovr, err := ints(6)
	if err != nil {
		return vm.NIL, err
	}
	wc, err := ints(7)
	if err != nil {
		return vm.NIL, err
	}
	ar, err := floats(8)
	if err != nil {
		return vm.NIL, err
	}
	ag, err := floats(9)
	if err != nil {
		return vm.NIL, err
	}
	ab, err := floats(10)
	if err != nil {
		return vm.NIL, err
	}
	npx := S * S
	if len(ovr) < npx || len(wc) < npx || len(ar) < npx || len(ag) < npx || len(ab) < npx {
		return vm.NIL, fmt.Errorf("pix/present-arena!: buffers shorter than %dx%d", S, S)
	}

	if cap(rgba) < npx*4 {
		rgba = make([]byte, npx*4)
	}
	rgba = rgba[:npx*4]
	kf, kinv := float64(k), 1/float64(k)
	for y := range S {
		Y := (float64(y)+0.5)*kinv - 24
		for x := range S {
			i := x + y*S
			base := ovr[i]
			if base < 0 {
				if ident {
					base = wc[i]
				} else {
					X := (float64(x)+0.5)*kinv - 24
					wx, wy := 24+c*X+s*Y, 24+c*Y-s*X
					if wx < 0 || wx >= 48 || wy < 0 || wy >= 48 {
						base = outside
					} else {
						base = wc[int(wx*kf)+int(wy*kf)*S]
					}
				}
			}
			r, g, b := byte(base>>16), byte(base>>8), byte(base)
			if lr, lg, lb := ar[i], ag[i], ab[i]; lr != 0 || lg != 0 || lb != 0 {
				r, g, b = post8(float64(r)+lr), post8(float64(g)+lg), post8(float64(b)+lb)
			}
			o := i * 4
			rgba[o], rgba[o+1], rgba[o+2], rgba[o+3] = r, g, b, 255
		}
	}
	clear(ar[:npx])
	clear(ag[:npx])
	clear(ab[:npx])
	for i := range ovr[:npx] {
		ovr[i] = -1
	}
	if ta, ok := a[11].(*vm.TypedArray); ok && ta.Kind() == vm.ArrayObject {
		for i, v := range ta.Unbox().([]vm.Value) {
			if v != vm.NIL && v != nil {
				ta.Unbox().([]vm.Value)[i] = vm.NIL
			}
		}
	}
	for _, j := range []int{12, 13, 14} {
		if v, err := ints(j); err == nil {
			clear(v)
		}
	}

	if !isOpen.Load() {
		// a window about 2.5 px per arena pixel at k=4, within a laptop screen
		side := min(900, max(480, S*2))
		if !openWindow(side, side, "TESSERAE") {
			return vm.TRUE, nil
		}
	}
	setFrame(rgba, S, S)
	stats = append(stats, time.Since(t0))
	return vm.Boolean(closed.Load()), nil
}

// writeStats saves frame timings to $PIX_STATS, when set: Go compose time and
// the gap between frames (the whole frame as lg runs it), median and p95.
func writeStats() {
	path := os.Getenv("PIX_STATS")
	if path == "" || len(stats) == 0 {
		return
	}
	pct := func(d []time.Duration, p float64) float64 {
		s := slices.Clone(d)
		slices.Sort(s)
		return float64(s[int(math.Min(float64(len(s)-1), p*float64(len(s))))].Microseconds()) / 1000
	}
	out := fmt.Sprintf("frames %d\ncompose-ms median %.2f p95 %.2f\n", len(stats), pct(stats, 0.5), pct(stats, 0.95))
	if len(gaps) > 0 {
		out += fmt.Sprintf("frame-ms median %.2f p95 %.2f\n", pct(gaps, 0.5), pct(gaps, 0.95))
	}
	_ = os.WriteFile(path, []byte(out), 0o644)
}
