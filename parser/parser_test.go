package parser

import (
	"fmt"
	"lizardscript/ast"
	"lizardscript/lexer"
	"strings"
	"testing"
)

// parse runs the parser and fails the test on any error.
func parse(t *testing.T, input string) *ast.Program {
	t.Helper()

	p := New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("input %q - unexpected errors:\n%s", input, strings.Join(p.Errors(), "\n"))
	}
	return program
}

// TestStatements checks the tree of each construct through its String form.
func TestStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let x = 5;", "let x = 5;"},
		{"let x = 5", "let x = 5;"},
		{"let x1 = y2;", "let x1 = y2;"},
		{"return x;", "return x;"},
		{"return;", "return;"},
		{"return", "return;"},
		{"function f() { return }", "let f = function() { return; };"},
		{"while (a) { break; }", "while a { break; }"},
		{"for (;;) { break }", "for (; ; ) { break; }"},
		{"while (a) { if (b) { break; } }", "while a { if b { break; } }"},
		{";;", ""},

		{"if (a) { 1 }", "if a { 1 }"},
		{"if a { 1 }", "if a { 1 }"},
		{"if (a) { 1 } else { 2 }", "if a { 1 } else { 2 }"},
		{"if (a) { 1 } elif (b) { 2 }", "if a { 1 } else { if b { 2 } }"},
		{"if (a) { 1 } elif (b) { 2 } elif (c) { 3 } else { 4 }",
			"if a { 1 } else { if b { 2 } else { if c { 3 } else { 4 } } }"},
		{"if (a) { 1 } else if (b) { 2 }", "if a { 1 } else { if b { 2 } }"},
		{"if (a) { 1 } else if (b) { 2 } else if (c) { 3 } else { 4 }",
			"if a { 1 } else { if b { 2 } else { if c { 3 } else { 4 } } }"},
		{"if (a) { 1 } elif (b) { 2 } else if (c) { 3 } else { 4 }",
			"if a { 1 } else { if b { 2 } else { if c { 3 } else { 4 } } }"},
		{"if (a) { 1 }\nelse\nif (b) { 2 }", "if a { 1 } else { if b { 2 } }"},
		{"if (a) { if (b) { 1 } }", "if a { if b { 1 } }"},
		{"let x = if (a) { 1 } else { 2 };", "let x = if a { 1 } else { 2 };"},

		{"while (x > 0) { x = x - 1; }", "while (x > 0) { (x = (x - 1)) }"},
		{"for (let i = 0; i < 3; i = i + 1) { f(i); }",
			"for (let i = 0; (i < 3); (i = (i + 1))) { f(i) }"},
		{"for (; i < 3;) { }", "for (; (i < 3); ) { }"},
		{"for (i = 0;;) { }", "for ((i = 0); ; ) { }"},

		{"function add(x, y) { return x + y; }", "let add = function(x, y) { return (x + y); };"},
		{"function add(x, y) { return x + y; };", "let add = function(x, y) { return (x + y); };"},
		{"function none() { }", "let none = function() { };"},
		{"let add = function(x, y) { x + y };", "let add = function(x, y) { (x + y) };"},
		{"function(x) { x }(5);", "function(x) { x }(5)"},
		{"f();", "f()"},
		{"f(1, 2 * 3, g(4));", "f(1, (2 * 3), g(4))"},

		{"[]", "[]"},
		{`[1, 2 + 3, "a", [true]]`, `[1, (2 + 3), "a", [true]]`},
		{"a[0]", "(a[0])"},
		{"m[i][j]", "((m[i])[j])"},
		{"a[0] = 1;", "((a[0]) = 1)"},
		{"{}", "{}"},
		{`{"a": 1, "b": 2 + 3}`, `{"a": 1, "b": (2 + 3)}`},
		{`{"a": 1,}`, `{"a": 1}`},
		{`{1: true, x: [2]}`, `{1: true, x: [2]}`},
		{`let h = {"a": {"b": 1}}; h["a"]["b"]`, `let h = {"a": {"b": 1}};((h["a"])["b"])`},

		{"-~> a comment\nlet x = 5; -~> another one\n-~> and a last one", "let x = 5;"},
	}

	for _, tt := range tests {
		if got := parse(t, tt.input).String(); got != tt.expected {
			t.Errorf("input %q\n expected: %s\n got:      %s", tt.input, tt.expected, got)
		}
	}
}

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 * 2 + 3", "((1 * 2) + 3)"},
		{"1 + 2 + 3", "((1 + 2) + 3)"},
		{"1 - 2 - 3", "((1 - 2) - 3)"},
		{"8 / 4 // 2 % 3", "(((8 / 4) // 2) % 3)"},
		{"(1 + 2) * 3", "((1 + 2) * 3)"},

		{"-a * b", "((-a) * b)"},
		{"!a == b", "((!a) == b)"},
		{"!!a", "(!(!a))"},
		{"- -a", "(-(-a))"},

		{"2 ** 3 ** 2", "(2 ** (3 ** 2))"},
		{"-2 ** 2", "(-(2 ** 2))"},
		{"2 ** -1", "(2 ** (-1))"},
		{"2 * 3 ** 2", "(2 * (3 ** 2))"},

		{"1 + 2 < 3 * 4", "((1 + 2) < (3 * 4))"},
		{"a < b == c > d", "((a < b) == (c > d))"},
		{"a <= b != c >= d", "((a <= b) != (c >= d))"},
		{"a === b !== c", "((a === b) !== c)"},

		{"a || b && c", "(a || (b && c))"},
		{"a && b || c", "((a && b) || c)"},
		{"a || b || c", "((a || b) || c)"},
		{"a && b && c", "((a && b) && c)"},
		{"a == b && c < d", "((a == b) && (c < d))"},
		{"!a && b", "((!a) && b)"},

		{"a = b = 1", "(a = (b = 1))"},
		{"a = 1 + 2", "(a = (1 + 2))"},
		{"a = b || c", "(a = (b || c))"},

		{"a[0] + f(1)", "((a[0]) + f(1))"},
		{"a * b[0]", "(a * (b[0]))"},
		{"-a[0]", "(-(a[0]))"},
		{"-f(x)", "(-f(x))"},
		{"f(x)[0]", "(f(x)[0])"},
		{"2 ** f(x)", "(2 ** f(x))"},
	}

	for _, tt := range tests {
		if got := parse(t, tt.input).String(); got != tt.expected {
			t.Errorf("input %q\n expected: %s\n got:      %s", tt.input, tt.expected, got)
		}
	}
}

func TestLiteralValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"42", 42},
		{"010", 10}, // decimal, not octal
		{"08", 8},
		{"9223372036854775807", 9223372036854775807},
	}

	for _, tt := range tests {
		statement := parse(t, tt.input).Statements[0].(*ast.ExpressionStatement)
		literal, ok := statement.Expression.(*ast.IntegerLiteral)
		if !ok {
			t.Fatalf("input %q - expected *ast.IntegerLiteral, got %T", tt.input, statement.Expression)
		}
		if literal.Value != tt.expected {
			t.Errorf("input %q - expected %d, got %d", tt.input, tt.expected, literal.Value)
		}
	}

	statement := parse(t, `"hello -~> world"`).Statements[0].(*ast.ExpressionStatement)
	if literal := statement.Expression.(*ast.StringLiteral); literal.Value != "hello -~> world" {
		t.Errorf("wrong string value: %q", literal.Value)
	}

	statement = parse(t, "false").Statements[0].(*ast.ExpressionStatement)
	if literal := statement.Expression.(*ast.Boolean); literal.Value {
		t.Errorf("false parsed as true")
	}
}

