package vm

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	lizardast "lizardscript/ast"
	lizardtoken "lizardscript/token"
)

// outcome is everything a run shows: what it printed, its value or error,
// the steps it took and the hook calls it made.
type outcome struct {
	output string
	value  string
	steps  int
	hooks  []string
}

func (o outcome) String() string {
	return fmt.Sprintf("value: %s\nsteps: %d\noutput:\n%s\nhooks:\n%s", o.value, o.steps, o.output, strings.Join(o.hooks, "\n"))
}

// describe writes a value with its type, and an error with its position
// and whether a try can catch it.
func describe(result object.Object) string {
	switch result := result.(type) {
	case nil:
		return "nil"
	case *object.Error:
		return fmt.Sprintf("ERROR %q line %d column %d uncatchable %t", result.Message, result.Line, result.Column, result.Uncatchable)
	default:
		return string(result.Type()) + " " + result.Inspect()
	}
}

// runEngine runs a parsed program on one engine with a step limit and both
// hooks recording.
func runEngine(program *lizardast.Program, engine func(*lizardast.Program, *object.Environment) object.Object, hooks bool) outcome {
	var out bytes.Buffer
	var events []string
	evaluator.Output = &out
	evaluator.MaxSteps = 20000
	evaluator.ResetSteps()
	evaluator.ResetMath()
	if hooks {
		evaluator.OnStatement = func(node lizardast.Statement) {
			events = append(events, fmt.Sprintf("stmt %p %s", node, firstLine(node.String())))
		}
		evaluator.OnLayer = func(name string, enter bool, tok lizardtoken.Token) {
			events = append(events, fmt.Sprintf("layer %s %t %d:%d", name, enter, tok.Line, tok.Column))
		}
	}
	defer func() {
		evaluator.Output = os.Stdout
		evaluator.MaxSteps = 0
		evaluator.OnStatement, evaluator.OnLayer = nil, nil
	}()

	result := engine(program, object.NewEnvironment())
	return outcome{output: out.String(), value: describe(result), steps: evaluator.Steps(), hooks: events}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// compareEngines runs input on both engines and reports any difference.
func compareEngines(t *testing.T, input string, hooks bool) bool {
	t.Helper()
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return false
	}
	tree := runEngine(program, func(p *lizardast.Program, env *object.Environment) object.Object {
		return evaluator.Eval(p, env)
	}, hooks)
	compiled := runEngine(program, Run, hooks)
	if tree.output != compiled.output || tree.value != compiled.value || tree.steps != compiled.steps ||
		strings.Join(tree.hooks, "\n") != strings.Join(compiled.hooks, "\n") {
		t.Errorf("the engines differ on %q\n--- evaluator:\n%s\n--- vm:\n%s", input, tree, compiled)
	}
	return true
}

// knownDifferences are inputs that are meant to give a different result:
// the evaluator's "nest too deeply" limit protects Go's stack from its own
// recursion, which the VM doesn't have.
var knownDifferences = map[string]bool{}

// testStrings collects every string literal in the evaluator's tests and
// the example programs: thousands of small programs, valid and failing.
func testStrings(t *testing.T) []string {
	files, err := filepath.Glob(filepath.Join("..", "evaluator", "*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no evaluator tests found: %v", err)
	}
	seen := map[string]bool{}
	var inputs []string
	for _, file := range files {
		parsed, err := goparser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(n ast.Node) bool {
			literal, ok := n.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(literal.Value)
			if err == nil && strings.TrimSpace(text) != "" && !seen[text] {
				seen[text] = true
				inputs = append(inputs, text)
			}
			return true
		})
	}

	for _, pattern := range []string{"../examples/*.liz", "../examples/errors/*.liz", "../tests/valid/*.liz", "../tests/errors/*.liz"} {
		programs, _ := filepath.Glob(pattern)
		for _, file := range programs {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			inputs = append(inputs, strings.ReplaceAll(string(source), "\r\n", "\n"))
		}
	}
	return inputs
}

