package evaluator

import (
	"lizardscript/ast"
	"lizardscript/object"
	"lizardscript/token"
	"math"
)

// Colors and points. Both are read-only values: their parts are read with
// dot access (c.r, p.x) and they are replaced, never changed.

// pointNames are the names of the coordinates, in order.
var pointNames = []string{"x", "y", "z"}

// evalPointLiteral makes the point of (x, y) or (x, y, z). Each coordinate
// must be a number.
func evalPointLiteral(node *ast.PointLiteral, env *object.Environment) object.Object {
	coords := make([]object.Object, 0, len(node.Coords))
	for _, expression := range node.Coords {
		value := Eval(expression, env)
		if isError(value) {
			return value
		}
		if !isNumber(value) {
			return newError(node.Token, "a point coordinate must be INTEGER or FLOAT, got %s", value.Type())
		}
		coords = append(coords, value)
	}
	return &object.Point{Coords: coords}
}

// evalColorProperty reads c.r, c.g, c.b or c.a.
func evalColorProperty(tok token.Token, color *object.Color, index object.Object) object.Object {
	name, ok := index.(*object.String)
	if !ok {
		return newError(tok, "index operator not supported: COLOR")
	}
	switch name.Value {
	case "r":
		return &object.Integer{Value: int64(color.R)}
	case "g":
		return &object.Integer{Value: int64(color.G)}
	case "b":
		return &object.Integer{Value: int64(color.B)}
	case "a":
		return &object.Integer{Value: int64(color.A)}
	default:
		return newError(tok, "a COLOR has no %q, use r, g, b or a", name.Value)
	}
}

// evalPointProperty reads p.x, p.y, or p.z of a point with three
// coordinates.
func evalPointProperty(tok token.Token, point *object.Point, index object.Object) object.Object {
	name, ok := index.(*object.String)
	if !ok {
		return newError(tok, "index operator not supported: POINT")
	}
	for i, coordName := range pointNames {
		if name.Value == coordName && i < len(point.Coords) {
			return point.Coords[i]
		}
	}
	return newError(tok, "a point of %d coordinates has no %q", len(point.Coords), name.Value)
}

// numberInfix applies a number operator to two numbers (integer or float)
// with the same rules as the operator between them: integers stay integers,
// and a float on either side makes a float.
func numberInfix(tok token.Token, operator string, left, right object.Object) object.Object {
	li, lok := left.(*object.Integer)
	ri, rok := right.(*object.Integer)
	if lok && rok {
		return evalIntegerInfixExpression(tok, operator, li.Value, ri.Value)
	}
	return evalFloatInfixExpression(tok, operator, toFloat(left), toFloat(right))
}

// mapPoint makes a point out of f applied to every coordinate. It stops at
// the first error.
func mapPoint(point *object.Point, f func(object.Object) object.Object) object.Object {
	coords := make([]object.Object, len(point.Coords))
	for i, c := range point.Coords {
		result := f(c)
		if isError(result) {
			return result
		}
		coords[i] = result
	}
	return &object.Point{Coords: coords}
}

// evalPointInfix handles the operators with a point: p + q, p - q, p * n,
// n * p and p / n. ok is false for anything else, which then fails like any
// other mismatched types.
func evalPointInfix(tok token.Token, operator string, left, right object.Object) (object.Object, bool) {
	leftPoint, leftIsPoint := left.(*object.Point)
	rightPoint, rightIsPoint := right.(*object.Point)

	switch {
	case leftIsPoint && rightIsPoint && (operator == "+" || operator == "-"):
		if len(leftPoint.Coords) != len(rightPoint.Coords) {
			return newError(tok, "can't mix points of %d and %d coordinates: POINT %s POINT",
				len(leftPoint.Coords), len(rightPoint.Coords), operator), true
		}
		coords := make([]object.Object, len(leftPoint.Coords))
		for i := range coords {
			result := numberInfix(tok, operator, leftPoint.Coords[i], rightPoint.Coords[i])
			if isError(result) {
				return result, true
			}
			coords[i] = result
		}
		return &object.Point{Coords: coords}, true

	case leftIsPoint && isNumber(right) && (operator == "*" || operator == "/"):
		return mapPoint(leftPoint, func(c object.Object) object.Object {
			return numberInfix(tok, operator, c, right)
		}), true

	case rightIsPoint && isNumber(left) && operator == "*":
		return mapPoint(rightPoint, func(c object.Object) object.Object {
			return numberInfix(tok, operator, left, c)
		}), true
	}
	return nil, false
}

// negatePoint is -p: every coordinate changes sign.
func negatePoint(point *object.Point) object.Object {
	return mapPoint(point, func(c object.Object) object.Object {
		if float, ok := c.(*object.Float); ok {
			return &object.Float{Value: -float.Value}
		}
		return &object.Integer{Value: -c.(*object.Integer).Value}
	})
}

