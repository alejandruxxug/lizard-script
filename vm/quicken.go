package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
	"slices"
)

// Quickening. The compiler writes plain stack instructions, one per step of
// the evaluator. Before a function first runs, the VM rewrites them into its
// own form (code.Function.Quick and Bare), which does the same steps with
// fewer instructions, and so fewer trips through the dispatch:
//
//   - Runs of instructions that come up all the time become one: an
//     operator on two locals or constants (n - 1, i % 7, x * 0.5), an
//     operator on the top value and one (... + 1.5), an operator whose
//     result goes straight into a local (t = a + b), a comparison of two
//     locals or constants and the jump that tests it (i < n), x += k on a
//     local or a captured variable, two locals pushed together (f(a, b)),
//     an assignment and the pop that drops its value, and a loop's jump
//     back with the step counted at its top.
//   - An index (container[key]) gets a cache of its own, for the hashes
//     with string keys it reads (code.IndexCache): a table of constants
//     read with a token's type or a name is two comparisons, not a map
//     lookup.
//   - The instructions with nothing left to do go: OpSkipScope; OpFormula,
//     whose usual instructions make no objects either now that numbers are
//     unboxed, and quicken like any other; OpCheckKey after a constant that
//     is a valid key; and, in the code for when no OnStatement hook is set
//     (see quick), the hook's OpStatement.
//
// A run is only fused when no jump lands inside it, and every jump is moved
// to where its target went. Quickened code is words, not bytes: one for the
// opcode and one for each operand, so reading an operand is one load.

// The opcodes only quickened code has, after the compiler's own. The
// operators with fused forms come in the order of arithmetic and
// comparisons, so each form is a block of opcodes.
const (
	// slot, slot: push two locals.
	opGetLocals = code.OpSetIndex + 1 + iota
	// slot, const: push a local and a constant.
	opGetLocalConst
	// tok, jump: count a step at tok and jump; a loop's way back to its top.
	opLoop
	// free: pop into a captured variable (OpAssignFree then OpPop).
	opSetFree
	// slot, const, tok: local += constant, or -=, as a statement.
	opIncLocal
	opDecLocal
	// free, const, tok: captured variable += constant, as a statement.
	opIncFree

	// a, b, tok: push a op b, where a and b are sources (see source).
	opAddX
	opSubX
	opMulX
	opDivX
	opModX
	opFloorDivX
	// b, tok: replace the top value t with t op b, b a source.
	opAddS
	opSubS
	opMulS
	opDivS
	opModS
	opFloorDivS
	// slot, tok: pop b and a, and store a op b in the local.
	opAddSet
	opSubSet
	opMulSet
	opDivSet
	opModSet
	opFloorDivSet

	// tok, jump: pop b and a, and jump unless a cmp b.
	opJumpUnlessLess
	opJumpUnlessGreater
	opJumpUnlessLessEqual
	opJumpUnlessGreaterEqual
	opJumpUnlessEqual
	opJumpUnlessNotEqual
	// a, b, tok, jump: jump unless a cmp b, where a and b are sources.
	opJumpUnlessLessX
	opJumpUnlessGreaterX
	opJumpUnlessLessEqualX
	opJumpUnlessGreaterEqualX
	opJumpUnlessEqualX
	opJumpUnlessNotEqualX

	// tok, index: OpIndex with its own code.IndexCache, fn.Indexes[index].
	opIndexHash
)

