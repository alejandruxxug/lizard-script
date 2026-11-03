package evaluator

import (
	"fmt"
	"lizardscript/ast"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"lizardscript/token"
	"strings"
	"testing"
)

func TestLayerEvaluation(t *testing.T) {
	checkAll(t, []expectation{
		{"layer a\n  1", "null"},
		{"layer a", "null"},
		{"let x = 1\nlayer a\n  let x = 2\nx", "1"},
		{"let x = 1\nlayer a\n  x = 2\nx", "2"},
		{"let x = 1\nlayer a\n  let y = x + 1\n  y", "null"},
		{"layer a\n  let inner = 5\nlet r = 0\nlayer b\n  r = 7\nr", "7"},
		{"let r = 0\nlayer a\n  layer b\n    r = 3\nr", "3"},
		{"function f() {\n  layer a\n    return 5\n  9\n}\nf()", "5"},
		{"function f() {\n  layer a\n    1\n  9\n}\nf()", "9"},
		{"let n = 0\nwhile (n < 5) {\n  layer a\n    n = n + 1\n    if (n == 2) { break }\n}\nn", "2"},
		{"let layer = 1\nlayer + 1", "2"},
		{"let layer = 1\nlayer = 2\nlayer", "2"},
		{"let layer = function(x) { x * 2 }\nlayer(4)", "8"},
		{"let layer = 3\nlayer", "3"},
	})
}

func TestLayerErrorsPassThrough(t *testing.T) {
	checkErrors(t, []expectation{
		{"layer a\n  1 + true", "[line 2, column 5] runtime error: type mismatch: INTEGER + BOOLEAN"},
		{"layer a\n  nope", "[line 2, column 3] runtime error: identifier not found: nope"},
	})
}

// recorder installs both hooks and collects what they see. It returns the
// events and a function that puts the hooks back.
func recorder() (*[]string, func()) {
	events := &[]string{}
	OnStatement = func(node ast.Statement) {
		*events = append(*events, "stmt "+strings.ReplaceAll(node.String(), "\n", "|"))
	}
	OnLayer = func(name string, enter bool, tok token.Token) {
		word := "exit"
		if enter {
			word = "enter"
		}
		*events = append(*events, fmt.Sprintf("%s %s @%d:%d", word, name, tok.Line, tok.Column))
	}
	return events, func() { OnStatement, OnLayer = nil, nil }
}

func TestHooksAreNilByDefault(t *testing.T) {
	if OnStatement != nil || OnLayer != nil {
		t.Fatal("the hooks must be nil until a host sets them")
	}
}

func TestHooksOrder(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"let x = 1\nx", []string{"stmt let x = 1;", "stmt x"}},
		{"layer sun\n  let a = 1\nlet b = 2", []string{
			"stmt layer sun|  let a = 1;",
			"enter sun @1:1",
			"stmt let a = 1;",
			"exit sun @1:1",
			"stmt let b = 2;",
		}},
		{"layer \"big sun\"\n  layer ray\n    1\n  2", []string{
			"stmt layer \"big sun\"|  layer ray|    1|  2",
			"enter big sun @1:1",
			"stmt layer ray|  1",
			"enter ray @2:3",
			"stmt 1",
			"exit ray @2:3",
			"stmt 2",
			"exit big sun @1:1",
		}},
		{"function f() {\n  layer a\n    return 1\n}\nf()", []string{
			"stmt let f = function() { layer a|  return 1; };",
			"stmt f()",
			"stmt layer a|  return 1;",
			"enter a @2:3",
			"stmt return 1;",
			"exit a @2:3",
		}},
		{"layer a\n  nope\n1", []string{
			"stmt layer a|  nope",
			"enter a @1:1",
			"stmt nope",
			"exit a @1:1",
		}},
		{"while (true) {\n  layer a\n    break\n}", []string{
			"stmt while true { layer a|  break; }",
			"stmt layer a|  break;",
			"enter a @2:3",
			"stmt break;",
			"exit a @2:3",
		}},
	}
	for _, tt := range tests {
		events, restore := recorder()
		run(t, tt.input)
		restore()
		if strings.Join(*events, "\n") != strings.Join(tt.expected, "\n") {
			t.Errorf("input %q\nexpected:\n%s\n     got:\n%s", tt.input,
				strings.Join(tt.expected, "\n"), strings.Join(*events, "\n"))
		}
	}
}

func TestHookStatementCarriesItsPlace(t *testing.T) {
	var lines []int
	OnStatement = func(node ast.Statement) {
		if let, ok := node.(*ast.LetStatement); ok {
			lines = append(lines, let.Token.Line)
		}
	}
	defer func() { OnStatement = nil }()

	p := parser.New(lexer.New("let a = 1\n\nlayer x\n  let b = 2"))
	Eval(p.ParseProgram(), object.NewEnvironment())
	if len(lines) != 2 || lines[0] != 1 || lines[1] != 4 {
		t.Errorf("expected lets on lines [1 4], got %v", lines)
	}
}
