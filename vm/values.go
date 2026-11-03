package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// Cell holds a local variable that closures share. It is empty (Value nil)
// until the variable's let runs. Cells only live in frame slots and in
// closures; a program never sees one.
type Cell struct {
	Value object.Object
}

func (c *Cell) Type() object.ObjectType { return "CELL" }
func (c *Cell) Inspect() string         { return "cell" }

// integer is the value of an integer result, without an allocation when it
// is small (see object.IntegerOf).
func integer(value int64) *object.Integer {
	return object.IntegerOf(value)
}

// operators are the operator texts of the infix opcodes, for the evaluator.
var operators = map[code.Opcode]string{
	code.OpAdd:          "+",
	code.OpSub:          "-",
	code.OpMul:          "*",
	code.OpDiv:          "/",
	code.OpFloorDiv:     "//",
	code.OpMod:          "%",
	code.OpPow:          "**",
	code.OpEqual:        "==",
	code.OpNotEqual:     "!=",
	code.OpExact:        "===",
	code.OpNotExact:     "!==",
	code.OpLess:         "<",
	code.OpGreater:      ">",
	code.OpLessEqual:    "<=",
	code.OpGreaterEqual: ">=",
}

// operand reads the 4-byte operand at ins[at].
func operand(ins code.Instructions, at int) int {
	return int(uint32(ins[at])<<24 | uint32(ins[at+1])<<16 | uint32(ins[at+2])<<8 | uint32(ins[at+3]))
}

// integerInfix is an infix opcode on two integers, for the operators whose
// result is simple. ok is false for the ones left to the evaluator.
func integerInfix(op code.Opcode, left, right int64) (object.Object, bool) {
	switch op {
	case code.OpAdd:
		return integer(left + right), true
	case code.OpSub:
		return integer(left - right), true
	case code.OpMul:
		return integer(left * right), true
	case code.OpDiv:
		if right == 0 {
			return nil, false
		}
		return integer(left / right), true
	case code.OpMod:
		if right == 0 {
			return nil, false
		}
		// The evaluator's: the remainder that goes with //.
		quotient := left / right
		if left%right != 0 && (left < 0) != (right < 0) {
			quotient--
		}
		return integer(left - quotient*right), true
	case code.OpEqual, code.OpExact:
		return evaluator.NativeBool(left == right), true
	case code.OpNotEqual, code.OpNotExact:
		return evaluator.NativeBool(left != right), true
	case code.OpLess:
		return evaluator.NativeBool(left < right), true
	case code.OpGreater:
		return evaluator.NativeBool(left > right), true
	case code.OpLessEqual:
		return evaluator.NativeBool(left <= right), true
	case code.OpGreaterEqual:
		return evaluator.NativeBool(left >= right), true
	}
	return nil, false
}

func truthy(value object.Object) bool {
	if b, ok := value.(*object.Boolean); ok {
		return b.Value
	}
	return evaluator.IsTruthy(value)
}