// TestDifferentialCorpus runs every program of the evaluator's tests on both
// engines, with and without hooks.
func TestDifferentialCorpus(t *testing.T) {
	ran := 0
	for _, input := range testStrings(t) {
		if knownDifferences[input] {
			continue
		}
		if compareEngines(t, input, false) {
			compareEngines(t, input, true)
			ran++
		}
	}
	if ran < 500 {
		t.Fatalf("only %d programs ran, the corpus is too small", ran)
	}
	t.Logf("%d programs compared", ran)
}

// TestDifferentialHandWritten covers the corners of scopes, signals and
// errors that the corpus may miss. Each runs on both engines and must give
// the same result.
func TestDifferentialHandWritten(t *testing.T) {
	for _, input := range handWritten {
		if !compareEngines(t, input, true) {
			t.Errorf("syntax error in %q", input)
		}
	}
}

var handWritten = []string{
	// A let reads the outer name before it defines its own.
	"let x = 1; function f() { let y = x; let x = 2; [y, x] } f()",
	"let x = 1; if (true) { let y = x; let x = 2; puts(y, x) }",
	"function f() { let x = x; x } let x = 5; f()",
	"function f(x) { if (true) { let x = x + 1; x } } f(1)",
	// A closure made before a later let sees it once it exists.
	"function f() { let g = function() { y }; let a = 0; try { g() } catch (e) { a = e.message }; let y = 7; [a, g()] } f()",
	"function f() { let g = function() { y }; let y = 1; let y = 2; g() } f()",
	"let y = 0; function f() { let g = function() { y }; let r = g(); let y = 1; [r, g()] } f()",
	"function f() { let g = function() { y }; if (true) { let y = 9 }; g() } let y = 3; f()",
	// By reference: = is seen by every closure that shares the variable.
	"function counter() { let n = 0; [function() { n += 1 }, function() { n }] } let c = counter(); c[0](); c[0](); c[1]()",
	"let make = function() { let v = 1; let set = function(x) { v = x }; let get = function() { v }; {\"set\": set, \"get\": get} }; let o = make(); o.set(42); o.get()",
	// Each round of a loop has its own variables; the for header has one for the whole loop.
	"let fs = []; for (let i = 0; i < 3; i += 1) { let j = i; push(fs, function() { [i, j] }) } [fs[0](), fs[1](), fs[2]()]",
	"let fs = []; let i = 0; while (i < 3) { let k = i * 10; push(fs, function() { k }); i += 1 } [fs[0](), fs[2]()]",
	"let fs = []; for (let i = 0; i < 3; i += 1) { push(fs, function() { i }) } fs[0]()",
	// let twice in a scope replaces; = on an undeclared name fails.
	"let a = 1; let a = a + 1; a",
	"function f() { let a = 1; let g = function() { a }; let a = 5; g() } f()",
	"function f() { b = 1 } f()",
	"function f() { let g = function() { q = 1 }; g() } f()",
	"function f() { let g = function() { q = 1 }; let r = 0; try { g() } catch (e) { r = e.message }; let q = 0; g(); [r, q] } f()",
	// Names after the scopes: builtins, null, the libraries, and a let that hides them.
	"let len = function(x) { 99 }; len([1])",
	"function f() { let len = 3; len } [f(), len([1, 2])]",
	"function f() { let g = function() { len }; let len = 4; g() } f()",
	"function f() { let g = function() { math }; let r = type(g()); let math = 1; [r, g()] } f()",
	"[null, type(text), type(list)]",
	"let null = 5; null",
	"function f() { nope } f()",
	"function f() { let g = function() { nope }; g() } f()",
	// Default parameters, at each call, in the call's scope.
	"function f(a, b = a * 2) { [a, b] } [f(1), f(1, 5)]",
	"let b = 100; function f(a = b, b = 1) { [a, b] } [f(), f(7), f(7, 8)]",
	"function f(a = []) { push(a, 1); a } [f(), f()]",
	"function f(a, b = function() { a }) { b() } f(3)",
	"function f(a = function() { b }, b = 2) { a() } f()",
	"function f(a, a = 5) { a } [f(1), f(1, 2)]",
	"function f(a, a) { a } f(1, 2)",
	"function f(a = if (true) { return 3 }) { a } f()",
	"function f(a = nope) { a } f()",
	"function f(a, b = 1) { a } f()",
	"function f(a, b = 1) { a } f(1, 2, 3)",
	"function f(a) { a } f()",
	// Values of statements.
	"function f() { if (true) { 5 } } f()",
	"function f() { let x = 1 } f()",
	"function f() { while (false) {} } f()",
	"function f() { return } f()",
	"return 5; puts(1)",
	"if (true) { return 6 }; 7",
	"let r = for (let i = 0; i < 3; i += 1) { if (i == 1) { break } }; r",
	"function f() {} f()",
	"",
	// A signal used as a value: what the evaluator gives back.
	"function f() { let x = if (true) { return 1 }; puts(\"after\"); 2 } f()",
	"function f() { let x = if (true) { return 1 }; x; puts(\"after\"); 2 } f()",
	"function f() { let x = if (true) { return 1 }; type(x) } f()",
	"function f() { let x = if (true) { return 1 }; x } f()",
	"function f() { let x = if (true) { return 1 }; return x } let v = f(); type(v)",
	"function f() { let x = if (true) { return 1 }; return x } f(); puts(\"next\")",
	"function f() { let x = if (true) { return 1 }; return x } function g() { f(); puts(\"g goes on\"); 3 } g()",
	"let n = 0; while (n < 5) { n += 1; let b = if (n == 2) { break }; puts(n, type(b)) }",
	"let n = 0; while (n < 5) { n += 1; let b = if (n == 2) { break }; b; puts(n) }; n",
	"let n = 0; while (n < 5) { n += 1; let b = if (n == 2) { continue }; b; puts(n) }; n",
	"function f() { while (true) { let b = if (true) { break }; return b } } let r = f(); type(r)",
	"function f() { while (true) { let b = if (true) { break }; return b } } function g() { while (true) { f(); puts(\"broke?\"); return 1 } } g()",
	"function f() { let b = 0; while (true) { b = if (true) { break }; break }; b } let v = f(); [type(v), v]",
	"function f() { let b = 0; while (true) { b = if (true) { break }; break }; b } function g() { let i = 0; while (i < 3) { i += 1; f(); puts(i) } } g()",
	"let x = while (true) { break }; x",
	"function f() { let x = while (true) { return 4 }; puts(type(x)); 1 } f()",
	"function f() { puts(if (true) { return 9 }); 1 } f()",
	"function f() { [if (true) { return 9 }] } let a = f(); [type(a[0]), a[0]]",
	"function f() { let a = [if (true) { return 9 }]; a[0]; 2 } f()",
	"function f() { 1 + if (true) { return 9 } } f()",
	"function f() { -if (true) { return 9 } } f()",
	"function f() { if (if (true) { return 1 }) { 2 } } f()",
	"function f() { let h = {\"k\": if (true) { return 1 }}; h.k; 5 } f()",
	"function f() { let rv = if (true) { return 1 }; let g = function() { rv }; g(); 7 } f()",
	"function f() { let rv = if (true) { return 1 }; rv == rv } f()",
	"function f() { let rv = if (true) { return 1 }; [str(rv), rv] } f()",
	"let x = if (true) { return 3 }; puts(\"top goes on\"); x",
	"let x = if (true) { return 3 }; x; puts(\"stopped?\")",
	"function f() { let b = 0; while (true) { b = if (true) { break }; break }; b } let v = f(); v; puts(\"top continues after a break value\"); 1",
	"function f() { let b = 0; while (true) { b = if (true) { break }; break }; b } let v = f(); layer a\n  v\n  puts(\"skipped\")\nputs(\"after layer\")",
	"function f() { try { let x = if (true) { return 1 }; x } catch (e) { 2 }; 3 } f()",
	"for (let i = if (true) { return 1 }; false; ) {}; 5",
	"for (let i = 0; i < 2; if (true) { return 1 }) { puts(i) }; 5",
	"function f() { let g = function() { return if (true) { return 1 } }; let r = g(); r; 2 } f()",
	// try and catch.
	"function f() { try { 1 / 0 } catch (e) { [e.message, e.line, e.column] } } f()",
	"function f(n) { if (n == 0) { 1 + true } else { f(n - 1) } } try { f(20) } catch (e) { e.message }",
	"let log = []; try { push(log, 1); nope; push(log, 2) } catch (e) { push(log, e.message) }; log",
	"try { try { nope } catch (e) { error(\"again\") } } catch (e) { e.message }",
	"function f() { try { return 1 } catch (e) { 2 } } f()",
	"let i = 0; while (i < 5) { i += 1; try { if (i == 3) { break } } catch (e) {} }; i",
	"let fs = []; for (let i = 0; i < 2; i += 1) { try { nope } catch (e) { push(fs, function() { e.message }) } } fs[1]()",
	"try { while (true) {} } catch (e) { 1 }",
	"try { list.sort([3, 1, 2], function(a, b) { nope }) } catch (e) { e.message }",
	"list.sort([3, 1, 2], function(a, b) { a > b })",
	"list.sort([3, 1, 2], function(a) { a })",
	"try { error(\"x\") } catch { 5 }",
	"function f() { let x = 1; try { let x = 2; nope } catch (e) { x } } f()",
	"function f() { let g = function() { e }; try { nope } catch (e) { g() } } f()",
	// Layers and hooks, also when a layer ends by an error, a return or a break.
	"layer a\n  puts(1)\n  layer b\n    puts(2)\nputs(3)",
	"function f() {\n  layer a\n    return 5\n  9\n}\nf()",
	"let n = 0\nwhile (n < 5) {\n  layer a\n    n += 1\n    if (n == 2) { break }\n}\nn",
	"let n = 0\nwhile (n < 3) {\n  n += 1\n  layer a\n    if (n == 2) { continue }\n    puts(n)\n}",
	"layer a\n  nope",
	"try {\n  layer a\n    layer b\n      nope\n} catch (e) { e.message }",
	"function f() {\n  layer inner\n    1 + true\n}\ntry { f() } catch (e) { e.line }",
	"let x = if (true) {\n  layer a\n    return 1\n}\nx",
	"layer a\n  let v = 3\n  function g() { v }\n  puts(g())",
	// Steps: the limit stops at the same loop or call.
	"let i = 0; while (true) { i += 1 }",
	"function f(n) { f(n + 1) } f(0)",
	"function f(n) { if (n == 0) { 0 } else { 1 + f(n - 1) } } f(15000)",
	"let a = []; for (let i = 0; i < 100; i += 1) { push(a, i) }; len(a)",
	"for (;;) { puts(1) }",
	// Functions are values.
	"let f = function(a, b = 2) { a + b }; [type(f), f]",
	"function f() {} puts(f); f == f",
	"function f() {} function g() {} [f == g, f === f, f != g]",
	"let h = {\"add\": function(a, b) { a + b }}; h.add(2, 3)",
	"function fib(n) { if (n < 2) { n } else { fib(n - 1) + fib(n - 2) } } fib(15)",
	"let compose = function(f, g) { function(x) { f(g(x)) } }; compose(function(x) { x * 2 }, function(x) { x + 1 })(5)",
	"5(1)",
	"let x = 1; x()",
	"len(1, 2)",
	"error(\"stop\")",
	"error(1, 2)",
	"push(1, 2)",
	// Assignments.
	"let a = [1, 2]; a[0] += 5; a",
	"let h = {}; h.n = 1; h.n += 2; h",
	"let h = {}; h.n += 2",
	"let s = \"ab\"; s[0] = \"x\"",
	"let a = [1]; a[5] = 1",
	"x += 1",
	"let len = 1; len += 1; len",
	"len += 1",
	"function f() { let a = 1; let g = function() { a += 1 }; g(); g(); a } f()",
	// Hashes and points check as they go.
	"puts({[1]: puts(\"value\")})",
	"{puts(\"k\"): 1}",
	"({\"a\": 1, \"a\": 2, \"b\": 3})",
	"(1, puts(\"x\"))",
	"(\"a\", puts(\"never\"))",
	"let p = (1, 2.5); [p.x, p.y, p + p, -p]",
	"-\"a\"",
	"!5",
	"!!null",
	"1 && 0",
	"0 || \"\"",
	"puts(1) || puts(2)",
	"let r = 1 && nope",
	// Big numbers and the small-integer cache.
	"let a = 1000; let b = 1000; [a == b, a === b, a + 24, 9223372036854775807 + 1]",
	"let h = {}; h[1000 + 1] = 1; h[1001]",
	"-9223372036854775807 - 1",
	"(-9223372036854775807 - 1) / -1",
	"[7 % -2, -7 % 2, -7 // 2, 7 / 0]",
	"5 % 0",
	"2 ** -1",
}

