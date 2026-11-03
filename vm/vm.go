// Package vm runs lizard-script programs compiled to bytecode (packages
// code and compiler). It is a second engine next to the tree-walking
// evaluator, faster for hosts that run a lot of code, with the same
// results: the same values, output, errors (messages and positions), steps
// and hook calls. It shares the evaluator's settings and state: MaxSteps and
// the step count, Output, OnStatement and OnLayer, the builtins and the math,
// text, list and hash libraries.
//
// The two engines work together: a program run with Run can call functions
// a program run with evaluator.Eval left in the environment, and the other
// way round, and builtins that call functions (list.sort) take functions of
// either engine.
package vm

import (
	"lizardscript/ast"
	"lizardscript/code"
	"lizardscript/compiler"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// Run compiles and runs program in env, with the contract of
// evaluator.Eval(program, env): it returns the program's value, or an
// *object.Error. The program's top-level lets are defined in env, and names
// it doesn't declare are looked up in env when they are read, so a host can
// fill env before and read it after, and run more programs in it.
func Run(program *ast.Program, env *object.Environment) object.Object {
	unit, err := compiler.Compile(program)
	if err != nil {
		return &object.Error{Message: "compile error: " + err.Error()}
	}
	return RunUnit(unit, env)
}

// RunUnit runs a program compiled with compiler.Compile in env. A host that
// runs the same program many times can compile it once.
func RunUnit(unit *code.Unit, env *object.Environment) object.Object {
	vm := getVM()
	defer putVM(vm)
	// The program's own frame is not a call.
	vm.depthBase = evaluator.CallDepth() - 1
	vm.start(&Closure{Fn: unit.Main, Globals: env}, nil)
	return vm.run()
}

// VM is one run: of a program, or of one closure called from Go.
type VM struct {
	stack    []object.Object
	sp       int
	frames   []frame
	handlers []handler
	layers   []layer
	// depthBase plus the frames is the number of calls open, for the
	// call depth limit.
	depthBase int
}

// VMs are kept for the next run, since builtins like list.sort start one
// per call.
var pool []*VM

func getVM() *VM {
	if n := len(pool); n > 0 {
		vm := pool[n-1]
		pool = pool[:n-1]
		return vm
	}
	return &VM{stack: make([]object.Object, 256)}
}

func putVM(vm *VM) {
	if len(vm.stack) > 1<<14 || len(pool) >= 16 {
		return
	}
	clear(vm.stack)
	vm.frames = vm.frames[:0]
	vm.handlers = vm.handlers[:0]
	vm.layers = vm.layers[:0]
	pool = append(pool, vm)
}

// start makes the first frame, for cl called with args.
func (vm *VM) start(cl *Closure, args []object.Object) {
	fn := cl.Fn
	vm.ensure(fn.NumLocals + fn.MaxStack)
	copy(vm.stack, args)
	clear(vm.stack[len(args):fn.NumLocals])
	vm.sp = fn.NumLocals
	vm.frames = append(vm.frames[:0], frame{cl: cl, given: len(args), profile: profiler})
	if profiler != nil {
		profiler.enterFunction(fn)
	}
}

// ensure grows the stack to at least size slots.
func (vm *VM) ensure(size int) {
	if size <= len(vm.stack) {
		return
	}
	grown := make([]object.Object, max(size, 2*len(vm.stack)))
	copy(grown, vm.stack)
	vm.stack = grown
}
