package parser

import (
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
    return hanoi(n - 1, origen, auxiliar, destino)
        + [[origen, destino]]
        + hanoi(n - 1, auxiliar, destino, origen);
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
