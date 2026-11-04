package vm

import (
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/engine/compiler"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
)

// moduleUnits keeps the code compiled for the programs of modules. The
// evaluator gives a module file the same program for as long as its text
// stays the same (see evaluator.Use), so the VM compiles each module once,
// not once per run.
var moduleUnits = map[*ast.Program]*code.Unit{}

// maxModuleUnits is how many compiled modules are kept; past it they are
// all forgotten, which drops the ones of files that changed.
const maxModuleUnits = 256

// RunModule is the engine use runs a module on: Run, with the code
// compiled for the program before when there is one.
func RunModule(program *ast.Program, env *object.Environment) object.Object {
	unit, ok := moduleUnits[program]
	if !ok {
		var err error
		if unit, err = compiler.Compile(program); err != nil {
			return &object.Error{Message: "compile error: " + err.Error()}
		}
		if len(moduleUnits) >= maxModuleUnits {
			clear(moduleUnits)
		}
		moduleUnits[program] = unit
	}
	return RunUnit(unit, env)
}