// TestControlStatementsEndAtTheirBrace checks that an if, while, for or
// function that starts a statement doesn't swallow the next line.
func TestControlStatementsEndAtTheirBrace(t *testing.T) {
	tests := []struct {
		input      string
		statements int
	}{
		{"if (a) { 1 }\n-5;", 2},
		{"if (a) { 1 }\n[1, 2];", 2},
		{"if (a) { 1 } else { 2 }\n(b);", 2},
		{"while (a) { b }\n(c);", 2},
		{"while (a) { b }\n-c;", 2},
		{"for (;;) { b }\n[c];", 2},
		{"function f() { }\n-1;", 2},
		{"function f() { }\n(1);", 2},
		{"if (a) { 1 }; -5;", 2},
		{"if (a) { 1 } if (b) { 2 }", 2},
		{"let x = 1\nlet y = 2\nx + y", 3},
	}

	for _, tt := range tests {
		if got := len(parse(t, tt.input).Statements); got != tt.statements {
			t.Errorf("input %q - expected %d statements, got %d", tt.input, tt.statements, got)
		}
	}
}

func TestFunctionDeclarationTree(t *testing.T) {
	program := parse(t, "function add(x, y) { return x + y; }")

	statement, ok := program.Statements[0].(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected *ast.LetStatement, got %T", program.Statements[0])
	}
	if statement.Name.Value != "add" {
		t.Errorf("wrong name: %q", statement.Name.Value)
	}

	function, ok := statement.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("expected *ast.FunctionLiteral, got %T", statement.Value)
	}
	if len(function.Parameters) != 2 || function.Parameters[1].Value != "y" {
		t.Errorf("wrong parameters: %v", function.Parameters)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let = 5", `[line 1, column 5] syntax error: expected identifier, got "="`},
		{"let x 5", `[line 1, column 7] syntax error: expected "=", got "5"`},
		{"let x = ;", `[line 1, column 9] syntax error: expected an expression, got ";"`},
		{"let", `[line 1, column 4] syntax error: expected identifier, got end of input`},
		{"let x = 5 6;", `[line 1, column 11] syntax error: expected end of statement, got "6"`},
		{"x y", `[line 1, column 3] syntax error: expected end of statement, got identifier "y"`},
		{"return 1 2", `[line 1, column 10] syntax error: expected end of statement, got "2"`},
		{"(1 + 2", `[line 1, column 7] syntax error: expected ")", got end of input`},
		{"1 +", `[line 1, column 4] syntax error: expected an expression, got end of input`},
		{"* 2", `[line 1, column 1] syntax error: expected an expression, got "*"`},
		{")", `[line 1, column 1] syntax error: expected an expression, got ")"`},
		{"}", `[line 1, column 1] syntax error: expected an expression, got "}"`},
		{"else { }", `[line 1, column 1] syntax error: expected an expression, got "else"`},

		{"if (x {", `[line 1, column 7] syntax error: expected ")", got "{"`},
		{"if (x) y", `[line 1, column 8] syntax error: expected "{", got identifier "y"`},
		{"if (x) { } else y", `[line 1, column 17] syntax error: expected "{", got identifier "y"`},
		{"if (x) {", `[line 1, column 8] syntax error: block is never closed: expected "}"`},
		{"while (a) {\n b;", `[line 1, column 11] syntax error: block is never closed: expected "}"`},
		{"while a", `[line 1, column 8] syntax error: expected "{", got end of input`},
		{"for i", `[line 1, column 5] syntax error: expected "(", got identifier "i"`},
		{"for (let i = 0\ni < 3) { }", `[line 2, column 1] syntax error: expected ";", got identifier "i"`},
		{"for (; i { }", `[line 1, column 10] syntax error: expected ";", got "{"`},
		{"for (;; i { }", `[line 1, column 11] syntax error: expected ")", got "{"`},

		{"function (1) { }", `[line 1, column 11] syntax error: expected identifier, got "1"`},
		{"function f(a, ) { }", `[line 1, column 15] syntax error: expected identifier, got ")"`},
		{"function f(a b) { }", `[line 1, column 14] syntax error: expected ")", got identifier "b"`},
		{"function f() return", `[line 1, column 14] syntax error: expected "{", got "return"`},
		{"f(1, 2", `[line 1, column 7] syntax error: expected ")", got end of input`},
		{"f(1,)", `[line 1, column 5] syntax error: expected an expression, got ")"`},
		{"[1, 2", `[line 1, column 6] syntax error: expected "]", got end of input`},
		{"a[1", `[line 1, column 4] syntax error: expected "]", got end of input`},
		{"a[]", `[line 1, column 3] syntax error: expected an expression, got "]"`},
		{`{"a" 1}`, `[line 1, column 6] syntax error: expected ":", got "1"`},
		{`{"a": 1 "b": 2}`, `[line 1, column 9] syntax error: expected ",", got "b"`},
		{`{"a": 1`, `[line 1, column 8] syntax error: expected ",", got end of input`},
		{"{ let x = 1; }", `[line 1, column 3] syntax error: expected an expression, got "let"`},

		{"1 = 2", `[line 1, column 3] syntax error: cannot assign to 1`},
		{"a + b = 2", `[line 1, column 7] syntax error: cannot assign to (a + b)`},
		{"f() = 2", `[line 1, column 5] syntax error: cannot assign to f()`},

		{"break;", `[line 1, column 1] syntax error: break outside of a loop`},
		{"if (a) { break; }", `[line 1, column 10] syntax error: break outside of a loop`},
		{"while (a) { function() { break; }; }", `[line 1, column 26] syntax error: break outside of a loop`},
		{"while (a) { }\nbreak;", `[line 2, column 1] syntax error: break outside of a loop`},

		{"99999999999999999999", `[line 1, column 1] syntax error: integer 99999999999999999999 is too large`},

		{"@", `[line 1, column 1] lexical error: unexpected character "@"`},
		{"let x = @;", `[line 1, column 9] lexical error: unexpected character "@"`},
		{"a @ b;", `[line 1, column 3] lexical error: unexpected character "@"`},
		{"a & b;", `[line 1, column 3] lexical error: unexpected character "&"`},
		{"-~", `[line 1, column 2] lexical error: unexpected character "~"`},
		{"let año = 1;", `[line 1, column 6] lexical error: unexpected character "ñ"`},
		{`"canción" @`, `[line 1, column 11] lexical error: unexpected character "@"`},
		{"a\x00b", `[line 1, column 2] lexical error: unexpected character "\x00"`},

		{"for (return 1;;) { }", `[line 1, column 6] syntax error: expected an expression, got "return"`},
		{"while (a) { for (break;;) { } }", `[line 1, column 18] syntax error: expected an expression, got "break"`},
		{"for (; @; i) { a; }", `[line 1, column 8] lexical error: unexpected character "@"`},
		{"for (f(1 2); ;) { a; }", `[line 1, column 10] syntax error: expected ")", got "2"`},
		{`let s = "open;`, `[line 1, column 9] lexical error: unterminated string`},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()

		errors := p.Errors()
		if len(errors) != 1 {
			t.Errorf("input %q - expected 1 error, got %d:\n%s",
				tt.input, len(errors), strings.Join(errors, "\n"))
			continue
		}
		if errors[0] != tt.expected {
			t.Errorf("input %q\n expected: %s\n got:      %s", tt.input, tt.expected, errors[0])
		}
	}
}

