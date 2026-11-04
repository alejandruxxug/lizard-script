package vm

import (
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
)

// step runs one instruction, the one at the ip of the innermost frame,
// whatever it is and whatever its values: everything the fast loop (see
// run.go) has no fast path for comes here, so this is the whole meaning of
// each instruction. It reports whether the run is over, with its value (or
// its error) then.
func (vm *VM) step() (object.Value, bool) {
	stack, sp := vm.stack, vm.sp
	fr := &vm.frames[len(vm.frames)-1]
	cl := fr.cl
	unit := cl.Fn.Unit
	ins := fr.code
	free := cl.Free
	ip, bp := fr.ip, fr.bp
	op := code.Opcode(ins[ip])
	var err *object.Error

	switch op {
	case code.OpConstant:
		stack[sp] = unit.Values[int(ins[ip+1])]
		sp++
		ip += 2

	case code.OpNull:
		stack[sp] = nullValue
		sp++
		ip++

	case code.OpTrue:
		stack[sp] = trueValue
		sp++
		ip++

	case code.OpFalse:
		stack[sp] = falseValue
		sp++
		ip++

	case code.OpPop:
		sp--
		stack[sp] = object.Value{}
		ip++

	case code.OpBreakValue:
		stack[sp] = breakValue
		sp++
		ip++

	case code.OpContinueValue:
		stack[sp] = continueValue
		sp++
		ip++

	case code.OpMakeReturn:
		stack[sp-1] = object.ObjectValue(&object.ReturnValue{Value: stack[sp-1].Object()})
		ip++

	case code.OpUnwrapReturn:
		if returned, ok := stack[sp-1].Ref().(*object.ReturnValue); ok {
			stack[sp-1] = object.ValueOf(returned.Value)
		}
		ip++

	// Variables

	case code.OpGetLocal:
		stack[sp] = stack[bp+int(ins[ip+1])]
		sp++
		ip += 2

	case code.OpSetLocal:
		sp--
		stack[bp+int(ins[ip+1])] = stack[sp]
		stack[sp] = object.Value{}
		ip += 2

	case code.OpAssignLocal:
		stack[bp+int(ins[ip+1])] = stack[sp-1]
		ip += 2

	case code.OpGetCell:
		stack[sp] = cellAt(stack[bp+int(ins[ip+1])]).Value
		sp++
		ip += 2

	case code.OpSetCell:
		sp--
		cellAt(stack[bp+int(ins[ip+1])]).Value = stack[sp]
		stack[sp] = object.Value{}
		ip += 2

	case code.OpAssignCell:
		cellAt(stack[bp+int(ins[ip+1])]).Value = stack[sp-1]
		ip += 2

	case code.OpGetFree:
		stack[sp] = free[int(ins[ip+1])].Value
		sp++
		ip += 2

	case code.OpAssignFree:
		free[int(ins[ip+1])].Value = stack[sp-1]
		ip += 2

	case code.OpGetName:
		value, e := lookup(&unit.Names[int(ins[ip+1])], free, cl.Globals)
		if e != nil {
			err = e
			break
		}
		stack[sp] = value
		sp++
		ip += 2

	case code.OpAssignName:
		if e := assign(&unit.Names[int(ins[ip+1])], stack[sp-1], free, cl.Globals); e != nil {
			err = e
			break
		}
		ip += 2

	case code.OpDefineGlobal:
		sp--
		cl.Globals.Define(unit.Names[int(ins[ip+1])].Name, stack[sp].Object())
		stack[sp] = object.Value{}
		ip += 2

	case code.OpEnterScope:
		enterScope(&unit.Scopes[int(ins[ip+1])], stack[bp:])
		ip += 2

	case code.OpMoveIfGiven:
		if int(ins[ip+1]) < fr.given {
			stack[bp+int(ins[ip+2])] = stack[bp+int(ins[ip+1])]
		}
		ip += 3

	case code.OpJumpIfGiven:
		if int(ins[ip+1]) < fr.given {
			ip = int(ins[ip+2])
		} else {
			ip += 3
		}

	// Operators

	case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpFloorDiv, code.OpMod, code.OpPow,
		code.OpEqual, code.OpNotEqual, code.OpExact, code.OpNotExact,
		code.OpLess, code.OpGreater, code.OpLessEqual, code.OpGreaterEqual:
		result, e := infix(op, unit.Tokens[int(ins[ip+1])], stack[sp-2], stack[sp-1])
		if e != nil {
			err = e
			break
		}
		sp--
		stack[sp] = object.Value{}
		stack[sp-1] = result
		ip += 2

	case code.OpMinus, code.OpPrefix:
		if result, ok := minus(stack[sp-1]); ok && op == code.OpMinus {
			stack[sp-1] = result
			ip += 2
			break
		}
		node := unit.Nodes[int(ins[ip+1])].(*ast.PrefixExpression)
		result := evaluator.Prefix(node, stack[sp-1].Object())
		if e, ok := result.(*object.Error); ok {
			err = e
			break
		}
		stack[sp-1] = object.ValueOf(result)
		ip += 2

	case code.OpNot:
		stack[sp-1] = boolValue(!truthy(stack[sp-1]))
		ip++

	case code.OpAnd:
		if !truthy(stack[sp-1]) {
			stack[sp-1] = falseValue
			ip = int(ins[ip+1])
		} else {
			sp--
			stack[sp] = object.Value{}
			ip += 2
		}

	case code.OpOr:
		if truthy(stack[sp-1]) {
			stack[sp-1] = trueValue
			ip = int(ins[ip+1])
		} else {
			sp--
			stack[sp] = object.Value{}
			ip += 2
		}

	case code.OpToBool:
		stack[sp-1] = boolValue(truthy(stack[sp-1]))
		ip++

	// Control flow

	case code.OpJump:
		ip = int(ins[ip+1])

	case code.OpJumpNotTruthy:
		sp--
		condition := stack[sp]
		stack[sp] = object.Value{}
		if truthy(condition) {
			ip += 2
		} else {
			ip = int(ins[ip+1])
		}

	case code.OpStep:
		if evaluator.MaxSteps != 0 {
			if e := evaluator.CountStep(unit.Tokens[int(ins[ip+1])]); e != nil {
				err = e
				break
			}
		}
		ip += 2

	case code.OpStatement:
		if hook := evaluator.OnStatement; hook != nil {
			hook(unit.Statements[int(ins[ip+1])])
		}
		ip += 2

	case code.OpCheckSignal:
		switch stack[sp-1].Ref().(type) {
		case *object.ReturnValue:
			ip = int(ins[ip+1])
		case *object.Break:
			ip = int(ins[ip+2])
		case *object.Continue:
			ip = int(ins[ip+3])
		default:
			ip += 4
		}

	case code.OpReturnValue:
		value := stack[sp-1]
		if fr.profile != nil {
			vm.leaveFrames(len(vm.frames) - 1)
		}
		vm.closeLayers(fr.layers)
		vm.handlers = vm.handlers[:fr.handlers]
		if len(vm.frames) == 1 {
			clear(stack[:sp])
			vm.frames = vm.frames[:0]
			vm.sp = 0
			return value, true
		}
		clear(stack[fr.bp:sp])
		sp = fr.bp - 1
		stack[sp] = value
		vm.sp = sp + 1
		vm.frames = vm.frames[:len(vm.frames)-1]
		return object.Value{}, false

	case code.OpPushHandler:
		vm.handlers = append(vm.handlers, handler{
			frame:  len(vm.frames) - 1,
			sp:     sp,
			catch:  int(ins[ip+1]),
			layers: len(vm.layers),
		})
		ip += 2

	case code.OpPopHandler:
		vm.handlers = vm.handlers[:len(vm.handlers)-1]
		ip++

	case code.OpLayerEnter:
		vm.openLayer(unit.Nodes[int(ins[ip+1])].(*ast.LayerStatement))
		ip += 2

	case code.OpLayerExit:
		vm.closeLayers(len(vm.layers) - 1)
		ip++

	case code.OpUse:
		result := evaluator.Use(unit.Nodes[int(ins[ip+1])].(*ast.UseStatement), cl.Globals, RunModule)
		if e, ok := result.(*object.Error); ok {
			err = e
			break
		}
		stack[sp] = object.ValueOf(result)
		sp++
		ip += 2

	case code.OpFail:
		failure := unit.Failures[int(ins[ip+1])]
		err = evaluator.NewError(failure.Token, "%s", failure.Message)

	// Functions

	case code.OpCall:
		count := int(ins[ip+1])
		tok := int(ins[ip+2])
		callee := stack[sp-1-count]
		next, ok := callee.Ref().(*Closure)
		if !ok && count == 2 && callee.Ref() == pushBuiltin && evaluator.MaxSteps == 0 && profiler == nil {
			// push(list, x), the builtin's work without the call.
			if array, isArray := stack[sp-2].Ref().(*object.Array); isArray {
				array.Elements = append(array.Elements, stack[sp-1].Object())
				clear(stack[sp-count : sp])
				sp -= count
				stack[sp-1] = object.ObjectValue(array)
				ip += 3
				break
			}
		}
		if !ok {
			result, e := vm.callForeign(tok, unit, callee.Object(), stack[sp-count:sp])
			if e != nil {
				err = e
				break
			}
			clear(stack[sp-count : sp])
			sp -= count
			stack[sp-1] = result
			ip += 3
			break
		}
		if evaluator.MaxSteps != 0 {
			if e := evaluator.CountStep(unit.Tokens[tok]); e != nil {
				err = e
				break
			}
		}
		if e := arityError(unit.Tokens[tok], next.Fn, count); e != nil {
			err = e
			break
		}
		if vm.depthBase+len(vm.frames) >= evaluator.MaxCallDepth {
			err = overflow(unit.Tokens[tok])
			break
		}
		fr.ip = ip + 3
		vm.sp = sp
		vm.pushFrame(next, count)
		return object.Value{}, false

	case code.OpClosure:
		fn := unit.Functions[int(ins[ip+1])]
		cells := captureCells(unit.Captures[int(ins[ip+2])], stack[bp:], free)
		stack[sp] = object.ObjectValue(&Closure{Fn: fn, Free: cells, Globals: cl.Globals})
		sp++
		ip += 3

	// Data structures

	case code.OpArray:
		count := int(ins[ip+1])
		array := arrayOf(stack[sp-count : sp])
		clear(stack[sp-count : sp])
		sp -= count
		stack[sp] = object.ObjectValue(array)
		sp++
		ip += 2

	case code.OpHash:
		count := int(ins[ip+1])
		hash := makeHash(stack[sp-2*count : sp])
		clear(stack[sp-2*count : sp])
		sp -= 2 * count
		stack[sp] = object.ObjectValue(hash)
		sp++
		ip += 2

	case code.OpCheckKey:
		key := stack[sp-1]
		if _, ok := key.Ref().(object.Hashable); !ok && !key.IsInt() {
			err = evaluator.NewError(unit.Tokens[int(ins[ip+1])], "unusable as hash key: %s", key.Object().Type())
			break
		}
		ip += 2

	case code.OpPoint:
		count := int(ins[ip+1])
		point := newPoint(stack[sp-count : sp])
		clear(stack[sp-count : sp])
		sp -= count
		stack[sp] = object.ObjectValue(point)
		sp++
		ip += 2

	case code.OpCheckCoord:
		if coord := stack[sp-1]; !coord.IsNumber() {
			err = evaluator.NewError(unit.Tokens[int(ins[ip+1])],
				"a point coordinate must be INTEGER or FLOAT, got %s", coord.Ref().Type())
			break
		}
		ip += 2

	case code.OpIndex, code.OpIndexKeep:
		found := index(unit.Tokens[int(ins[ip+1])], stack[sp-2], stack[sp-1])
		if e, ok := found.(*object.Error); ok {
			err = e
			break
		}
		if op == code.OpIndex {
			sp--
			stack[sp] = object.Value{}
			stack[sp-1] = object.ValueOf(found)
		} else {
			stack[sp] = object.ValueOf(found)
			sp++
		}
		ip += 2

	case opIndexHash:
		left, key := stack[sp-2], stack[sp-1]
		found := index(unit.Tokens[int(ins[ip+1])], left, key)
		if e, ok := found.(*object.Error); ok {
			err = e
			break
		}
		value := object.ValueOf(found)
		if h, ok := left.Ref().(*object.Hash); ok {
			if s, ok := key.Ref().(*object.String); ok {
				remember(&cl.Fn.Indexes[int(ins[ip+2])], h, s, value)
			}
		}
		sp--
		stack[sp] = object.Value{}
		stack[sp-1] = value
		ip += 3

	case code.OpGetField:
		result, e := getField(&unit.Fields[int(ins[ip+1])], int(ins[ip+2]), unit, stack[sp-1])
		if e != nil {
			err = e
			break
		}
		stack[sp-1] = result
		ip += 3

	case code.OpGetLocalField, code.OpGetCellField, code.OpGetNameField:
		var value object.Value
		switch op {
		case code.OpGetLocalField:
			value = stack[bp+int(ins[ip+1])]
		case code.OpGetCellField:
			value = cellAt(stack[bp+int(ins[ip+1])]).Value
		default:
			if value, err = lookup(&unit.Names[int(ins[ip+1])], free, cl.Globals); err != nil {
				break
			}
		}
		if err != nil {
			break
		}
		result, e := getField(&unit.Fields[int(ins[ip+2])], int(ins[ip+3]), unit, value)
		if e != nil {
			err = e
			break
		}
		stack[sp] = result
		sp++
		ip += 4

	case code.OpSetIndex:
		container, index, value := stack[sp-3], stack[sp-2], stack[sp-1]
		array, isArray := container.Ref().(*object.Array)
		if i := index.Int(); isArray && index.IsInt() && i >= 0 && i < int64(len(array.Elements)) {
			array.Elements[i] = value.Object()
		} else {
			result := evaluator.SetIndex(unit.Tokens[int(ins[ip+1])], container.Object(), index.Object(), value.Object())
			if e, ok := result.(*object.Error); ok {
				err = e
				break
			}
		}
		clear(stack[sp-3 : sp])
		sp -= 2
		stack[sp-1] = value
		ip += 2

	// The fused instructions of quickened code (see quicken.go): each does
	// what the instructions it stands for would.

	case opGetLocals:
		stack[sp] = stack[bp+int(ins[ip+1])]
		stack[sp+1] = stack[bp+int(ins[ip+2])]
		sp += 2
		ip += 3

	case opGetLocalConst:
		stack[sp] = stack[bp+int(ins[ip+1])]
		stack[sp+1] = unit.Values[int(ins[ip+2])]
		sp += 2
		ip += 3

	case opLoop:
		if evaluator.MaxSteps != 0 {
			if e := evaluator.CountStep(unit.Tokens[int(ins[ip+1])]); e != nil {
				err = e
				break
			}
		}
		ip = int(ins[ip+2])

	case opSetFree:
		sp--
		free[int(ins[ip+1])].Value = stack[sp]
		stack[sp] = object.Value{}
		ip += 2

	case opIncLocal, opDecLocal, opIncFree:
		operator := code.OpAdd
		if op == opDecLocal {
			operator = code.OpSub
		}
		var variable *object.Value
		if op == opIncFree {
			variable = &free[int(ins[ip+1])].Value
		} else {
			variable = &stack[bp+int(ins[ip+1])]
		}
		result, e := infix(operator, unit.Tokens[int(ins[ip+3])], *variable, unit.Values[int(ins[ip+2])])
		if e != nil {
			err = e
			break
		}
		*variable = result
		ip += 4

	case opAddX, opSubX, opMulX, opDivX, opModX, opFloorDivX:
		a := source(stack, bp, unit.Values, int(ins[ip+1]))
		b := source(stack, bp, unit.Values, int(ins[ip+2]))
		result, e := infix(arithmetic[op-opAddX], unit.Tokens[int(ins[ip+3])], a, b)
		if e != nil {
			err = e
			break
		}
		stack[sp] = result
		sp++
		ip += 4

	case opAddS, opSubS, opMulS, opDivS, opModS, opFloorDivS:
		b := source(stack, bp, unit.Values, int(ins[ip+1]))
		result, e := infix(arithmetic[op-opAddS], unit.Tokens[int(ins[ip+2])], stack[sp-1], b)
		if e != nil {
			err = e
			break
		}
		stack[sp-1] = result
		ip += 3

	case opAddSet, opSubSet, opMulSet, opDivSet, opModSet, opFloorDivSet:
		result, e := infix(arithmetic[op-opAddSet], unit.Tokens[int(ins[ip+2])], stack[sp-2], stack[sp-1])
		if e != nil {
			err = e
			break
		}
		sp -= 2
		clear(stack[sp : sp+2])
		stack[bp+int(ins[ip+1])] = result
		ip += 3

	case opJumpUnlessLess, opJumpUnlessGreater, opJumpUnlessLessEqual,
		opJumpUnlessGreaterEqual, opJumpUnlessEqual, opJumpUnlessNotEqual:
		result, e := infix(comparisons[op-opJumpUnlessLess], unit.Tokens[int(ins[ip+1])], stack[sp-2], stack[sp-1])
		if e != nil {
			err = e
			break
		}
		sp -= 2
		clear(stack[sp : sp+2])
		if truthy(result) {
			ip += 3
		} else {
			ip = int(ins[ip+2])
		}

	case opJumpUnlessLessX, opJumpUnlessGreaterX, opJumpUnlessLessEqualX,
		opJumpUnlessGreaterEqualX, opJumpUnlessEqualX, opJumpUnlessNotEqualX:
		a := source(stack, bp, unit.Values, int(ins[ip+1]))
		b := source(stack, bp, unit.Values, int(ins[ip+2]))
		result, e := infix(comparisons[op-opJumpUnlessLessX], unit.Tokens[int(ins[ip+3])], a, b)
		if e != nil {
			err = e
			break
		}
		if truthy(result) {
			ip += 5
		} else {
			ip = int(ins[ip+4])
		}

	default:
		panic("vm: unknown instruction " + opName(op))
	}

	if err != nil {
		// To the innermost try that catches it, or out of the run.
		vm.sp = sp
		if !vm.throw(err) {
			clear(stack[:sp])
			vm.sp = 0
			return object.ObjectValue(err), true
		}
		return object.Value{}, false
	}
	vm.sp, fr.ip = sp, ip
	return object.Value{}, false
}