// quickNames are the names of the opcodes above, for messages and tests.
var quickNames = map[code.Opcode]string{
	opGetLocals:               "opGetLocals",
	opGetLocalConst:           "opGetLocalConst",
	opLoop:                    "opLoop",
	opSetFree:                 "opSetFree",
	opIncLocal:                "opIncLocal",
	opDecLocal:                "opDecLocal",
	opIncFree:                 "opIncFree",
	opAddX:                    "opAddX",
	opSubX:                    "opSubX",
	opMulX:                    "opMulX",
	opDivX:                    "opDivX",
	opModX:                    "opModX",
	opFloorDivX:               "opFloorDivX",
	opAddS:                    "opAddS",
	opSubS:                    "opSubS",
	opMulS:                    "opMulS",
	opDivS:                    "opDivS",
	opModS:                    "opModS",
	opFloorDivS:               "opFloorDivS",
	opAddSet:                  "opAddSet",
	opSubSet:                  "opSubSet",
	opMulSet:                  "opMulSet",
	opDivSet:                  "opDivSet",
	opModSet:                  "opModSet",
	opFloorDivSet:             "opFloorDivSet",
	opJumpUnlessLess:          "opJumpUnlessLess",
	opJumpUnlessGreater:       "opJumpUnlessGreater",
	opJumpUnlessLessEqual:     "opJumpUnlessLessEqual",
	opJumpUnlessGreaterEqual:  "opJumpUnlessGreaterEqual",
	opJumpUnlessEqual:         "opJumpUnlessEqual",
	opJumpUnlessNotEqual:      "opJumpUnlessNotEqual",
	opJumpUnlessLessX:         "opJumpUnlessLessX",
	opJumpUnlessGreaterX:      "opJumpUnlessGreaterX",
	opJumpUnlessLessEqualX:    "opJumpUnlessLessEqualX",
	opJumpUnlessGreaterEqualX: "opJumpUnlessGreaterEqualX",
	opJumpUnlessEqualX:        "opJumpUnlessEqualX",
	opJumpUnlessNotEqualX:     "opJumpUnlessNotEqualX",
	opIndexHash:               "opIndexHash",
}

// opName is the name of any opcode of quickened code.
func opName(op code.Opcode) string {
	if name, ok := quickNames[op]; ok {
		return name
	}
	if def, err := code.Lookup(byte(op)); err == nil {
		return def.Name
	}
	return "?"
}

// arithmetic and comparisons are the operators with fused forms, in the
// order of their blocks of opcodes.
var (
	arithmetic  = []code.Opcode{code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpFloorDiv}
	comparisons = []code.Opcode{code.OpLess, code.OpGreater, code.OpLessEqual, code.OpGreaterEqual, code.OpEqual, code.OpNotEqual}
)

// jumpOperands are the operands that are places in the code, by opcode.
var jumpOperands = func() map[code.Opcode][]int {
	jumps := map[code.Opcode][]int{
		code.OpJump:          {0},
		code.OpJumpNotTruthy: {0},
		code.OpAnd:           {0},
		code.OpOr:            {0},
		code.OpFormula:       {1},
		code.OpCheckSignal:   {0, 1, 2},
		code.OpPushHandler:   {0},
		code.OpJumpIfGiven:   {1},
		opLoop:               {1},
	}
	for i := range comparisons {
		jumps[opJumpUnlessLess+code.Opcode(i)] = []int{1}
		jumps[opJumpUnlessLessX+code.Opcode(i)] = []int{3}
	}
	return jumps
}()

// instruction is one decoded instruction: at is where it starts in the
// compiler's code.
type instruction struct {
	op       code.Opcode
	operands []int
	at       int
}

// quick is the quickened code a call of fn runs now: with the OnStatement
// hook's instructions while a hook is set for it (see hooked), without them
// (Bare) otherwise.
// Each is made the first time it is needed.
//
// A frame keeps the code it started with, so a hook set in the middle of
// a call is first seen by the calls made after it. Hosts set their hooks
// before a run (and lizard.run around the program it runs), when no frame
// of that run has started yet.
func quick(fn *code.Function) []uint32 {
	if hooked(fn) {
		if fn.Quick == nil {
			fn.Quick = quicken(fn, true)
		}
		return fn.Quick
	}
	if fn.Bare == nil {
		fn.Bare = quicken(fn, false)
	}
	return fn.Bare
}

// hooked says whether a call of fn starting now calls the OnStatement
// hook: while one is set, unless fn is a function (not a program) and
// evaluator.HookFunctions leaves function bodies out.
func hooked(fn *code.Function) bool {
	return evaluator.OnStatement != nil && (evaluator.HookFunctions || fn.Literal == nil)
}

