package window

import (
	"github.com/alejandruxxug/lizard-script/terrarium/internal/pixels"
	"github.com/alejandruxxug/lizard-script/terrarium/pen"
	"image"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The Ebitengine side of the window. Only WritePixels shows pixels; none of
// Ebitengine's drawing, text or vector functions are used.

// ebitenLoop opens the window and runs frame once a frame. It must be
// called from the program's main goroutine, which is where a lizard-script
// program runs.
func ebitenLoop(w *Window, frame func(events []Event) bool) error {
	ebiten.SetWindowTitle(w.title)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetScreenClearedEveryFrame(false)
	g := &game{w: w, frame: frame, lastX: -1, lastY: -1}
	g.applyChrome()
	options := &ebiten.RunGameOptions{ScreenTransparent: w.transparent}
	if err := ebiten.RunGameWithOptions(g, options); err != nil && err != ebiten.Termination {
		return err
	}
	return nil
}

// game is what Ebitengine runs.
type game struct {
	w     *Window
	frame func(events []Event) bool

	started        bool
	layoutW        int // the buffer size the window has room for
	layoutH        int
	expect         int    // frames to wait for the system to take a new size
	shownW, shownH int    // the size of the screen last written
	premultiplied  []byte // the buffer as the screen takes it
	lastX, lastY   int
	presence       presence
	cursor         string // the cursor shown
}

// applyChrome gives the system window the options the program chose: its
// frame, size, scale, limits and speed.
func (g *game) applyChrome() {
	w := g.w
	sw, sh := w.screenSize()
	ebiten.SetWindowSize(sw, sh)
	ebiten.SetWindowDecorated(!w.frameless)
	ebiten.SetWindowFloating(w.floating)
	if w.resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	} else {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	}
	minW, minH := -1, -1
	if w.minW > 0 || w.minH > 0 {
		minW, minH = max(1, w.minW)*w.scale, max(1, w.minH)*w.scale
	}
	ebiten.SetWindowSizeLimits(minW, minH, -1, -1)
	ebiten.SetTPS(w.fps)
	if w.center {
		if m := ebiten.Monitor(); m != nil {
			mw, mh := m.Size()
			ebiten.SetWindowPosition(max(0, (mw-sw)/2), max(0, (mh-sh)/2))
		}
		w.center = false
	}
	w.chrome = false
	g.expect = 30
}

// Layout makes the screen the size of the frame. A resizable window's
// buffer follows the window: its size divided by the scale.
func (g *game) Layout(outW, outH int) (int, int) {
	w := g.w
	if !w.resizable {
		return w.frameSize()
	}
	g.layoutW, g.layoutH = max(1, outW/w.scale), max(1, outH/w.scale)
	u := w.screen.unit
	return g.layoutW * w.scale / u, g.layoutH * w.scale / u
}

// Update runs one frame of the program.
func (g *game) Update() error {
	w := g.w
	if !g.started {
		g.started = true
		pen.Start()
		startPinch()
	}
	events := g.poll()
	if g.expect > 0 {
		// The system takes a few frames to give the window the size the
		// program asked for; the old size meanwhile is not a resize.
		g.expect--
		if g.layoutW == w.buf.W && g.layoutH == w.buf.H {
			g.expect = 0
		}
	} else if w.resizable && g.layoutW > 0 && (g.layoutW != w.buf.W || g.layoutH != w.buf.H) {
		w.resize(g.layoutW, g.layoutH)
		events = append(events, Event{Type: "resize", W: g.layoutW, H: g.layoutH})
	}
	closing := ebiten.IsWindowBeingClosed()
	g.applyCursor()
	if closing {
		events = append(events, Event{Type: "close"})
	}
	keepGoing := g.frame(events)
	if w.titleChanged {
		ebiten.SetWindowTitle(w.title)
		w.titleChanged = false
	}
	if w.chrome {
		g.applyChrome()
	}
	if !keepGoing || closing {
		return ebiten.Termination
	}
	return nil
}

