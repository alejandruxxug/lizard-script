package terrarium

import (
	"bytes"
	"lizardscript/ast"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"lizardscript/vm"
	"strings"
	"testing"
)

// engine runs a parsed program: the tree-walker or the VM.
type engine func(program *ast.Program, env *object.Environment) object.Object

// engines are both engines, which must give the same for every program.
var engines = []struct {
	name string
	run  engine
}{
	{"evaluator", func(p *ast.Program, env *object.Environment) object.Object { return evaluator.Eval(p, env) }},
	{"vm", vm.Run},
}

// run runs source with the Terrarium installed on the tree-walker and
// returns its value.
func run(t *testing.T, source string) object.Object {
	t.Helper()
	return runOn(t, engines[0].run, source)
}

// runOn runs source like run, on the given engine.
func runOn(t *testing.T, run engine, source string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors in %q: %v", source, p.Errors())
	}
	env := object.NewEnvironment()
	Install(env)
	return run(program, env)
}

// testCase is a program and what it gives: the value as puts shows it, or,
// when err is set, a text the error message must contain.
type testCase struct {
	name   string
	source string
	want   string
	err    bool
}

// check runs every case on both engines.
func check(t *testing.T, cases []testCase) {
	t.Helper()
	for _, e := range engines {
		for _, tc := range cases {
			t.Run(e.name+"/"+tc.name, func(t *testing.T) {
				checkCase(t, tc, runOn(t, e.run, tc.source))
			})
		}
	}
}

// checkCase compares what a case gave with what it wants.
func checkCase(t *testing.T, tc testCase, got object.Object) {
	t.Helper()
	if err, ok := got.(*object.Error); ok {
		if !tc.err {
			t.Fatalf("got error %q, want %s", err.Message, tc.want)
		}
		if !strings.Contains(err.Message, tc.want) {
			t.Fatalf("error %q doesn't contain %q", err.Message, tc.want)
		}
		return
	}
	if tc.err {
		t.Fatalf("got %s, want an error with %q", got.Inspect(), tc.want)
	}
	if got.Inspect() != tc.want {
		t.Fatalf("got %s, want %s", got.Inspect(), tc.want)
	}
}

// quote makes a lizard-script string literal of s (s must not need escapes
// other than backslash and quote).
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func TestInstall(t *testing.T) {
	check(t, []testCase{
		{"core modules", `[type(fs), type(os), type(time), type(json), type(img), type(clip), type(io), type(lizard)]`,
			`["HASH", "HASH", "HASH", "HASH", "HASH", "HASH", "HASH", "HASH"]`, false},
		{"functions are builtins", `fs.read`, "builtin fs.read", false},
		{"json helpers stay hidden", `keys(json)`, `["parse", "write"]`, false},
		{"a let hides a module", `let fs = 5; fs`, "5", false},
		{"modules are new each install", `fs.read = 1; fs.read`, "1", false},
	})
	if run(t, `fs.read`).Inspect() != "builtin fs.read" {
		t.Fatal("a change to a module lived on into the next run")
	}
}

func TestInstallCoreHasNoWindow(t *testing.T) {
	env := object.NewEnvironment()
	InstallCore(env)
	for _, name := range []string{"window", "pen"} {
		if _, ok := env.Get(name); ok {
			t.Errorf("InstallCore defined %s", name)
		}
	}
}

func TestRegister(t *testing.T) {
	extrasMu.Lock()
	saved := extras
	extrasMu.Unlock()
	defer func() {
		extrasMu.Lock()
		extras = saved
		extrasMu.Unlock()
	}()
	Register(func(env *object.Environment) { env.Define("extra", &object.Integer{Value: 7}) })
	if got := run(t, `extra`).Inspect(); got != "7" {
		t.Fatalf("a registered module gave %s, want 7", got)
	}
}

func TestModulesAreSorted(t *testing.T) {
	check(t, []testCase{
		{"fs", `keys(fs)`, `["append", "copy", "exists", "home", "info", "join", "list", "mkdir", "read", "remove", "rename", "trash", "write"]`, false},
		{"os", `keys(os)`, `["args", "config_dir", "env", "exit", "locale", "memory", "open_url", "platform", "program_dir", "reveal"]`, false},
		{"img", `keys(img)`, `["blit", "clear", "clip", "composite", "fill", "from_pixels", "get", "load", "new", "pixels", "png_base64", "pop_layer", "push_layer", "save", "set", "size", "span", "span_gradient", "spans", "spans_gradient", "triangles", "unclip"]`, false},
	})
}

// Output from puts while a test runs goes here, so it doesn't mix with the
// test's own.
func init() {
	evaluator.Output = &bytes.Buffer{}
}