// quicken rewrites the compiler's instructions of fn into the VM's, with or
// without the instructions of the OnStatement hook.
func quicken(fn *code.Function, hooks bool) []uint32 {
	ins := fn.Instructions
	q := &quickener{fn: fn, constants: fn.Unit.Constants, targets: map[int]bool{}, hooks: hooks}
	q.decode(ins)
	var out []instruction
	moved := make(map[int]int, len(q.list)+1) // compiler offset -> index in out
	for i := 0; i < len(q.list); {
		moved[q.list[i].at] = len(out)
		if fused, n := q.fuse(i); n > 0 {
			out = append(out, fused)
			i += n
			continue
		}
		if in, keep := q.alone(i); keep {
			out = append(out, in)
		}
		i++
	}
	moved[len(ins)] = len(out)
	return encode(out, moved)
}

// quickener holds the instructions of one function being quickened.
type quickener struct {
	fn        *code.Function
	hooks     bool // whether OpStatement stays
	list      []instruction
	targets   map[int]bool // where jumps land, by compiler offset
	steps     map[int]int  // the token of each OpStep, by compiler offset
	constants []object.Object
}

// decode reads every instruction of ins, the places jumps go to (but the
// OpFormula's, which goes) and the steps.
func (q *quickener) decode(ins code.Instructions) {
	q.steps = map[int]int{}
	for at := 0; at < len(ins); {
		op := code.Opcode(ins[at])
		def, err := code.Lookup(byte(op))
		if err != nil {
			panic("vm: can't quicken an unknown opcode")
		}
		operands, read := code.ReadOperands(def, ins[at+1:])
		q.list = append(q.list, instruction{op: op, operands: operands, at: at})
		if op != code.OpFormula {
			for _, j := range jumpOperands[op] {
				q.targets[operands[j]] = true
			}
		}
		if op == code.OpStep {
			q.steps[at] = operands[0]
		}
		at += 1 + read
	}
}

// anyOp matches any instruction in run; the caller checks which it is.
const anyOp code.Opcode = 255

// run reports whether the instructions from i have these opcodes and no
// jump lands after the first of them.
func (q *quickener) run(i int, ops ...code.Opcode) bool {
	if i+len(ops) > len(q.list) {
		return false
	}
	for k, op := range ops {
		in := q.list[i+k]
		if (op != anyOp && in.op != op) || (k > 0 && q.targets[in.at]) {
			return false
		}
	}
	return true
}

// source is instruction i as a source operand, when it pushes a local or a
// constant.
func (q *quickener) source(i int) (int, bool) {
	if i >= len(q.list) {
		return 0, false
	}
	switch in := q.list[i]; in.op {
	case code.OpGetLocal:
		return in.operands[0], true
	case code.OpConstant:
		return in.operands[0] | constantSource, true
	}
	return 0, false
}