// TestErrorRecovery checks that each broken statement gives one error and
// that the good statements around it are still parsed.
func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		input      string
		errors     []string
		statements int
	}{
		{
			"let = 1;\nlet y = 2;\nlet z = ;\nlet w = 3;",
			[]string{
				`[line 1, column 5] syntax error: expected identifier, got "="`,
				`[line 3, column 9] syntax error: expected an expression, got ";"`,
			},
			2,
		},
		{
			"function f() {\n  let = 1;\n  return 2;\n}\nlet y = 1;",
			[]string{`[line 2, column 7] syntax error: expected identifier, got "="`},
			2,
		},
		{
			"if (x {\n  y;\n}\nlet z = 1;",
			[]string{`[line 1, column 7] syntax error: expected ")", got "{"`},
			1,
		},
		{
			"while (a) { let = 1 }\nlet z = 1;",
			[]string{`[line 1, column 17] syntax error: expected identifier, got "="`},
			2,
		},
		{
			`let h = {"a": 1 2}; let z = 1;`,
			[]string{`[line 1, column 17] syntax error: expected ",", got "2"`},
			1,
		},
		{
			"if (a) {\n  let h = {\"a\": 1 2};\n  let y = 2;\n}\nlet z = 1;",
			[]string{`[line 2, column 19] syntax error: expected ",", got "2"`},
			2,
		},
		{
			"for (let = 0; i < 3; i = i + 1) {\n  a;\n  b;\n}\nlet z = 1;",
			[]string{`[line 1, column 10] syntax error: expected identifier, got "="`},
			1,
		},
		{
			"let h = {\"f\": function() { let = 1; }};\nlet z = 1;",
			[]string{`[line 1, column 32] syntax error: expected identifier, got "="`},
			2,
		},
		{
			"let h = {\"a\": if (x) { let = 1; }, \"b\": 2};\nlet z = 1;",
			[]string{`[line 1, column 28] syntax error: expected identifier, got "="`},
			2,
		},
		{
			"}\nlet h = {\"a\": 1 2; \"b\": 3};\nlet z = 1;",
			[]string{
				`[line 1, column 1] syntax error: expected an expression, got "}"`,
				`[line 2, column 17] syntax error: expected ",", got "2"`,
			},
			1,
		},
		{
			"1 +\n@\nlet z = 1;",
			[]string{`[line 2, column 1] lexical error: unexpected character "@"`},
			0,
		},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		program := p.ParseProgram()

		got := strings.Join(p.Errors(), "\n")
		if expected := strings.Join(tt.errors, "\n"); got != expected {
			t.Errorf("input %q\n expected:\n%s\n got:\n%s", tt.input, expected, got)
		}
		if len(program.Statements) != tt.statements {
			t.Errorf("input %q - expected %d statements, got %d",
				tt.input, tt.statements, len(program.Statements))
		}
	}
}

// TestTruncatedInputs cuts a program at every position, which leaves broken
// input of all kinds. The parser must always finish, without panicking, and
// the tree it returns must be safe to print.
func TestTruncatedInputs(t *testing.T) {
	input := `function f(a, b) { for (let i = 0; i < a[1]; i = i + 1) { if (!a && b) { break; } elif (a) { return -1; } else { g({"k": [1, 2]}, "s"); } } } while (x) { y = 2 ** 3; }`

	for end := range len(input) {
		for start := range min(end, 40) {
			p := New(lexer.New(input[start:end]))
			_ = p.ParseProgram().String()
		}
	}
}

// TestWholePrograms parses complete programs: the README taste of the
// language and the example of the course's reference project.
func TestWholePrograms(t *testing.T) {
	readme := `let five = 5;
let ten = 10;

function add(x, y) {
  return x + y;
}

let result = add(five, ten); -~> this is a comment

while (result > 0) {
  if (result > 10 && five != 0) {
    result = result // 2;
  } elif (result == 10) {
    result = result - 1;
  } else {
    break;
  }
}

let lizard = {"name": "Liz", "toes": [5, 5, 5, 5]};
`

	reference := `let numeros = [2, 4, 6];

let sumar = function(lista) {
    let total = 0;
    for (let i = 0; i < 3; i = i + 1) {
        total = total + lista[i];
    }
    return total;
};

let datos = {"curso": "Compiladores", "factor": 2};
let resultado = sumar(numeros) * datos["factor"];

let ackermann = function(m, n) {
    if (m == 0) {
        return n + 1;
    }
    if (n == 0) {
        return ackermann(m - 1, 1);
    }
    return ackermann(m - 1, ackermann(m, n - 1));
};

let fibonacci = function(n) {
    if (n <= 1) {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
};

let hanoi = function(n, origen, destino, auxiliar) {
    if (n == 0) {
        return [];
    }
    return hanoi(n - 1, origen, auxiliar, destino) +
        [[origen, destino]] +
        hanoi(n - 1, auxiliar, destino, origen);
};

{"resultado": resultado, "ackermann": ackermann(2, 2), "fibonacci": fibonacci(34), "hanoi": hanoi(3, "A", "C", "B")};
`

	if got := len(parse(t, readme).Statements); got != 6 {
		t.Errorf("readme - expected 6 statements, got %d", got)
	}

	program := parse(t, reference)
	if got := len(program.Statements); got != 8 {
		t.Fatalf("reference - expected 8 statements, got %d", got)
	}

	hanoi := program.Statements[6].String()
	expected := "return ((hanoi((n - 1), origen, auxiliar, destino) + [[origen, destino]]) + hanoi((n - 1), auxiliar, destino, origen));"
	if !strings.Contains(hanoi, expected) {
		t.Errorf("hanoi body is grouped wrong:\n%s", hanoi)
	}
}

// TestNestingLimit checks that deep but reasonable nesting parses, and that
// nesting past the limit is one syntax error instead of a crash.
func TestNestingLimit(t *testing.T) {
	deep := []string{
		strings.Repeat("(", 5000) + "1" + strings.Repeat(")", 5000),
		strings.Repeat("-", 5000) + "1",
		strings.Repeat("[", 5000) + strings.Repeat("]", 5000),
		strings.Repeat("if (a) { ", 3000) + "1" + strings.Repeat(" }", 3000),
		"if (a) { 1 }" + strings.Repeat(" elif (a) { 1 }", 5000) + " else { 2 }",
	}
	for _, input := range deep {
		parse(t, input)
	}

	tooDeep := []string{
		strings.Repeat("(", 1000000) + "1" + strings.Repeat(")", 1000000),
		strings.Repeat("-", 1000000) + "1",
		strings.Repeat("{\"a\": ", 1000000) + "1" + strings.Repeat("}", 1000000),
		strings.Repeat("if (a) { ", 1000000) + "1" + strings.Repeat(" }", 1000000),
		strings.Repeat("function() { ", 1000000),
		strings.Repeat("x = ", 1000000) + "1",
	}
	for _, input := range tooDeep {
		p := New(lexer.New(input))
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) != 1 || !strings.HasSuffix(errors[0], "syntax error: the program is nested too deeply") {
			t.Errorf("input %q... - expected one nesting error, got %d: %.200s", input[:30], len(errors), strings.Join(errors, "\n"))
		}
	}
}