// channel turns a number into a color channel: rounded, and from 0 to 255.
// arg is the number as the program wrote it, so an error shows 300 for an
// integer and 300.5 for a float.
func channel(name string, arg object.Object) (uint8, *object.Error) {
	rounded := math.Round(toFloat(arg))
	if rounded < 0 || rounded > 255 {
		return 0, argError("%s must be from 0 to 255, got %s", name, arg.Inspect())
	}
	return uint8(rounded), nil
}

// rgb(r, g, b) and rgba(r, g, b, a) are colors from channels of 0 to 255.
// Floats are rounded to the nearest integer.
func builtinRGB(args ...object.Object) object.Object {
	return colorFromChannels(args, 3)
}

func builtinRGBA(args ...object.Object) object.Object {
	return colorFromChannels(args, 4)
}

func colorFromChannels(args []object.Object, count int) object.Object {
	if _, err := numberArguments(args, count); err != nil {
		return err
	}
	names := []string{"red", "green", "blue", "alpha"}
	channels := []uint8{0, 0, 0, 255}
	for i := range args {
		value, err := channel(names[i], args[i])
		if err != nil {
			return err
		}
		channels[i] = value
	}
	return &object.Color{R: channels[0], G: channels[1], B: channels[2], A: channels[3]}
}

// hsl(h, s, l) is a color from a hue in degrees (0 to 360) and a saturation
// and lightness in percent (0 to 100).
func builtinHSL(args ...object.Object) object.Object {
	numbers, err := numberArguments(args, 3)
	if err != nil {
		return err
	}
	hue, saturation, lightness := numbers[0], numbers[1], numbers[2]
	if hue < 0 || hue > 360 {
		return argError("hue must be from 0 to 360, got %s", args[0].Inspect())
	}
	if saturation < 0 || saturation > 100 {
		return argError("saturation must be from 0 to 100, got %s", args[1].Inspect())
	}
	if lightness < 0 || lightness > 100 {
		return argError("lightness must be from 0 to 100, got %s", args[2].Inspect())
	}

	s, l := saturation/100, lightness/100
	chroma := (1 - math.Abs(2*l-1)) * s
	h := hue / 60
	x := chroma * (1 - math.Abs(math.Mod(h, 2)-1))
	var r, g, b float64
	switch {
	case h < 1:
		r, g, b = chroma, x, 0
	case h < 2:
		r, g, b = x, chroma, 0
	case h < 3:
		r, g, b = 0, chroma, x
	case h < 4:
		r, g, b = 0, x, chroma
	case h < 5:
		r, g, b = x, 0, chroma
	default:
		r, g, b = chroma, 0, x
	}
	m := l - chroma/2
	return &object.Color{
		R: uint8(math.Round((r + m) * 255)),
		G: uint8(math.Round((g + m) * 255)),
		B: uint8(math.Round((b + m) * 255)),
		A: 255,
	}
}

// point(x, y) and point(x, y, z) make a point, the same as (x, y).
func builtinPoint(args ...object.Object) object.Object {
	if len(args) != 2 && len(args) != 3 {
		return wrongCount("2 or 3", len(args))
	}
	if _, err := numberArguments(args, len(args)); err != nil {
		return err
	}
	return &object.Point{Coords: append([]object.Object{}, args...)}
}

// pointArguments checks that a builtin got two points with the same number
// of coordinates.
func pointArguments(args []object.Object) (*object.Point, *object.Point, *object.Error) {
	if len(args) != 2 {
		return nil, nil, wrongCount("2", len(args))
	}
	p, pok := args[0].(*object.Point)
	q, qok := args[1].(*object.Point)
	if !pok || !qok {
		return nil, nil, argError("arguments must be POINT, got %s and %s", args[0].Type(), args[1].Type())
	}
	if len(p.Coords) != len(q.Coords) {
		return nil, nil, argError("can't mix points of %d and %d coordinates", len(p.Coords), len(q.Coords))
	}
	return p, q, nil
}

// math.dist(p, q) is the distance between two points.
func mathDist(args ...object.Object) object.Object {
	p, q, err := pointArguments(args)
	if err != nil {
		return err
	}
	distance := 0.0
	for i := range p.Coords {
		distance = math.Hypot(distance, toFloat(q.Coords[i])-toFloat(p.Coords[i]))
	}
	return finite(distance)
}

// math.angle(p, q) is the angle in radians of the line from p to q, from
// the positive x direction, like atan2. It is for points of two
// coordinates.
func mathAngle(args ...object.Object) object.Object {
	p, q, err := pointArguments(args)
	if err != nil {
		return err
	}
	if len(p.Coords) != 2 {
		return argError("angle is for points of two coordinates, got %d", len(p.Coords))
	}
	return finite(math.Atan2(toFloat(q.Coords[1])-toFloat(p.Coords[1]),
		toFloat(q.Coords[0])-toFloat(p.Coords[0])))
}

// pointInfixError is the error of an operator with a point that evalPointInfix
// doesn't handle, worded like the same mistake with any other types.
func pointInfixError(tok token.Token, operator string, left, right object.Object) object.Object {
	if left.Type() != right.Type() {
		return newError(tok, "type mismatch: %s %s %s", left.Type(), operator, right.Type())
	}
	return newError(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
}