// fuse is the run of instructions from i as one, and how many it took, or
// 0 when none starts there. Longer runs are tried first.
func (q *quickener) fuse(i int) (instruction, int) {
	op := func(k int) code.Opcode { return q.list[i+k].op }
	operand := func(k int) int { return q.list[i+k].operands[0] }
	fused := func(op code.Opcode, operands ...int) instruction { return instruction{op: op, operands: operands} }

	// x += k or x -= k on a local, x += k on a captured variable.
	if q.run(i, code.OpGetLocal, code.OpConstant, anyOp, code.OpAssignLocal, code.OpPop) && operand(0) == operand(3) {
		switch op(2) {
		case code.OpAdd:
			return fused(opIncLocal, operand(0), operand(1), operand(2)), 5
		case code.OpSub:
			return fused(opDecLocal, operand(0), operand(1), operand(2)), 5
		}
	}
	if q.run(i, code.OpGetFree, code.OpConstant, code.OpAdd, code.OpAssignFree, code.OpPop) && operand(0) == operand(3) {
		return fused(opIncFree, operand(0), operand(1), operand(2)), 5
	}

	a, aok := q.source(i)
	b, bok := q.source(i + 1)
	// a cmp b, and the jump that tests it.
	if aok && bok && q.run(i, anyOp, anyOp, anyOp, code.OpJumpNotTruthy) {
		if c := slices.Index(comparisons, op(2)); c >= 0 {
			return fused(opJumpUnlessLessX+code.Opcode(c), a, b, operand(2), operand(3)), 4
		}
	}
	// a op b.
	if aok && bok && q.run(i, anyOp, anyOp, anyOp) {
		if k := slices.Index(arithmetic, op(2)); k >= 0 {
			return fused(opAddX+code.Opcode(k), a, b, operand(2)), 3
		}
	}
	// An operator whose result is stored in a local and dropped.
	if q.run(i, anyOp, code.OpAssignLocal, code.OpPop) {
		if k := slices.Index(arithmetic, op(0)); k >= 0 {
			return fused(opAddSet+code.Opcode(k), operand(1), operand(0)), 3
		}
	}
	// The top value op a.
	if aok && q.run(i, anyOp, anyOp) {
		if k := slices.Index(arithmetic, op(1)); k >= 0 {
			return fused(opAddS+code.Opcode(k), a, operand(1)), 2
		}
	}

	if !q.run(i, anyOp, anyOp) {
		return instruction{}, 0
	}
	// A local before an operator on two sources stays alone, so the
	// operator reads them where they are.
	_, second := q.source(i + 1)
	_, third := q.source(i + 2)
	if second && third && q.run(i+1, anyOp, anyOp, anyOp) &&
		(slices.Contains(arithmetic, op(3)) || slices.Contains(comparisons, op(3))) {
		return instruction{}, 0
	}
	switch first, second := op(0), op(1); {
	case first == code.OpGetLocal && second == code.OpGetLocal:
		return fused(opGetLocals, operand(0), operand(1)), 2
	case first == code.OpGetLocal && second == code.OpConstant:
		return fused(opGetLocalConst, operand(0), operand(1)), 2
	case slices.Contains(comparisons, first) && second == code.OpJumpNotTruthy:
		return fused(opJumpUnlessLess+code.Opcode(slices.Index(comparisons, first)), operand(0), operand(1)), 2
	case first == code.OpAssignLocal && second == code.OpPop:
		return fused(code.OpSetLocal, operand(0)), 2
	case first == code.OpAssignCell && second == code.OpPop:
		return fused(code.OpSetCell, operand(0)), 2
	case first == code.OpAssignFree && second == code.OpPop:
		return fused(opSetFree, operand(0)), 2
	}
	return instruction{}, 0
}

// alone is instruction i quickened on its own, and whether it stays: the
// ones with nothing left to do go, and a jump back to a loop's step becomes
// the step and a jump past it.
func (q *quickener) alone(i int) (instruction, bool) {
	in := q.list[i]
	switch in.op {
	case code.OpSkipScope, code.OpFormula:
		return in, false
	case code.OpStatement:
		return in, q.hooks
	case code.OpCheckKey:
		// After a constant string, integer or boolean, which is a key.
		if prev := q.list[i-1]; prev.op == code.OpConstant && !q.targets[in.at] {
			_, hashable := q.constants[prev.operands[0]].(object.Hashable)
			return in, !hashable
		}
	case code.OpIndex:
		// With a cache of its own, for the hashes it reads.
		q.fn.Indexes = append(q.fn.Indexes, nil)
		return instruction{op: opIndexHash, operands: []int{in.operands[0], len(q.fn.Indexes) - 1}}, true
	case code.OpJump:
		if tok, ok := q.steps[in.operands[0]]; ok {
			return instruction{op: opLoop, operands: []int{tok, in.operands[0] + 5}}, true
		}
	}
	return in, true
}

// encode writes the quickened instructions as words, with every jump moved
// to where its target went.
func encode(list []instruction, moved map[int]int) []uint32 {
	// Where each instruction of list starts.
	starts := make([]int, len(list)+1)
	for i, in := range list {
		starts[i+1] = starts[i] + 1 + len(in.operands)
	}
	out := make([]uint32, 0, starts[len(list)])
	for _, in := range list {
		out = append(out, uint32(in.op))
		jumps := jumpOperands[in.op]
		for j, o := range in.operands {
			if slices.Contains(jumps, j) {
				o = starts[moved[o]]
			}
			out = append(out, uint32(o))
		}
	}
	return out
}

// constantSource marks an operand of a fused instruction that is a
// constant; without it the operand is a local's slot.
const constantSource = 1 << 31

// source is the value an operand of a fused instruction names.
func source(stack []object.Value, bp int, constants []object.Value, operand int) object.Value {
	if operand&constantSource != 0 {
		return constants[operand&^constantSource]
	}
	return stack[bp+operand]
}
