package window

import (
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"lizardscript/terrarium/pen"
	"math"
	"path/filepath"
	"strings"
)

// Mods are the modifier keys held during an event.
type Mods struct{ Shift, Alt, Cmd, Ctrl bool }

// Pen is what the pointer knows besides its position.
type Pen struct {
	Pressure float64    // 0 to 1, after the device's response curve; 1 for a mouse
	Raw      float64    // 0 to 1, as the device reported it
	Tilt     [2]float64 // tablet tilt, -1 to 1 on each axis
	Source   string     // "mouse", "trackpad" or "tablet"
	Eraser   bool       // the eraser end of a tablet pen
}

// Spot is where the pointer is on a surface: its name and the position in
// the surface's own pixels. Name is "" when the pointer is over no surface.
type Spot struct {
	Name string
	X, Y int
}

// Sample is one point of a drag: every one the system sent between two
// frames, not only the last. X and Y are buffer pixels, T is pen.Now().
type Sample struct {
	X, Y     int
	Spot     Spot
	Pressure float64
	Raw      float64
	T        float64
}

// Event is one input event. X and Y are buffer pixels.
type Event struct {
	Type    string // down, up, move, scroll, drop, key, type, resize, focus, leave, enter or close
	X, Y    int    // the pointer, for down, up, move and scroll
	Spot    Spot   // the surface under the pointer
	Button  string // down and up: "left", "right" or "middle"
	Key     string // key: the key's name, like "A", "Escape" or "ArrowLeft"
	Text    string // type: the character typed
	DY      float64
	W, H    int  // resize: the new size
	Focused bool // focus: the window got (true) or lost (false) the focus
	Mods    Mods
	Pen     Pen      // down, up and move
	Samples []Sample // move: every point since the last frame, the last one at X, Y
	Paths   []string // drop: the files dropped, as absolute paths
	T       float64  // when it happened, pen.Now()
}

// pointer reports whether an event has a position.
func (e Event) pointer() bool {
	switch e.Type {
	case "down", "up", "move", "scroll", "drop":
		return true
	}
	return false
}

// hash is the event as the program sees it. Only the keys that mean
// something for its type are there.
func (e Event) hash() *object.Hash {
	h := object.NewHash()
	arg.Set(h, "type", arg.Str(e.Type))
	if e.pointer() {
		arg.Set(h, "pos", arg.PointOf(e.X, e.Y))
		setSpot(h, e.Spot)
	}
	switch e.Type {
	case "down", "up":
		arg.Set(h, "button", arg.Str(e.Button))
	case "key":
		arg.Set(h, "key", arg.Str(e.Key))
	case "type":
		arg.Set(h, "text", arg.Str(e.Text))
	case "scroll":
		arg.Set(h, "dy", arg.Float(e.DY))
	case "resize":
		arg.Set(h, "size", arg.PointOf(e.W, e.H))
	case "focus":
		arg.Set(h, "focused", arg.Bool(e.Focused))
	case "drop":
		list := &object.Array{Elements: make([]object.Object, len(e.Paths))}
		for i, p := range e.Paths {
			list.Elements[i] = arg.Str(p)
		}
		arg.Set(h, "paths", list)
	}
	switch e.Type {
	case "resize", "close", "focus", "leave", "enter":
	default:
		arg.Set(h, "mods", modsHash(e.Mods))
	}
	switch e.Type {
	case "down", "up", "move":
		arg.Set(h, "pen", penHash(e.Pen))
	}
	if e.Type == "move" {
		arg.Set(h, "samples", e.samplesList())
	}
	return h
}

// setSpot adds the surface under the pointer to h, when there is one.
func setSpot(h *object.Hash, s Spot) {
	if s.Name != "" {
		arg.Set(h, "surface", arg.Str(s.Name))
		arg.Set(h, "surface_pos", arg.PointOf(s.X, s.Y))
	}
}

// samplesList is the samples of a move event. A move with no samples of
// the system's (a system without them, an injected move) has one, its own
// point.
func (e Event) samplesList() *object.Array {
	samples := e.Samples
	if len(samples) == 0 {
		samples = []Sample{{X: e.X, Y: e.Y, Spot: e.Spot, Pressure: e.Pen.Pressure, Raw: e.Pen.Raw, T: e.T}}
	}
	list := &object.Array{Elements: make([]object.Object, len(samples))}
	for i, s := range samples {
		h := object.NewHash()
		arg.Set(h, "pos", arg.PointOf(s.X, s.Y))
		setSpot(h, s.Spot)
		arg.Set(h, "pressure", arg.Float(s.Pressure))
		arg.Set(h, "raw", arg.Float(s.Raw))
		arg.Set(h, "t", arg.Float(s.T))
		list.Elements[i] = h
	}
	return list
}

func modsHash(m Mods) *object.Hash {
	h := object.NewHash()
	arg.Set(h, "shift", arg.Bool(m.Shift))
	arg.Set(h, "alt", arg.Bool(m.Alt))
	arg.Set(h, "cmd", arg.Bool(m.Cmd))
	arg.Set(h, "ctrl", arg.Bool(m.Ctrl))
	return h
}

func penHash(p Pen) *object.Hash {
	h := object.NewHash()
	arg.Set(h, "pressure", arg.Float(p.Pressure))
	arg.Set(h, "raw", arg.Float(p.Raw))
	arg.Set(h, "source", arg.Str(p.Source))
	arg.Set(h, "tilt", &object.Point{Coords: []object.Object{arg.Float(p.Tilt[0]), arg.Float(p.Tilt[1])}})
	arg.Set(h, "eraser", arg.Bool(p.Eraser))
	return h
}

