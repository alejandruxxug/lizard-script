package vm

import (
	"lizardscript/evaluator"
	"lizardscript/object"
)

// toCaller moves err, on its way out of the frames a throw leaves, to the
// call into the file that raised it, when it asks for that (see
// evaluator.AtCaller): the innermost call of those frames made from
// another file. The frames left are the ones above the try that catches
// err, or all of them.
func (vm *VM) toCaller(err *object.Error) {
	if !err.AtCaller {
		return
	}
	stop := 0
	if !err.Uncatchable && len(vm.handlers) > 0 {
		stop = vm.handlers[len(vm.handlers)-1].frame
	}
	for i := len(vm.frames) - 1; i > stop; i-- {
		caller := &vm.frames[i-1]
		// The caller's ip is just past its OpCall, whose last operand is
		// the token of the call.
		tok := caller.cl.Fn.Unit.Tokens[int(caller.code[caller.ip-1])]
		if evaluator.AtCaller(err, tok) {
			return
		}
	}
}