// TestDotAccess checks that a.b builds the same tree as a["b"].
func TestDotAccess(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a.b", `(a["b"])`},
		{"a.b.c", `((a["b"])["c"])`},
		{"a.b[0]", `((a["b"])[0])`},
		{"a.b(1)", `(a["b"])(1)`},
		{"f().x", `(f()["x"])`},
		{"a[0].b", `((a[0])["b"])`},
		{"a.b = 1", `((a["b"]) = 1)`},
		{"a.b.c = a.d", `(((a["b"])["c"]) = (a["d"]))`},
		{"-a.b", `(-(a["b"]))`},
		{"a.b ** 2", `((a["b"]) ** 2)`},
		{"a.b + c.d", `((a["b"]) + (c["d"]))`},
		{"a . b", `(a["b"])`},
		{"a.\nb", `(a["b"])`},
		{"{\"x\": 1}.x", `({"x": 1}["x"])`},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)
		if got := program.String(); got != tt.expected {
			t.Errorf("input %q - expected %s, got %s", tt.input, tt.expected, got)
		}
		// The same tree as the index form.
		if tt.input == "a.b" {
			if got := parse(t, `a["b"]`).String(); got != program.String() {
				t.Errorf("a.b and a[\"b\"] differ: %s and %s", program.String(), got)
			}
		}
	}
}

func TestDotErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a.if", `[line 1, column 3] syntax error: expected a name after "."`},
		{"a.", `[line 1, column 3] syntax error: expected a name after "."`},
		{"a.5", `[line 1, column 3] syntax error: expected a name after "."`},
		{`a."b"`, `[line 1, column 3] syntax error: expected a name after "."`},
		{"a..b", `[line 1, column 3] syntax error: expected a name after "."`},
		{".a", `[line 1, column 1] syntax error: expected an expression, got "."`},
		{`a."\q"`, `[line 1, column 3] lexical error: unknown escape \q in string`},
		{`let s = "a\qb"`, `[line 1, column 9] lexical error: unknown escape \q in string`},
		{`let s = "a\u12"`, `[line 1, column 9] lexical error: invalid unicode escape \u12: \u needs exactly 4 hex digits`},
		{`let s = "\uD83D\uDE00"`, `[line 1, column 9] lexical error: invalid unicode escape \uD83D: surrogate halves (D800 to DFFF) are not characters`},
		{`let s = "\u{110000}"`, `[line 1, column 9] lexical error: invalid unicode escape \u{110000}: code point above 10FFFF`},
		{`let s = "\u{41"`, `[line 1, column 9] lexical error: invalid unicode escape \u{41: \u{...} needs 1 to 6 hex digits and a closing }`},
		{`let s = "\u{}"`, `[line 1, column 9] lexical error: invalid unicode escape \u{}: \u{...} needs 1 to 6 hex digits and a closing }`},
		{`let s = "a" \ 1`, `[line 1, column 13] lexical error: unexpected character "\\"`},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) != 1 || errors[0] != tt.expected {
			t.Errorf("input %q\n expected: %s\n got:      %v", tt.input, tt.expected, errors)
		}
	}
}

func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1.5", 1.5},
		{"0.25", 0.25},
		{"10.0", 10},
		{"007.5", 7.5},
		{"45deg", 0.7853981633974483},
		{"1.5rad", 1.5},
		{"0.25turn", 1.5707963267948966},
		{"2.5px", 2.5},
		{"1e3", 1000},
		{"2.5e-4", 0.00025},
		{"1E+6", 1000000},
		{"3e0", 3},
		{"1e3rad", 1000},
	}
	for _, tt := range tests {
		statement := parse(t, tt.input).Statements[0].(*ast.ExpressionStatement)
		literal, ok := statement.Expression.(*ast.FloatLiteral)
		if !ok {
			t.Fatalf("input %q - expected *ast.FloatLiteral, got %T", tt.input, statement.Expression)
		}
		if literal.Value != tt.expected {
			t.Errorf("input %q - expected %v, got %v", tt.input, tt.expected, literal.Value)
		}
	}

	// px keeps an integer an integer.
	statement := parse(t, "10px").Statements[0].(*ast.ExpressionStatement)
	if literal, ok := statement.Expression.(*ast.IntegerLiteral); !ok || literal.Value != 10 {
		t.Errorf("10px - expected the integer 10, got %T", statement.Expression)
	}
}

func TestFloatTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1.5 + 2 * 0.5", "(1.5 + (2 * 0.5))"},
		{"-1.5 ** 2", "(-(1.5 ** 2))"},
		{"a.b * 1.5", "((a[\"b\"]) * 1.5)"},
		{"1.5.x", "(1.5[\"x\"])"},
		{"1.x", "(1[\"x\"])"},
	}
	for _, tt := range tests {
		if got := parse(t, tt.input).String(); got != tt.expected {
			t.Errorf("input %q - expected %s, got %s", tt.input, tt.expected, got)
		}
	}
}

func TestFloatErrors(t *testing.T) {
	huge := strings.Repeat("9", 400)
	tests := []struct{ input, expected string }{
		{"5kg", `[line 1, column 1] syntax error: unknown unit "kg"`},
		{"let x = 1.5kg", `[line 1, column 9] syntax error: unknown unit "kg"`},
		{"1x", `[line 1, column 1] syntax error: unknown unit "x"`},
		{"1e", `[line 1, column 1] syntax error: unknown unit "e"`},
		{"1e5kg", `[line 1, column 1] syntax error: unknown unit "kg"`},
		{"1e400", `[line 1, column 1] syntax error: number 1e400 is too large`},
		{"puts(5kg)", `[line 1, column 6] syntax error: unknown unit "kg"`},
		{"f(1, 2.5meters)", `[line 1, column 6] syntax error: unknown unit "meters"`},
		{"[1, 2kg]", `[line 1, column 5] syntax error: unknown unit "kg"`},
		{`{"a": 5kg}`, `[line 1, column 7] syntax error: unknown unit "kg"`},
		{"(5kg)", `[line 1, column 2] syntax error: unknown unit "kg"`},
		{"a[1x]", `[line 1, column 3] syntax error: unknown unit "x"`},
		{"5degx", `[line 1, column 1] syntax error: unknown unit "degx"`},
		{"5 kg", `[line 1, column 3] syntax error: expected end of statement, got identifier "kg"`},
		{"1 + 2.5meters", `[line 1, column 5] syntax error: unknown unit "meters"`},
		{huge + ".0", `[line 1, column 1] syntax error: number ` + huge + `.0 is too large`},
		{huge + "turn", `[line 1, column 1] syntax error: number +Inf is too large`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) != 1 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

func TestColorLiterals(t *testing.T) {
	tests := []struct {
		input      string
		r, g, b, a uint8
	}{
		{"#000", 0, 0, 0, 255},
		{"#fff", 255, 255, 255, 255},
		{"#f80", 255, 136, 0, 255},
		{"#3fa34d", 63, 163, 77, 255},
		{"#3FA34D", 63, 163, 77, 255},
		{"#3fa34d80", 63, 163, 77, 128},
		{"#00000000", 0, 0, 0, 0},
	}
	for _, tt := range tests {
		statement := parse(t, tt.input).Statements[0].(*ast.ExpressionStatement)
		color, ok := statement.Expression.(*ast.ColorLiteral)
		if !ok {
			t.Fatalf("input %q - expected *ast.ColorLiteral, got %T", tt.input, statement.Expression)
		}
		if color.R != tt.r || color.G != tt.g || color.B != tt.b || color.A != tt.a {
			t.Errorf("input %q - got %d %d %d %d", tt.input, color.R, color.G, color.B, color.A)
		}
	}
}

func TestPointTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"(1, 2)", "(1, 2)"},
		{"(1, 2, 3)", "(1, 2, 3)"},
		{"(1 + 2, 3 * 4)", "((1 + 2), (3 * 4))"},
		{"((1, 2))", "(1, 2)"},
		{"(1)", "1"},
		{"(1 + 2)", "(1 + 2)"},
		{"((1, 2), 3)", "((1, 2), 3)"},
		{"(1, 2) + (3, 4)", "((1, 2) + (3, 4))"},
		{"2 * (1, 2)", "(2 * (1, 2))"},
		{"-(1, 2)", "(-(1, 2))"},
		{"(1, 2).x", "((1, 2)[\"x\"])"},
		{"f((1, 2))", "f((1, 2))"},
		{"f((1, 2), 3)", "f((1, 2), 3)"},
		{"f(1, 2)", "f(1, 2)"},
		{"[(1, 2), (3, 4)]", "[(1, 2), (3, 4)]"},
		{"(a, b)", "(a, b)"},
		{"(f(1), g(2, 3))", "(f(1), g(2, 3))"},
		{"(1,\n2)", "(1, 2)"},
	}
	for _, tt := range tests {
		if got := parse(t, tt.input).String(); got != tt.expected {
			t.Errorf("input %q - expected %s, got %s", tt.input, tt.expected, got)
		}
	}
}

func TestColorAndPointErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"#12", `[line 1, column 1] lexical error: bad color #12`},
		{"let c = #12345", `[line 1, column 9] lexical error: bad color #12345`},
		{"#ggg", `[line 1, column 1] lexical error: bad color #ggg`},
		{"#1234", `[line 1, column 1] lexical error: bad color #1234`},
		{"f(#12)", `[line 1, column 3] lexical error: bad color #12`},
		{"(#12, 1)", `[line 1, column 2] lexical error: bad color #12`},
		{"#", `[line 1, column 1] lexical error: unexpected character "#"`},
		{"()", `[line 1, column 1] syntax error: a point has two or three coordinates`},
		{"let p = ()", `[line 1, column 9] syntax error: a point has two or three coordinates`},
		{"(1, 2, 3, 4)", `[line 1, column 1] syntax error: a point has two or three coordinates`},
		{"(1, 2, 3, 4, 5)", `[line 1, column 1] syntax error: a point has two or three coordinates`},
		{"(1, )", `[line 1, column 5] syntax error: expected an expression, got ")"`},
		{"(1, 2", `[line 1, column 6] syntax error: expected ")", got end of input`},
		{"(, 1)", `[line 1, column 2] syntax error: expected an expression, got ","`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) != 1 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

// lines prints each top-level statement of a program on its own line.
func lines(program *ast.Program) string {
	parts := make([]string, len(program.Statements))
	for i, s := range program.Statements {
		parts[i] = s.String()
	}
	return strings.Join(parts, "\n")
}

func TestLayerStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with a body", "layer sun\n  let a = 1\n  puts(a)", "layer sun\n  let a = 1;\n  puts(a)"},
		{"empty", "layer sun", "layer sun"},
		{"empty before another statement", "layer sun\nlet a = 1", "layer sun\nlet a = 1;"},
		{"the body ends at a dedent", "layer sun\n  let a = 1\nlet b = 2\n  let c = 3", "layer sun\n  let a = 1;\nlet b = 2;\nlet c = 3;"},
		{"nested", "layer a\n  layer b\n    1\n  2\n3", "layer a\n  layer b\n    1\n  2\n3"},
		{"siblings", "layer a\n  1\nlayer b\n  2", "layer a\n  1\nlayer b\n  2"},
		{"a string name", "layer \"big sun\"\n  1", "layer \"big sun\"\n  1"},
		{"a string that is a plain name", "layer \"sun\"", "layer sun"},
		{"a trailing comment", "layer sun -~> the sun\n  1", "layer sun\n  1"},
		{"a comment line inside", "layer sun\n  -~> note\n  1", "layer sun\n  1"},
		{"a semicolon after the name", "layer sun;\n  1", "layer sun\n  1"},
		{"in a function", "function f() {\n  layer a\n    1\n  2\n}", "let f = function() { layer a\n  1 2 };"},
		{"in a loop", "while (x) {\n  layer a\n    1\n}", "while x { layer a\n  1 }"},
		{"an if inside", "layer a\n  if (x) {\n    1\n  }\n2", "layer a\n  if x { 1 }\n2"},
		{"deeper than needed", "layer a\n      1\n   2", "layer a\n  1\n  2"},
		{"tabs count as one column", "layer a\n\t1\nlayer b\n\t\tlayer c\n\t\t\t2", "layer a\n  1\nlayer b\n  layer c\n    2"},
		{"the same line is not the body", "layer a; 1", "layer a\n1"},
		{"a multi-line statement stays whole", "layer a\n  let x = 1 +\n2\n3", "layer a\n  let x = (1 + 2);\n3"},
		{"inside a layer, the column is the layer's", "  layer a\n    1\n  2", "layer a\n  1\n2"},
		{"a name that is a keyword in a string", "layer \"let\"", "layer \"let\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lines(parse(t, tt.input)); got != tt.expected {
				t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
			}
		})
	}
}

// Where the body of a layer ends is a matter of columns, so check the tree
// and not only the printed form.
func TestLayerBodyTree(t *testing.T) {
	program := parse(t, "layer a\n  layer b\n    1\n    2\n  3\n4")
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 top-level statements, got %d", len(program.Statements))
	}
	a, ok := program.Statements[0].(*ast.LayerStatement)
	if !ok || a.Name != "a" || len(a.Body.Statements) != 2 {
		t.Fatalf("layer a wrong: %#v", program.Statements[0])
	}
	b, ok := a.Body.Statements[0].(*ast.LayerStatement)
	if !ok || b.Name != "b" || len(b.Body.Statements) != 2 {
		t.Fatalf("layer b wrong: %#v", a.Body.Statements[0])
	}
	if b.Token.Line != 2 || b.Token.Column != 3 {
		t.Errorf("layer b token at %d:%d, want 2:3", b.Token.Line, b.Token.Column)
	}
}

