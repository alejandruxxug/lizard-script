package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// The VM computes with object.Value: numbers inside the value, everything
// else as its object (see object/value.go). Values become objects (boxed)
// only at the edges: builtins, the evaluator, arrays, hashes, points and
// the host's environment.

// Cell holds a local variable that closures share. It is empty until the
// variable's let runs. Cells only live in frame slots and in closures; a
// program never sees one.
type Cell struct {
	Value object.Value
}

func (c *Cell) Type() object.ObjectType { return "CELL" }
func (c *Cell) Inspect() string         { return "cell" }

// cellAt is the cell in a frame slot of a shared local.
func cellAt(slot object.Value) *Cell {
	return slot.Ref().(*Cell)
}

// Values of the evaluator's own objects.
var (
	nullValue     = object.ObjectValue(evaluator.NULL)
	trueValue     = object.ObjectValue(evaluator.TRUE)
	falseValue    = object.ObjectValue(evaluator.FALSE)
	breakValue    = object.ObjectValue(evaluator.BREAK)
	continueValue = object.ObjectValue(evaluator.CONTINUE)
)

// boolValue is the value of true or false.
func boolValue(b bool) object.Value {
	if b {
		return trueValue
	}
	return falseValue
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

// truthy is whether v counts as true in a condition, like the evaluator's
// IsTruthy: a number is true when it is not zero.
func truthy(v object.Value) bool {
	switch v.Kind() {
	case object.KindInt:
		return v.Int() != 0
	case object.KindFloat:
		return v.Float() != 0
	}
	if b, ok := v.Ref().(*object.Boolean); ok {
		return b.Value
	}
	return evaluator.IsTruthy(v.Ref())
}

// box is values as objects, written into out, which has room for them.
func box(out []object.Object, values []object.Value) []object.Object {
	for i, v := range values {
		out[i] = v.Object()
	}
	return out[:len(values)]
}