// Draw shows what changed: the whole buffer when the window has no
// surfaces, else the changed rectangles of the composed frame. A new
// screen gets everything.
func (g *game) Draw(screen *ebiten.Image) {
	w := g.w
	b := screen.Bounds()
	fw, fh := w.frameSize()
	if b.Dx() != fw || b.Dy() != fh {
		return
	}
	fresh := b.Dx() != g.shownW || b.Dy() != g.shownH
	g.shownW, g.shownH = b.Dx(), b.Dy()
	if w.screen.frame == nil {
		if !w.changed && !fresh {
			return
		}
		g.write(screen, w.buf, w.buf.Bounds())
		w.changed = false
		return
	}
	if fresh {
		w.screen.all = true
	}
	for _, r := range w.compose() {
		g.write(screen, w.screen.frame, r)
	}
	w.changed = false
}

// write puts the rectangle r of src on the screen, premultiplied as the
// screen takes it.
func (g *game) write(screen *ebiten.Image, src *pixels.Image, r image.Rectangle) {
	if r == src.Bounds() {
		if len(g.premultiplied) != len(src.Pix) {
			g.premultiplied = make([]byte, len(src.Pix))
		}
		src.Premultiplied(g.premultiplied)
		screen.WritePixels(g.premultiplied)
		return
	}
	part := src.Crop(r)
	out := make([]byte, len(part.Pix))
	part.Premultiplied(out)
	screen.SubImage(r).(*ebiten.Image).WritePixels(out)
}

// DrawFinalScreen scales the frame to the window with sharp pixels, so a
// scale of 2 shows each pixel as a clean 2x2 square.
func (g *game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geoM, Filter: ebiten.FilterNearest})
}

var buttons = []struct {
	button ebiten.MouseButton
	name   string
}{
	{ebiten.MouseButtonLeft, "left"},
	{ebiten.MouseButtonRight, "right"},
	{ebiten.MouseButtonMiddle, "middle"},
}

// poll reads the input of this frame as events.
func (g *game) poll() []Event {
	w := g.w
	var events []Event
	cx, cy := ebiten.CursorPosition()
	u := float64(w.screen.unit)
	sx, sy := float64(cx)*u, float64(cy)*u // in screen pixels
	mods := Mods{
		Shift: ebiten.IsKeyPressed(ebiten.KeyShift),
		Alt:   ebiten.IsKeyPressed(ebiten.KeyAlt),
		Cmd:   ebiten.IsKeyPressed(ebiten.KeyMeta),
		Ctrl:  ebiten.IsKeyPressed(ebiten.KeyControl),
	}
	now := pen.Now()
	native := pen.Drain()
	pointer := func(kind, button string, p Pen) {
		e := w.at(Event{Type: kind, Button: button, Mods: mods, Pen: p, T: now}, sx, sy)
		if kind == "move" {
			e.Samples = w.dragSamples(native, e)
		}
		events = append(events, e)
	}

	held := false
	for _, b := range buttons {
		if inpututil.IsMouseButtonJustPressed(b.button) {
			pointer("down", b.name, penAt(native, pen.PhaseDown, true))
		}
		held = held || ebiten.IsMouseButtonPressed(b.button)
	}
	moved := cx != g.lastX || cy != g.lastY
	dragged := held && hasPhase(native, pen.PhaseDrag)
	if (moved || dragged) && (held || w.hover) {
		pointer("move", "", penAt(native, pen.PhaseDrag, held))
	}
	g.lastX, g.lastY = cx, cy
	for _, b := range buttons {
		if inpututil.IsMouseButtonJustReleased(b.button) {
			pointer("up", b.name, penAt(native, pen.PhaseUp, false))
		}
	}
	if dx, dy := ebiten.Wheel(); dx != 0 || dy != 0 {
		events = append(events, w.at(Event{Type: "scroll", DX: dx, DY: dy, Mods: mods, T: now}, sx, sy))
	}

	if scale, ok := drainPinch(); ok {
		events = append(events, w.at(Event{Type: "pinch", Scale: scale, Mods: mods, T: now}, sx, sy))
	}

	if paths := droppedPaths(); len(paths) > 0 {
		events = append(events, w.at(Event{Type: "drop", Paths: paths, Mods: mods, T: now}, sx, sy))
	}

	for _, k := range inpututil.AppendPressedKeys(nil) {
		if g.repeats(k) {
			events = append(events, Event{Type: "key", Key: k.String(), Mods: mods, T: now})
		}
	}
	for _, k := range inpututil.AppendJustReleasedKeys(nil) {
		events = append(events, Event{Type: "keyup", Key: k.String(), Mods: mods, T: now})
	}
	// The text as the system composed it (dead keys, option combinations).
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= 32 && r != 127 {
			events = append(events, Event{Type: "type", Text: string(r), Mods: mods, T: now})
		}
	}
	fw, fh := w.frameSize()
	return append(events, g.presence.update(ebiten.IsFocused(), cx, cy, fw, fh)...)
}

