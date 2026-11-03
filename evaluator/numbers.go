package evaluator

import (
	"lizardscript/object"
	"lizardscript/token"
	"math"
	"strconv"
)

// notANumber is the message of every operation whose result would be
// infinite or not a number, like 1e300 * 1e300 or 0.0 ** -1.
const notANumber = "the result is not a number"

// isNumber reports whether obj is an integer or a float.
func isNumber(obj object.Object) bool {
	t := obj.Type()
	return t == object.INTEGER_OBJ || t == object.FLOAT_OBJ
}

// toFloat returns the value of an integer or float as a float64. The caller
// has checked it with isNumber.
func toFloat(obj object.Object) float64 {
	if integer, ok := obj.(*object.Integer); ok {
		return float64(integer.Value)
	}
	return obj.(*object.Float).Value
}

// newFloat wraps result, or returns the runtime error at tok when it is
// infinite or not a number. That is how floats never hold either of them.
func newFloat(tok token.Token, result float64) object.Object {
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return newError(tok, notANumber)
	}
	return &object.Float{Value: result}
}

// evalFloatInfixExpression handles an operator between two numbers when at
// least one is a float; the other was converted to a float too.
func evalFloatInfixExpression(tok token.Token, operator string, left, right float64) object.Object {
	switch operator {
	case "+":
		return newFloat(tok, left+right)
	case "-":
		return newFloat(tok, left-right)
	case "*":
		return newFloat(tok, left*right)
	case "/", "//", "%":
		if right == 0 {
			return newError(tok, "division by zero")
		}
		switch operator {
		case "/":
			return newFloat(tok, left/right)
		case "//":
			return newFloat(tok, math.Floor(left/right))
		default:
			return newFloat(tok, floatRemainder(left, right))
		}
	case "**":
		return newFloat(tok, math.Pow(left, right))
	case "<":
		return nativeBoolToBooleanObject(left < right)
	case ">":
		return nativeBoolToBooleanObject(left > right)
	case "<=":
		return nativeBoolToBooleanObject(left <= right)
	case ">=":
		return nativeBoolToBooleanObject(left >= right)
	default:
		// No operator gets here today; it is the same message as the integers.
		return newError(tok, "unknown operator: FLOAT %s FLOAT", operator)
	}
}

// floatRemainder is the remainder that goes with //: it has the sign of the
// right side, so a == (a // b) * b + a % b. math.Mod gives the remainder with
// the sign of the left side, which is moved when the signs differ.
func floatRemainder(left, right float64) float64 {
	remainder := math.Mod(left, right)
	if remainder != 0 && (remainder < 0) != (right < 0) {
		remainder += right
	}
	return remainder
}

// floatToInteger converts a float with no decimals to an integer. It fails
// when the value doesn't fit in 64 bits.
func floatToInteger(value float64) (*object.Integer, bool) {
	if value < -(1<<63) || value >= 1<<63 {
		return nil, false
	}
	return object.IntegerOf(int64(value)), true
}

// looseFloat is the conversion of == when one side is a float: integers,
// floats, booleans and strings with a number become a float64.
func looseFloat(obj object.Object) (float64, bool) {
	switch obj := obj.(type) {
	case *object.Float:
		return obj.Value, true
	case *object.String:
		return parseNumber(obj.Value)
	default:
		number, ok := toNumber(obj)
		return float64(number), ok
	}
}

// parseNumber reads a string like "5", "-2" or "1.5" as a float64. Unlike
// strconv.ParseFloat it accepts nothing else, so "inf", "1e3" and "0x10"
// are not numbers.
func parseNumber(s string) (float64, bool) {
	if whole, err := strconv.ParseInt(s, 10, 64); err == nil {
		return float64(whole), true
	}
	digits := s
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	dot := -1
	for i := 0; i < len(digits); i++ {
		switch {
		case digits[i] >= '0' && digits[i] <= '9':
		case digits[i] == '.' && dot < 0 && i > 0 && i < len(digits)-1:
			dot = i
		default:
			return 0, false
		}
	}
	if len(digits) == 0 {
		return 0, false
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

// builtinFloat is float(x): an integer, a float or a string with a number as
// a float.
func builtinFloat(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongCount("1", len(args))
	}
	switch arg := args[0].(type) {
	case *object.Integer, *object.Float:
		return &object.Float{Value: toFloat(arg)}
	case *object.String:
		value, ok := parseNumber(arg.Value)
		if !ok {
			return argError("not a number: %q", arg.Value)
		}
		return &object.Float{Value: value}
	default:
		return argError("argument not supported, got %s", arg.Type())
	}
}
