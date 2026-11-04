package canvas

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/pixels"
	"image/color"
)

// triangles(target, list, options = {}) fills every triangle of list, the
// pixel step of 3D drawing. Each triangle is
//
//	[a, b, c, color]                 one color
//	[a, color_a, b, color_b, c, color_c]  colors mixed from the corners
//
// with the corners points (x, y, z): x and y in pixels, z the depth,
// smaller nearer ((x, y) is depth 0). The target keeps a depth buffer, so
// a pixel is only painted when it is nearer than what was painted there
// before. The options:
//
//	"depth": false        paint in order, with no depth test
//	"clear_depth": true   make the depth buffer empty first
func triangles(args ...object.Object) object.Object {
	t, err := startOr(args, 2, 3)
	if err != nil {
		return err
	}
	list, err := arg.Array(args, 1, "list of triangles")
	if err != nil {
		return err
	}
	depth, clearDepth := true, false
	if len(args) == 3 {
		if depth, clearDepth, err = triangleOptions(args); err != nil {
			return err
		}
	}
	m := t.Pixels()
	defer t.Touched()
	if clearDepth {
		m.ClearDepth()
	}
	for i, item := range list.Elements {
		v, ok := triangleOf(item)
		if !ok {
			return arg.Errorf("triangle %d must be [a, b, c, color] or [a, color, b, color, c, color] with points (x, y, z), got %s", i, item.Inspect())
		}
		m.Triangle(v, depth)
	}
	return arg.Null
}

// triangleOptions reads the options hash of triangles: depth and
// clear_depth, true or false.
func triangleOptions(args []object.Object) (depth, clearDepth bool, err *object.Error) {
	options, err := arg.Hash(args, 2, "options")
	if err != nil {
		return false, false, err
	}
	depth = true
	options.Each(func(pair object.HashPair) {
		key := pair.Key.Inspect()
		value, isBool := pair.Value.(*object.Boolean)
		switch {
		case err != nil:
		case key != "depth" && key != "clear_depth":
			err = arg.Errorf("unknown option %s (the options are clear_depth, depth)", object.Quote(key))
		case !isBool:
			err = arg.Errorf("the option %s must be true or false, got %s", key, arg.Name(pair.Value))
		case key == "depth":
			depth = value.Value
		default:
			clearDepth = value.Value
		}
	})
	return depth, clearDepth, err
}

// triangleOf reads one triangle, in either of its forms.
func triangleOf(item object.Object) ([3]pixels.Vertex, bool) {
	var v [3]pixels.Vertex
	parts, ok := item.(*object.Array)
	if !ok {
		return v, false
	}
	e := parts.Elements
	switch len(e) {
	case 4:
		c, okC := colorOf(e[3])
		for i := range v {
			if v[i], ok = vertexOf(e[i], c); !ok {
				return v, false
			}
		}
		return v, okC
	case 6:
		for i := range v {
			c, okC := colorOf(e[2*i+1])
			if v[i], ok = vertexOf(e[2*i], c); !ok || !okC {
				return v, false
			}
		}
		return v, true
	}
	return v, false
}

// vertexOf reads a corner, a point (x, y, z) or (x, y), with its color.
func vertexOf(o object.Object, c color.NRGBA) (pixels.Vertex, bool) {
	x, y, ok := arg.XY(o)
	if !ok {
		return pixels.Vertex{}, false
	}
	v := pixels.Vertex{X: x, Y: y, C: c}
	if p := o.(*object.Point); len(p.Coords) == 3 {
		if v.Z, ok = arg.Number(p.Coords[2]); !ok {
			return v, false
		}
	}
	return v, true
}

// colorOf reads a color.
func colorOf(o object.Object) (color.NRGBA, bool) {
	c, ok := o.(*object.Color)
	if !ok {
		return color.NRGBA{}, false
	}
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}, true
}
