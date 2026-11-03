package vm

import (
	"lizardscript/ast"
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// run runs the frames until the first one returns, and returns its value,
// or the error that stopped the run.
func (vm *VM) run() object.Object {
	stack := vm.stack
	sp := vm.sp
	fr := &vm.frames[len(vm.frames)-1]
	cl := fr.cl
	ins := cl.Fn.Instructions
	unit := cl.Fn.Unit
	free := cl.Free
	ip := fr.ip
	bp := fr.bp

	for {
		var err *object.Error
		op := code.Opcode(ins[ip])

		switch op {
		case code.OpConstant:
			stack[sp] = unit.Constants[operand(ins, ip+1)]
			sp++
			ip += 5

		case code.OpNull:
			stack[sp] = evaluator.NULL
			sp++
			ip++

		case code.OpTrue:
			stack[sp] = evaluator.TRUE
			sp++
			ip++

		case code.OpFalse:
			stack[sp] = evaluator.FALSE
			sp++
			ip++

		case code.OpPop:
			sp--
			stack[sp] = nil
			ip++

		case code.OpBreakValue:
			stack[sp] = evaluator.BREAK
			sp++
			ip++

		case code.OpContinueValue:
			stack[sp] = evaluator.CONTINUE
			sp++
			ip++

		case code.OpMakeReturn:
			stack[sp-1] = &object.ReturnValue{Value: stack[sp-1]}
			ip++

		case code.OpUnwrapReturn:
			if returned, ok := stack[sp-1].(*object.ReturnValue); ok {
				stack[sp-1] = returned.Value
			}
			ip++

		// Variables

		case code.OpGetLocal:
			stack[sp] = stack[bp+operand(ins, ip+1)]
			sp++
			ip += 5

		case code.OpSetLocal:
			sp--
			stack[bp+operand(ins, ip+1)] = stack[sp]
			stack[sp] = nil
			ip += 5

		case code.OpAssignLocal:
			stack[bp+operand(ins, ip+1)] = stack[sp-1]
			ip += 5

		case code.OpGetCell:
			stack[sp] = stack[bp+operand(ins, ip+1)].(*Cell).Value
			sp++
			ip += 5

		case code.OpSetCell:
			sp--
			stack[bp+operand(ins, ip+1)].(*Cell).Value = stack[sp]
			stack[sp] = nil
			ip += 5

		case code.OpAssignCell:
			stack[bp+operand(ins, ip+1)].(*Cell).Value = stack[sp-1]
			ip += 5

		case code.OpGetFree:
			stack[sp] = free[operand(ins, ip+1)].Value
			sp++
			ip += 5

		case code.OpAssignFree:
			free[operand(ins, ip+1)].Value = stack[sp-1]
			ip += 5

		case code.OpGetName:
			value, e := lookup(&unit.Names[operand(ins, ip+1)], free, cl.Globals)
			if e != nil {
				err = e
				break
			}
			stack[sp] = value
			sp++
			ip += 5

		case code.OpAssignName:
			if e := assign(&unit.Names[operand(ins, ip+1)], stack[sp-1], free, cl.Globals); e != nil {
				err = e
				break
			}
			ip += 5

		case code.OpDefineGlobal:
			sp--
			cl.Globals.Define(unit.Names[operand(ins, ip+1)].Name, stack[sp])
			stack[sp] = nil
			ip += 5

		case code.OpEnterScope:
			enterScope(&unit.Scopes[operand(ins, ip+1)], stack[bp:])
			ip += 5

		case code.OpSkipScope:
			ip += 5

		case code.OpMoveIfGiven:
			if operand(ins, ip+1) < fr.given {
				stack[bp+operand(ins, ip+5)] = stack[bp+operand(ins, ip+1)]
			}
			ip += 9

		case code.OpJumpIfGiven:
			if operand(ins, ip+1) < fr.given {
				ip = operand(ins, ip+5)
			} else {
				ip += 9
			}

		// Operators

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpFloorDiv, code.OpMod, code.OpPow,
			code.OpEqual, code.OpNotEqual, code.OpExact, code.OpNotExact,
			code.OpLess, code.OpGreater, code.OpLessEqual, code.OpGreaterEqual:
			left, right := stack[sp-2], stack[sp-1]
			if l, ok := left.(*object.Integer); ok {
				if r, ok := right.(*object.Integer); ok {
					if result, ok := integerInfix(op, l.Value, r.Value); ok {
						sp--
						stack[sp] = nil
						stack[sp-1] = result
						ip += 5
						continue
					}
				}
			}
			result, ok := fastInfix(op, left, right)
			if !ok {
				result = evaluator.Infix(unit.Tokens[operand(ins, ip+1)], operators[op], left, right)
				if e, ok := result.(*object.Error); ok {
					err = e
					break
				}
			}
			sp--
			stack[sp] = nil
			stack[sp-1] = result
			ip += 5

		case code.OpMinus:
			if i, ok := stack[sp-1].(*object.Integer); ok {
				stack[sp-1] = integer(-i.Value)
				ip += 5
				continue
			}
			fallthrough

		case code.OpPrefix:
			node := unit.Nodes[operand(ins, ip+1)].(*ast.PrefixExpression)
			result := evaluator.Prefix(node, stack[sp-1])
			if e, ok := result.(*object.Error); ok {
				err = e
				break
			}
			stack[sp-1] = result
			ip += 5

		case code.OpNot:
			stack[sp-1] = evaluator.NativeBool(!truthy(stack[sp-1]))
			ip++

		case code.OpAnd:
			if !truthy(stack[sp-1]) {
				stack[sp-1] = evaluator.FALSE
				ip = operand(ins, ip+1)
			} else {
				sp--
				stack[sp] = nil
				ip += 5
			}

		case code.OpOr:
			if truthy(stack[sp-1]) {
				stack[sp-1] = evaluator.TRUE
				ip = operand(ins, ip+1)
			} else {
				sp--
				stack[sp] = nil
				ip += 5
			}

		case code.OpToBool:
			stack[sp-1] = evaluator.NativeBool(truthy(stack[sp-1]))
			ip++

		// Control flow

		case code.OpJump:
			ip = operand(ins, ip+1)

		case code.OpJumpNotTruthy:
			sp--
			condition := stack[sp]
			stack[sp] = nil
			if truthy(condition) {
				ip += 5
			} else {
				ip = operand(ins, ip+1)
			}

		case code.OpStep:
			if evaluator.MaxSteps != 0 {
				if e := evaluator.CountStep(unit.Tokens[operand(ins, ip+1)]); e != nil {
					err = e
					break
				}
			}
			ip += 5

		case code.OpStatement:
			if hook := evaluator.OnStatement; hook != nil {
				hook(unit.Statements[operand(ins, ip+1)])
			}
			ip += 5

		case code.OpCheckSignal:
			switch stack[sp-1].(type) {
			case *object.ReturnValue:
				ip = operand(ins, ip+1)
			case *object.Break:
				ip = operand(ins, ip+5)
			case *object.Continue:
				ip = operand(ins, ip+9)
			default:
				ip += 13
			}

		case code.OpReturnValue:
			value := stack[sp-1]
			if fr.profile != nil {
				vm.leaveFrames(len(vm.frames) - 1)
			}
			vm.closeLayers(fr.layers)
			vm.handlers = vm.handlers[:fr.handlers]
			if len(vm.frames) == 1 {
				vm.frames = vm.frames[:0]
				vm.sp = 0
				return value
			}
			clear(stack[fr.bp:sp])
			sp = fr.bp - 1
			stack[sp] = value
			sp++
			vm.frames = vm.frames[:len(vm.frames)-1]
			fr = &vm.frames[len(vm.frames)-1]
			cl = fr.cl
			ins, unit, free = cl.Fn.Instructions, cl.Fn.Unit, cl.Free
			ip, bp = fr.ip, fr.bp

		case code.OpPushHandler:
			vm.handlers = append(vm.handlers, handler{
				frame:  len(vm.frames) - 1,
				sp:     sp,
				catch:  operand(ins, ip+1),
				layers: len(vm.layers),
			})
			ip += 5

		case code.OpPopHandler:
			vm.handlers = vm.handlers[:len(vm.handlers)-1]
			ip++

		case code.OpLayerEnter:
			vm.openLayer(unit.Nodes[operand(ins, ip+1)].(*ast.LayerStatement))
			ip += 5

		case code.OpLayerExit:
			vm.closeLayers(len(vm.layers) - 1)
			ip++

		case code.OpFail:
			failure := unit.Failures[operand(ins, ip+1)]
			err = evaluator.NewError(failure.Token, "%s", failure.Message)

		// Functions

		case code.OpCall:
			count := operand(ins, ip+1)
			tok := operand(ins, ip+5)
			callee := stack[sp-1-count]
			next, ok := callee.(*Closure)
			if !ok {
				// The arguments are lent from the stack, without a copy (see
				// object.BuiltinFunction); the cap keeps an append off it.
				result := vm.callForeign(tok, unit, callee, stack[sp-count:sp:sp])
				if e, ok := result.(*object.Error); ok {
					err = e
					break
				}
				clear(stack[sp-count : sp])
				sp -= count
				stack[sp-1] = result
				ip += 9
				continue
			}

			if evaluator.MaxSteps != 0 {
				if e := evaluator.CountStep(unit.Tokens[tok]); e != nil {
					err = e
					break
				}
			}
			fn := next.Fn
			if e := arityError(unit.Tokens[tok], fn, count); e != nil {
				err = e
				break
			}
			if vm.depthBase+len(vm.frames) >= evaluator.MaxCallDepth {
				err = overflow(unit.Tokens[tok])
				break
			}

			fr.ip = ip + 9
			vm.sp = sp
			vm.pushFrame(next, count)
			stack, sp = vm.stack, vm.sp
			fr = &vm.frames[len(vm.frames)-1]
			cl = next
			ins, unit, free = fn.Instructions, fn.Unit, next.Free
			ip, bp = 0, fr.bp

		case code.OpClosure:
			fn := unit.Functions[operand(ins, ip+1)]
			cells := captureCells(unit.Captures[operand(ins, ip+5)], stack[bp:], free)
			stack[sp] = &Closure{Fn: fn, Free: cells, Globals: cl.Globals}
			sp++
			ip += 9

		// Data structures

		case code.OpArray:
			count := operand(ins, ip+1)
			array := arrayOf(stack[sp-count : sp])
			clear(stack[sp-count : sp])
			sp -= count
			stack[sp] = array
			sp++
			ip += 5

		case code.OpHash:
			count := operand(ins, ip+1)
			hash := makeHash(stack[sp-2*count : sp])
			clear(stack[sp-2*count : sp])
			sp -= 2 * count
			stack[sp] = hash
			sp++
			ip += 5

		case code.OpCheckKey:
			if _, ok := stack[sp-1].(object.Hashable); !ok {
				err = evaluator.NewError(unit.Tokens[operand(ins, ip+1)], "unusable as hash key: %s", stack[sp-1].Type())
				break
			}
			ip += 5

		case code.OpPoint:
			count := int(ins[ip+1])
			point := pointOf(stack[sp-count : sp])
			clear(stack[sp-count : sp])
			sp -= count
			stack[sp] = point
			sp++
			ip += 2

		case code.OpCheckCoord:
			if !evaluator.IsNumber(stack[sp-1]) {
				err = evaluator.NewError(unit.Tokens[operand(ins, ip+1)],
					"a point coordinate must be INTEGER or FLOAT, got %s", stack[sp-1].Type())
				break
			}
			ip += 5

		case code.OpIndex:
			left, index := stack[sp-2], stack[sp-1]
			var result object.Object
			array, isArray := left.(*object.Array)
			i, isInteger := index.(*object.Integer)
			if isArray && isInteger {
				if i.Value >= 0 && i.Value < int64(len(array.Elements)) {
					result = array.Elements[i.Value]
				} else {
					result = evaluator.NULL
				}
			} else {
				result = evaluator.Index(unit.Tokens[operand(ins, ip+1)], left, index)
				if e, ok := result.(*object.Error); ok {
					err = e
					break
				}
			}
			sp--
			stack[sp] = nil
			stack[sp-1] = result
			ip += 5

		case code.OpIndexKeep:
			result := evaluator.Index(unit.Tokens[operand(ins, ip+1)], stack[sp-2], stack[sp-1])
			if e, ok := result.(*object.Error); ok {
				err = e
				break
			}
			stack[sp] = result
			sp++
			ip += 5

		case code.OpGetField:
			result := getField(&unit.Fields[operand(ins, ip+1)], operand(ins, ip+5), unit, stack[sp-1])
			if e, ok := result.(*object.Error); ok {
				err = e
				break
			}
			stack[sp-1] = result
			ip += 9

		case code.OpGetLocalField, code.OpGetCellField:
			value := stack[bp+operand(ins, ip+1)]
			if op == code.OpGetCellField {
				value = value.(*Cell).Value
			}
			result := getField(&unit.Fields[operand(ins, ip+5)], operand(ins, ip+9), unit, value)
			if e, ok := result.(*object.Error); ok {
				err = e
				break
			}
			stack[sp] = result
			sp++
			ip += 13

		case code.OpGetNameField:
			value, e := lookup(&unit.Names[operand(ins, ip+1)], free, cl.Globals)
			if e != nil {
				err = e
				break
			}
			result := getField(&unit.Fields[operand(ins, ip+5)], operand(ins, ip+9), unit, value)
			if e, ok := result.(*object.Error); ok {
				err = e
				break
			}
			stack[sp] = result
			sp++
			ip += 13

		case code.OpSetIndex:
			container, index, value := stack[sp-3], stack[sp-2], stack[sp-1]
			array, isArray := container.(*object.Array)
			i, isInteger := index.(*object.Integer)
			if isArray && isInteger && i.Value >= 0 && i.Value < int64(len(array.Elements)) {
				array.Elements[i.Value] = value
			} else {
				result := evaluator.SetIndex(unit.Tokens[operand(ins, ip+1)], container, index, value)
				if e, ok := result.(*object.Error); ok {
					err = e
					break
				}
			}
			clear(stack[sp-2 : sp])
			sp -= 2
			stack[sp-1] = value
			ip += 5

		case code.OpFormula:
			if result, ok := formula(&unit.Formulas[operand(ins, ip+1)], unit, stack[bp:], free, cl.Globals); ok {
				stack[sp] = result
				sp++
				ip = operand(ins, ip+5)
			} else {
				ip += 9
			}

		default:
			def, _ := code.Lookup(byte(op))
			name := "?"
			if def != nil {
				name = def.Name
			}
			panic("vm: unknown instruction " + name)
		}

		if err == nil {
			continue
		}

		// An error: to the innermost try that catches it, or out of the run.
		vm.sp = sp
		if !vm.throw(err) {
			clear(stack[:sp])
			vm.sp = 0
			return err
		}
		stack = vm.stack
		sp = vm.sp
		fr = &vm.frames[len(vm.frames)-1]
		cl = fr.cl
		ins, unit, free = cl.Fn.Instructions, cl.Fn.Unit, cl.Free
		ip, bp = fr.ip, fr.bp
	}
}

// enterScope gives the shared locals of a scope that starts their cells:
// new and empty for a block, holding the argument for a function's scope.
func enterScope(scope *code.Scope, locals []object.Object) {
	for _, slot := range scope.Slots {
		if scope.Wrap {
			locals[slot] = &Cell{Value: locals[slot]}
		} else {
			locals[slot] = &Cell{}
		}
	}
}

// captureCells collects the cells a new closure captures: locals of the
// frame that makes it, or that frame's own captured cells.
func captureCells(captures []code.Capture, locals []object.Object, free []*Cell) []*Cell {
	cells := make([]*Cell, len(captures))
	for i, capture := range captures {
		if capture.Local {
			cells[i] = locals[capture.Index].(*Cell)
		} else {
			cells[i] = free[capture.Index]
		}
	}
	return cells
}

// makeHash builds a hash from key, value pairs, in order. The keys were
// checked with OpCheckKey.
func makeHash(pairs []object.Object) *object.Hash {
	hash := object.NewHashSized(len(pairs) / 2)
	for i := 0; i < len(pairs); i += 2 {
		hash.Set(pairs[i].(object.Hashable), pairs[i+1])
	}
	return hash
}
