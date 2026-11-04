package canvas

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/pixels"
	"image/color"
	"math"
	"strings"
)

// The drawing functions that work pixel by pixel: the clip, gradient spans
// and blend layers.
//
//	clip(target, rect)                 drawing is cut to rect until unclip
//	unclip(target)                     drawing reaches the whole target again
//	span_gradient(target, y, x0, x1, gradient)
//	spans_gradient(target, list, gradient, offset)
//	push_layer(target)                 drawing goes to a new transparent layer
//	pop_layer(target, mode, opacity)   mixes it onto what is below
//	composite(target, source, mode, opacity, pos)

// clip(target, rect) cuts every drawing on target to the rectangle rect,
// [x, y, w, h], until unclip. A new clip replaces the old one.
func clip(args ...object.Object) object.Object {
	t, err := start(args, 2)
	if err != nil {
		return err
	}
	r, err := RectArg(args, 1)
	if err != nil {
		return err
	}
	t.Pixels().SetClip(r)
	return arg.Null
}

// unclip(target) lets drawing reach the whole target again.
func unclip(args ...object.Object) object.Object {
	t, err := start(args, 1)
	if err != nil {
		return err
	}
	t.Pixels().Unclip()
	return arg.Null
}

// spanGradient is span_gradient(target, y, x0, x1, gradient): span, with
// each pixel's color taken from the gradient.
func spanGradient(args ...object.Object) object.Object {
	t, err := start(args, 5)
	if err != nil {
		return err
	}
	var v [3]float64
	for k, what := range []string{"y", "x0", "x1"} {
		if v[k], err = arg.NumberArg(args, k+1, what); err != nil {
			return err
		}
	}
	g, err := GradientArg(args, 4)
	if err != nil {
		return err
	}
	t.Pixels().GradientSpan(int(math.Floor(v[0])), v[1], v[2], g, 0, 0)
	t.Touched()
	return arg.Null
}

// spansGradient is spans_gradient(target, list, gradient, offset = (0, 0)):
// spans, with each pixel's color taken from the gradient. The gradient is
// in the space of the list, before the offset, so a moved shape keeps it.
func spansGradient(args ...object.Object) object.Object {
	t, err := startOr(args, 3, 4)
	if err != nil {
		return err
	}
	list, err := arg.Array(args, 1, "list of spans")
	if err != nil {
		return err
	}
	g, err := GradientArg(args, 2)
	if err != nil {
		return err
	}
	dx, dy, err := offsetArg(args, 3)
	if err != nil {
		return err
	}
	m := t.Pixels()
	defer t.Touched()
	for i, item := range list.Elements {
		y, x0, x1, ok := spanOf(item)
		if !ok {
			return arg.Errorf("span %d must be [y, x0, x1] of numbers, got %s", i, item.Inspect())
		}
		m.GradientSpan(int(math.Floor(y+dy)), x0+dx, x1+dx, g, dx, dy)
	}
	return arg.Null
}

// GradientArg reads argument i as a gradient, a hash:
//
//	{"kind": "linear", "origin": point, "axis": point, "stops": [[t, color], ...], "steps": n}
//	{"kind": "radial", "origin": point, "scale": number, "stops": ..., "steps": n}
//
// A linear gradient's t is (p - origin) · axis, so axis is the direction
// divided by its squared length; a radial one's is the distance from
// origin times scale (1 / radius). The stops go in order of t, two at the
// same t make a hard edge, and steps (optional, 2 or more) cuts the
// gradient into that many flat bands.
func GradientArg(args []object.Object, i int) (*pixels.Gradient, *object.Error) {
	h, err := arg.Hash(args, i, "gradient")
	if err != nil {
		return nil, err
	}
	bad := func(format string, a ...any) (*pixels.Gradient, *object.Error) {
		return nil, arg.Errorf("bad gradient: "+format, a...)
	}
	g := &pixels.Gradient{}
	kind, _ := get(h, "kind").(*object.String)
	if kind == nil || (kind.Value != "linear" && kind.Value != "radial") {
		return bad(`"kind" must be "linear" or "radial"`)
	}
	g.Radial = kind.Value == "radial"
	var ok bool
	if g.OX, g.OY, ok = arg.XY(get(h, "origin")); !ok {
		return bad(`"origin" must be a point`)
	}
	if g.Radial {
		if g.Scale, ok = arg.Number(get(h, "scale")); !ok {
			return bad(`a radial gradient needs a number "scale" (1 / radius)`)
		}
	} else if g.AX, g.AY, ok = arg.XY(get(h, "axis")); !ok {
		return bad(`a linear gradient needs a point "axis"`)
	}
	list, _ := get(h, "stops").(*object.Array)
	if list == nil || len(list.Elements) == 0 {
		return bad(`"stops" must be a list of [t, color], at least one`)
	}
	for k, item := range list.Elements {
		stop, ok := stopOf(item)
		if !ok {
			return bad("stop %d must be [t, color], got %s", k, item.Inspect())
		}
		g.Stops = append(g.Stops, stop)
	}
	if steps, ok := arg.Number(get(h, "steps")); ok && steps >= 2 {
		g.Steps = int(steps)
	}
	return g, nil
}

