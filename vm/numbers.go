package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/token"
	"math"
)

// The operators on numbers, computed on the values' own bits, so nothing is
// boxed. Each gives exactly the evaluator's value; where the evaluator would
// report an error (a division by zero, a negative exponent, a result that is
// not a number) ok is false, and the evaluator does it, with its message.

// infix applies an infix opcode to two values: here for numbers, points and
// the common string cases, by the evaluator for the rest and for every
// error.
func infix(op code.Opcode, tok token.Token, left, right object.Value) (object.Value, *object.Error) {
	if left.IsNumber() && right.IsNumber() {
		if result, ok := numberInfix(op, left, right); ok {
			return result, nil
		}
	} else if result, ok := objectInfix(op, left, right); ok {
		return result, nil
	}
	result := evaluator.Infix(tok, operators[op], left.Object(), right.Object())
	if err, ok := result.(*object.Error); ok {
		return object.Value{}, err
	}
	return object.ValueOf(result), nil
}

// numberInfix is an infix opcode on two numbers: integers stay integers,
// and a float on either side makes a float.
func numberInfix(op code.Opcode, left, right object.Value) (object.Value, bool) {
	if left.IsInt() && right.IsInt() {
		return integerInfix(op, left.Int(), right.Int())
	}
	switch op {
	case code.OpExact:
		// === never takes an integer for a float.
		return boolValue(left.Kind() == right.Kind() && left.Float() == right.Float()), true
	case code.OpNotExact:
		return boolValue(left.Kind() != right.Kind() || left.Float() != right.Float()), true
	}
	return floatInfix(op, left.Number(), right.Number())
}

// integerInfix is an infix opcode on two integers.
func integerInfix(op code.Opcode, a, b int64) (object.Value, bool) {
	switch op {
	case code.OpAdd:
		return object.IntValue(a + b), true
	case code.OpSub:
		return object.IntValue(a - b), true
	case code.OpMul:
		return object.IntValue(a * b), true
	case code.OpDiv:
		if b == 0 {
			return object.Value{}, false
		}
		return object.IntValue(a / b), true
	case code.OpFloorDiv:
		if b == 0 {
			return object.Value{}, false
		}
		return object.IntValue(floorDiv(a, b)), true
	case code.OpMod:
		if b == 0 {
			return object.Value{}, false
		}
		return object.IntValue(a - floorDiv(a, b)*b), true
	case code.OpPow:
		if b < 0 {
			return object.Value{}, false
		}
		return object.IntValue(power(a, b)), true
	case code.OpEqual, code.OpExact:
		return boolValue(a == b), true
	case code.OpNotEqual, code.OpNotExact:
		return boolValue(a != b), true
	case code.OpLess:
		return boolValue(a < b), true
	case code.OpGreater:
		return boolValue(a > b), true
	case code.OpLessEqual:
		return boolValue(a <= b), true
	case code.OpGreaterEqual:
		return boolValue(a >= b), true
	}
	return object.Value{}, false
}

// floatInfix is an infix opcode on two numbers, at least one a float, both
// as floats. == and != compare them that way too (1 == 1.0).
func floatInfix(op code.Opcode, a, b float64) (object.Value, bool) {
	var result float64
	switch op {
	case code.OpAdd:
		result = a + b
	case code.OpSub:
		result = a - b
	case code.OpMul:
		result = a * b
	case code.OpDiv:
		result = a / b
	case code.OpFloorDiv:
		result = math.Floor(a / b)
	case code.OpMod:
		if b == 0 {
			return object.Value{}, false
		}
		result = floatRemainder(a, b)
	case code.OpPow:
		result = math.Pow(a, b)
	case code.OpEqual:
		return boolValue(a == b), true
	case code.OpNotEqual:
		return boolValue(a != b), true
	case code.OpLess:
		return boolValue(a < b), true
	case code.OpGreater:
		return boolValue(a > b), true
	case code.OpLessEqual:
		return boolValue(a <= b), true
	case code.OpGreaterEqual:
		return boolValue(a >= b), true
	default:
		return object.Value{}, false
	}
	// A division by zero gives an infinity or NaN, and infinity minus itself
	// is NaN, so this catches every result that is not a number.
	if result-result != 0 {
		return object.Value{}, false
	}
	return object.FloatValue(result), true
}

// floorDiv divides rounding down, so -7 // 2 is -4, like the evaluator.
func floorDiv(a, b int64) int64 {
	quotient := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		quotient--
	}
	return quotient
}

// power is base ** exponent by squaring, wrapping around like the
// evaluator's.
func power(base, exponent int64) int64 {
	result := int64(1)
	for exponent > 0 {
		if exponent&1 == 1 {
			result *= base
		}
		base *= base
		exponent >>= 1
	}
	return result
}

// floatRemainder is the remainder that goes with //, with the sign of the
// right side, like the evaluator's.
func floatRemainder(a, b float64) float64 {
	remainder := math.Mod(a, b)
	if remainder != 0 && (remainder < 0) != (b < 0) {
		remainder += b
	}
	return remainder
}

// objectInfix is an infix opcode with an object on at least one side, for
// the cases that need no evaluator: points, and strings compared or
// joined. ok is false for everything else.
func objectInfix(op code.Opcode, left, right object.Value) (object.Value, bool) {
	if ls, ok := left.Ref().(*object.String); ok {
		if rs, ok := right.Ref().(*object.String); ok {
			return stringInfix(op, ls, rs)
		}
		return object.Value{}, false
	}
	return pointInfix(op, left, right)
}

// stringInfix is an infix opcode on two strings: the comparisons, and +
// (see object.Concat).
func stringInfix(op code.Opcode, a, b *object.String) (object.Value, bool) {
	switch op {
	case code.OpEqual, code.OpExact:
		return boolValue(a.Value == b.Value), true
	case code.OpNotEqual, code.OpNotExact:
		return boolValue(a.Value != b.Value), true
	case code.OpLess:
		return boolValue(a.Value < b.Value), true
	case code.OpGreater:
		return boolValue(a.Value > b.Value), true
	case code.OpLessEqual:
		return boolValue(a.Value <= b.Value), true
	case code.OpGreaterEqual:
		return boolValue(a.Value >= b.Value), true
	}
	return object.Value{}, false
}

// minus is -v for a number, false for anything else.
func minus(v object.Value) (object.Value, bool) {
	switch v.Kind() {
	case object.KindInt:
		return object.IntValue(-v.Int()), true
	case object.KindFloat:
		return object.FloatValue(-v.Float()), true
	}
	return object.Value{}, false
}
