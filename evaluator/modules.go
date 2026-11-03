package evaluator

import (
	"errors"
	"fmt"
	"io/fs"
	"lizardscript/ast"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"os"
	"path/filepath"
	"strings"
)

// Modules: use "draw/shapes" runs the file draw/shapes.liz once per run, in
// a top scope of its own, and lets the file that says it see the names it
// defines (see object.Environment). use "draw/shapes" as shapes puts them in
// a hash instead.

// ReadModule reads the file of a module. A host can replace it to read
// modules from somewhere else, like the files built into a program.
var ReadModule = os.ReadFile

// ModulePath lists the folders where use looks for a module that is not
// next to the file that uses it, in order.
var ModulePath []string

// ModulePrelude, when it is not nil, fills the scope around every module's
// top scope, once per run: a host gives modules what it gives the program,
// like the Terrarium. Without it a module has the builtins and the
// libraries, like any program.
var ModulePrelude func(env *object.Environment)

// Runner runs a parsed program in env, like Eval does: the engine that runs
// a module.
type Runner func(program *ast.Program, env *object.Environment) object.Object

// module is a file loaded by use.
type module struct {
	name    string              // the path it was found at, its source name
	env     *object.Environment // its top scope
	names   *object.Hash        // its names, for use ... as
	loading bool                // it is running now: a use of it is a cycle
}

// moduleState is what use keeps for one run.
type moduleState struct {
	loaded  map[string]*module // by moduleKey
	chain   []*module          // the modules loading now, the innermost last
	prelude *object.Environment
}

var modules = newModuleState()

func newModuleState() *moduleState {
	return &moduleState{loaded: map[string]*module{}}
}

// ResetModules forgets the modules loaded so far, so the next use of each
// runs its file again. The REPL and Run call it before a run, like
// ResetMath; a host that runs programs one after another calls it when the
// files may have changed.
func ResetModules() {
	modules = newModuleState()
}

// Use runs the statement node in env, the top scope of a file: it loads the
// module with run, if this run hasn't loaded it yet, and makes its names
// seen from env, or defines the hash of its names. Its value is null, or the
// error that stopped the module.
func Use(node *ast.UseStatement, env *object.Environment, run Runner) object.Object {
	m, err := loadModule(node, run)
	if err != nil {
		return err
	}
	if node.Name != nil {
		env.Define(node.Name.Value, m.names)
	} else {
		env.Use(m.env)
	}
	return NULL
}

// loadModule finds the module of node and runs it, once per run.
func loadModule(node *ast.UseStatement, run Runner) (*module, *object.Error) {
	name, source, err := findModule(node)
	if err != nil {
		return nil, err
	}
	key := moduleKey(name)
	if m, ok := modules.loaded[key]; ok {
		if m.loading {
			return nil, newError(node.Token, "%s", cycleMessage(m))
		}
		return m, nil
	}

	p := parser.New(lexer.NewNamed(string(source), name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, syntaxErrors(name, p.Errors())
	}

	m := &module{name: name, env: object.NewEnclosedEnvironment(modules.preludeEnv()), loading: true}
	state := modules
	state.loaded[key] = m
	state.chain = append(state.chain, m)
	result := run(program, m.env)
	state.chain = state.chain[:len(state.chain)-1]
	m.loading = false
	if err, ok := result.(*object.Error); ok {
		// A module that failed is tried again by the next use, so a REPL
		// can fix the file and go on.
		delete(state.loaded, key)
		return nil, err
	}
	m.names = namesHash(m.env)
	return m, nil
}

// FindModule finds the file of the module that node uses, the way use
// does, without running it: the path it was found at and the file. A tool
// that bundles a program with the modules it uses calls it.
func FindModule(node *ast.UseStatement) (string, []byte, *object.Error) {
	return findModule(node)
}

// findModule finds the file of the module node names: next to the file
// that says it (its token's source name), then in each folder of
// ModulePath. ".liz" is added to a path without it. It returns the path
// that was found, which becomes the module's source name, and the file.
func findModule(node *ast.UseStatement) (string, []byte, *object.Error) {
	spec := filepath.FromSlash(node.Path.Value)
	if !strings.HasSuffix(spec, ".liz") {
		spec += ".liz"
	}
	var candidates []string
	if filepath.IsAbs(spec) {
		candidates = []string{spec}
	} else {
		candidates = append(candidates, filepath.Join(filepath.Dir(node.Token.Source), spec))
		for _, dir := range ModulePath {
			candidates = append(candidates, filepath.Join(dir, spec))
		}
	}

	for _, candidate := range candidates {
		source, err := ReadModule(candidate)
		if err == nil {
			return filepath.ToSlash(candidate), source, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", nil, newError(node.Token, "can't read the module %q: %v", node.Path.Value, unwrapPathError(err))
		}
	}
	for i := range candidates {
		candidates[i] = filepath.ToSlash(candidates[i])
	}
	return "", nil, newError(node.Token, "can't find the module %q: there is no %s",
		node.Path.Value, orList(candidates))
}

// moduleKey is what tells two modules apart: the absolute path of the file,
// so two ways of writing the same path load it once.
func moduleKey(name string) string {
	if abs, err := filepath.Abs(filepath.FromSlash(name)); err == nil {
		return abs
	}
	return name
}

// preludeEnv is the scope around every module's top scope, made the first
// time a module loads.
func (s *moduleState) preludeEnv() *object.Environment {
	if s.prelude == nil {
		s.prelude = object.NewEnvironment()
		if ModulePrelude != nil {
			ModulePrelude(s.prelude)
		}
	}
	return s.prelude
}

// namesHash is the value of use ... as: a hash of the names the module
// defined, as they are when its file has just run.
func namesHash(env *object.Environment) *object.Hash {
	names := object.NewHash()
	for _, name := range env.Names() {
		if object.Exported(name) {
			value, _ := env.Get(name)
			names.Set(&object.String{Value: name}, value)
		}
	}
	return names
}

// cycleMessage explains a use of m while m is still loading: the chain of
// uses from m back to it, like "a.liz uses b.liz, which uses a.liz".
func cycleMessage(m *module) string {
	chain := modules.chain
	for len(chain) > 0 && chain[0] != m {
		chain = chain[1:]
	}
	message := "use cycle: " + m.name + " uses "
	for _, step := range chain[1:] {
		message += step.name + ", which uses "
	}
	if len(chain) <= 1 {
		return message + "itself"
	}
	return message + m.name
}

// syntaxErrors is the error of a module whose file doesn't parse: every
// syntax error, one per line, at the position of the first.
func syntaxErrors(name string, messages []string) *object.Error {
	err := &object.Error{Message: strings.Join(messages, "\n"), Source: name}
	fmt.Sscanf(messages[0], "[line %d, column %d]", &err.Line, &err.Column)
	return err
}

// unwrapPathError drops Go's "open path:" from a file error.
func unwrapPathError(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}

// orList writes a, b or c.
func orList(items []string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
}