// dragSamples are the system's drag samples of this frame as the samples
// of the move event e; the last one is put on e's own point. The system
// gives them in screen pixels.
func (w *Window) dragSamples(native []pen.Sample, e Event) []Sample {
	var out []Sample
	for _, s := range native {
		if s.Phase != pen.PhaseDrag {
			continue
		}
		at := w.at(Event{}, s.X, s.Y)
		out = append(out, Sample{X: at.X, Y: at.Y, Spot: at.Spot, Pressure: pen.Shape(s.Source, s.Pressure), Raw: s.Pressure, T: s.T})
	}
	if n := len(out); n > 0 {
		out[n-1].X, out[n-1].Y, out[n-1].Spot = e.X, e.Y, e.Spot
	}
	return out
}

// mouse is the pen of a plain mouse.
var mouse = Pen{Pressure: 1, Raw: 1, Source: "mouse"}

// eventTypes are the types window.inject takes.
var eventTypes = []string{"close", "down", "drop", "enter", "focus", "key", "leave", "move", "resize", "scroll", "type", "up"}

// window.inject(w, event) adds an event, a hash like the ones tick gets, to
// the next frame, as if the user made it: how a program tests itself with
// a headless window. Keys left out take a plain value: the left button, no
// mods, a mouse. A pointer event needs a pos; its surface is worked out.
func inject(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	h, err := arg.Hash(args, 1, "event")
	if err != nil {
		return err
	}
	e, err := w.eventOf(h)
	if err != nil {
		return err
	}
	if e.Type == "resize" {
		w.setSize(e.W, e.H)
		return arg.Null
	}
	w.queued = append(w.queued, e)
	return arg.Null
}

// eventOf reads an event hash for window.inject.
func (w *Window) eventOf(h *object.Hash) (Event, *object.Error) {
	r := reader{h: h}
	e := Event{Type: r.str("type", ""), Button: "left", Pen: mouse, T: pen.Now()}
	if !contains(eventTypes, e.Type) {
		return e, arg.Errorf("an event's type must be one of %s, got %s", strings.Join(eventTypes, ", "), r.show("type"))
	}
	if e.pointer() {
		x, y, ok := arg.XY(r.get("pos"))
		if !ok {
			return e, arg.Errorf("a %s event needs a point pos, got %s", e.Type, r.show("pos"))
		}
		e = w.at(e, float64(x)*float64(w.scale), float64(y)*float64(w.scale))
	}
	e.Button = r.str("button", e.Button)
	e.Key = r.str("key", "")
	e.Text = r.str("text", "")
	e.DY = r.num("dy", 0)
	e.Focused = r.flag("focused", true)
	if e.Type == "drop" {
		list, ok := r.get("paths").(*object.Array)
		if !ok {
			return e, arg.Errorf("a drop event needs a list of paths, got %s", r.show("paths"))
		}
		for _, el := range list.Elements {
			s, ok := el.(*object.String)
			if !ok {
				return e, arg.Errorf("a drop event's paths are strings, got %s", el.Inspect())
			}
			e.Paths = append(e.Paths, absPath(s.Value))
		}
	}
	if e.Type == "resize" {
		width, height, ok := arg.XY(r.get("size"))
		if !ok {
			return e, arg.Errorf("a resize event needs a point size, got %s", r.show("size"))
		}
		e.W, e.H = int(width), int(height)
	}
	if mods, ok := r.get("mods").(*object.Hash); ok {
		m := reader{h: mods}
		e.Mods = Mods{Shift: m.flag("shift", false), Alt: m.flag("alt", false), Cmd: m.flag("cmd", false), Ctrl: m.flag("ctrl", false)}
	}
	if p, ok := r.get("pen").(*object.Hash); ok {
		pr := reader{h: p}
		raw := math.Max(0, math.Min(1, pr.num("raw", pr.num("pressure", 1))))
		e.Pen = Pen{Raw: raw, Pressure: pr.num("pressure", raw), Source: pr.str("source", "mouse"), Eraser: pr.flag("eraser", false)}
	}
	return e, nil
}

// at is e at the screen point (sx, sy): its buffer pixel and its surface.
func (w *Window) at(e Event, sx, sy float64) Event {
	e.X, e.Y = int(math.Floor(sx/float64(w.scale))), int(math.Floor(sy/float64(w.scale)))
	e.Spot = w.spotAt(sx, sy)
	return e
}

// reader reads the keys of a hash, with a value for the ones left out.
type reader struct{ h *object.Hash }

func (r reader) get(key string) object.Object {
	v, _ := r.h.Get(&object.String{Value: key})
	return v
}

func (r reader) show(key string) string {
	if v := r.get(key); v != nil {
		return v.Inspect()
	}
	return "nothing"
}

func (r reader) str(key, otherwise string) string {
	if s, ok := r.get(key).(*object.String); ok {
		return s.Value
	}
	return otherwise
}

func (r reader) num(key string, otherwise float64) float64 {
	if n, ok := arg.Number(r.get(key)); ok {
		return n
	}
	return otherwise
}

func (r reader) flag(key string, otherwise bool) bool {
	if b, ok := r.get(key).(*object.Boolean); ok {
		return b.Value
	}
	return otherwise
}

// absPath is p made absolute, as a dropped file's path is; a path that can't
// be (no working folder) stays as it is.
func absPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
