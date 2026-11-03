package vm

import (
	"lizardscript/code"
	"lizardscript/object"
)

// Point arithmetic (p + q, p - q, p * n, n * p, p / n), the evaluator's
// rules for each coordinate, with one allocation per point: the point, its
// coordinates and their floats share one block of memory. Points and floats
// can't change, so sharing it is invisible to a program. Point literals get
// one block too.

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
func newPoint(coords []number) *object.Point {
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
func fillCoords(dst []object.Object, floats []object.Float, coords []number) []object.Object {
	for i, c := range coords {
		if c.isFloat {
			floats[i].Value = c.f
			dst[i] = &floats[i]
		} else {
			dst[i] = integer(c.i)
		}
	}
	return dst[:len(coords):len(coords)]
}

// literal2 and literal3 are a point and its coordinates, for a point
// literal, whose coordinates are already values.
type literal2 struct {
	point  object.Point
	coords [2]object.Object
}

type literal3 struct {
	point  object.Point
	coords [3]object.Object
}

// pointOf makes the point of a literal from values already checked to be
// numbers, two or three.
func pointOf(values []object.Object) *object.Point {
	if len(values) == 2 {
		block := &literal2{coords: [2]object.Object{values[0], values[1]}}
		block.point.Coords = block.coords[:]
		return &block.point
	}
	block := &literal3{}
	n := copy(block.coords[:], values)
	block.point.Coords = block.coords[:n:n]
	return &block.point
}

// pointInfix is an operator with a point on one side. ok is false where the
// evaluator must answer: an error, or an operator points don't have.
func pointInfix(op code.Opcode, left, right object.Object) (object.Object, bool) {
	lp, leftIsPoint := left.(*object.Point)
	rp, rightIsPoint := right.(*object.Point)
	switch {
	case leftIsPoint && rightIsPoint:
		if (op != code.OpAdd && op != code.OpSub) || len(lp.Coords) != len(rp.Coords) {
			return nil, false
		}
		return addPoints(op, lp.Coords, rp.Coords)
	case leftIsPoint && (op == code.OpMul || op == code.OpDiv):
		scale, ok := toNumber(right)
		if !ok {
			return nil, false
		}
		return scalePoint(op, lp.Coords, scale, false)
	case rightIsPoint && op == code.OpMul:
		scale, ok := toNumber(left)
		if !ok {
			return nil, false
		}
		return scalePoint(op, rp.Coords, scale, true)
	}
	return nil, false
}

// addPoints is p + q or p - q, coordinate by coordinate.
func addPoints(op code.Opcode, p, q []object.Object) (object.Object, bool) {
	var coords [3]number
	if len(p) > len(coords) {
		return nil, false
	}
	for i := range p {
		a, aok := toNumber(p[i])
		b, bok := toNumber(q[i])
		if !aok || !bok {
			return nil, false
		}
		c, ok := arithmetic(op, a, b)
		if !ok {
			return nil, false
		}
		coords[i] = c
	}
	return newPoint(coords[:len(p)]), true
}

// scalePoint is p * n or p / n, or n * p when first is true.
func scalePoint(op code.Opcode, p []object.Object, scale number, first bool) (object.Object, bool) {
	var coords [3]number
	if len(p) > len(coords) {
		return nil, false
	}
	for i := range p {
		a, ok := toNumber(p[i])
		if !ok {
			return nil, false
		}
		var c number
		if first {
			c, ok = arithmetic(op, scale, a)
		} else {
			c, ok = arithmetic(op, a, scale)
		}
		if !ok {
			return nil, false
		}
		coords[i] = c
	}
	return newPoint(coords[:len(p)]), true
}