// cursorShapes are the cursors window.cursor takes, as Ebitengine's shapes.
// Ebitengine has no open-hand cursor, so "grab" is its four-arrow move.
var cursorShapes = map[string]ebiten.CursorShapeType{
	"default":   ebiten.CursorShapeDefault,
	"text":      ebiten.CursorShapeText,
	"pointer":   ebiten.CursorShapePointer,
	"grab":      ebiten.CursorShapeMove,
	"crosshair": ebiten.CursorShapeCrosshair,
	"resize-ew": ebiten.CursorShapeEWResize,
	"resize-ns": ebiten.CursorShapeNSResize,
}

// applyCursor shows the cursor the program chose, when it changed.
func (g *game) applyCursor() {
	if g.w.cursor != g.cursor {
		g.cursor = g.w.cursor
		ebiten.SetCursorShape(cursorShapes[g.cursor])
	}
}

// repeats reports whether a held key sends an event this frame: when it
// goes down, then after half a second about 15 times a second.
func (g *game) repeats(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	delay, every := max(1, g.w.fps/2), max(1, g.w.fps/15)
	return d == 1 || (d >= delay && (d-delay)%every == 0)
}

// hasPhase reports whether one of the native samples is of the phase.
func hasPhase(native []pen.Sample, phase string) bool {
	for _, s := range native {
		if s.Phase == phase {
			return true
		}
	}
	return false
}

// penAt is the pen of a pointer event: the newest native sample of the
// phase when there is one, else the latest reading while a button is held,
// else a plain mouse.
func penAt(native []pen.Sample, phase string, held bool) Pen {
	latest := pen.Latest()
	for i := len(native) - 1; i >= 0; i-- {
		if s := native[i]; s.Phase == phase && s.Source != "mouse" {
			return newPen(s.Pressure, s.Tilt, s.Source, latest.Eraser)
		}
	}
	if held && latest.OK && latest.Source != "" && latest.Source != "mouse" {
		return newPen(latest.Pressure, latest.Tilt, latest.Source, latest.Eraser)
	}
	return newPen(1, [2]float64{}, "mouse", false)
}

// newPen is a pen with the device's response curve applied to the raw
// pressure.
func newPen(raw float64, tilt [2]float64, source string, eraser bool) Pen {
	return Pen{Pressure: pen.Shape(source, raw), Raw: raw, Tilt: tilt, Source: source, Eraser: eraser}
}

// droppedPaths are the files dropped on the window this frame, by their
// real paths. A file the system gives with no real path (a browser) is left
// out.
func droppedPaths() []string {
	return pathsOf(ebiten.DroppedFiles())
}

// pathsOf is the absolute paths of the files in a drop. Ebitengine gives
// nil on every frame where nothing was dropped.
func pathsOf(files fs.FS) []string {
	if files == nil {
		return nil
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if p, ok := entry.(ebiten.AbsPather); ok {
			paths = append(paths, p.AbsPath())
		}
	}
	return paths
}
