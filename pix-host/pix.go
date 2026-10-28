package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/nooga/let-go/pkg/rt"
	"github.com/nooga/let-go/pkg/vm"
)

// The pix namespace, as lg sees it:
//   (pix/open! w h title)   open the window (w x h logical px); returns once it's up
//   (pix/present! rgba w h) show a frame: a byte-array of w*h RGBA, copied here
//   (pix/closed?)           true once the window has gone (closed by the user, or close!)
//   (pix/close!)            close the window
//   (pix/events)            window input since the last call (input.go)
//   (pix/open-arena! S)     open the window for an S-px arena and its panel
//   (pix/text! ...), (pix/panel! ...), (pix/card! ...)  type over and beside
//                            the arena (text.go)
//   (pix/present-arena! ...) the game's arena, composed here from its own
//                            buffers (arena.go); opens the window on first use
//
// lg and Ebitengine run on different threads. present! copies the frame under
// a lock and Draw uploads the newest one, so lg never waits on the display
// and a slow frame just shows the previous one longer.

type winReq struct {
	w, h  int
	title string
	up    chan struct{}
}

var (
	openReq = make(chan winReq)

	mu     sync.Mutex
	frame  []byte
	light  []byte // nil when the frame came from present!
	fw, fh int
	fresh  bool

	quit   atomic.Bool
	closed atomic.Bool
	isOpen atomic.Bool
)

type game struct {
	up      chan struct{}
	img     *ebiten.Image
	light   *ebiten.Image
	fx      fx
	snapped bool
}

// PIX_SNAPSHOT=<file.png> saves the window as drawn after Update tick 60 (or
// PIX_SNAPSHOT_AT, on the same clock as PIX_INJECT), so a headless check can see what the window showed
// without a screen capture.
var snapshot = os.Getenv("PIX_SNAPSHOT")
var snapshotAt = func() int {
	if n, err := strconv.Atoi(os.Getenv("PIX_SNAPSHOT_AT")); err == nil {
		return n
	}
	return 60
}()

func (g *game) Update() error {
	if g.up != nil {
		close(g.up)
		g.up = nil
	}
	if quit.Load() {
		return ebiten.Termination
	}
	pollInput()
	fxKeys()
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	mu.Lock()
	if fresh {
		if g.img == nil || g.img.Bounds().Dx() != fw || g.img.Bounds().Dy() != fh {
			g.img = ebiten.NewImage(fw, fh)
		}
		g.img.WritePixels(frame)
		if len(light) == fw*fh*4 {
			g.light = sized(g.light, fw, fh)
			g.light.WritePixels(light)
		} else if g.light != nil {
			g.light.Deallocate()
			g.light = nil
		}
		fresh = false
	}
	mu.Unlock()
	// Largest whole-number scale that fits beside the panel, centred in what
	// is left: square pixels stay square. A window too narrow drops the panel.
	// Before the first arena frame (a level card) the arena is the square
	// that fits, at scale 1, so the card has somewhere to go.
	screen.Fill(color.Black)
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	aw, px := sw-panelPx, sw-panelPx+16
	iw, ih := min(aw, sh), min(aw, sh)
	if g.img != nil {
		iw, ih = g.img.Bounds().Dx(), g.img.Bounds().Dy()
	}
	if aw < iw {
		aw, px = sw, -1
	}
	k := max(1, min(aw/iw, sh/ih))
	ox, oy := (aw-iw*k)/2, (sh-ih*k)/2
	if g.img != nil {
		setView(ox, oy, k, iw, ih)
		g.fx.drawArena(screen, g.img, g.light, ox, oy, k)
	}
	drawOverlay(screen, ox, oy, k, iw, ih, px)
	drawNote(screen)
	if snapshot != "" && ticks >= snapshotAt && !g.snapped {
		g.snapped = true
		saveSnapshot(screen)
	}
}

