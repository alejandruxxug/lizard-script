package vm

import (
	"lizardscript/ast"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// A run keeps three stacks next to the value stack: the frames of the calls
// being run, the trys open (handlers) and the layers open. A frame remembers
// how many handlers and layers were open when it started, so leaving it
// (return) closes the ones it opened, and a handler remembers its frame and
// stack height, so an error can go back to it from any depth.

// frame is a call being run.
type frame struct {
	cl    *Closure
	ip    int
	bp    int // where its locals start in the stack
	given int // how many arguments the call gave
	// the handlers and layers open when it started
	handlers, layers int
	// profile is the profile that counts this call, nil when it is off.
	profile *profileState
}

// handler is an open try.
type handler struct {
	frame  int // the frame of the try
	sp     int // the stack height at the try
	catch  int // where its catch code starts
	layers int // the layers open at the try
}

// layer is an open layer statement. hooked says whether OnLayer was told it
// started, and so must be told it ended.
type layer struct {
	node   *ast.LayerStatement
	hooked bool
}

// openLayer starts a layer and tells OnLayer, if it is set now.
func (vm *VM) openLayer(node *ast.LayerStatement) {
	hooked := evaluator.OnLayer != nil
	vm.layers = append(vm.layers, layer{node: node, hooked: hooked})
	if hooked {
		evaluator.OnLayer(node.Name, true, node.Token)
	}
}

// closeLayers ends the layers above depth, innermost first, telling OnLayer.
func (vm *VM) closeLayers(depth int) {
	for len(vm.layers) > depth {
		l := vm.layers[len(vm.layers)-1]
		vm.layers = vm.layers[:len(vm.layers)-1]
		if l.hooked && evaluator.OnLayer != nil {
			evaluator.OnLayer(l.node.Name, false, l.node.Token)
		}
	}
}

// throw sends err to the innermost try of this run that catches it, and
// reports whether there was one: its frame is the current one then, with
// the stack as it was at the try plus the error's hash. Otherwise every
// layer of the run ends and the run stops with err.
func (vm *VM) throw(err *object.Error) bool {
	if !err.Uncatchable && len(vm.handlers) > 0 {
		h := vm.handlers[len(vm.handlers)-1]
		vm.handlers = vm.handlers[:len(vm.handlers)-1]
		vm.closeLayers(h.layers)
		vm.leaveFrames(h.frame + 1)
		vm.frames = vm.frames[:h.frame+1]
		vm.frames[h.frame].ip = h.catch
		vm.sp = h.sp
		vm.stack[vm.sp] = evaluator.ErrorHash(err)
		vm.sp++
		return true
	}
	vm.closeLayers(0)
	vm.leaveFrames(0)
	vm.frames = vm.frames[:0]
	vm.handlers = vm.handlers[:0]
	return false
}

// pushFrame starts a call of cl whose count arguments are the top of the
// stack: they become its first locals, and the rest of its locals start
// empty (unset) above them.
func (vm *VM) pushFrame(cl *Closure, count int) {
	fn := cl.Fn
	base := vm.sp - count
	vm.ensure(base + fn.NumLocals + fn.MaxStack)
	clear(vm.stack[vm.sp : base+fn.NumLocals])
	vm.sp = base + fn.NumLocals
	vm.frames = append(vm.frames, frame{
		cl:       cl,
		bp:       base,
		given:    count,
		handlers: len(vm.handlers),
		layers:   len(vm.layers),
		profile:  profiler,
	})
	if profiler != nil {
		profiler.enterFunction(fn)
	}
}
