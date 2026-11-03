package vm

import (
	"lizardscript/code"
	"lizardscript/object"
)

// Point arithmetic (p + q, p - q, p * n, n * p, p / n), the evaluator's
// rules for each coordinate, with one allocation per point: the point, its
// coordinates and their floats share one block of memory. Points and floats
// can't change, so sharing it is invisible to a program. A point literal
// gets one block too.

// point2 and point3 are a point with room for its coordinates and their
// floats, for two and three coordinates.
type point2 struct {
	point  object.Point
	coords [2]object.Object
	floats [2]object.Float
}

type point3 struct {
	point  object.Point
	coords [3]object.Object
	floats [3]object.Float
}

// newPoint makes the point of coords, two or three numbers.
func newPoint(coords []object.Value) *object.Point {
	if len(coords) == 2 {
		block := &point2{}
		block.point.Coords = fillCoords(block.coords[:], block.floats[:], coords)
		return &block.point
	}
	block := &point3{}
	block.point.Coords = fillCoords(block.coords[:], block.floats[:], coords)
	return &block.point
}

// fillCoords writes coords into a block's coordinates, the floats in the
// block's own floats, and returns the coordinates.
func fillCoords(dst []object.Object, floats []object.Float, coords []object.Value) []object.Object {
	for i, c := range coords {
		if c.IsFloat() {
			floats[i].Value = c.Float()
			dst[i] = &floats[i]
		} else {
			dst[i] = object.IntegerOf(c.Int())
		}
	}
	return dst[:len(coords):len(coords)]
}

// pointInfix is an operator with a point on one side. ok is false where the
// evaluator must answer: an error, or an operator points don't have.
func pointInfix(op code.Opcode, left, right object.Value) (object.Value, bool) {
	lp, leftIsPoint := left.Ref().(*object.Point)
	rp, rightIsPoint := right.Ref().(*object.Point)
	var point *object.Point
	var ok bool
	switch {
	case leftIsPoint && rightIsPoint:
		if (op != code.OpAdd && op != code.OpSub) || len(lp.Coords) != len(rp.Coords) {
			return object.Value{}, false
		}
		point, ok = addPoints(op, lp.Coords, rp.Coords)
	case leftIsPoint && right.IsNumber() && (op == code.OpMul || op == code.OpDiv):
		point, ok = scalePoint(op, lp.Coords, right, false)
	case rightIsPoint && left.IsNumber() && op == code.OpMul:
		point, ok = scalePoint(op, rp.Coords, left, true)
	}
	if !ok {
		return object.Value{}, false
	}
	return object.ObjectValue(point), true
}

// addPoints is p + q or p - q, coordinate by coordinate.
func addPoints(op code.Opcode, p, q []object.Object) (*object.Point, bool) {
	var coords [3]object.Value
	if len(p) > len(coords) {
		return nil, false
	}
	for i := range p {
		a, b := object.ValueOf(p[i]), object.ValueOf(q[i])
		if !a.IsNumber() || !b.IsNumber() {
			return nil, false
		}
		c, ok := numberInfix(op, a, b)
		if !ok {
			return nil, false
		}
		coords[i] = c
	}
	return newPoint(coords[:len(p)]), true
}

// scalePoint is p * n or p / n, or n * p when first is true.
func scalePoint(op code.Opcode, p []object.Object, scale object.Value, first bool) (*object.Point, bool) {
	var coords [3]object.Value
	if len(p) > len(coords) {
		return nil, false
	}
	for i := range p {
		coord := object.ValueOf(p[i])
		if !coord.IsNumber() {
			return nil, false
		}
		var c object.Value
		var ok bool
		if first {
			c, ok = numberInfix(op, scale, coord)
		} else {
			c, ok = numberInfix(op, coord, scale)
		}
		if !ok {
			return nil, false
		}
		coords[i] = c
	}
	return newPoint(coords[:len(p)]), true
}
