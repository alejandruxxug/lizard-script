package evaluator

import (
	"errors"
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Modules: use "draw/shapes" runs the file draw/shapes.liz once per run, in
// a top scope of its own, and lets the file that says it see the names it
// defines (see object.Environment). use "draw/shapes" as shapes puts them in
// a hash instead. Modules can use each other in a cycle: a use of a module
// that is still loading sees what it has defined so far.

// ReadModule reads the file of a module. A host can replace it to read
// modules from somewhere else, like the files built into a program.
var ReadModule = os.ReadFile

// ModulePath lists the folders where use looks for a module that is not
// next to the file that uses it, in order.
var ModulePath []string

// StandardModules holds the standard modules: big libraries that come with
// the language but are not installed into every program, like a 2D
// graphics library. use finds one after the program's own folder and
// ModulePath, at the path StandardRoot + "/" + its name, like
// "std:/lily2d.liz". A host sets it (the Terrarium does); nil has none.
var StandardModules fs.FS

// StandardRoot starts the source name of a standard module's file.
const StandardRoot = "std:"

// IsStandardModule reports whether path is the source name of a standard
// module's file. A tool that bundles a program skips those: they are part
// of every lizardscript.
func IsStandardModule(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), StandardRoot+"/")
}

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
	from    string              // the source name of the file whose use loaded it first
	env     *object.Environment // its top scope
	names   *object.Hash        // its names, for use ... as
	program *ast.Program        // its statements
	loading bool                // it is running now: a use of it is a cycle
	cycle   string              // the first use cycle back to it, or ""
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
// files may have changed. The parsed files stay (see parsedModules): a file
// whose text is the same is not parsed again.
func ResetModules() {
	modules = newModuleState()
}

// parsedModule is a module file parsed before: its text and its program.
type parsedModule struct {
	source  string
	program *ast.Program
}

// parsedModules keeps every module file parsed so far, by moduleKey, so a
// later run that reads the same text uses the same program instead of
// parsing it again. An engine that compiles programs keeps its code by
// program too (the VM does), so a module is compiled once, not once per
// run. A file whose text changed is parsed again and replaces its entry.
var parsedModules = map[string]parsedModule{}

// Use runs the statement node in env, the top scope of a file: it loads the
// module with run, if this run hasn't loaded it yet, and makes its names
// seen from env, or defines the hash of its names. Its value is null, or the
// error that stopped the module.
func Use(node *ast.UseStatement, env *object.Environment, run Runner) object.Object {
	m, err := loadModule(node, run)
	if err != nil {
		return err
	}
	switch {
	case node.Name != nil:
		env.Define(node.Name.Value, m.names)
	case m.env != env:
		// A file that uses itself already sees its names.
		env.Use(m.env)
	}
	return NULL
}

// loadModule finds the module of node and runs it, once per run. A use of
// a module that is still loading, a cycle, gets it as it is so far: the
// names it defined before the use that started the cycle are there, and
// the rest appear as it goes on, since its names are alive.
func loadModule(node *ast.UseStatement, run Runner) (*module, *object.Error) {
	name, source, err := findModule(node)
	if err != nil {
		return nil, err
	}
	key := moduleKey(name)
	if m, ok := modules.loaded[key]; ok {
		if m.loading {
			if m.cycle == "" {
				m.cycle = cycleMessage(m, modules.chain)
			}
			fillNames(m.names, m.env)
		}
		return m, nil
	}

	program, err := parseModule(key, name, source)
	if err != nil {
		return nil, err
	}
	m := &module{
		name:    name,
		from:    node.Token.Source,
		env:     object.NewEnclosedEnvironment(modules.preludeEnv()),
		names:   object.NewHash(),
		program: program,
		loading: true,
	}
	state := modules
	state.loaded[key] = m
	state.chain = append(state.chain, m)
	result := run(program, m.env)
	if err, ok := result.(*object.Error); ok {
		// Explained while the chain still holds the modules of the cycle.
		result = explainNotLoaded(err, state)
	}
	state.chain = state.chain[:len(state.chain)-1]
	m.loading = false
	if err, ok := result.(*object.Error); ok {
		// A module that failed is tried again by the next use, so a REPL
		// can fix the file and go on.
		delete(state.loaded, key)
		return nil, err
	}
	fillNames(m.names, m.env)
	return m, nil
}

// parseModule is the program of the module file name, with the text
// source: the one parsed before when the text is the same.
func parseModule(key, name string, source []byte) (*ast.Program, *object.Error) {
	if parsed, ok := parsedModules[key]; ok && parsed.source == string(source) {
		return parsed.program, nil
	}
	p := parser.New(lexer.NewNamed(string(source), name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, syntaxErrors(name, p.Errors())
	}
	parsedModules[key] = parsedModule{source: string(source), program: program}
	return program, nil
}

// FindModule finds the file of the module that node uses, the way use
// does, without running it: the path it was found at and the file. A tool
// that bundles a program with the modules it uses calls it.
func FindModule(node *ast.UseStatement) (string, []byte, *object.Error) {
	return findModule(node)
}

// findModule finds the file of the module node names: next to the file
// that says it (its token's source name), then in each folder of
// ModulePath, then among the standard modules. ".liz" is added to a path
// without it. It returns the path that was found, which becomes the
// module's source name, and the file. The not-found message lists only the
// files on disk.
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
		source, err := readCandidate(candidate)
		if err == nil {
			return filepath.ToSlash(candidate), source, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", nil, newError(node.Token, "can't read the module %q: %v", node.Path.Value, unwrapPathError(err))
		}
	}
	if !filepath.IsAbs(spec) {
		if name, source, ok := standardModule(spec); ok {
			return name, source, nil
		}
	}
	for i := range candidates {
		candidates[i] = filepath.ToSlash(candidates[i])
	}
	return "", nil, newError(node.Token, "can't find the module %q: there is no %s",
		node.Path.Value, orList(candidates))
}

