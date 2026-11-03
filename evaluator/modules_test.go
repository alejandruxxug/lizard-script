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
	"testing/fstest"
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
	"ring_a.liz":    "let first = 1\nuse \"ring_b\"\nlet later = 2\nfunction from_a() { return from_b() + later }",
	"ring_b.liz":    "use \"ring_a\"\nuse \"ring_a\" as a\nlet seen = first\nlet seen_in_hash = a.first\nfunction from_b() { return later * 10 }\nfunction hash_later() { return a.later }",
	"early_a.liz":   "use \"early_b\"\nlet late = 5",
	"early_b.liz":   "use \"early_a\"\nlet x = late + 1",
	"typo_a.liz":    "use \"typo_b\"\nlet late = 5",
	"typo_b.liz":    "use \"typo_a\"\nlet x = nope",
	"checks.liz": "function _need(v) {\n  if (type(v) != \"INTEGER\") { error(\"twice: want an INTEGER\", {\"at\": \"caller\"}) }\n}\n" +
		"function twice(v) { _need(v); return v * 2 }\nfunction inside() { try { _need(\"x\") } catch (e) { return [e.line, e.column] } }\n" +
		"function each(f) { return f() }",
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
		{"cycle", "use \"cycle_a\"", "null"},
		{"uses itself", "use \"self\"", "null"},
		{"a cycle sees the names so far, then the rest", "use \"ring_a\"\nuse \"ring_b\"\n[seen, seen_in_hash, from_a(), hash_later()]", "[1, 1, 22, 2]"},
		{"a name used before its module defines it", "use \"early_a\"",
			"identifier not found: late: early_a.liz defines it further down, but it is still loading (use cycle: " +
				"early_a.liz uses early_b.liz, which uses early_a.liz)"},
		{"syntax errors", "use \"broken\"", "[line 1, column 5] syntax error: let needs a name before the \"=\", like let x = 1\n[line 2, column 5]"},
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
		{"use \"early_a\"", dir + "/early_b.liz", 2, 9},
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

// TestUseCycleNamesNoOtherFile checks that a name no module of the cycle
// defines is only "not found".
func TestUseCycleNamesNoOtherFile(t *testing.T) {
	dir := writeModules(t)
	got, _ := runModuleProgram(t, dir, "use \"typo_a\"")
	if err, ok := got.(*object.Error); !ok || !strings.HasSuffix(err.Message, "identifier not found: nope") {
		t.Errorf("got %s, want only identifier not found: nope", got.Inspect())
	}
}

// TestModulesAreParsedOnce checks that a module file is parsed again only
// when its text changed.
func TestModulesAreParsedOnce(t *testing.T) {
	dir := writeModules(t)
	path := dir + "/shapes.liz"
	program := func() any {
		runModuleProgram(t, dir, "use \"shapes\"")
		return parsedModules[moduleKey(path)].program
	}
	first := program()
	if first == nil || program() != first {
		t.Fatalf("the second run parsed shapes.liz again")
	}
	if err := os.WriteFile(filepath.FromSlash(path), []byte("let sides = 4"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := runModuleProgram(t, dir, "use \"shapes\"\nsides"); got.Inspect() != "4" {
		t.Errorf("a changed file gave %s, want 4", got.Inspect())
	}
	if program() == first {
		t.Errorf("a changed file kept its old program")
	}
}

// TestErrorAtCaller checks that error(message, {"at": "caller"}) in a
// module is reported at the program's call into it.
func TestErrorAtCaller(t *testing.T) {
	dir := writeModules(t)
	tests := []struct {
		name, input, want string
		line, column      int
		source            string
	}{
		{"at the call into the module", "use \"checks\"\n\ntwice(\"a\")", "[line 3, column 6] runtime error: twice: want an INTEGER", 3, 6, dir + "/main.liz"},
		{"a value when it is right", "use \"checks\"\ntwice(4)", "8", 0, 0, ""},
		{"a try in the program sees it moved", "use \"checks\"\ntry { twice(1.5) } catch (e) { [e.line, e.column] }", "[2, 12]", 0, 0, ""},
		{"a try in the module sees it where it was raised", "use \"checks\"\ninside()", "[2, 36]", 0, 0, ""},
		{"through a function of the program", "use \"checks\"\nfunction go() { twice(\"b\") }\ngo()", "[line 2, column 22] runtime error: twice: want an INTEGER", 2, 22, dir + "/main.liz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := runModuleProgram(t, dir, tt.input)
			if got.Inspect() != tt.want {
				t.Fatalf("got %s, want %s", got.Inspect(), tt.want)
			}
			if err, ok := got.(*object.Error); ok && (err.Line != tt.line || err.Column != tt.column || err.Source != tt.source || err.AtCaller) {
				t.Errorf("the error is at %s %d:%d (at caller %t), want %s %d:%d", err.Source, err.Line, err.Column, err.AtCaller, tt.source, tt.line, tt.column)
			}
		})
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

func TestStandardModules(t *testing.T) {
	dir := writeModules(t)
	saved := StandardModules
	defer func() { StandardModules, ModulePath = saved, nil }()
	StandardModules = fstest.MapFS{
		"kit.liz":        {Data: []byte("use \"kit/parts\"\nlet kit = {\"size\": size}")},
		"kit/parts.liz":  {Data: []byte("use \"more\"\nlet size = 3 + more")},
		"kit/more.liz":   {Data: []byte("let more = 1")},
		"shapes.liz":     {Data: []byte("let sides = 100")},
		"found.liz":      {Data: []byte("let found = \"standard\"")},
		"bad.liz":        {Data: []byte("let x = 1 + \"a\"")},
		"nested/odd.liz": {Data: []byte("use \"../kit\"\nlet odd = kit.size")},
	}
	tests := []struct {
		name, input, want string
	}{
		{"a standard module", "use \"kit\"\nkit.size", "4"},
		{"with as", "use \"kit\" as k\nk.kit.size", "4"},
		{"its own files are found next to it", "use \"kit/parts\"\nsize", "4"},
		{"a parent folder inside the standard modules", "use \"nested/odd\"\nodd", "4"},
		{"the program's folder comes first", "use \"shapes\"\nsides", "3"},
		{"its errors carry its name", "use \"bad\"", "type mismatch"},
		{"missing lists only the files on disk", "use \"nope\"", `can't find the module "nope": there is no ` + dir + "/nope.liz"},
		{"no climbing out", "use \"../kit\"", `can't find the module "../kit"`},
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

	got, _ := runModuleProgram(t, dir, "use \"bad\"")
	if err, ok := got.(*object.Error); !ok || err.Source != "std:/bad.liz" {
		t.Errorf("the error of a standard module is %#v, want one from std:/bad.liz", got)
	}
	ModulePath = []string{dir + "/lib"}
	if got, _ := runModuleProgram(t, t.TempDir(), "use \"found\"\nfound"); got.Inspect() != "on the path" {
		t.Errorf("ModulePath must come before the standard modules: %s", got.Inspect())
	}
	if !IsStandardModule("std:/kit.liz") || IsStandardModule("kit.liz") {
		t.Errorf("IsStandardModule is wrong")
	}
}