// TestInteropBothWays calls functions of one engine from the other, through
// the same environment, both directly and from builtins.
func TestInteropBothWays(t *testing.T) {
	parse := func(input string) *lizardast.Program {
		p := parser.New(lexer.New(input))
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("syntax errors in %q: %v", input, p.Errors())
		}
		return program
	}

	tests := []struct {
		first, second string
		firstOnVM     bool
		expected      string
	}{
		// A tree-walker function called from the VM, and from list.sort there.
		{"function twice(x) { x * 2 } let base = 10", "twice(base) + 1", false, "21"},
		{"function less(a, b) { a > b }", "list.sort([1, 3, 2], less)", false, "[3, 2, 1]"},
		{"let n = 0; function bump() { n += 1 }", "bump(); bump(); n", false, "2"},
		// A VM closure called from the evaluator, and from list.sort there.
		{"function twice(x) { x * 2 } let base = 10", "twice(base) + 1", true, "21"},
		{"function less(a, b) { a > b }", "list.sort([1, 3, 2], less)", true, "[3, 2, 1]"},
		{"function make() { let n = 0; function() { n += 1 } } let c = make()", "c(); c(); c()", true, "3"},
		{"function f(a, b = a + 1) { [a, b] }", "f(1)", true, "[1, 2]"},
		{"function f(a) { a }", "f()", true, "[line 1, column 2] runtime error: wrong number of arguments: want 1, got 0"},
		{"function f() { nope }", "try { f() } catch (e) { e.message }", true, "\"identifier not found: nope\""},
		{"function f() {}", "[type(f), f == f]", true, "[\"FUNCTION\", true]"},
		{"function f(a, b = 2) { a }", "puts(f); str(f)", true, "\"function(a, b = 2) { a }\""},
		// A global defined later by the host is found by name.
		{"function g() { later }", "let later = 5; g()", true, "5"},
	}

	for _, tt := range tests {
		for _, mode := range []string{"vm-then-tree", "tree-then-vm"} {
			if (mode == "vm-then-tree") != tt.firstOnVM {
				continue
			}
			var out bytes.Buffer
			evaluator.Output = &out
			env := object.NewEnvironment()
			first, second := parse(tt.first), parse(tt.second)
			var result object.Object
			if tt.firstOnVM {
				if r := Run(first, env); r != nil && r.Type() == object.ERROR_OBJ {
					t.Fatalf("%q: %s", tt.first, r.Inspect())
				}
				result = evaluator.Eval(second, env)
			} else {
				if r := evaluator.Eval(first, env); r != nil && r.Type() == object.ERROR_OBJ {
					t.Fatalf("%q: %s", tt.first, r.Inspect())
				}
				result = Run(second, env)
			}
			evaluator.Output = os.Stdout
			got := result.Inspect()
			if result.Type() == object.STRING_OBJ {
				got = "\"" + got + "\""
			}
			if got != tt.expected {
				t.Errorf("%s: %q then %q: expected %s, got %s", mode, tt.first, tt.second, tt.expected, got)
			}
		}
	}
}
