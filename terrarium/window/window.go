// Package window is the window module of the Terrarium: a window with a
// pixel buffer the program draws on, the surfaces shown over it, and the
// input it gets.
//
// Ebitengine is used only to open the window, read its input and show the
// finished frame; every pixel is drawn by the program, with the drawing
// functions every target shares (package canvas). A headless window runs
// the same ticks with no screen at all, for tests and benchmarks. Importing
// this package adds window and pen to terrarium.Install:
//
//	import _ "github.com/alejandruxxug/lizard-script/terrarium/window"
package window

import (
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/canvas"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/pixels"
	"github.com/alejandruxxug/lizard-script/terrarium/pen"
	"image/color"
	"os"
	"strings"
	"time"
)

func init() {
	terrarium.Register(func(env *object.Environment) {
		env.Define("window", Module())
		env.Define("pen", pen.Module())
	})
}

// Kind is the kind of the natives that are windows.
const Kind = "window"

// maxSide is the biggest width or height of a window's buffer.
const maxSide = 8192

// black is the color of a new window.
var black = color.NRGBA{A: 255}

// Window is a window and the buffer it shows. Its size is the buffer's, in
// pixels; with a scale of 2 each one is shown as 2x2 on the screen.
type Window struct {
	buf         *pixels.Image
	title       string
	scale       int
	fps         int
	resizable   bool
	frameless   bool
	transparent bool // chosen at open: the system can't change it later
	floating    bool // above every other window
	minW, minH  int  // the smallest buffer the user can resize it to
	hover       bool // move events with no button down
	onDemand    bool // tick runs only when there is something to do
	headless    bool // no screen: ticks, events and screenshots only

	changed      bool // drawn on since it was last shown
	drew         bool // it or a surface was drawn on this frame
	titleChanged bool // title changed while it runs
	chrome       bool // an option changed while it runs (window.set)
	center       bool // move it to the middle of the screen
	wake         bool // window.needs_paint: run tick next frame
	started      bool // tick ran once
	idle         bool // the last frame had nothing to do, so tick didn't run
	running      bool

	screen screen  // the surfaces and the composed frame (surface.go)
	queued []Event // events for the next frame: injected, or made by window.set
	auto   *automation
	stats  stats
}

// stats are the numbers automation's Stats reports.
type stats struct {
	ticks, frames, events int
	tickMS, composeMS     float64
}

// Base, Pixels and Touched make a window a drawing target.
func (w *Window) Base() *pixels.Image   { return w.buf }
func (w *Window) Pixels() *pixels.Image { return w.buf.Top() }
func (w *Window) Touched()              { w.changed, w.drew, w.screen.base.touched = true, true, true }

// Describe is the size, for Inspect: <window 600x400>.
func (w *Window) Describe() string { return w.buf.Describe() }

// resize gives the window a buffer of the new size, keeping the pixels
// that still fit.
func (w *Window) resize(width, height int) {
	if bigger, err := w.buf.Resized(width, height); err == nil {
		w.buf = bigger
		w.changed = true
		w.layout()
	}
}

// Module is the window module: open, run, title, set and the rest of this
// package, and the shared drawing functions of package canvas.
func Module() *object.Hash {
	return arg.Module("window", canvas.Add(map[string]object.BuiltinFunction{
		"open":        open,
		"run":         run,
		"title":       title,
		"set":         set,
		"needs_paint": needsPaint,
		"inject":      inject,
		"surface":     surfaceFunc,
		"dirty":       dirty,
		"screenshot":  screenshot,
		"automation":  automationFunc,
	}))
}

// windowArg reads argument i as a window.
func windowArg(args []object.Object, i int) (*Window, *object.Error) {
	v, err := arg.Native(args, i, Kind)
	if err != nil {
		return nil, err
	}
	return v.(*Window), nil
}

