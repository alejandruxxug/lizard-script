package terrarium

import (
	"bytes"
	"lizardscript/evaluator"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLizardTokens(t *testing.T) {
	check(t, []testCase{
		{"kinds", `let ks = []; for (let i = 0; i < 11; i += 1) { ks = ks + [lizard.tokens("let x = \"a\" + 2.5 -~> hi\n#fff(y)")[i].kind] }; ks`,
			`["keyword", "name", "operator", "string", "operator", "number", "comment", "color", "punctuation", "name", "punctuation"]`, false},
		{"one token", `lizard.tokens("  x1")[0]`,
			`{"kind": "name", "type": "IDENT", "text": "x1", "value": "x1", "line": 1, "column": 3, "end_line": 1, "end_column": 5}`, false},
		{"text and value", `let t = lizard.tokens("\"a\\\"b\" 45deg"); [t[0].text, t[0].value, t[1].text, t[1].type]`,
			`["\"a\\\"b\"", "a\"b", "45deg", "FLOAT"]`, false},
		{"a string on two lines", `let t = lizard.tokens("x = \"é\nü\"")[2]; [t.line, t.column, t.end_line, t.end_column]`, "[1, 5, 2, 3]", false},
		{"an error token", `lizard.tokens("a & b")[1].kind`, "error", false},
		{"nothing", `lizard.tokens("  -~>")[0].type`, "COMMENT", false},
		{"empty", `lizard.tokens("")`, "[]", false},
		{"a number", `lizard.tokens(5)`, "want a STRING code, got INTEGER", true},
	})
}

func TestLizardParse(t *testing.T) {
	check(t, []testCase{
		{"a let", `let s = lizard.parse("let x = 1 + y").program.statements[0]
[s.kind, s.name.value, s.value.kind, s.value.operator, s.value.right.value, s.declaration, s.end_column]`,
			`["let", "x", "infix", "+", "y", false, 14]`, false},
		{"a declaration", `let s = lizard.parse("function f(a, b = (1, 2)) { a }").program.statements[0]
let f = s.value
[s.declaration, f.kind, f.params[1].value, f.defaults[0], f.defaults[1].kind, f.body.statements[0].kind]`,
			`[true, "function", "b", null, "point", "expression"]`, false},
		{"positions", `let e = lizard.parse("\n  (1 + 2) * 3").program.statements[0].expression
[e.line, e.column, e.end_line, e.end_column, e.left.column, e.left.end_column]`,
			"[2, 3, 2, 14, 3, 10]", false},
		{"dot and index", `let e = lizard.parse("a.b[0]").program.statements[0].expression
[e.kind, e.left.kind, e.left.name.value, e.index.value]`, `["index", "dot", "b", 0]`, false},
		{"calls and hashes", `let e = lizard.parse("f({\"k\": #fff})").program.statements[0].expression
[e.callee.value, e.arguments[0].pairs[0].key.value, e.arguments[0].pairs[0].value.value]`, `["f", "k", #ffffff]`, false},
		{"layers, use and try", `let st = lizard.parse("use \"m\" as m\nlayer sun\n  try { 1 } catch (e) { 2 }").program.statements
[st[0].kind, st[0].path.value, st[0].as.value, st[1].name, st[1].body.statements[0].error_name.value]`,
			`["use", "m", "m", "sun", "e"]`, false},
		{"if and for", `let st = lizard.parse("if a { 1 } elif b { 2 }\nfor (;;) { break }").program.statements
[st[0].expression.alternative.statements[0].expression.condition.value, st[1].expression.init, st[1].expression.body.statements[0].kind]`,
			`["b", null, "break"]`, false},
		{"assign", `let e = lizard.parse("x += 1").program.statements[0].expression; [e.kind, e.operator, e.target.value]`,
			`["assign", "+=", "x"]`, false},
		{"errors", `lizard.parse("let = 1;\nlet y = 5kg").errors`,
			`[{"kind": "syntax", "message": "let needs a name before the \"=\", like let x = 1", "line": 1, "column": 5}, {"kind": "syntax", "message": "unknown unit \"kg\"", "line": 2, "column": 9}]`, false},
		{"a lexical error", `lizard.parse("x = \"open").errors[0].kind`, "lexical", false},
		{"what parsed stays", `len(lizard.parse("let a = 1;\nlet = 2;\nlet c = 3").program.statements)`, "2", false},
		{"a name", `lizard.parse("x", "a.liz").errors`, "[]", false},
		{"a wrong name", `lizard.parse("x", 1)`, "want a STRING name, got INTEGER", true},
	})
}

func TestLizardRun(t *testing.T) {
	check(t, []testCase{
		{"the value", `lizard.run("1 + 2").value`, "3", false},
		{"globals in and out", `let r = lizard.run("let y = x * 2", {"globals": {"x": 21}}); [r.globals.y, r.globals.x, r.value]`, "[42, 21, null]", false},
		{"output is kept", `let r = lizard.run("puts(1)\nputs(2)"); r.output`, "1\n2\n", false},
		{"a runtime error", `lizard.run("let a = 1\na + \"x\"", {"name": "drawing"}).error`,
			`{"kind": "runtime", "message": "type mismatch: INTEGER + STRING", "line": 2, "column": 3, "source": "drawing"}`, false},
		{"a syntax error", `lizard.run("let = 1").error`,
			`{"kind": "syntax", "message": "let needs a name before the \"=\", like let x = 1", "line": 1, "column": 5, "source": ""}`, false},
		{"the step limit", `let r = lizard.run("while (true) {}", {"steps": 100}); [r.error.message, r.steps]`,
			`["the program ran too long (more than 100 steps)", 101]`, false},
		{"steps are counted", `lizard.run("for (let i = 0; i < 10; i += 1) {}", {"steps": 1000}).steps`, "11", false},
		{"no Terrarium inside", `lizard.run("fs").error.message`, "identifier not found: fs", false},
		{"a module can be given", `lizard.run("json.write([1])", {"globals": {"json": json}}).value`, "[1]", false},
		{"the vm", `lizard.run("let f = function(n) { if (n < 2) { n } else { f(n - 1) + f(n - 2) } }; f(15)", {"engine": "vm"}).value`, "610", false},
		{"random starts fresh", `math.random(); let a = lizard.run("math.random()").value; a == lizard.run("math.random()").value`, "true", false},
		{"the caller goes on", `let before = math.random()
math.seed(7); let x = math.random(); math.seed(7); lizard.run("math.seed(1); math.random()"); x == math.random()`, "true", false},
		{"statement hook", `let seen = []
lizard.run("let a = 1\nlet b = 2", {"on_statement": function(line, column, source) { seen = seen + [line] }})
seen`, "[1, 2]", false},
		{"layer hook", `let seen = []
lizard.run("layer sun\n  1", {"on_layer": function(name, enter, line) { seen = seen + [[name, enter, line]] }})
seen`, `[["sun", true, 1], ["sun", false, 1]]`, false},
		{"hook output is the caller's", `lizard.run("1", {"on_statement": function(l, c, s) { puts("hooked") }}).output`, "", false},
		{"a failing hook", `lizard.run("1\n2", {"on_statement": function(l, c, s) { 1 + "x" }})`, "lizard.run: a hook failed: type mismatch", true},
		{"unknown option", `lizard.run("1", {"speed": 1})`, `unknown option "speed"`, true},
		{"bad steps", `lizard.run("1", {"steps": -1})`, "the option steps must be an INTEGER, 0 or more, got INTEGER", true},
		{"bad engine", `lizard.run("1", {"engine": "fast"})`, `the option engine must be "evaluator" or "vm"`, true},
		{"bad globals", `lizard.run("1", {"globals": 1})`, "the option globals must be a HASH, got INTEGER", true},
	})
}

// TestLizardRunModules checks the options use and hook_source, on both
// engines inside.
func TestLizardRunModules(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	for name, source := range map[string]string{
		"counter.liz": "let count = 0\nfunction bump() { count += 1; return count }",
		"twice.liz":   "function twice(n) { return n * 2 }",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var cases []testCase
	for _, engine := range []string{"evaluator", "vm"} {
		options := func(more string) string {
			return `{"name": "` + dir + `/drawing.liz", "engine": "` + engine + `"` + more + `}`
		}
		cases = append(cases,
			testCase{engine + "/live modules", `lizard.run("count = 10\nbump()", ` + options(`, "use": ["counter", "twice"]`) + `).value`, "11", false},
			testCase{engine + "/several in order", `lizard.run("twice(bump())", ` + options(`, "use": ["counter", "twice"]`) + `).value`, "2", false},
			testCase{engine + "/not in globals", `keys(lizard.run("let mine = 1", ` + options(`, "use": ["counter"]`) + `).globals)`, `["mine"]`, false},
			testCase{engine + "/a missing module", `lizard.run("1", ` + options(`, "use": ["nope"]`) + `).error.message`,
				`can't find the module "nope": there is no ` + dir + `/nope.liz`, false},
			testCase{engine + "/the hook sees every source", `let seen = []
lizard.run("use \"twice\"\ntwice(2)", ` + options(`, "on_statement": function(l, c, s) { seen = seen + [s] }`) + `)
len(seen)`, "4", false},
			testCase{engine + "/hook_source keeps the run's own", `let seen = []
lizard.run("use \"twice\"\ntwice(2)", ` + options(`, "hook_source": "`+dir+`/drawing.liz", "on_statement": function(l, c, s) { seen = seen + [l] }`) + `)
seen`, "[1, 2]", false},
		)
	}
	cases = append(cases,
		testCase{"bad use", `lizard.run("1", {"use": "counter"})`, "the option use must be a list of module paths, got STRING", true},
		testCase{"bad use item", `lizard.run("1", {"use": [1]})`, "the option use must be a list of module paths, got INTEGER in it", true},
		testCase{"bad hook_source", `lizard.run("1", {"hook_source": 1})`, "the option hook_source must be a STRING, got INTEGER", true},
	)
	check(t, cases)
}

// TestLizardRunKeepsTheCaller checks that a run inside a run leaves the
// caller's step count, limit and output as they were.
func TestLizardRunKeepsTheCaller(t *testing.T) {
	var out bytes.Buffer
	saved := evaluator.Output
	evaluator.Output = &out
	evaluator.MaxSteps = 1000
	evaluator.ResetSteps()
	defer func() { evaluator.Output, evaluator.MaxSteps = saved, 0 }()
	for _, e := range engines {
		// The same program without the run, for the steps the caller
		// takes (installing the Terrarium takes some too).
		evaluator.ResetSteps()
		runOn(t, e.run, `puts("a"); let r = {"output": "b\n"}; puts("c"); r.output`)
		want := evaluator.Steps() + 1

		out.Reset()
		evaluator.ResetSteps()
		got := runOn(t, e.run, `puts("a"); let r = lizard.run("puts(\"b\"); while (true) {}", {"steps": 50}); puts("c"); r.output`)
		if got.Inspect() != "b\n" || out.String() != "a\nc\n" {
			t.Errorf("%s: got %s and output %q", e.name, got.Inspect(), out.String())
		}
		if steps := evaluator.Steps(); steps != want {
			t.Errorf("%s: the caller has %d steps, want %d", e.name, steps, want)
		}
	}
}

func TestLizardDocs(t *testing.T) {
	code := strings.Join([]string{
		"-~> not this one",
		"",
		"-~> A circle around center.",
		"-~> It is round.",
		"-~>",
		"-~> center: the middle",
		"-~>   of it",
		"-~> radius: how big",
		"-~> example: circle((10, 10), 5)",
		"-~>   puts(1)",
		"function circle(center, radius = (1 + 1)) { center }",
		"let plain = function(a) { a }",
		"let x = 5",
		"layer l",
		"  function hidden() { 1 }",
	}, "\n")
	check(t, []testCase{
		{"names", `keys(lizard.docs(` + quote(code) + `))`, `["circle", "plain"]`, false},
		{"circle", `lizard.docs(` + quote(code) + `).circle`,
			`{"name": "circle", "line": 11, "summary": "A circle around center. It is round.", "params": [{"name": "center", "default": null, "doc": "the middle of it"}, {"name": "radius", "default": "(1 + 1)", "doc": "how big"}], "example": "circle((10, 10), 5)\nputs(1)"}`, false},
		{"no comment", `lizard.docs(` + quote(code) + `).plain`,
			`{"name": "plain", "line": 12, "summary": "", "params": [{"name": "a", "default": null, "doc": null}], "example": null}`, false},
		{"a broken file", `keys(lizard.docs("function a() {}\nlet = 1"))`, `["a"]`, false},
	})
}

func TestLizardCheck(t *testing.T) {
	check(t, []testCase{
		{"nothing to say", `lizard.check("let x = 1\nputs(x)")`, "[]", false},
		{"an unknown name", `lizard.check("puts(lenn([1]))")[0]`,
			`{"severity": "error", "rule": "unknown-name", "message": "identifier not found: lenn", "hint": "did you mean ` + "`len`" + `?", "line": 1, "column": 6, "end_line": 1, "end_column": 10}`, false},
		{"a warning has no hint", `let d = lizard.check("puts(1 == 1)")[0]; [d.severity, d.rule, d.hint, d.column, d.end_column]`,
			`["warning", "literals", null, 6, 12]`, false},
		{"the Terrarium is known", `lizard.check("puts(fs.read(\"a\"), lizard.check)")`, "[]", false},
		{"a member of a module", `lizard.check("fs.raed(\"a\")")[0].hint`, "did you mean `fs.read`?", false},
		{"names of the host", `[len(lizard.check("draw(canvas)")), len(lizard.check("draw(canvas)", "", ["draw", "canvas"]))]`, "[2, 0]", false},
		{"syntax errors", `let d = lizard.check("let = 1")[0]; [d.rule, d.line, d.column]`, `["syntax", 1, 5]`, false},
		{"a wrong name", `lizard.check("x", 1)`, "want a STRING name, got INTEGER", true},
		{"wrong names", `lizard.check("x", "", [1])`, "want names as STRING, got INTEGER", true},
	})
}