func saveSnapshot(screen *ebiten.Image) {
	b := screen.Bounds()
	rgba := image.NewRGBA(b)
	screen.ReadPixels(rgba.Pix)
	f, err := os.Create(snapshot)
	if err != nil {
		fmt.Println("pix: snapshot:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, rgba); err != nil {
		fmt.Println("pix: snapshot:", err)
	}
}

func (g *game) Layout(ow, oh int) (int, int) { return ow, oh }

// runWindow runs on the main thread until the window closes.
func runWindow(r winReq) {
	quit.Store(false)
	closed.Store(false)
	isOpen.Store(true)
	defer isOpen.Store(false)
	ebiten.SetWindowSize(r.w, r.h)
	ebiten.SetWindowTitle(r.title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	g := &game{up: r.up}
	err := ebiten.RunGame(g)
	closed.Store(true)
	if g.up != nil { // RunGame failed before the first Update
		close(g.up)
	}
	if err != nil {
		fmt.Println("pix: window:", err)
	}
}

func installPix() {
	ns := vm.NewNamespace("pix")
	ns.Def("open!", native("open!", 3, func(a []vm.Value) (vm.Value, error) {
		return vm.Boolean(openWindow(toInt(a[0]), toInt(a[1]), fmt.Sprint(a[2].Unbox()))), nil
	}))
	ns.Def("present-arena!", native("present-arena!", len(arenaArgs), presentArena))
	ns.Def("present!", native("present!", 3, func(a []vm.Value) (vm.Value, error) {
		ta, ok := a[0].(*vm.TypedArray)
		if !ok || ta.Kind() != vm.ArrayByte {
			return vm.NIL, fmt.Errorf("pix/present!: wants a byte-array, got %s", a[0].Type())
		}
		px := ta.Unbox().([]byte)
		w, h := toInt(a[1]), toInt(a[2])
		if len(px) < w*h*4 {
			return vm.NIL, fmt.Errorf("pix/present!: %d bytes is short of %dx%d RGBA", len(px), w, h)
		}
		setFrame(px, nil, w, h)
		return vm.NIL, nil
	}))
	ns.Def("closed?", native("closed?", 0, func([]vm.Value) (vm.Value, error) {
		return vm.Boolean(closed.Load()), nil
	}))
	ns.Def("events", native("events", 0, drainEvents))
	ns.Def("panel!", native("panel!", 2, setPanel))
	ns.Def("text!", native("text!", 4, addText))
	ns.Def("card!", native("card!", 4, setCard))
	ns.Def("open-arena!", native("open-arena!", 1, func(a []vm.Value) (vm.Value, error) {
		return vm.Boolean(isOpen.Load() || openArena(toInt(a[0]))), nil
	}))
	ns.Def("close!", native("close!", 0, func([]vm.Value) (vm.Value, error) {
		quit.Store(true)
		return vm.NIL, nil
	}))
	rt.RegisterNS(ns)
}

// openWindow asks main to open the window and waits until it is up. Reports
// whether it is.
func openWindow(w, h int, title string) bool {
	r := winReq{w: w, h: h, title: title, up: make(chan struct{})}
	openReq <- r
	<-r.up
	return !closed.Load()
}

// setFrame hands a finished RGBA frame, and its light for bloom (or nil), to Draw.
func setFrame(px, lt []byte, w, h int) {
	mu.Lock()
	if cap(frame) < w*h*4 {
		frame = make([]byte, w*h*4)
	}
	frame = frame[:w*h*4]
	copy(frame, px)
	light = light[:0]
	if lt != nil {
		light = append(light, lt[:w*h*4]...)
	}
	fw, fh, fresh = w, h, true
	mu.Unlock()
}

func toInt(v vm.Value) int {
	switch n := v.Unbox().(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

func toFloat(v vm.Value) float64 {
	switch n := v.Unbox().(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

func native(name string, arity int, fn func([]vm.Value) (vm.Value, error)) vm.Value {
	v, _ := vm.NativeFnType.Wrap(func(a []vm.Value) (vm.Value, error) {
		if len(a) != arity {
			return vm.NIL, fmt.Errorf("pix/%s: wants %d argument(s), got %d", name, arity, len(a))
		}
		return fn(a)
	})
	f := v.(*vm.NativeFn).WithArity(arity, false)
	f.SetName(name)
	return f
}