// "layer" is an ordinary identifier anywhere but at the start of a statement
// followed by a name on the same line.
func TestLayerAsAnIdentifier(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let layer = 1", "let layer = 1;"},
		{"layer + 1", "(layer + 1)"},
		{"layer = 2", "(layer = 2)"},
		{"f(layer)", "f(layer)"},
		{"layer", "layer"},
		{"layer\nsun", "layer\nsun"},
		{"layer(1)", "layer(1)"},
		{"layer[0]", "(layer[0])"},
		{"layer.x", "(layer[\"x\"])"},
		{"let layer = function(a) { layer }", "let layer = function(a) { layer };"},
		{"[layer, layer]", "[layer, layer]"},
		{"layer -1", "(layer - 1)"},
		{"layer == 1", "(layer == 1)"},
		{"layer\n  1", "layer\n1"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

func TestLayerErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"layer sun foo", `[line 1, column 11] syntax error: a layer is "layer name" alone on its line, got identifier "foo" after the name`},
		{"layer sun 1", `[line 1, column 11] syntax error: a layer is "layer name" alone on its line, got "1" after the name`},
		{"layer \"sun\" + 1", `[line 1, column 13] syntax error: a layer is "layer name" alone on its line, got "+" after the name`},
		{"layer a b c\n  1", `[line 1, column 9] syntax error: a layer is "layer name" alone on its line, got identifier "b" after the name`},
		{"layer sun\n  let = 1", `[line 2, column 7] syntax error: expected identifier, got "="`},
		{"layer sun\n  break", `[line 2, column 3] syntax error: break outside of a loop`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) != 1 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

func TestDefaultParameterTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"function f(a, b = 2) { a }", "let f = function(a, b = 2) { a };"},
		{"function f(a = 1, b = a * 2) { b }", "let f = function(a = 1, b = (a * 2)) { b };"},
		{"function f(t = null, v = [1, 2], h = {\"a\": 1}) { }", "let f = function(t = null, v = [1, 2], h = {\"a\": 1}) { };"},
		{"function f(a = function(x) { x }) { }", "let f = function(a = function(x) { x }) { };"},
		{"function f(a = g(1, 2)) { }", "let f = function(a = g(1, 2)) { };"},
		{"function f(a = (1, 2)) { }", "let f = function(a = (1, 2)) { };"},
		{"function f(a = -1) { }", "let f = function(a = (-1)) { };"},
		{"let f = function(k, v = 0) { v }", "let f = function(k, v = 0) { v };"},
		{"function(a = 1) { a }(5)", "function(a = 1) { a }(5)"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}

	// The tree keeps one default per parameter.
	literal := parse(t, "let f = function(a, b = 1, c = 2) { }").Statements[0].(*ast.LetStatement).Value.(*ast.FunctionLiteral)
	if len(literal.Parameters) != 3 || len(literal.Defaults) != 3 ||
		literal.Defaults[0] != nil || literal.Defaults[1] == nil || literal.Defaults[2] == nil {
		t.Errorf("wrong defaults: %v", literal.Defaults)
	}
	plain := parse(t, "let f = function(a, b) { }").Statements[0].(*ast.LetStatement).Value.(*ast.FunctionLiteral)
	if plain.Defaults != nil {
		t.Errorf("a function without defaults should have none, got %v", plain.Defaults)
	}
}

func TestDefaultParameterErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"function f(a = 1, b) { }", `[line 1, column 19] syntax error: parameter b needs a default value, it comes after one that has it`},
		{"function f(a, b = 1, c) { }", `[line 1, column 22] syntax error: parameter c needs a default value, it comes after one that has it`},
		{"let f = function(a = 1, b) { }", `[line 1, column 25] syntax error: parameter b needs a default value, it comes after one that has it`},
		{"function f(a = ) { }", `[line 1, column 16] syntax error: expected an expression, got ")"`},
		{"function f(= 1) { }", `[line 1, column 12] syntax error: expected identifier, got "="`},
		{"function f(a = 1 b = 2) { }", `[line 1, column 18] syntax error: expected ")", got identifier "b"`},
		{"function f(a == 1) { }", `[line 1, column 14] syntax error: expected ")", got "=="`},
		{"function f(1 = 1) { }", `[line 1, column 12] syntax error: expected identifier, got "1"`},
		{"while (true) { function f(a = break) { } }", `[line 1, column 31] syntax error: expected an expression, got "break"`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

func TestTryTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"try { f() } catch (e) { g(e) }", "try { f() } catch (e) { g(e) }"},
		{"try { f() } catch { g() }", "try { f() } catch { g() }"},
		{"try { } catch (e) { }", "try { } catch (e) { }"},
		{"try { f() }\ncatch (e) { g() }", "try { f() } catch (e) { g() }"},
		{"try { f() } catch (e) { g() };", "try { f() } catch (e) { g() }"},
		{"try { f() } catch (e) { g() }\nh()", "try { f() } catch (e) { g() }\nh()"},
		{"try { try { f() } catch (a) { g() } } catch (b) { h() }", "try { try { f() } catch (a) { g() } } catch (b) { h() }"},
		{"while (a) { try { break } catch (e) { continue } }", "while a { try { break; } catch (e) { continue; } }"},
		{"function f() { try { return 1 } catch (e) { return 2 } }", "let f = function() { try { return 1; } catch (e) { return 2; } };"},
		{"if (a) { try { 1 } catch { 2 } }", "if a { try { 1 } catch { 2 } }"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

// try and catch are only special in the shapes above: everywhere else they
// are ordinary names.
func TestTryAsAnIdentifier(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let try = 1", "let try = 1;"},
		{"try + 1", "(try + 1)"},
		{"try = 2", "(try = 2)"},
		{"try", "try"},
		{"try(1)", "try(1)"},
		{"try[0]", "(try[0])"},
		{"try.x", "(try[\"x\"])"},
		{"f(try)", "f(try)"},
		{"if try { 1 }", "if try { 1 }"},
		{"if (try) { 1 }", "if try { 1 }"},
		{"while try { break }", "while try { break; }"},
		{"let catch = 1", "let catch = 1;"},
		{"catch + 1", "(catch + 1)"},
		{"catch(1)", "catch(1)"},
		{"try\n{ \"a\": 1 }", "try\n{\"a\": 1}"},
		{"function try(catch) { catch }", "let try = function(catch) { catch };"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

func TestTryErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"try { 1 }", `[line 1, column 10] syntax error: expected "catch" after the try block, got end of input`},
		{"try { 1 } 5", `[line 1, column 11] syntax error: expected "catch" after the try block, got "5"`},
		{"try { 1 } finally { 2 }", `[line 1, column 11] syntax error: expected "catch" after the try block, got identifier "finally"`},
		{"try { 1 } catch", `[line 1, column 16] syntax error: expected "{", got end of input`},
		{"try { 1 } catch () { 2 }", `[line 1, column 18] syntax error: expected identifier, got ")"`},
		{"try { 1 } catch (e, f) { 2 }", `[line 1, column 19] syntax error: expected ")", got ","`},
		{"try { 1 } catch (e) 2", `[line 1, column 21] syntax error: expected "{", got "2"`},
		{"try { 1 } catch e { 2 }", `[line 1, column 17] syntax error: expected "{", got identifier "e"`},
		{"try { 1 catch (e) { 2 }", `[line 1, column 9] syntax error: expected end of statement, got identifier "catch"`},
		{"try { 1 } catch (e) { 2", `[line 1, column 21] syntax error: block is never closed: expected "}"`},
		{"try { break } catch { }", `[line 1, column 7] syntax error: break outside of a loop`},
		{"try { 1 } catch (e) { break }", `[line 1, column 23] syntax error: break outside of a loop`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

func TestContinueTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"while (a) { continue }", "while a { continue; }"},
		{"while (a) { continue; }", "while a { continue; }"},
		{"for (;;) { continue }", "for (; ; ) { continue; }"},
		{"while (a) { continue\nb() }", "while a { continue; b() }"},
		{"while (a) { if (b) { continue } }", "while a { if b { continue; } }"},
		{"while (a) { for (;;) { continue } }", "while a { for (; ; ) { continue; } }"},
		{"while (a) { layer x\n    continue }", "while a { layer x continue; }"},
		// Still names.
		{"while (a) { continue + 1 }", "while a { (continue + 1) }"},
		{"while (a) { continue = 1 }", "while a { (continue = 1) }"},
		{"while (a) { continue(1) }", "while a { continue(1) }"},
		{"while (a) { continue[0] }", "while a { (continue[0]) }"},
		{"while (a) { continue.x }", "while a { (continue[\"x\"]) }"},
		{"while (a) { continue -\n1 }", "while a { (continue - 1) }"},
		{"while (a) { f(continue) }", "while a { f(continue) }"},
		{"while (a) { let continue = 1 }", "while a { let continue = 1; }"},
		{"let continue = 1", "let continue = 1;"},
		{"continue + 1", "(continue + 1)"},
		{"f(continue)", "f(continue)"},
		{"function continue() { }", "let continue = function() { };"},
		// Outside a loop it is still a name, as long as it is not alone.
		{"function f() { continue + 1 }", "let f = function() { (continue + 1) };"},
		{"while (a) { function f() { let continue = 1\ncontinue + 1 } }", "while a { let f = function() { let continue = 1; (continue + 1) }; }"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

func TestContinueErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"while (a) { continue 5 }", `[line 1, column 22] syntax error: expected end of statement, got "5"`},
		{"continue", "[line 1, column 1] syntax error: continue outside of a loop"},
		{"continue;", "[line 1, column 1] syntax error: continue outside of a loop"},
		{"if (a) { continue }", "[line 1, column 10] syntax error: continue outside of a loop"},
		{"function f() { continue }", "[line 1, column 16] syntax error: continue outside of a loop"},
		{"while (a) { function f() { continue } }", "[line 1, column 28] syntax error: continue outside of a loop"},
		{"while (a) { let f = function() { continue } }", "[line 1, column 34] syntax error: continue outside of a loop"},
		{"try { continue } catch { }", "[line 1, column 7] syntax error: continue outside of a loop"},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

func TestCompoundAssignTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"x += 1", "(x += 1)"},
		{"x -= 1", "(x -= 1)"},
		{"x *= 2", "(x *= 2)"},
		{"x /= 2", "(x /= 2)"},
		{"x+=1", "(x += 1)"},
		{"x += 1 + 2", "(x += (1 + 2))"},
		{"x *= a + b", "(x *= (a + b))"},
		{"x += y -= 1", "(x += (y -= 1))"},
		{"x = y += 1", "(x = (y += 1))"},
		{"a[i] += 1", "((a[i]) += 1)"},
		{"a.b += 1", "((a[\"b\"]) += 1)"},
		{"a.b.c -= f(1)", "(((a[\"b\"])[\"c\"]) -= f(1))"},
		{"a[i + 1] *= 2", "((a[(i + 1)]) *= 2)"},
		{"x = 1", "(x = 1)"},
		{"for (let i = 0; i < 3; i += 1) { }", "for (let i = 0; (i < 3); (i += 1)) { }"},
		{"for (i += 1;;) { }", "for ((i += 1); ; ) { }"},
		{"x += 1\ny -= 2", "(x += 1)\n(y -= 2)"},
		{"x -= -1", "(x -= (-1))"},
		{"x - -1", "(x - (-1))"},
		{"x / y", "(x / y)"},
		{"x // y", "(x // y)"},
		{"x ** y", "(x ** y)"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

func TestCompoundAssignErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1 += 2", `[line 1, column 3] syntax error: cannot assign to 1`},
		{"f() -= 2", `[line 1, column 5] syntax error: cannot assign to f()`},
		{"(a + b) *= 2", `[line 1, column 9] syntax error: cannot assign to (a + b)`},
		{"x += ", `[line 1, column 6] syntax error: expected an expression, got end of input`},
		{"let x += 1", `[line 1, column 7] syntax error: expected "=", got "+="`},
		{"+= 1", `[line 1, column 1] syntax error: expected an expression, got "+="`},
		{"x **= 2", `[line 1, column 5] syntax error: expected an expression, got "="`},
		{"x %= 2", `[line 1, column 4] syntax error: expected an expression, got "="`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}

// TestStatementsEndAtLineEnd checks LEP 0001: a statement ends at the end of
// its line, unless the line ends with an operator, a ',' or an opening
// bracket, or a '(', '[' or hash is still open.
func TestStatementsEndAtLineEnd(t *testing.T) {
	tests := []struct{ input, expected string }{
		// A new line ends the statement, even before a token that could go on.
		{"let x = 10\n-1", "let x = 10;\n(-1)"},
		{"let f = g\n(1)", "let f = g;\n1"},
		{"let a = b\n[1, 2]", "let a = b;\n[1, 2]"},
		{"a\n-b\nc", "a\n(-b)\nc"},
		{"x = 1\n-2", "(x = 1)\n(-2)"},
		{"f(1)\n(2, 3)", "f(1)\n(2, 3)"},
		// An operator at the end of the line goes on to the next one.
		{"let x = 10 -\n1", "let x = (10 - 1);"},
		{"let t = a +\n  b +\n  c", "let t = ((a + b) + c);"},
		{"a &&\nb ||\nc", "((a && b) || c)"},
		{"a ==\nb", "(a == b)"},
		{"a **\nb", "(a ** b)"},
		{"x =\n5", "(x = 5)"},
		{"x +=\n5", "(x += 5)"},
		{"let x =\n  5", "let x = 5;"},
		{"a.\nb", "(a[\"b\"])"},
		{"!\na", "(!a)"},
		// So do a ',' and an opening bracket.
		{"f(1,\n2)", "f(1, 2)"},
		{"f(\n1\n)", "f(1)"},
		{"[\n1,\n2\n]", "[1, 2]"},
		{"a[\n0\n]", "(a[0])"},
		// Inside '(' and '[' a new line is just a space.
		{"f(a\n+ b)", "f((a + b))"},
		{"(a\n- b)", "(a - b)"},
		{"[a\n- 1, b]", "[(a - 1), b]"},
		{"if (a\n&& b) { c }", "if (a && b) { c }"},
		{"(1\n, 2)", "(1, 2)"},
		{"for (let i = 0; i < n\n; i += 1) { }", "for (let i = 0; (i < n); (i += 1)) { }"},
		{"function f(a = 1\n+ 2) { a }", "let f = function(a = (1 + 2)) { a };"},
		// So it is inside a hash.
		{"let h = {\n\"a\": 1\n+ 2,\n\"b\": [3]\n}", "let h = {\"a\": (1 + 2), \"b\": [3]};"},
		{"let h = {\"a\"\n: 1}", "let h = {\"a\": 1};"},
		// A block inside brackets starts statements again, until its '}'.
		{"f(function() {\nlet x = 1\n-2\n})", "f(function() { let x = 1; (-2) })"},
		{"f(function() {\nx\n}\n+ 1)", "f((function() { x } + 1))"},
		{"let h = {\"f\": function() {\na\n-1\n}}", "let h = {\"f\": function() { a (-1) }};"},
		// A closed bracket doesn't keep the line going.
		{"f(a)\n-1", "f(a)\n(-1)"},
		// ';' still separates statements on one line, and blank lines and
		// comments at the end of a line are only space.
		{"a; -1", "a\n(-1)"},
		{"let x = 1; let y = 2", "let x = 1;\nlet y = 2;"},
		{"a +\n\n\nb", "(a + b)"},
		{"a + -~> more below\nb", "(a + b)"},
		{"a -~> done\n-1", "a\n(-1)"},
		{"a +\n-~> a whole line of comment\nb", "(a + b)"},
		// A string over several lines is one token: what follows it on its
		// last line is on the same line.
		{"let s = \"one\ntwo\" + \"!\"", "let s = (\"one\ntwo\" + \"!\");"},
		// return takes a value that starts on its line.
		{"function f() {\nreturn\n5\n}", "let f = function() { return; 5 };"},
		{"function f() {\nreturn 5 +\n1\n}", "let f = function() { return (5 + 1); };"},
		{"function f() {\nreturn (\n5)\n}", "let f = function() { return 5; };"},
		// A continue alone on its line is a continue.
		{"while (a) { continue\n- 1 }", "while a { continue; (-1) }"},
		// An else, an elif and a catch still may start the line after the '}'.
		{"if (a) { 1 }\nelse { 2 }", "if a { 1 } else { 2 }"},
		{"if (a) { 1 }\nelif (b) { 2 }", "if a { 1 } else { if b { 2 } }"},
		{"try { a }\ncatch (e) { b }", "try { a } catch (e) { b }"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

// TestLeadingOperatorErrors checks the error for a line that starts with an
// operator that can only go between two values. The line is still read as
// part of the line before, so each one is reported and nothing else is.
func TestLeadingOperatorErrors(t *testing.T) {
	message := func(line, column int, operator string) string {
		return fmt.Sprintf("[line %d, column %d] syntax error: a line can't start with %s; "+
			"put the %s at the end of the previous line", line, column, operator, operator)
	}
	tests := []struct {
		input    string
		expected []string
	}{
		{"a\n+ b", []string{message(2, 1, "+")}},
		{"let total = first\n  + second\n  * third", []string{message(2, 3, "+"), message(3, 3, "*")}},
		{"a\n&& b", []string{message(2, 1, "&&")}},
		{"a\n|| b", []string{message(2, 1, "||")}},
		{"a\n== b", []string{message(2, 1, "==")}},
		{"a\n!== b", []string{message(2, 1, "!==")}},
		{"a\n<= b", []string{message(2, 1, "<=")}},
		{"a\n// b", []string{message(2, 1, "//")}},
		{"a\n% b", []string{message(2, 1, "%")}},
		{"a\n** b", []string{message(2, 1, "**")}},
		{"a\n.b", []string{message(2, 1, ".")}},
		{"x\n= 5", []string{message(2, 1, "=")}},
		{"x\n+= 5", []string{message(2, 1, "+=")}},
		{"if a\n&& b { c }", []string{message(2, 1, "&&")}},
		{"while (x) {\nlet y = a\n/ 2\n}", []string{message(3, 1, "/")}},
		{"a -~> comment\n+ b", []string{message(2, 1, "+")}},
		// The parse goes on, so a mistake after it is found too.
		{"a\n+ b\nlet = 1", []string{message(2, 1, "+"),
			`[line 3, column 5] syntax error: expected identifier, got "="`}},
		// A statement that starts with an operator is still an ordinary
		// error: no line before it went on.
		{"* 2", []string{`[line 1, column 1] syntax error: expected an expression, got "*"`}},
		{"if (a) { 1 }\n+ 2", []string{`[line 2, column 1] syntax error: expected an expression, got "+"`}},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if got := strings.Join(p.Errors(), "\n"); got != strings.Join(tt.expected, "\n") {
			t.Errorf("input %q\nexpected: %s\n     got: %s", tt.input, strings.Join(tt.expected, "\n"), got)
		}
	}
}

// TestLineChanges checks the list of lines whose meaning LEP 0001 changed,
// which a migration tool shows: each one once, and nothing for code that
// meant the same before.
func TestLineChanges(t *testing.T) {
	newStatement := func(line, column int, operator string) string {
		return fmt.Sprintf("[line %d, column %d] this line no longer goes on with the line before: "+
			"%s starts a new statement; to go on, end the line before with an operator or join the lines",
			line, column, operator)
	}
	leading := func(line, column int, operator string) string {
		return fmt.Sprintf("[line %d, column %d] a line can't start with %s; "+
			"put the %s at the end of the previous line", line, column, operator, operator)
	}
	tests := []struct {
		input    string
		expected []string
	}{
		{"let x = 10\n-1", []string{newStatement(2, 1, "-")}},
		{"let x = a * b\n-1", []string{newStatement(2, 1, "-")}},
		{"x = a + b\n(1)", []string{newStatement(2, 1, "(")}},
		{"f\n[0]", []string{newStatement(2, 1, "[")}},
		{"a\n+ b", []string{leading(2, 1, "+")}},
		{"function f() {\nreturn\nx\n}", []string{"[line 3, column 1] this line is no longer the value of " +
			"the return before it; start the value on the return line"}},
		// The same before and after LEP 0001.
		{"let x = 10 -\n1", nil},
		{"f(a\n- b)", nil},
		{"if (a) { 1 }\n-1", nil},
		{"while (a) { b }\n(c)", nil},
		{"function f() { }\n[1]", nil},
		{"let x = 1;\n-1", nil},
		{"while (a) { break\n-1 }", nil},
		{"function f() {\nreturn\n}", nil},
		{"let x = 1\nlet y = 2", nil},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if got := strings.Join(p.LineChanges(), "\n"); got != strings.Join(tt.expected, "\n") {
			t.Errorf("input %q\nexpected: %s\n     got: %s", tt.input, strings.Join(tt.expected, "\n"), got)
		}
	}
}

func TestUseTrees(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`use "shapes"`, `use "shapes";`},
		{`use "draw/shapes";`, `use "draw/shapes";`},
		{`use "draw/shapes" as shapes`, `use "draw/shapes" as shapes;`},
		{"use \"a\"\nuse \"b\" as b\nx", "use \"a\";\nuse \"b\" as b;\nx"},
		{"use \"a\"\nas", "use \"a\";\nas"},
		{`use "a"; f()`, "use \"a\";\nf()"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

// use and as are only special in the shapes above: everywhere else they
// are ordinary names.
func TestUseAsAnIdentifier(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let use = 1", "let use = 1;"},
		{"use + 1", "(use + 1)"},
		{"use(\"a\")", "use(\"a\")"},
		{"use\n\"a\"", "use\n\"a\""},
		{"let as = use", "let as = use;"},
		{"function use(as) { as }", "let use = function(as) { as };"},
		{"f(use)", "f(use)"},
	}
	for _, tt := range tests {
		if got := lines(parse(t, tt.input)); got != tt.expected {
			t.Errorf("input %q\nexpected: %q\n     got: %q", tt.input, tt.expected, got)
		}
	}
}

func TestUseErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`use ""`, `[line 1, column 5] syntax error: use needs the path of a module, like use "shapes"`},
		{`use "a" as`, `[line 1, column 11] syntax error: expected a name after "as", got end of input`},
		{`use "a" as 5`, `[line 1, column 12] syntax error: expected a name after "as", got "5"`},
		{"use \"a\" as\nb", `[line 2, column 1] syntax error: expected a name after "as", got identifier "b"`},
		{`use "a" b`, `[line 1, column 9] syntax error: expected end of statement, got identifier "b"`},
		{`use "a" as b c`, `[line 1, column 14] syntax error: expected end of statement, got identifier "c"`},
		{`if (x) { use "a" }`, `[line 1, column 10] syntax error: use must be at the top level of a file, not inside a block or a layer`},
		{`function f() { use "a" }`, `[line 1, column 16] syntax error: use must be at the top level of a file, not inside a block or a layer`},
		{"layer sun\n  use \"a\"", `[line 2, column 3] syntax error: use must be at the top level of a file, not inside a block or a layer`},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || p.Errors()[0] != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %v", tt.input, tt.expected, p.Errors())
		}
	}
}