// stopOf reads one [t, color].
func stopOf(item object.Object) (pixels.Stop, bool) {
	pair, ok := item.(*object.Array)
	if !ok || len(pair.Elements) != 2 {
		return pixels.Stop{}, false
	}
	t, okT := arg.Number(pair.Elements[0])
	c, okC := pair.Elements[1].(*object.Color)
	if !okT || !okC {
		return pixels.Stop{}, false
	}
	return pixels.Stop{T: t, C: color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}}, true
}

// get is h[key] for a string key, or nil.
func get(h *object.Hash, key string) object.Object {
	v, _ := h.Get(&object.String{Value: key})
	return v
}

// pushLayer is push_layer(target): drawing on target goes to a new
// transparent layer, with the clip it had, until pop_layer. Layers nest.
func pushLayer(args ...object.Object) object.Object {
	t, err := start(args, 1)
	if err != nil {
		return err
	}
	m := t.Base()
	if m.Layers() >= maxLayers {
		return arg.Errorf("too many layers open (at most %d): is a pop_layer missing?", maxLayers)
	}
	m.PushLayer()
	return arg.Null
}

// maxLayers is how many layers a target can have open at once.
const maxLayers = 64

// popLayer is pop_layer(target, mode = "normal", opacity = 1): closes the
// newest layer and mixes it onto what is below with the blend mode and the
// opacity (0 to 1).
func popLayer(args ...object.Object) object.Object {
	t, err := startOr(args, 1, 3)
	if err != nil {
		return err
	}
	mode, opacity, err := blendArgs(args, 1)
	if err != nil {
		return err
	}
	if !t.Base().PopLayer(mode, opacity) {
		return arg.Errorf("no layer to pop: call push_layer first")
	}
	t.Touched()
	return arg.Null
}

// composite(target, source, mode = "normal", opacity = 1, pos = (0, 0))
// mixes source onto target with its top left at pos, with a blend mode and
// an opacity: blit with a blend mode.
func composite(args ...object.Object) object.Object {
	t, err := startOr(args, 2, 5)
	if err != nil {
		return err
	}
	src, err := TargetArg(args, 1, "source")
	if err != nil {
		return err
	}
	mode, opacity, err := blendArgs(args, 2)
	if err != nil {
		return err
	}
	x, y, err := offsetArg(args, 4)
	if err != nil {
		return err
	}
	from, to := src.Pixels(), t.Pixels()
	if from == to {
		from = &pixels.Image{W: from.W, H: from.H, Pix: append([]byte(nil), from.Pix...)}
	}
	to.Composite(from, int(math.Floor(x)), int(math.Floor(y)), mode, opacity)
	t.Touched()
	return arg.Null
}

// blendArgs reads the optional mode and opacity starting at argument i.
func blendArgs(args []object.Object, i int) (pixels.Mode, float64, *object.Error) {
	mode, opacity := pixels.Modes["normal"], 1.0
	if len(args) > i {
		name, err := arg.String(args, i, "blend mode")
		if err != nil {
			return nil, 0, err
		}
		if mode = pixels.Modes[name]; mode == nil {
			return nil, 0, arg.Errorf("unknown blend mode %q (the modes are %s)", name, strings.Join(pixels.ModeNames(), ", "))
		}
	}
	if len(args) > i+1 {
		var err *object.Error
		if opacity, err = arg.NumberArg(args, i+1, "opacity"); err != nil {
			return nil, 0, err
		}
		if opacity < 0 || opacity > 1 {
			return nil, 0, arg.Errorf("the opacity must be from 0 to 1, got %s", args[i+1].Inspect())
		}
	}
	return mode, opacity, nil
}
