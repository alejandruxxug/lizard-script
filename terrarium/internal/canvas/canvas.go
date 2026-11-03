// Package canvas has the drawing functions every target shares: an image
// (img) and a window (window) are drawn on the same way, so a drawing
// library can take either one.
//
//	size(target)                  (width, height) in pixels
//	clear(target, color)          every pixel becomes color
//	fill(target, rect, color)     paints the rectangle [x, y, w, h]
//	span(target, y, x0, x1, color) paints row y from x0 up to x1
//	spans(target, list, color)    paints every [y, x0, x1] of list
//	blit(target, source, pos)     paints the target source with its top left at pos
//
// Painting mixes the color over what is there (alpha counts); clear
// replaces. Ends of spans and sides of rectangles can be fractional: the
// pixel they cut is covered in part, so edges are soft.
package canvas

import (
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
	if err := arg.Count(args, count, count); err != nil {
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

// clear(target, color) replaces every pixel with color.
func clear(args ...object.Object) object.Object {
	t, err := start(args, 2)
	if err != nil {
		return err
	}
	c, err := arg.Color(args, 1, "color")
	if err != nil {
		return err
	}
	t.Pixels().Clear(c)
	t.Touched()
	return arg.Null
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

// spans(target, list, color) paints every [y, x0, x1] of list in one color,
// the fast way to fill a shape a program has worked out row by row.
func spans(args ...object.Object) object.Object {
	t, err := start(args, 3)
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
	m := t.Pixels()
	defer t.Touched()
	for i, item := range list.Elements {
		y, x0, x1, ok := spanOf(item)
		if !ok {
			return arg.Errorf("span %d must be [y, x0, x1] of numbers, got %s", i, item.Inspect())
		}
		m.Span(int(math.Floor(y)), x0, x1, c)
	}
	return arg.Null
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
