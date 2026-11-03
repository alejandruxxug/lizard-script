package evaluator

import (
	"bytes"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// moduleFiles are the modules the tests of use load, written to a folder of
// their own.
var moduleFiles = map[string]string{
	"shapes.liz":    "let sides = 3\nfunction area(w, h) { return w * h }\nlet _secret = 1",
	"counter.liz":   "let count = 0\nfunction bump() { count += 1; return count }",
	"loud.liz":      "puts(\"loading loud\")\nlet loud = true",
	"cycle_a.liz":   "use \"cycle_b\"",
	"cycle_b.liz":   "use \"cycle_a\"",
	"self.liz":      "use \"self\"",
	"broken.liz":    "let = 1;\nlet = 2",
	"fails.liz":     "let a = 1\nlet b = a + \"x\"",
	"peek.liz":      "let seen = secret",
	"greet.liz":     "function hi() { return greeting }",
	"sub/inner.liz": "use \"../shapes\"\nlet both = sides * 2",
	"uses.liz":      "use \"shapes\"\nlet mine = 1",
	"lib/found.liz": "let found = \"on the path\"",
}

// writeModules writes moduleFiles into a new folder and returns it, with "/".
func writeModules(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, source := range moduleFiles {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.ToSlash(dir)
}

// runModuleProgram runs source as the file main.liz of dir on the
// evaluator, after a ResetModules, and returns its value and output.
func runModuleProgram(t *testing.T, dir, source string) (object.Object, string) {
	t.Helper()
	p := parser.New(lexer.NewNamed(source, dir+"/main.liz"))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors in %q: %v", source, p.Errors())
	}
	var out bytes.Buffer
	saved := Output
	Output = &out
	defer func() { Output = saved }()
	ResetModules()
	return Eval(program, object.NewEnvironment()), out.String()
}

func TestUse(t *testing.T) {
	dir := writeModules(t)
	tests := []struct {
		name, input, want string
	}{
		{"all names", "use \"shapes\"\narea(2, sides)", "6"},
		{"a hash of names", "use \"shapes\" as s\n[s.sides, s.area(2, 3), keys(s)]", `[3, 6, ["area", "sides"]]`},
		{"the value of use", "use \"shapes\"", "null"},
		{"with .liz", "use \"shapes.liz\"\nsides", "3"},
		{"names are alive", "use \"counter\"\nbump()\nbump()\ncount", "2"},
		{"= reaches the module", "use \"counter\"\ncount = 10\nbump()", "11"},
		{"compound assign reaches the module", "use \"counter\"\ncount += 5\nbump()", "6"},
		{"an own let hides a module name", "use \"counter\"\nlet count = 5\nbump()\ncount", "5"},
		{"an own let before the use too", "let count = 5\nuse \"counter\"\nbump()\ncount", "5"},
		{"the hash is a snapshot", "use \"counter\" as c\nc.bump()\nc.count", "0"},
		{"the same hash each time", "use \"counter\" as a\nuse \"counter\" as b\nsame(a, b)", "true"},
		{"one module for both forms", "use \"counter\"\nuse \"counter\" as c\nc.bump()\ncount", "1"},
		{"relative to the file that uses it", "use \"sub/inner\"\nboth", "6"},
		{"names a module uses are its own", "use \"uses\"\nmine", "1"},
		{"a module's own use is not seen", "use \"uses\"\nsides", "identifier not found: sides"},
		{"private names", "use \"shapes\"\n_secret", "identifier not found: _secret"},
		{"private names in the hash", "use \"shapes\" as s\ns._secret", "null"},
		{"a module doesn't see the program", "let secret = 1\nuse \"peek\"", "identifier not found: secret"},
		{"missing", "use \"nope\"", `can't find the module "nope": there is no ` + dir + "/nope.liz"},
		{"cycle", "use \"cycle_a\"", "use cycle: " + dir + "/cycle_a.liz uses " + dir + "/cycle_b.liz, which uses " + dir + "/cycle_a.liz"},
		{"uses itself", "use \"self\"", "use cycle: " + dir + "/self.liz uses itself"},
		{"syntax errors", "use \"broken\"", "[line 1, column 5] syntax error: expected identifier, got \"=\"\n[line 2, column 5]"},
		{"runtime errors", "use \"fails\"", "[line 2, column 11] runtime error: type mismatch: INTEGER + STRING"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := runModuleProgram(t, dir, tt.input)
			text := got.Inspect()
			if err, ok := got.(*object.Error); ok {
				text = err.Message
			}
			if !strings.Contains(text, tt.want) {
				t.Errorf("got %s, want %s", text, tt.want)
			}
		})
	}
}

