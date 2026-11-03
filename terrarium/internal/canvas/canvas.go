// Package canvas has the drawing functions every target shares: an image
// (img) and a window (window) are drawn on the same way, so a drawing
// library can take either one.
//
//	size(target)                  (width, height) in pixels
//	clear(target, color, rect)    every pixel (of rect) becomes color
//	fill(target, rect, color)     paints the rectangle [x, y, w, h]
//	span(target, y, x0, x1, color) paints row y from x0 up to x1
//	spans(target, list, color, offset) paints every [y, x0, x1] of list
//	blit(target, source, pos)     paints the target source with its top left at pos
//
// and, in paint.go, clip and unclip, the gradient spans, the blend layers
// and composite.
//
// Painting mixes the color over what is there (alpha counts); clear
// replaces. Ends of spans and sides of rectangles can be fractional: the
// pixel they cut is covered in part, so edges are soft. Nothing reaches
// outside the target's clip.
package canvas

import (
	"image"
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"lizardscript/terrarium/internal/pixels"
	"math"
)

// Add puts the shared drawing builtins into functions, a module's own,
// and returns it.
func Add(functions map[string]object.BuiltinFunction) map[string]object.BuiltinFunction {
	for name, fn := range map[string]object.BuiltinFunction{
		"size":  size,
		"clear": clear,
		"fill":  fill,
		"span":  span,
		"spans": spans,
		"blit":  blit,

		"clip":           clip,
		"unclip":         unclip,
		"span_gradient":  spanGradient,
		"spans_gradient": spansGradient,
		"push_layer":     pushLayer,
		"pop_layer":      popLayer,
		"composite":      composite,
	} {
		functions[name] = fn
	}
	return functions
}

// TargetArg reads argument i as a target: a native whose value is one.
func TargetArg(args []object.Object, i int, what string) (pixels.Target, *object.Error) {
	if n, ok := args[i].(*object.Native); ok {
		if t, ok := n.Value.(pixels.Target); ok {
			return t, nil
		}
	}
	return nil, arg.Errorf("want an image or a window as the %s, got %s", what, arg.Name(args[i]))
}

// start checks the count and reads the target, which is always first.
func start(args []object.Object, count int) (pixels.Target, *object.Error) {
	return startOr(args, count, count)
}

// startOr is start for a function with optional arguments: from least to
// most of them.
func startOr(args []object.Object, least, most int) (pixels.Target, *object.Error) {
	if err := arg.Count(args, least, most); err != nil {
		return nil, err
	}
	return TargetArg(args, 0, "target")
}

// size(target) is (width, height).
func size(args ...object.Object) object.Object {
	t, err := start(args, 1)
	if err != nil {
		return err
	}
	m := t.Pixels()
	return arg.PointOf(m.W, m.H)
}

// clear(target, color, rect = all) replaces every pixel, or every pixel of
// the rectangle rect, with color. A transparent color erases.
func clear(args ...object.Object) object.Object {
	t, err := startOr(args, 2, 3)
	if err != nil {
		return err
	}
	c, err := arg.Color(args, 1, "color")
	if err != nil {
		return err
	}
	m := t.Pixels()
	area := m.Bounds()
	if len(args) == 3 {
		if area, err = RectArg(args, 2); err != nil {
			return err
		}
	}
	m.ClearRect(area, c)
	t.Touched()
	return arg.Null
}

// RectArg reads argument i as a rectangle [x, y, w, h] of whole pixels:
// the sides are rounded out, so the rectangle covers every pixel it
// touches.
func RectArg(args []object.Object, i int) (image.Rectangle, *object.Error) {
	r, err := rectArg(args, i)
	if err != nil {
		return image.Rectangle{}, err
	}
	return image.Rect(int(math.Floor(r[0])), int(math.Floor(r[1])),
		int(math.Ceil(r[0]+r[2])), int(math.Ceil(r[1]+r[3]))), nil
}

// rectArg reads argument i as a rectangle [x, y, w, h] of numbers.
func rectArg(args []object.Object, i int) ([4]float64, *object.Error) {
	var r [4]float64
	list, ok := args[i].(*object.Array)
	if ok && len(list.Elements) == 4 {
		for k, e := range list.Elements {
			if r[k], ok = arg.Number(e); !ok {
				break
			}
		}
	}
	if !ok || len(list.Elements) != 4 {
		return r, arg.Errorf("want a rect [x, y, w, h] of numbers, got %s", args[i].Inspect())
	}
	return r, nil
}

// fill(target, rect, color) paints the rectangle [x, y, w, h].
func fill(args ...object.Object) object.Object {
	t, err := start(args, 3)
	if err != nil {
		return err
	}
	r, err := rectArg(args, 1)
	if err != nil {
		return err
	}
	c, err := arg.Color(args, 2, "color")
	if err != nil {
		return err
	}
	t.Pixels().Fill(r[0], r[1], r[2], r[3], c)
	t.Touched()
	return arg.Null
}

// span(target, y, x0, x1, color) paints row y from x0 up to x1. A
// fractional y is rounded down.
func span(args ...object.Object) object.Object {
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
	c, err := arg.Color(args, 4, "color")
	if err != nil {
		return err
	}
	t.Pixels().Span(int(math.Floor(v[0])), v[1], v[2], c)
	t.Touched()
	return arg.Null
}

// spans(target, list, color, offset = (0, 0)) paints every [y, x0, x1] of
// list in one color, moved by offset: the fast way to fill a shape a
// program has worked out row by row.
func spans(args ...object.Object) object.Object {
	t, err := startOr(args, 3, 4)
	if err != nil {
		return err
	}
	list, err := arg.Array(args, 1, "list of spans")
	if err != nil {
		return err
	}
	c, err := arg.Color(args, 2, "color")
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
		m.Span(int(math.Floor(y+dy)), x0+dx, x1+dx, c)
	}
	return arg.Null
}

// offsetArg reads the optional point argument i, (0, 0) when it isn't
// there.
func offsetArg(args []object.Object, i int) (x, y float64, err *object.Error) {
	if len(args) <= i {
		return 0, 0, nil
	}
	return arg.Point(args, i, "offset")
}

// spanOf reads one [y, x0, x1].
func spanOf(item object.Object) (y, x0, x1 float64, ok bool) {
	run, isArray := item.(*object.Array)
	if !isArray || len(run.Elements) != 3 {
		return 0, 0, 0, false
	}
	y, okY := arg.Number(run.Elements[0])
	x0, ok0 := arg.Number(run.Elements[1])
	x1, ok1 := arg.Number(run.Elements[2])
	return y, x0, x1, okY && ok0 && ok1
}

// blit(target, source, pos) paints the pixels of source, an image or a
// window, over target with source's top left at pos (rounded down to whole
// pixels).
func blit(args ...object.Object) object.Object {
	t, err := start(args, 3)
	if err != nil {
		return err
	}
	src, err := TargetArg(args, 1, "source")
	if err != nil {
		return err
	}
	x, y, err := arg.Point(args, 2, "pos")
	if err != nil {
		return err
	}
	t.Pixels().Blit(src.Pixels(), int(math.Floor(x)), int(math.Floor(y)))
	t.Touched()
	return arg.Null
}
