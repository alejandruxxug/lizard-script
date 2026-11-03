package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// Formulas (OpFormula, see code.Formula and compiler/formulas.go): a whole
// arithmetic expression in one instruction. computeFormula reads every leaf
// and works on Go numbers, so only the result becomes an object. When it
// can't (a leaf is not a number or can't be read, or an operator would fail)
// it gives up, and the usual instructions that follow the OpFormula run
// instead, so the value and any error are theirs.

// formulaBackoff is how many runs of a formula go straight to the usual
// instructions after computeFormula gave up, for formulas on points or
// strings, which would give up every time.
const formulaBackoff = 32

// formula is the value of f, or false when the usual instructions must
// compute it.
func formula(f *code.Formula, unit *code.Unit, locals []object.Object, free []*Cell, globals *object.Environment) (object.Object, bool) {
	if f.Skip > 0 {
		f.Skip--
		return nil, false
	}
	value, ok := computeFormula(f, unit, locals, free, globals)
	if !ok {
		f.Skip = formulaBackoff
	}
	return value, ok
}

// computeFormula is the value of f computed on Go numbers, or false when
// it can't be. Floats on both sides, the commonest case, are done
// right here; everything else goes through arithmetic.
func computeFormula(f *code.Formula, unit *code.Unit, locals []object.Object, free []*Cell, globals *object.Environment) (object.Object, bool) {
	var values [code.MaxFormulaDepth]number
	n := 0
	for i := range f.Steps {
		step := &f.Steps[i]
		if step.Leaf >= 0 {
			leaf := &f.Leaves[step.Leaf]
			var value object.Object
			switch {
			case leaf.Fields == nil && leaf.Kind == code.LeafLocal:
				value = locals[leaf.Index]
			case leaf.Fields == nil && leaf.Kind == code.LeafConstant:
				value = unit.Constants[leaf.Index]
			default:
				var err *object.Error
				if value, err = readLeaf(leaf, unit, locals, free, globals); err != nil {
					return nil, false
				}
			}
			switch value := value.(type) {
			case *object.Float:
				values[n] = number{isFloat: true, f: value.Value}
			case *object.Integer:
				values[n] = number{i: value.Value}
			default:
				return nil, false
			}
			n++
			continue
		}
		n--
		left, right := &values[n-1], &values[n]
		if left.isFloat && right.isFloat {
			if result, ok := floatArithmetic(step.Op, left.f, right.f); ok {
				left.f = result
				continue
			}
		}
		if result, ok := arithmetic(step.Op, *left, *right); ok {
			*left = result
			continue
		}
		// A comparison is always the last step.
		if result, ok := compareNumbers(step.Op, *left, *right); ok {
			return evaluator.NativeBool(result), true
		}
		return nil, false
	}
	return values[0].object(), true
}

// floatArithmetic is + - * or / on two floats, false when the result is not
// a number or the operator is another one.
func floatArithmetic(op code.Opcode, a, b float64) (float64, bool) {
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
	default:
		return 0, false
	}
	// Infinity minus itself is NaN, so this catches both, and a division
	// by zero gives one of them.
	if result-result != 0 {
		return 0, false
	}
	return result, true
}

// readLeaf reads a leaf like the instructions OpGetLocal, OpGetCell,
// OpGetFree, OpGetName or OpConstant, then OpGetField for each dot access.
func readLeaf(leaf *code.Leaf, unit *code.Unit, locals []object.Object, free []*Cell, globals *object.Environment) (object.Object, *object.Error) {
	var value object.Object
	switch leaf.Kind {
	case code.LeafLocal:
		value = locals[leaf.Index]
	case code.LeafCell:
		value = locals[leaf.Index].(*Cell).Value
	case code.LeafFree:
		value = free[leaf.Index].Value
	case code.LeafConstant:
		value = unit.Constants[leaf.Index]
	case code.LeafName:
		var err *object.Error
		if value, err = lookup(&unit.Names[leaf.Index], free, globals); err != nil {
			return nil, err
		}
	}
	for _, read := range leaf.Fields {
		value = getField(&unit.Fields[read.Field], read.Token, unit, value)
		if err, ok := value.(*object.Error); ok {
			return nil, err
		}
	}
	return value, nil
}

// compareNumbers is < > <= or >= on two numbers: as integers when both are,
// as floats otherwise, like the instructions.
func compareNumbers(op code.Opcode, left, right number) (bool, bool) {
	if left.isFloat || right.isFloat {
		return compare(op, left.float(), right.float())
	}
	a, b := left.i, right.i
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
