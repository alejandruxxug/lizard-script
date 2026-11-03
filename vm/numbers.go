package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
	"math"
)

// Fast paths for the operators on floats and points, next to the integer
// ones in values.go. Each gives exactly the evaluator's value; where the
// evaluator would report an error (division by zero, a result that is not
// a number, points of different sizes) or the case is rare (//, %, **),
// ok is false and the evaluator does it.

// number is an integer or a float without its object, so a computation
// allocates only for its result.
type number struct {
	isFloat bool
	i       int64
	f       float64
}

func toNumber(obj object.Object) (number, bool) {
	switch obj := obj.(type) {
	case *object.Integer:
		return number{i: obj.Value}, true
	case *object.Float:
		return number{isFloat: true, f: obj.Value}, true
	}
	return number{}, false
}

func (n number) float() float64 {
	if n.isFloat {
		return n.f
	}
	return float64(n.i)
}

// object makes the value of n, shared when it is a small whole number.
func (n number) object() object.Object {
	if n.isFloat {
		return object.FloatOf(n.f)
	}
	return integer(n.i)
}

// arithmetic is + - * or / on two numbers, with the evaluator's rules:
// integers stay integers, and a float on either side makes a float.
func arithmetic(op code.Opcode, left, right number) (number, bool) {
	if !left.isFloat && !right.isFloat {
		a, b := left.i, right.i
		switch op {
		case code.OpAdd:
			return number{i: a + b}, true
		case code.OpSub:
			return number{i: a - b}, true
		case code.OpMul:
			return number{i: a * b}, true
		case code.OpDiv:
			if b != 0 {
				return number{i: a / b}, true
			}
		}
		return number{}, false
	}
	a, b := left.float(), right.float()
	var result float64
	switch op {
	case code.OpAdd:
		result = a + b
	case code.OpSub:
		result = a - b
	case code.OpMul:
		result = a * b
	case code.OpDiv:
		if b == 0 {
			return number{}, false
		}
		result = a / b
	default:
		return number{}, false
	}
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return number{}, false
	}
	return number{isFloat: true, f: result}, true
}

// compare is < > <= or >= on two numbers, at least one a float.
func compare(op code.Opcode, a, b float64) (bool, bool) {
	switch op {
	case code.OpLess:
		return a < b, true
	case code.OpGreater:
		return a > b, true
	case code.OpLessEqual:
		return a <= b, true
	case code.OpGreaterEqual:
		return a >= b, true
	}
	return false, false
}

// fastInfix is an infix opcode on anything but two integers: floats,
// mixed numbers and points.
func fastInfix(op code.Opcode, left, right object.Object) (object.Object, bool) {
	l, lok := toNumber(left)
	r, rok := toNumber(right)
	if !lok || !rok {
		return pointInfix(op, left, right)
	}
	if result, ok := arithmetic(op, l, r); ok {
		return result.object(), true
	}
	if result, ok := compare(op, l.float(), r.float()); ok {
		return evaluator.NativeBool(result), true
	}
	return nil, false
}