// readCandidate reads a file use may find: a standard module's for a path
// under StandardRoot (a module next to a standard one), else with
// ReadModule.
func readCandidate(path string) ([]byte, error) {
	if IsStandardModule(path) {
		name, source, ok := standardModule(strings.TrimPrefix(filepath.ToSlash(path), StandardRoot+"/"))
		if !ok || name != filepath.ToSlash(path) {
			return nil, fs.ErrNotExist
		}
		return source, nil
	}
	return ReadModule(path)
}

// standardModule reads the standard module at spec, a path with ".liz",
// and gives its source name. A spec that climbs out with ".." is none.
func standardModule(spec string) (string, []byte, bool) {
	if StandardModules == nil {
		return "", nil, false
	}
	name := filepath.ToSlash(filepath.Clean(spec))
	if !fs.ValidPath(name) {
		return "", nil, false
	}
	source, err := fs.ReadFile(StandardModules, name)
	if err != nil {
		return "", nil, false
	}
	return StandardRoot + "/" + name, source, true
}

// moduleKey is what tells two modules apart: the absolute path of the file,
// so two ways of writing the same path load it once. A standard module's
// source name is already one of a kind.
func moduleKey(name string) string {
	if IsStandardModule(name) {
		return name
	}
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

// fillNames puts in names, the value of use ... as, the names the module
// env defined so far, as they are now. A module's hash is made when it
// starts loading and filled when its file has run, so every use ... as of
// it, even one in a cycle, gets the same hash.
func fillNames(names *object.Hash, env *object.Environment) {
	for _, name := range env.Names() {
		if object.Exported(name) {
			value, _ := env.Get(name)
			names.Set(&object.String{Value: name}, value)
		}
	}
}

// notFound starts the message of a name that isn't defined.
const notFound = "identifier not found: "

// explainNotLoaded adds to err, when it is a name that isn't defined, why:
// a module of the cycle that is still loading defines that name further
// down its file, so it isn't there yet. Any other error comes back as it
// is.
func explainNotLoaded(err *object.Error, state *moduleState) *object.Error {
	at := strings.Index(err.Message, notFound)
	if at < 0 || strings.Contains(err.Message, "still loading") {
		return err
	}
	name := err.Message[at+len(notFound):]
	if end := strings.IndexByte(name, '\n'); end >= 0 {
		name = name[:end]
	}
	for _, m := range state.chain {
		if m.cycle == "" || !definesLater(m, name) {
			continue
		}
		explained := *err
		explained.Message = strings.Replace(err.Message, notFound+name,
			fmt.Sprintf("%s%s: %s defines it further down, but it is still loading (%s)",
				notFound, name, shownName(m, state.chain), m.cycle), 1)
		return &explained
	}
	return err
}

// definesLater reports whether the file of m defines name at its top level
// (in a layer too) and hasn't yet.
func definesLater(m *module, name string) bool {
	if _, done := m.env.Get(name); done {
		return false
	}
	var defines func(statements []ast.Statement) bool
	defines = func(statements []ast.Statement) bool {
		for _, st := range statements {
			switch st := st.(type) {
			case *ast.LetStatement:
				if st.Name != nil && st.Name.Value == name {
					return true
				}
			case *ast.LayerStatement:
				if st.Body != nil && defines(st.Body.Statements) {
					return true
				}
			}
		}
		return false
	}
	return defines(m.program.Statements)
}

// cycleMessage is the cycle of a use of m while it is loading, from the
// modules loading now: "use cycle: a.liz uses b.liz, which uses a.liz".
func cycleMessage(m *module, chain []*module) string {
	for len(chain) > 0 && chain[0] != m {
		chain = chain[1:]
	}
	message := "use cycle: " + shownName(m, chain) + " uses "
	for _, step := range chain[1:] {
		message += shownName(step, chain) + ", which uses "
	}
	if len(chain) <= 1 {
		return message + "itself"
	}
	return message + shownName(m, chain)
}

// shownName is how a message names the file of m: from the folder of the
// program that started loading the chain, the way an error's file is shown,
// so the message is the same wherever the program is run from.
func shownName(m *module, chain []*module) string {
	if len(chain) == 0 || IsStandardModule(m.name) {
		return m.name
	}
	root := chain[0].from
	if root == "" {
		return m.name
	}
	if rel, err := filepath.Rel(filepath.Dir(root), m.name); err == nil {
		return filepath.ToSlash(rel)
	}
	return m.name
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
