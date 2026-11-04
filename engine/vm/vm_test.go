package vm

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/compiler"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"strings"
	"testing"
)

func parse(t *testing.T, input string) *ast.Program {
	t.Helper()
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("input %q - syntax errors: %v", input, p.Errors())
	}
	return program
}

// show writes a value the way the evaluator's tests do: strings in quotes.
func show(result object.Object) string {
	if result.Type() == object.STRING_OBJ {
		return "\"" + result.Inspect() + "\""
	}
	return result.Inspect()
}

// TestScopes has every case of the evaluator's dynamic scopes with its
// expected value (the differential tests compare them with the evaluator too).
func TestScopes(t *testing.T) {
	tests := []struct{ input, expected string }{
		// let y = x; let x = 2 reads the outer x first, then the new one.
		{"let x = 1; function f() { let y = x; let x = 2; [y, x] } f()", "[1, 2]"},
		{"let x = 1; function f() { if (true) { let y = x; let x = 3; [y, x] } } f()", "[1, 3]"},
		// A closure made before a later let sees it once it exists.
		{"let y = 0; function f() { let g = function() { y }; let a = g(); let y = 7; [a, g()] } f()", "[0, 7]"},
		// By reference.
		{"function f() { let n = 0; let inc = function() { n += 1 }; inc(); inc(); n } f()", "2"},
		// A fresh scope per round.
		{"let fs = []; for (let i = 0; i < 3; i += 1) { let j = i; push(fs, function() { j }) } [fs[0](), fs[2]()]", "[0, 2]"},
		// The for header is one scope for the whole loop.
		{"let fs = []; for (let i = 0; i < 3; i += 1) { push(fs, function() { i }) } fs[0]()", "3"},
		// let twice redefines, for closures too.
		{"function f() { let a = 1; let g = function() { a }; let a = 5; g() } f()", "5"},
		// The order after the scopes: builtins, null, libraries; a let hides them.
		{"function f() { let len = 3; len } [f(), len([1, 2])]", "[3, 2]"},
		{"[type(null), type(math), type(text), type(list)]", "[\"NULL\", \"HASH\", \"HASH\", \"HASH\"]"},
		// Defaults run at each call in the call's scope.
		{"let b = 100; function f(a = b, b = 1) { [a, b] } [f(), f(7)]", "[[100, 1], [7, 1]]"},
		{"function f(a = []) { push(a, 1) } [f(), f()]", "[[1], [1]]"},
	}
	for _, tt := range tests {
		result := Run(parse(t, tt.input), object.NewEnvironment())
		if got := show(result); got != tt.expected {
			t.Errorf("input %q - expected %s, got %s", tt.input, tt.expected, got)
		}
	}
}

func TestScopeErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"function f() { b = 1 } f()", "[line 1, column 16] runtime error: can't assign to b: it was never declared, use let b = ..."},
		{"function f() { let g = function() { y }; let r = g(); let y = 1 } f()", "[line 1, column 37] runtime error: identifier not found: y"},
		{"function f(a) { a } f()", "[line 1, column 22] runtime error: wrong number of arguments: want 1, got 0"},
		{"function f(a, b = 1) { a } f()", "[line 1, column 29] runtime error: wrong number of arguments: want 1 to 2, got 0"},
	}
	for _, tt := range tests {
		result := Run(parse(t, tt.input), object.NewEnvironment())
		if got := result.Inspect(); got != tt.expected {
			t.Errorf("input %q - expected %s, got %s", tt.input, tt.expected, got)
		}
	}
}

// TestEnvironment checks the contract of Run with the host's environment:
// top-level lets are in it afterwards, and names are looked up in it when
// they are read, so globals added after compiling are found.
func TestEnvironment(t *testing.T) {
	env := object.NewEnvironment()
	env.Define("given", &object.Integer{Value: 4})
	if result := Run(parse(t, "let made = given * 2; function later() { added }"), env); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if made, ok := env.Get("made"); !ok || made.Inspect() != "8" {
		t.Errorf("made should be 8 in the environment, got %v", made)
	}

	unit, err := compiler.Compile(parse(t, "later()"))
	if err != nil {
		t.Fatal(err)
	}
	if result := RunUnit(unit, env); !strings.Contains(result.Inspect(), "identifier not found: added") {
		t.Errorf("expected added to be missing, got %s", result.Inspect())
	}
	env.Define("added", &object.Integer{Value: 9})
	if result := RunUnit(unit, env); result.Inspect() != "9" {
		t.Errorf("a global added after compiling should be found, got %s", result.Inspect())
	}
}

