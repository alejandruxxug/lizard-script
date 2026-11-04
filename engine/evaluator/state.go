package evaluator

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"io"
	"math/rand"
)

// RunState is what the evaluator keeps for the program that is running:
// where puts writes, the step limit and count, the hooks (and the function
// bodies running, for HookFunctions), the random
// numbers, the libraries and the modules (both engines share all of it). A
// host that runs a program inside another one, like a sandbox, saves it,
// starts the inner run from scratch, and restores it after, so the outer
// program goes on as if nothing else had run.
type RunState struct {
	output      io.Writer
	maxSteps    int
	steps       int
	onStatement func(node ast.Statement)
	onLayer     func(name string, enter bool, tok token.Token)
	hookBodies  bool
	bodies      int
	random      *rand.Rand
	libraries   [4]*object.Hash
	modules     *moduleState
	prelude     func(env *object.Environment)
}

// SaveRunState returns the state of the run now.
func SaveRunState() RunState {
	return RunState{
		output:      Output,
		maxSteps:    MaxSteps,
		steps:       steps,
		onStatement: OnStatement,
		onLayer:     OnLayer,
		hookBodies:  HookFunctions,
		bodies:      functionBodies,
		random:      random,
		libraries:   [4]*object.Hash{mathHash, textHash, listHash, hashLibraryHash},
		modules:     modules,
		prelude:     ModulePrelude,
	}
}

// Restore puts the state back as it was saved.
func (s RunState) Restore() {
	Output, MaxSteps, steps = s.output, s.maxSteps, s.steps
	OnStatement, OnLayer = s.onStatement, s.onLayer
	HookFunctions, functionBodies = s.hookBodies, s.bodies
	random = s.random
	mathHash, textHash, listHash, hashLibraryHash = s.libraries[0], s.libraries[1], s.libraries[2], s.libraries[3]
	modules, ModulePrelude = s.modules, s.prelude
	// What a name of a library finds changed back: an engine that
	// remembered the inner run's libraries must look again.
	libraryResets++
}

// StartRun starts a run from scratch, like a program on its own: no steps
// taken, no function running, the default random seed, new libraries and
// no modules loaded. Output, the step limit, the hooks and the module
// prelude stay as they are, for the host to set.
func StartRun() {
	functionBodies = 0
	ResetSteps()
	ResetMath()
	ResetModules()
}

// Outside runs f as the program s was saved from: with that state in place,
// so its steps count there and its puts writes there. Then the state of the
// run of now comes back. A sandbox calls the outer program's hooks this way.
func (s *RunState) Outside(f func()) {
	inner := SaveRunState()
	s.Restore()
	f()
	*s = SaveRunState()
	inner.Restore()
}
