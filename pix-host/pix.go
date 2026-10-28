package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
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
	fw, fh int
	fresh  bool

	quit   atomic.Bool
	closed atomic.Bool
)

type game struct {
	up    chan struct{}
	img   *ebiten.Image
	draws int
}

// PIX_SNAPSHOT=<file.png> saves the window's 60th drawn frame, so a headless
// check can see what the window showed without a screen capture.
var snapshot = os.Getenv("PIX_SNAPSHOT")

func (g *game) Update() error {
	if g.up != nil {
		close(g.up)
		g.up = nil
	}
	if quit.Load() {
		return ebiten.Termination
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	mu.Lock()
	if fresh {
		if g.img == nil || g.img.Bounds().Dx() != fw || g.img.Bounds().Dy() != fh {
			g.img = ebiten.NewImage(fw, fh)
		}
		g.img.WritePixels(frame)
		fresh = false
	}
	mu.Unlock()
	if g.img == nil {
		return
	}
	// Largest whole-number scale that fits, centred: square pixels stay square.
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	iw, ih := g.img.Bounds().Dx(), g.img.Bounds().Dy()
	k := max(1, min(sw/iw, sh/ih))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(k), float64(k))
	op.GeoM.Translate(float64((sw-iw*k)/2), float64((sh-ih*k)/2))
	screen.DrawImage(g.img, op)
	if g.draws++; snapshot != "" && g.draws == 60 {
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
		w, h := toInt(a[0]), toInt(a[1])
		r := winReq{w: w, h: h, title: fmt.Sprint(a[2].Unbox()), up: make(chan struct{})}
		openReq <- r
		<-r.up
		return vm.Boolean(!closed.Load()), nil
	}))
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
		mu.Lock()
		if cap(frame) < w*h*4 {
			frame = make([]byte, w*h*4)
		}
		frame = frame[:w*h*4]
		copy(frame, px)
		fw, fh, fresh = w, h, true
		mu.Unlock()
		return vm.NIL, nil
	}))
	ns.Def("closed?", native("closed?", 0, func([]vm.Value) (vm.Value, error) {
		return vm.Boolean(closed.Load()), nil
	}))
	ns.Def("close!", native("close!", 0, func([]vm.Value) (vm.Value, error) {
		quit.Store(true)
		return vm.NIL, nil
	}))
	rt.RegisterNS(ns)
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
