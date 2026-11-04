package evaluator

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"strings"
	"testing"
)

// loadNamed parses input as the source called name and runs it in env.
func loadNamed(t *testing.T, input, name string, env *object.Environment) object.Object {
	t.Helper()

	p := parser.New(lexer.NewNamed(input, name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("source %q - unexpected syntax errors:\n%s", name, strings.Join(p.Errors(), "\n"))
	}
	return Eval(program, env)
}

// TestErrorSource checks that an error carries the name of the source it
// happened in, even when the function was called from another source, and
// that the message stays the same with or without names.
func TestErrorSource(t *testing.T) {
	env := object.NewEnvironment()
	loadNamed(t, "let a = 1\nfunction boom(x) {\n  x + nope\n}\n", "lib.liz", env)

	got := loadNamed(t, "let b = 2\nboom(b)\n", "user", env)
	e, ok := got.(*object.Error)
	if !ok {
		t.Fatalf("expected an error, got %v", got)
	}
	if e.Source != "lib.liz" || e.Line != 3 || e.Column != 7 {
		t.Errorf("error at %q line %d column %d, want lib.liz 3:7", e.Source, e.Line, e.Column)
	}
	want := "[line 3, column 7] runtime error: identifier not found: nope"
	if e.Message != want {
		t.Errorf("message %q, want %q", e.Message, want)
	}

	// An error in the caller's own source names that source.
	e, _ = loadNamed(t, "1 + true", "user", env).(*object.Error)
	if e == nil || e.Source != "user" || e.Line != 1 {
		t.Errorf("expected an error from user at line 1, got %+v", e)
	}

	// Unnamed sources give an empty Source and the same message.
	named := loadNamed(t, "\n1 + true", "x", object.NewEnvironment()).(*object.Error)
	plain := run(t, "\n1 + true").(*object.Error)
	if plain.Source != "" || plain.Message != named.Message {
		t.Errorf("unnamed: Source %q, message %q vs %q", plain.Source, plain.Message, named.Message)
	}
}

// TestCaughtErrorKeepsMessage checks that a caught error still shows the
// message without any source name.
func TestCaughtErrorKeepsMessage(t *testing.T) {
	env := object.NewEnvironment()
	got := loadNamed(t, `try { 1 + true } catch (e) { e.message }`, "lib.liz", env)
	if strings.Contains(got.Inspect(), "lib.liz") {
		t.Errorf("message mentions the source: %s", got.Inspect())
	}
}

// TestOnStatementSources checks that a host can read the source of each
// statement from its token, also for statements of a function that runs
// later from another source.
func TestOnStatementSources(t *testing.T) {
	var seen []string
	OnStatement = func(node ast.Statement) {
		switch n := node.(type) {
		case *ast.LetStatement:
			seen = append(seen, fmt.Sprintf("let@%s:%d", n.Token.Source, n.Token.Line))
		case *ast.ExpressionStatement:
			seen = append(seen, fmt.Sprintf("expr@%s:%d", n.Token.Source, n.Token.Line))
		}
	}
	defer func() { OnStatement = nil }()

	env := object.NewEnvironment()
	loadNamed(t, "function f() {\n  1\n}\n", "lib.liz", env)
	loadNamed(t, "f()\n", "user", env)

	want := "let@lib.liz:1 expr@user:1 expr@lib.liz:2"
	if got := strings.Join(seen, " "); got != want {
		t.Errorf("statements %q, want %q", got, want)
	}
}