// window.open(w, h, title, options = {}) makes a window of w by h pixels.
// It shows on the screen when window.run starts, so the program can draw
// its first picture before. The options are in openOptions.
func open(args ...object.Object) object.Object {
	if err := arg.Count(args, 3, 4); err != nil {
		return err
	}
	width, err := arg.IntIn(args, 0, "width", 1, maxSide)
	if err != nil {
		return err
	}
	height, err := arg.IntIn(args, 1, "height", 1, maxSide)
	if err != nil {
		return err
	}
	name, err := arg.String(args, 2, "title")
	if err != nil {
		return err
	}
	buf, berr := pixels.New(width, height, black)
	if berr != nil {
		return arg.Errorf("%v", berr)
	}
	w := &Window{buf: buf, title: name, scale: 1, fps: 60, hover: true, changed: true}
	w.layout()
	if len(args) == 4 {
		options, err := arg.Hash(args, 3, "options")
		if err != nil {
			return err
		}
		if err := w.setOptions(options, openOptions); err != nil {
			return err
		}
	}
	if os.Getenv(AutomationEnv) == "1" && w.auto == nil {
		w.auto = newAutomation()
	}
	return &object.Native{Kind: Kind, Value: w}
}

// openOptions are the keys window.open takes in its options, and
// setOptions the ones window.set takes.
var (
	openOptions = []string{"automation", "floating", "fps", "frameless", "headless", "hover", "min_size", "on_demand", "resizable", "scale", "transparent"}
	setOptions  = []string{"center", "floating", "fps", "frameless", "hover", "min_size", "on_demand", "resizable", "scale", "size", "title"}
)

// setOptions reads the options of window.open or window.set; allowed are
// the ones it takes.
func (w *Window) setOptions(options *object.Hash, allowed []string) *object.Error {
	var err *object.Error
	options.Each(func(pair object.HashPair) {
		if err != nil {
			return
		}
		name := pair.Key.Inspect()
		if s, ok := pair.Key.(*object.String); ok {
			name = s.Value
		}
		if !contains(allowed, name) {
			err = unknownOption(name, allowed)
			return
		}
		err = w.setOption(name, []object.Object{pair.Value})
	})
	return err
}

// unknownOption is the error of an option a function doesn't take.
func unknownOption(name string, allowed []string) *object.Error {
	if name == "transparent" || name == "headless" {
		return arg.Errorf("%s can only be chosen in window.open", name)
	}
	return arg.Errorf("unknown option %q (the options are %s)", name, strings.Join(allowed, ", "))
}

// setOption sets one option to value[0].
func (w *Window) setOption(name string, value []object.Object) *object.Error {
	var err *object.Error
	switch name {
	case "scale":
		var scale int
		if scale, err = arg.IntIn(value, 0, "scale", 1, 8); err == nil && scale != w.scale {
			w.scale = scale
			w.layout()
		}
	case "fps":
		w.fps, err = arg.IntIn(value, 0, "fps", 1, 240)
	case "resizable":
		w.resizable, err = boolOption(value, name)
	case "frameless":
		w.frameless, err = boolOption(value, name)
	case "transparent":
		w.transparent, err = boolOption(value, name)
	case "floating":
		w.floating, err = boolOption(value, name)
	case "hover":
		w.hover, err = boolOption(value, name)
	case "on_demand":
		w.onDemand, err = boolOption(value, name)
	case "headless":
		w.headless, err = boolOption(value, name)
	case "center":
		w.center, err = boolOption(value, name)
	case "automation":
		var on bool
		if on, err = boolOption(value, name); err == nil && on && w.auto == nil {
			w.auto = newAutomation()
		}
	case "min_size":
		w.minW, w.minH, err = sizeOption(value, name, 0)
	case "size":
		var width, height int
		if width, height, err = sizeOption(value, name, 1); err == nil {
			w.setSize(width, height)
		}
	case "title":
		if w.title, err = arg.String(value, 0, "title"); err == nil {
			w.titleChanged = true
		}
	}
	w.chrome = true
	return err
}

// setSize resizes the buffer to at least the smallest size, and tells the
// program with a resize event on the next frame.
func (w *Window) setSize(width, height int) {
	width, height = max(width, w.minW), max(height, w.minH)
	if width == w.buf.W && height == w.buf.H {
		return
	}
	w.resize(width, height)
	w.queued = append(w.queued, Event{Type: "resize", W: width, H: height})
}

// contains reports whether list has s.
func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// boolOption reads an option that is true or false.
func boolOption(value []object.Object, name string) (bool, *object.Error) {
	b, ok := value[0].(*object.Boolean)
	if !ok {
		return false, arg.Errorf("want a BOOLEAN %s, got %s", name, arg.Name(value[0]))
	}
	return b.Value, nil
}