func withMaxSteps(n int, f func()) {
	evaluator.MaxSteps = n
	evaluator.ResetSteps()
	defer func() { evaluator.MaxSteps = 0 }()
	f()
}

// TestStepLimit checks that the limit stops at the same token as the
// evaluator, and that a try doesn't catch it.
func TestStepLimit(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let i = 0\nwhile (true) { i += 1 }", "[line 2, column 1] runtime error: the program ran too long (more than 100 steps)"},
		{"function f() { f() }\nf()", "[line 1, column 17] runtime error: the program ran too long (more than 100 steps)"},
		{"try { for (;;) {} } catch (e) { 1 }", "[line 1, column 7] runtime error: the program ran too long (more than 100 steps)"},
		{"let a = []; for (let i = 0; i < 200; i += 1) { push(a, i) }", "[line 1, column 13] runtime error: the program ran too long (more than 100 steps)"},
	}
	for _, tt := range tests {
		withMaxSteps(100, func() {
			result := Run(parse(t, tt.input), object.NewEnvironment())
			err, ok := result.(*object.Error)
			if !ok || err.Message != tt.expected || !err.Uncatchable {
				t.Errorf("input %q - expected uncatchable %q, got %s", tt.input, tt.expected, result.Inspect())
			}
			if evaluator.Steps() != 101 {
				t.Errorf("input %q - expected 101 steps, got %d", tt.input, evaluator.Steps())
			}
		})
	}
}

func TestCallDepth(t *testing.T) {
	input := "function f(n) { f(n + 1) }\ntry { f(0) } catch (e) { 1 }"
	result := Run(parse(t, input), object.NewEnvironment())
	err, ok := result.(*object.Error)
	if !ok || !err.Uncatchable || err.Message != "[line 1, column 18] runtime error: stack overflow: more than 10000 nested calls" {
		t.Fatalf("expected an uncatchable stack overflow, got %s", result.Inspect())
	}
	if evaluator.CallDepth() != 0 {
		t.Errorf("the call depth should be back to 0, got %d", evaluator.CallDepth())
	}

	// 9999 nested calls are fine.
	result = Run(parse(t, "function f(n) { if (n == 0) { 0 } else { 1 + f(n - 1) } }\nf(9999)"), object.NewEnvironment())
	if result.Inspect() != "9999" {
		t.Errorf("expected 9999, got %s", result.Inspect())
	}
}

// TestHooks records the hooks on the VM, including the layer ends by an
// error, a return and a break.
func TestHooks(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"layer a\n  let x = 1\nx", []string{"stmt layer a|  let x = 1;", "enter a", "stmt let x = 1;", "exit a", "stmt x"}},
		{"function f() {\n  layer a\n    return 1\n}\nf()", []string{"stmt let f = function() { layer a|  return 1; };", "stmt f()", "stmt layer a|  return 1;", "enter a", "stmt return 1;", "exit a"}},
		{"while (true) {\n  layer a\n    break\n}", []string{"stmt while true { layer a|  break; }", "stmt layer a|  break;", "enter a", "stmt break;", "exit a"}},
		{"layer a\n  layer b\n    nope", []string{"stmt layer a|  layer b|    nope", "enter a", "stmt layer b|  nope", "enter b", "stmt nope", "exit b", "exit a"}},
	}
	for _, tt := range tests {
		var events []string
		evaluator.OnStatement = func(node ast.Statement) {
			events = append(events, "stmt "+strings.ReplaceAll(node.String(), "\n", "|"))
		}
		evaluator.OnLayer = func(name string, enter bool, tok token.Token) {
			events = append(events, fmt.Sprintf("%s %s", map[bool]string{true: "enter", false: "exit"}[enter], name))
		}
		Run(parse(t, tt.input), object.NewEnvironment())
		evaluator.OnStatement, evaluator.OnLayer = nil, nil
		if strings.Join(events, "\n") != strings.Join(tt.expected, "\n") {
			t.Errorf("input %q - hooks:\n%s\nwant:\n%s", tt.input, strings.Join(events, "\n"), strings.Join(tt.expected, "\n"))
		}
	}
}