func TestUseRunsAModuleOnce(t *testing.T) {
	dir := writeModules(t)
	got, out := runModuleProgram(t, dir, "use \"loud\"\nuse \"loud\"\nuse \"loud\" as l\nl.loud")
	if got.Inspect() != "true" || out != "loading loud\n" {
		t.Errorf("got %s and output %q, want true and one line", got.Inspect(), out)
	}
	// A new run loads it again.
	if _, out := runModuleProgram(t, dir, "use \"loud\""); out != "loading loud\n" {
		t.Errorf("the next run printed %q", out)
	}
}

func TestUseErrorsCarryTheModule(t *testing.T) {
	dir := writeModules(t)
	tests := []struct {
		input, source string
		line, column  int
	}{
		{"use \"fails\"", dir + "/fails.liz", 2, 11},
		{"use \"broken\"", dir + "/broken.liz", 1, 5},
		{"use \"peek\"", dir + "/peek.liz", 1, 12},
		{"\nuse \"nope\"", dir + "/main.liz", 2, 1},
		{"use \"cycle_a\"", dir + "/cycle_b.liz", 1, 1},
	}
	for _, tt := range tests {
		got, _ := runModuleProgram(t, dir, tt.input)
		err, ok := got.(*object.Error)
		if !ok || err.Source != tt.source || err.Line != tt.line || err.Column != tt.column {
			t.Errorf("%q gave %#v, want an error at %s %d:%d", tt.input, got, tt.source, tt.line, tt.column)
		}
	}
	got, _ := runModuleProgram(t, dir, "use \"broken\"")
	if lines := strings.Split(got.(*object.Error).Message, "\n"); len(lines) != 2 {
		t.Errorf("a module with two syntax errors gave %q", lines)
	}
}

func TestModulePreludeAndPath(t *testing.T) {
	dir := writeModules(t)
	saved := ModulePrelude
	defer func() { ModulePrelude, ModulePath = saved, nil }()
	ModulePrelude = func(env *object.Environment) { env.Define("greeting", &object.String{Value: "hi"}) }
	if got, _ := runModuleProgram(t, dir, "use \"greet\"\nhi()"); got.Inspect() != "hi" {
		t.Errorf("a module didn't see the prelude: %s", got.Inspect())
	}
	if got, _ := runModuleProgram(t, dir, "use \"greet\"\ngreeting"); !strings.Contains(got.Inspect(), "identifier not found") {
		t.Errorf("the prelude reached the program: %s", got.Inspect())
	}

	ModulePath = []string{dir + "/lib"}
	if got, _ := runModuleProgram(t, t.TempDir(), "use \"found\"\nfound"); got.Inspect() != "on the path" {
		t.Errorf("ModulePath: %s", got.Inspect())
	}
}

func TestReadModule(t *testing.T) {
	saved := ReadModule
	defer func() { ReadModule = saved }()
	ReadModule = func(path string) ([]byte, error) {
		if filepath.ToSlash(path) == "virtual/x.liz" {
			return []byte("let x = 42"), nil
		}
		return nil, os.ErrNotExist
	}
	if got, _ := runModuleProgram(t, "virtual", "use \"x\"\nx"); got.Inspect() != "42" {
		t.Errorf("got %s", got.Inspect())
	}
	ReadModule = func(path string) ([]byte, error) { return nil, os.ErrPermission }
	got, _ := runModuleProgram(t, "virtual", "use \"x\"")
	if !strings.Contains(got.Inspect(), `can't read the module "x": permission denied`) {
		t.Errorf("got %s", got.Inspect())
	}
}