// sizeOption reads an option that is a point (w, h) of whole numbers from
// least to maxSide.
func sizeOption(value []object.Object, name string, least int) (int, int, *object.Error) {
	x, y, ok := arg.XY(value[0])
	if !ok || x != float64(int(x)) || y != float64(int(y)) || x < float64(least) || y < float64(least) || x > maxSide || y > maxSide {
		return 0, 0, arg.Errorf("the %s must be a point (w, h) of whole numbers from %d to %d, got %s", name, least, maxSide, value[0].Inspect())
	}
	return int(x), int(y), nil
}

// window.title(w, text) changes the window's title.
func title(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	if w.title, err = arg.String(args, 1, "title"); err != nil {
		return err
	}
	w.titleChanged = true
	return arg.Null
}

// window.set(w, options) changes options of the window, while it runs or
// before: the frame, the size, the scale, centering it on the screen. The
// options are in setOptions; the window changes after the frame.
func set(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	options, err := arg.Hash(args, 1, "options")
	if err != nil {
		return err
	}
	if err := w.setOptions(options, setOptions); err != nil {
		return err
	}
	return arg.Null
}

// window.needs_paint(w) asks for one more frame: with on_demand, tick runs
// next frame even when nothing happens.
func needsPaint(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	w.wake = true
	return arg.Null
}

// loop shows w and calls frame once a frame with the events since the
// last one, until frame returns false or the window closes. Tests replace
// it with one that opens no window.
var loop = ebitenLoop

// ran is set once a window has run: Ebitengine can open only one window in
// the life of a program. Headless windows don't count.
var ran bool

// window.run(w, tick) shows the window and calls tick(events) every frame,
// with the list of events since the last frame (often empty), until tick
// returns false or the window is closed. Closing it sends one last "close"
// event first. An error in tick stops the window and is run's error.
func run(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	tick := args[1]
	if tick.Type() != object.FUNCTION_OBJ && tick.Type() != object.BUILTIN_OBJ {
		return arg.Errorf("want a FUNCTION tick, got %s", arg.Name(tick))
	}
	if w.running {
		return arg.Errorf("the window is already running")
	}
	show := headlessLoop
	if !w.headless {
		if ran {
			return arg.Errorf("a program can show only one window")
		}
		ran, show = true, loop
	}
	if w.auto != nil {
		if serr := w.auto.start(w); serr != nil {
			return arg.Errorf("can't start the automation: %v", serr)
		}
		defer w.auto.stop()
	}
	w.running = true
	defer func() { w.running = false }()

	var failed *object.Error
	loopErr := show(w, func(events []Event) bool {
		keepGoing, err := w.frame(tick, events)
		failed = err
		return keepGoing
	})
	if failed != nil {
		return failed
	}
	if loopErr != nil {
		return arg.Errorf("the window failed: %v", loopErr)
	}
	return arg.Null
}

// frame is one frame of a running window: the automation's commands, the
// events, tick, and the replies to the commands, which see what tick did.
// With on_demand, tick runs only when there is something to do.
func (w *Window) frame(tick object.Object, events []Event) (bool, *object.Error) {
	replies := w.auto.runJobs()
	events = append(events, w.queued...)
	w.queued = nil
	busy := len(events) > 0 || len(replies) > 0 || w.wake || !w.started
	keepGoing := true
	w.idle = !busy && w.onDemand
	if !w.idle {
		w.started, w.wake = true, false
		start := time.Now()
		result := call(tick, events)
		w.stats.tickMS = float64(time.Since(start).Microseconds()) / 1000
		w.stats.ticks++
		w.stats.events += len(events)
		if w.drew {
			w.stats.frames++
			w.drew = false
		}
		if e, ok := result.(*object.Error); ok {
			w.auto.reply(replies, true)
			return false, e
		}
		keepGoing = result != arg.False
	}
	w.auto.reply(replies, busy)
	return keepGoing && !w.auto.quitting(), nil
}

// call runs tick with the events as a list of hashes, the way list.sort
// calls its comparison: with the checks of a call written in the program,
// for a function of either engine.
func call(tick object.Object, events []Event) object.Object {
	list := &object.Array{Elements: make([]object.Object, len(events))}
	for i, e := range events {
		list.Elements[i] = e.hash()
	}
	return apply(tick, list)
}

// apply calls a function of the program with args.
func apply(fn object.Object, args ...object.Object) object.Object {
	return evaluator.ApplyFunction(token.Token{}, fn, args)
}
