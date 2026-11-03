package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
	"unsafe"
)

// run runs the frames until the first one returns, and returns its value,
// or the error that stopped the run.
//
// Its loop runs the commonest instructions on the commonest values
// (numbers, locals, jumps, calls of closures) right here, and hands any
// other to step, which runs it in full. The split is for speed: in Go a call
// can change every register, so a loop whose cases call functions has to
// save its variables to memory at each instruction. Here no case calls
// anything; after step, the variables are read again from the frame, so none
// of them has to survive the call either. A case either does its whole
// instruction and continues, or changes nothing and leaves it to step.
func (vm *VM) run() object.Value {
	var (
		stack     []object.Value
		sp, ip    int
		bp        int // where the frame's locals start
		fr        *frame
		ins       []uint32
		constants []object.Value
		free      []*Cell
	)

load:
	stack, sp = vm.stack, vm.sp
	fr = &vm.frames[len(vm.frames)-1]
	ins, constants, free = fr.code, fr.cl.Fn.Unit.Values, fr.cl.Free
	ip, bp = fr.ip, fr.bp

	for {
		switch code.Opcode(ins[ip]) {
		case code.OpConstant:
			stack[sp] = constants[int(ins[ip+1])]
			sp++
			ip += 2
			continue

		case code.OpNull:
			stack[sp] = nullValue
			sp++
			ip++
			continue

		case code.OpTrue:
			stack[sp] = trueValue
			sp++
			ip++
			continue

		case code.OpFalse:
			stack[sp] = falseValue
			sp++
			ip++
			continue

		case code.OpPop:
			sp--
			stack[sp] = object.Value{}
			ip++
			continue

		case code.OpGetLocal:
			stack[sp] = stack[bp+int(ins[ip+1])]
			sp++
			ip += 2
			continue

		case code.OpSetLocal:
			sp--
			stack[bp+int(ins[ip+1])] = stack[sp]
			stack[sp] = object.Value{}
			ip += 2
			continue

		case code.OpAssignLocal:
			stack[bp+int(ins[ip+1])] = stack[sp-1]
			ip += 2
			continue

		case code.OpGetCell:
			stack[sp] = cellAt(stack[bp+int(ins[ip+1])]).Value
			sp++
			ip += 2
			continue

		case code.OpSetCell:
			sp--
			cellAt(stack[bp+int(ins[ip+1])]).Value = stack[sp]
			stack[sp] = object.Value{}
			ip += 2
			continue

		case code.OpAssignCell:
			cellAt(stack[bp+int(ins[ip+1])]).Value = stack[sp-1]
			ip += 2
			continue

		case code.OpGetFree:
			stack[sp] = free[int(ins[ip+1])].Value
			sp++
			ip += 2
			continue

		case code.OpAssignFree:
			free[int(ins[ip+1])].Value = stack[sp-1]
			ip += 2
			continue

		case code.OpGetName:
			// A global or a builtin read before, unchanged since.
			n := &fr.cl.Fn.Unit.Names[int(ins[ip+1])]
			globals, memo := fr.cl.Globals, &n.Global
			version, all := globals.OwnVersion()
			if len(n.Checked) != 0 || n.Final >= 0 || memo.Env != globals || !all ||
				memo.Version != version || memo.Resets != evaluator.LibraryResets() {
				break
			}
			stack[sp] = n.Global.Value
			sp++
			ip += 2
			continue

		case code.OpGetLocalField, code.OpGetCellField, code.OpGetNameField, code.OpGetField:
			// A dot access on a hash it read before, unchanged since, or on
			// a point: its value is right here. The value read comes from
			// a local, a cell, a remembered global, or the stack.
			op := code.Opcode(ins[ip])
			unit := fr.cl.Fn.Unit
			var value object.Value
			var field *code.Field
			switch op {
			case code.OpGetLocalField:
				value, field = stack[bp+int(ins[ip+1])], &unit.Fields[int(ins[ip+2])]
			case code.OpGetCellField:
				value, field = cellAt(stack[bp+int(ins[ip+1])]).Value, &unit.Fields[int(ins[ip+2])]
			case code.OpGetNameField:
				n := &unit.Names[int(ins[ip+1])]
				globals, memo := fr.cl.Globals, &n.Global
				version, all := globals.OwnVersion()
				if len(n.Checked) == 0 && n.Final < 0 && memo.Env == globals && all &&
					memo.Version == version && memo.Resets == evaluator.LibraryResets() {
					value, field = memo.Value, &unit.Fields[int(ins[ip+2])]
				}
			default:
				value, field = stack[sp-1], &unit.Fields[int(ins[ip+1])]
			}
			if field == nil {
				break
			}
			var result object.Value
			if h, ok := value.Ref().(*object.Hash); ok && field.Cache.Hash == h && field.Cache.Version == h.Version() {
				result = field.Cache.Value
			} else if p, ok := value.Ref().(*object.Point); ok && field.Coord >= 0 && field.Coord < len(p.Coords) {
				// A coordinate is an integer or a float.
				switch c := p.Coords[field.Coord].(type) {
				case *object.Float:
					result = object.ValueOfFloat(c)
				case *object.Integer:
					result = object.ValueOfInteger(c)
				default:
					result = object.ObjectValue(c)
				}
			} else {
				break
			}
			if op == code.OpGetField {
				stack[sp-1] = result
				ip += 3
			} else {
				stack[sp] = result
				sp++
				ip += 4
			}
			continue

		case code.OpCheckCoord:
			if !stack[sp-1].IsNumber() {
				break
			}
			ip += 2
			continue

		case code.OpAnd, code.OpOr:
			// For numbers, booleans and null; step knows the rest.
			v := stack[sp-1]
			var truth bool
			if v.IsInt() {
				truth = v.Int() != 0
			} else if v.IsFloat() {
				truth = v.Float() != 0
			} else if v.Ref() == evaluator.TRUE {
				truth = true
			} else if v.Ref() != evaluator.FALSE && v.Ref() != evaluator.NULL {
				break
			}
			if op := code.Opcode(ins[ip]); op == code.OpAnd && !truth {
				stack[sp-1] = falseValue
				ip = int(ins[ip+1])
			} else if op == code.OpOr && truth {
				stack[sp-1] = trueValue
				ip = int(ins[ip+1])
			} else {
				sp--
				stack[sp] = object.Value{}
				ip += 2
			}
			continue

		case code.OpToBool:
			v := stack[sp-1]
			var truth bool
			if v.IsInt() {
				truth = v.Int() != 0
			} else if v.IsFloat() {
				truth = v.Float() != 0
			} else if v.Ref() == evaluator.TRUE {
				truth = true
			} else if v.Ref() != evaluator.FALSE && v.Ref() != evaluator.NULL {
				break
			}
			stack[sp-1] = boolValue(truth)
			ip++
			continue

		case opIndexHash:
			// An element of an array, or a string key that this index read
			// in the same hash before, unchanged since (see code.IndexCache).
			index := stack[sp-1]
			if array, ok := stack[sp-2].Ref().(*object.Array); ok {
				if !index.IsInt() || index.Int() < 0 || index.Int() >= int64(len(array.Elements)) {
					break
				}
				switch e := array.Elements[index.Int()].(type) {
				case *object.Integer:
					stack[sp-2] = object.ValueOfInteger(e)
				case *object.Float:
					stack[sp-2] = object.ValueOfFloat(e)
				default:
					stack[sp-2] = object.ObjectValue(e)
				}
			} else {
				h, ok := stack[sp-2].Ref().(*object.Hash)
				key, isString := index.Ref().(*object.String)
				cache := fr.cl.Fn.Indexes[int(ins[ip+2])]
				if !ok || !isString || cache == nil {
					break
				}
				hash := key.KnownHash()
				if hash == 0 {
					break
				}
				e := &cache[indexSlot(hash)]
				if e.Hash != h || e.Version != h.Version() || e.KeyHash != hash || len(e.Key) != len(key.Value) {
					break
				}
				// The same text: the same bytes, or equal ones.
				same := unsafe.StringData(e.Key) == unsafe.StringData(key.Value)
				if !same {
					same = true
					for i := 0; same && i < len(e.Key); i++ {
						same = e.Key[i] == key.Value[i]
					}
				}
				if !same {
					break
				}
				stack[sp-2] = e.Value
			}
			sp--
			stack[sp] = object.Value{}
			ip += 3
			continue

		case code.OpJump:
			ip = int(ins[ip+1])
			continue

		case code.OpJumpNotTruthy:
			v := stack[sp-1]
			var truth bool
			if v.IsInt() {
				truth = v.Int() != 0
			} else if v.IsFloat() {
				truth = v.Float() != 0
			} else if v.Ref() == evaluator.TRUE {
				truth = true
			} else if v.Ref() != evaluator.FALSE && v.Ref() != evaluator.NULL {
				// Strings, arrays, hashes: step knows them.
				break
			}
			sp--
			stack[sp] = object.Value{}
			if truth {
				ip += 2
			} else {
				ip = int(ins[ip+1])
			}
			continue

		case code.OpStep:
			if evaluator.MaxSteps != 0 {
				break
			}
			ip += 2
			continue

		case code.OpStatement:
			if evaluator.OnStatement != nil {
				break
			}
			ip += 2
			continue

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
			continue

		// The fused instructions of quickened code (see quicken.go).

		case opGetLocals:
			stack[sp] = stack[bp+int(ins[ip+1])]
			stack[sp+1] = stack[bp+int(ins[ip+2])]
			sp += 2
			ip += 3
			continue

		case opGetLocalConst:
			stack[sp] = stack[bp+int(ins[ip+1])]
			stack[sp+1] = constants[int(ins[ip+2])]
			sp += 2
			ip += 3
			continue

		case opLoop:
			if evaluator.MaxSteps != 0 {
				break
			}
			ip = int(ins[ip+2])
			continue

		case opSetFree:
			sp--
			free[int(ins[ip+1])].Value = stack[sp]
			stack[sp] = object.Value{}
			ip += 2
			continue

		// Operators on numbers, in every form quickening makes (see
		// quicken.go). Each is written out in full: Go inlines almost nothing
		// into a function this big, and a call would cost the loop its
		// registers. Anything that is not a number, and any result that would
		// be an error, goes to step.

		case code.OpAdd:
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() + b.Int())
			} else if f := a.Number() + b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-2] = result
			sp--
			ip += 2
			continue

		case code.OpSub:
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() - b.Int())
			} else if f := a.Number() - b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-2] = result
			sp--
			ip += 2
			continue

		case code.OpMul:
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() * b.Int())
			} else if f := a.Number() * b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-2] = result
			sp--
			ip += 2
			continue

		case code.OpDiv:
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() && b.Int() != 0 {
				result = object.IntValue(a.Int() / b.Int())
			} else if f := a.Number() / b.Number(); a.IsNumber() && b.IsNumber() && !(a.IsInt() && b.IsInt()) && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-2] = result
			sp--
			ip += 2
			continue

		case code.OpMod:
			a, b := stack[sp-2], stack[sp-1]
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(x - q*y)
			stack[sp-2] = result
			sp--
			ip += 2
			continue

		case code.OpFloorDiv:
			a, b := stack[sp-2], stack[sp-1]
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(q)
			stack[sp-2] = result
			sp--
			ip += 2
			continue

		case opAddX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() + b.Int())
			} else if f := a.Number() + b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp] = result
			sp++
			ip += 4
			continue

		case opSubX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() - b.Int())
			} else if f := a.Number() - b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp] = result
			sp++
			ip += 4
			continue

		case opMulX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() * b.Int())
			} else if f := a.Number() * b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp] = result
			sp++
			ip += 4
			continue

		case opDivX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var result object.Value
			if a.IsInt() && b.IsInt() && b.Int() != 0 {
				result = object.IntValue(a.Int() / b.Int())
			} else if f := a.Number() / b.Number(); a.IsNumber() && b.IsNumber() && !(a.IsInt() && b.IsInt()) && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp] = result
			sp++
			ip += 4
			continue

		case opModX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(x - q*y)
			stack[sp] = result
			sp++
			ip += 4
			continue

		case opFloorDivX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(q)
			stack[sp] = result
			sp++
			ip += 4
			continue

		case opAddS:
			a, b := stack[sp-1], source(stack, bp, constants, int(ins[ip+1]))
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() + b.Int())
			} else if f := a.Number() + b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-1] = result
			ip += 3
			continue

		case opSubS:
			a, b := stack[sp-1], source(stack, bp, constants, int(ins[ip+1]))
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() - b.Int())
			} else if f := a.Number() - b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-1] = result
			ip += 3
			continue

		case opMulS:
			a, b := stack[sp-1], source(stack, bp, constants, int(ins[ip+1]))
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() * b.Int())
			} else if f := a.Number() * b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-1] = result
			ip += 3
			continue

		case opDivS:
			a, b := stack[sp-1], source(stack, bp, constants, int(ins[ip+1]))
			var result object.Value
			if a.IsInt() && b.IsInt() && b.Int() != 0 {
				result = object.IntValue(a.Int() / b.Int())
			} else if f := a.Number() / b.Number(); a.IsNumber() && b.IsNumber() && !(a.IsInt() && b.IsInt()) && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[sp-1] = result
			ip += 3
			continue

		case opModS:
			a, b := stack[sp-1], source(stack, bp, constants, int(ins[ip+1]))
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(x - q*y)
			stack[sp-1] = result
			ip += 3
			continue

		case opFloorDivS:
			a, b := stack[sp-1], source(stack, bp, constants, int(ins[ip+1]))
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(q)
			stack[sp-1] = result
			ip += 3
			continue

		case opAddSet:
			// Both were numbers: the slots left need no clearing.
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() + b.Int())
			} else if f := a.Number() + b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[bp+int(ins[ip+1])] = result
			sp -= 2
			ip += 3
			continue

		case opSubSet:
			// Both were numbers: the slots left need no clearing.
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() - b.Int())
			} else if f := a.Number() - b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[bp+int(ins[ip+1])] = result
			sp -= 2
			ip += 3
			continue

		case opMulSet:
			// Both were numbers: the slots left need no clearing.
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() * b.Int())
			} else if f := a.Number() * b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[bp+int(ins[ip+1])] = result
			sp -= 2
			ip += 3
			continue

		case opDivSet:
			// Both were numbers: the slots left need no clearing.
			a, b := stack[sp-2], stack[sp-1]
			var result object.Value
			if a.IsInt() && b.IsInt() && b.Int() != 0 {
				result = object.IntValue(a.Int() / b.Int())
			} else if f := a.Number() / b.Number(); a.IsNumber() && b.IsNumber() && !(a.IsInt() && b.IsInt()) && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[bp+int(ins[ip+1])] = result
			sp -= 2
			ip += 3
			continue

		case opModSet:
			// Both were numbers: the slots left need no clearing.
			a, b := stack[sp-2], stack[sp-1]
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(x - q*y)
			stack[bp+int(ins[ip+1])] = result
			sp -= 2
			ip += 3
			continue

		case opFloorDivSet:
			// Both were numbers: the slots left need no clearing.
			a, b := stack[sp-2], stack[sp-1]
			if !a.IsInt() || !b.IsInt() || b.Int() == 0 {
				break
			}
			// The quotient rounded down, like floorDiv.
			x, y := a.Int(), b.Int()
			q := x / y
			if x%y != 0 && (x < 0) != (y < 0) {
				q--
			}
			result := object.IntValue(q)
			stack[bp+int(ins[ip+1])] = result
			sp -= 2
			ip += 3
			continue

		case opIncLocal:
			slot := bp + int(ins[ip+1])
			a, b := stack[slot], constants[int(ins[ip+2])]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() + b.Int())
			} else if f := a.Number() + b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[slot] = result
			ip += 4
			continue

		case opDecLocal:
			slot := bp + int(ins[ip+1])
			a, b := stack[slot], constants[int(ins[ip+2])]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() - b.Int())
			} else if f := a.Number() - b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			stack[slot] = result
			ip += 4
			continue

		case opIncFree:
			cell := free[int(ins[ip+1])]
			a, b := cell.Value, constants[int(ins[ip+2])]
			var result object.Value
			if a.IsInt() && b.IsInt() {
				result = object.IntValue(a.Int() + b.Int())
			} else if f := a.Number() + b.Number(); a.IsNumber() && b.IsNumber() && f-f == 0 {
				result = object.FloatValue(f)
			} else {
				break
			}
			cell.Value = result
			ip += 4
			continue

		case code.OpLess:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() < b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() < b.Number()
			} else {
				break
			}
			stack[sp-2] = boolValue(holds)
			sp--
			ip += 2
			continue

		case code.OpGreater:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() > b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() > b.Number()
			} else {
				break
			}
			stack[sp-2] = boolValue(holds)
			sp--
			ip += 2
			continue

		case code.OpLessEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() <= b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() <= b.Number()
			} else {
				break
			}
			stack[sp-2] = boolValue(holds)
			sp--
			ip += 2
			continue

		case code.OpGreaterEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() >= b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() >= b.Number()
			} else {
				break
			}
			stack[sp-2] = boolValue(holds)
			sp--
			ip += 2
			continue

		case code.OpEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() == b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() == b.Number()
			} else if x, y := a.Ref(), b.Ref(); !a.IsNumber() && !b.IsNumber() &&
				(x == evaluator.NULL || x == evaluator.TRUE || x == evaluator.FALSE) &&
				(y == evaluator.NULL || y == evaluator.TRUE || y == evaluator.FALSE) {
				holds = ((x == evaluator.NULL && y == evaluator.NULL) || (x == evaluator.TRUE && y == evaluator.TRUE) ||
					(x == evaluator.FALSE && y == evaluator.FALSE))
			} else if x, ok := a.Ref().(*object.String); ok {
				// Two strings, byte by byte: a call would cost more than
				// the short strings compared the most.
				y, ok := b.Ref().(*object.String)
				if !ok {
					break
				}
				same := len(x.Value) == len(y.Value)
				for i := 0; same && i < len(x.Value); i++ {
					same = x.Value[i] == y.Value[i]
				}
				holds = same
			} else {
				break
			}
			stack[sp-2] = boolValue(holds)
			sp--
			ip += 2
			continue

		case code.OpNotEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() != b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() != b.Number()
			} else if x, y := a.Ref(), b.Ref(); !a.IsNumber() && !b.IsNumber() &&
				(x == evaluator.NULL || x == evaluator.TRUE || x == evaluator.FALSE) &&
				(y == evaluator.NULL || y == evaluator.TRUE || y == evaluator.FALSE) {
				holds = !((x == evaluator.NULL && y == evaluator.NULL) || (x == evaluator.TRUE && y == evaluator.TRUE) ||
					(x == evaluator.FALSE && y == evaluator.FALSE))
			} else if x, ok := a.Ref().(*object.String); ok {
				// Two strings, byte by byte: a call would cost more than
				// the short strings compared the most.
				y, ok := b.Ref().(*object.String)
				if !ok {
					break
				}
				same := len(x.Value) == len(y.Value)
				for i := 0; same && i < len(x.Value); i++ {
					same = x.Value[i] == y.Value[i]
				}
				holds = !same
			} else {
				break
			}
			stack[sp-2] = boolValue(holds)
			sp--
			ip += 2
			continue

		case opJumpUnlessLess:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() < b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() < b.Number()
			} else {
				break
			}
			sp -= 2
			if holds {
				ip += 3
			} else {
				ip = int(ins[ip+2])
			}
			continue

		case opJumpUnlessGreater:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() > b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() > b.Number()
			} else {
				break
			}
			sp -= 2
			if holds {
				ip += 3
			} else {
				ip = int(ins[ip+2])
			}
			continue

		case opJumpUnlessLessEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() <= b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() <= b.Number()
			} else {
				break
			}
			sp -= 2
			if holds {
				ip += 3
			} else {
				ip = int(ins[ip+2])
			}
			continue

		case opJumpUnlessGreaterEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() >= b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() >= b.Number()
			} else {
				break
			}
			sp -= 2
			if holds {
				ip += 3
			} else {
				ip = int(ins[ip+2])
			}
			continue

		case opJumpUnlessEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() == b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() == b.Number()
			} else if x, y := a.Ref(), b.Ref(); !a.IsNumber() && !b.IsNumber() &&
				(x == evaluator.NULL || x == evaluator.TRUE || x == evaluator.FALSE) &&
				(y == evaluator.NULL || y == evaluator.TRUE || y == evaluator.FALSE) {
				holds = ((x == evaluator.NULL && y == evaluator.NULL) || (x == evaluator.TRUE && y == evaluator.TRUE) ||
					(x == evaluator.FALSE && y == evaluator.FALSE))
			} else if x, ok := a.Ref().(*object.String); ok {
				// Two strings, byte by byte: a call would cost more than
				// the short strings compared the most.
				y, ok := b.Ref().(*object.String)
				if !ok {
					break
				}
				same := len(x.Value) == len(y.Value)
				for i := 0; same && i < len(x.Value); i++ {
					same = x.Value[i] == y.Value[i]
				}
				holds = same
			} else {
				break
			}
			sp -= 2
			if holds {
				ip += 3
			} else {
				ip = int(ins[ip+2])
			}
			continue

		case opJumpUnlessNotEqual:
			a, b := stack[sp-2], stack[sp-1]
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() != b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() != b.Number()
			} else if x, y := a.Ref(), b.Ref(); !a.IsNumber() && !b.IsNumber() &&
				(x == evaluator.NULL || x == evaluator.TRUE || x == evaluator.FALSE) &&
				(y == evaluator.NULL || y == evaluator.TRUE || y == evaluator.FALSE) {
				holds = !((x == evaluator.NULL && y == evaluator.NULL) || (x == evaluator.TRUE && y == evaluator.TRUE) ||
					(x == evaluator.FALSE && y == evaluator.FALSE))
			} else if x, ok := a.Ref().(*object.String); ok {
				// Two strings, byte by byte: a call would cost more than
				// the short strings compared the most.
				y, ok := b.Ref().(*object.String)
				if !ok {
					break
				}
				same := len(x.Value) == len(y.Value)
				for i := 0; same && i < len(x.Value); i++ {
					same = x.Value[i] == y.Value[i]
				}
				holds = !same
			} else {
				break
			}
			sp -= 2
			if holds {
				ip += 3
			} else {
				ip = int(ins[ip+2])
			}
			continue

		case opJumpUnlessLessX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() < b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() < b.Number()
			} else {
				break
			}
			if holds {
				ip += 5
			} else {
				ip = int(ins[ip+4])
			}
			continue

		case opJumpUnlessGreaterX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() > b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() > b.Number()
			} else {
				break
			}
			if holds {
				ip += 5
			} else {
				ip = int(ins[ip+4])
			}
			continue

		case opJumpUnlessLessEqualX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() <= b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() <= b.Number()
			} else {
				break
			}
			if holds {
				ip += 5
			} else {
				ip = int(ins[ip+4])
			}
			continue

		case opJumpUnlessGreaterEqualX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() >= b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() >= b.Number()
			} else {
				break
			}
			if holds {
				ip += 5
			} else {
				ip = int(ins[ip+4])
			}
			continue

		case opJumpUnlessEqualX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() == b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() == b.Number()
			} else if x, y := a.Ref(), b.Ref(); !a.IsNumber() && !b.IsNumber() &&
				(x == evaluator.NULL || x == evaluator.TRUE || x == evaluator.FALSE) &&
				(y == evaluator.NULL || y == evaluator.TRUE || y == evaluator.FALSE) {
				holds = ((x == evaluator.NULL && y == evaluator.NULL) || (x == evaluator.TRUE && y == evaluator.TRUE) ||
					(x == evaluator.FALSE && y == evaluator.FALSE))
			} else if x, ok := a.Ref().(*object.String); ok {
				// Two strings, byte by byte: a call would cost more than
				// the short strings compared the most.
				y, ok := b.Ref().(*object.String)
				if !ok {
					break
				}
				same := len(x.Value) == len(y.Value)
				for i := 0; same && i < len(x.Value); i++ {
					same = x.Value[i] == y.Value[i]
				}
				holds = same
			} else {
				break
			}
			if holds {
				ip += 5
			} else {
				ip = int(ins[ip+4])
			}
			continue

		case opJumpUnlessNotEqualX:
			a, b := source(stack, bp, constants, int(ins[ip+1])), source(stack, bp, constants, int(ins[ip+2]))
			var holds bool
			if a.IsInt() && b.IsInt() {
				holds = a.Int() != b.Int()
			} else if a.IsNumber() && b.IsNumber() {
				holds = a.Number() != b.Number()
			} else if x, y := a.Ref(), b.Ref(); !a.IsNumber() && !b.IsNumber() &&
				(x == evaluator.NULL || x == evaluator.TRUE || x == evaluator.FALSE) &&
				(y == evaluator.NULL || y == evaluator.TRUE || y == evaluator.FALSE) {
				holds = !((x == evaluator.NULL && y == evaluator.NULL) || (x == evaluator.TRUE && y == evaluator.TRUE) ||
					(x == evaluator.FALSE && y == evaluator.FALSE))
			} else if x, ok := a.Ref().(*object.String); ok {
				// Two strings, byte by byte: a call would cost more than
				// the short strings compared the most.
				y, ok := b.Ref().(*object.String)
				if !ok {
					break
				}
				same := len(x.Value) == len(y.Value)
				for i := 0; same && i < len(x.Value); i++ {
					same = x.Value[i] == y.Value[i]
				}
				holds = !same
			} else {
				break
			}
			if holds {
				ip += 5
			} else {
				ip = int(ins[ip+4])
			}
			continue

		case code.OpMinus:
			v := stack[sp-1]
			if v.IsInt() {
				stack[sp-1] = object.IntValue(-v.Int())
			} else if v.IsFloat() {
				stack[sp-1] = object.FloatValue(-v.Float())
			} else {
				break
			}
			ip += 2
			continue

		// Calls of closures and returns, when nothing special is going on:
		// no step limit, no profiler, no statement hook for function bodies,
		// room on the stacks, no layer to close.

		case code.OpCall:
			count := int(ins[ip+1])
			callee := stack[sp-1-count].Ref()
			if b, ok := callee.(*object.Builtin); ok && b == lenBuiltin && count == 1 && evaluator.MaxSteps == 0 && profiler == nil {
				// len itself, with nothing to count: its value is right here.
				n := -1
				switch arg := stack[sp-1].Ref().(type) {
				case *object.Array:
					n = len(arg.Elements)
				case *object.Hash:
					n = len(arg.Pairs)
				case *object.String:
					n = arg.KnownLen()
				}
				if n >= 0 {
					sp--
					stack[sp] = object.Value{}
					stack[sp-1] = object.IntValue(int64(n))
					ip += 3
					continue
				}
			}
			next, ok := callee.(*Closure)
			if !ok || evaluator.MaxSteps != 0 || profiler != nil || evaluator.OnStatement != nil && evaluator.HookFunctions || next.Fn.Bare == nil {
				break
			}
			fn := next.Fn
			base := sp - count
			top := base + fn.NumLocals + fn.MaxStack
			if count < fn.Required || count > fn.NumParams || top > len(stack) ||
				len(vm.frames) == cap(vm.frames) || vm.depthBase+len(vm.frames) >= evaluator.MaxCallDepth {
				break
			}
			vm.used = max(vm.used, top)
			for i := sp; i < base+fn.NumLocals; i++ {
				stack[i] = object.Value{}
			}
			fr.ip = ip + 3
			vm.frames = vm.frames[:len(vm.frames)+1]
			fr = &vm.frames[len(vm.frames)-1]
			fr.cl, fr.code, fr.ip, fr.bp, fr.given, fr.profile = next, fn.Bare, 0, base, count, nil
			fr.handlers, fr.layers = len(vm.handlers), len(vm.layers)
			ins, constants, free = fn.Bare, fn.Unit.Values, next.Free
			sp, ip, bp = base+fn.NumLocals, 0, base
			continue

		case code.OpReturnValue:
			if len(vm.frames) == 1 || fr.profile != nil || len(vm.layers) > fr.layers {
				break
			}
			value := stack[sp-1]
			vm.handlers = vm.handlers[:fr.handlers]
			for i := bp; i < sp; i++ {
				stack[i] = object.Value{}
			}
			sp = bp
			stack[sp-1] = value
			vm.frames = vm.frames[:len(vm.frames)-1]
			fr = &vm.frames[len(vm.frames)-1]
			ins, constants, free = fr.code, fr.cl.Fn.Unit.Values, fr.cl.Free
			ip, bp = fr.ip, fr.bp
			continue
		}

		vm.sp, fr.ip = sp, ip
		if value, done := vm.step(); done {
			return value
		}
		goto load
	}
}