// index is left[index]: an array element or a string's character read
// here, anything else (and every error) by the evaluator.
func index(tok token.Token, left, index object.Value) object.Object {
	if index.IsInt() {
		i := index.Int()
		switch left := left.Ref().(type) {
		case *object.Array:
			if i >= 0 && i < int64(len(left.Elements)) {
				return left.Elements[i]
			}
			return evaluator.NULL
		case *object.String:
			if i >= 0 && i < int64(left.Len()) {
				return object.Char(left.RuneAt(int(i)))
			}
			return evaluator.NULL
		}
	}
	return evaluator.Index(tok, left.Object(), index.Object())
}

// remember keeps in an index's cache that key read value in h (see
// code.IndexCache), making the cache on its first hash.
func remember(cache **code.IndexCache, h *object.Hash, key *object.String, value object.Value) {
	if *cache == nil {
		*cache = &code.IndexCache{}
	}
	hash := key.HashKey().Value
	(*cache)[indexSlot(hash)] = code.IndexEntry{Hash: h, Version: h.Version(), KeyHash: hash, Key: key.Value, Value: value}
}

// indexSlot is the entry of a code.IndexCache for a key's hash: its bits
// mixed, since the hashes of one-letter keys differ in a few bits only.
func indexSlot(hash uint64) uint64 {
	return (hash ^ hash>>33) * 0xff51afd7ed558ccd >> (64 - 5)
}

// enterScope gives the shared locals of a scope that starts their cells:
// new and empty for a block, holding the argument for a function's scope.
func enterScope(scope *code.Scope, locals []object.Value) {
	for _, slot := range scope.Slots {
		if scope.Wrap {
			locals[slot] = object.ObjectValue(&Cell{Value: locals[slot]})
		} else {
			locals[slot] = object.ObjectValue(&Cell{})
		}
	}
}

// captureCells collects the cells a new closure captures: locals of the
// frame that makes it, or that frame's own captured cells.
func captureCells(captures []code.Capture, locals []object.Value, free []*Cell) []*Cell {
	cells := make([]*Cell, len(captures))
	for i, capture := range captures {
		if capture.Local {
			cells[i] = cellAt(locals[capture.Index])
		} else {
			cells[i] = free[capture.Index]
		}
	}
	return cells
}

// makeHash builds a hash from key, value pairs, in order. The keys were
// checked with OpCheckKey.
func makeHash(pairs []object.Value) *object.Hash {
	hash := object.NewHashSized(len(pairs) / 2)
	for i := 0; i < len(pairs); i += 2 {
		hash.Set(pairs[i].Object().(object.Hashable), pairs[i+1].Object())
	}
	return hash
}
