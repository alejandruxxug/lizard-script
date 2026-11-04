package parser

import (
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"strings"
	"testing"
)

// TestTeachingMessages checks the errors that say what was probably meant.
func TestTeachingMessages(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fucntion add(a) { a }", `[line 1, column 1] syntax error: "fucntion" is not a keyword; did you mean "function"?`},
		{"retrun 5", `[line 1, column 1] syntax error: "retrun" is not a keyword; did you mean "return"?`},
		{"whlie (x) { }", `[line 1, column 1] syntax error: "whlie" is not a keyword; did you mean "while"?`},
		{"lte x = 1", `[line 1, column 1] syntax error: "lte" is not a keyword; did you mean "let"?`},
		{"if (a) { 1 } elsif (b) { 2 }", `[line 1, column 14] syntax error: "elsif" is not a keyword; did you mean "elif"?`},
		{"if (a) { 1 }\nelseif (b) { 2 }", `[line 2, column 1] syntax error: "elseif" is not a keyword; did you mean "elif"?`},
		{"var x = 1", `[line 1, column 1] syntax error: "var" is not a keyword; did you mean "let"?`},
		{"def f() { 1 }", `[line 1, column 1] syntax error: "def" is not a keyword; did you mean "function"?`},
		{"form x", `[line 1, column 6] syntax error: expected end of statement, got identifier "x"`},
		{"total x", `[line 1, column 7] syntax error: expected end of statement, got identifier "x"`},

		{"if (x = 1) { }", `[line 1, column 7] syntax error: "=" in a condition gives x a value; did you mean "=="?`},
		{"if (a) { } elif (b = 2) { }", `[line 1, column 20] syntax error: "=" in a condition gives b a value; did you mean "=="?`},
		{"while (x = next()) { }", `[line 1, column 10] syntax error: "=" in a condition gives x a value; did you mean "=="?`},
		{"for (let i = 0; i = 3; i += 1) { }", `[line 1, column 19] syntax error: "=" in a condition gives i a value; did you mean "=="?`},

		{"let = 1", `[line 1, column 5] syntax error: let needs a name before the "=", like let x = 1`},
		{"let 5 = 1", `[line 1, column 5] syntax error: expected a name after let, got "5"`},
		{"let if = 1", `[line 1, column 5] syntax error: "if" is a reserved word, it can't be a name`},
		{"function f(a, let) { }", `[line 1, column 15] syntax error: "let" is a reserved word, it can't be a name`},
		{"let x: 1", `[line 1, column 6] syntax error: a let gives its value with "=", not ":"`},
		{`let h = {"a" = 1}`, `[line 1, column 14] syntax error: a hash puts ":" between a key and its value, not "="`},
		{`let h = {"a": 1, b = 2}`, `[line 1, column 20] syntax error: a hash puts ":" between a key and its value, not "="`},

		{")", `[line 1, column 1] syntax error: this ")" has nothing to close`},
		{"]", `[line 1, column 1] syntax error: this "]" has nothing to close`},
		{"puts(1)\n}", `[line 2, column 1] syntax error: this "}" has nothing to close`},
		{"puts(1\nlet x = 2", `[line 1, column 7] syntax error: the "(" at line 1, column 5 is never closed`},
		{"f(1,\n2", `[line 2, column 2] syntax error: the "(" at line 1, column 2 is never closed`},
		{"let a = [1, 2\nlet b = 3", `[line 1, column 14] syntax error: the "[" at line 1, column 9 is never closed`},
		{"let h = {\n  \"a\": 1\nlet x = 2", `[line 2, column 9] syntax error: the "{" at line 1, column 9 is never closed`},
		{"if (x) {\n  f(1\n}", `[line 2, column 6] syntax error: the "(" at line 2, column 4 is never closed`},
		{`let s = "abc`, `[line 1, column 9] lexical error: unterminated string: this " is never closed`},
		{"function f() {\n  if (x) {\n    1\n  2\n}\nf()", `[line 2, column 10] syntax error: block is never closed: expected "}"`},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()

		errors := p.Errors()
		if len(errors) != 1 || errors[0] != tt.expected {
			t.Errorf("input %q\n expected: %s\n got:      %s", tt.input, tt.expected, strings.Join(errors, "\n           "))
		}
	}
}

// TestTeachingMessagesKeepValidPrograms checks that the words and shapes the
// messages look for still work where they are valid.
func TestTeachingMessagesKeepValidPrograms(t *testing.T) {
	for _, input := range []string{
		"let fucntion = 1\nfucntion",
		"let var = 2\nvar + 1",
		"if ((x = 1) == 1) { }",
		"while (x += 1) { break }",
		`let h = {(x = 1): 2, "b": y = 3}`,
		"let f = function(elsif) { elsif }",
	} {
		parse(t, input)
	}
}

// TestRecoveryByLine checks that after a mistake the parser goes on with the
// next line, as LEP 0001 ends statements there, so each mistake gives one
// error and the lines after it are still checked.
func TestRecoveryByLine(t *testing.T) {
	tests := []struct {
		input  string
		errors []string
	}{
		{
			"if (x) {\n  let = 1\n  puts(2)\n  let = 3\n}\nlet = 4",
			[]string{
				`[line 2, column 7] syntax error: let needs a name before the "=", like let x = 1`,
				`[line 4, column 7] syntax error: let needs a name before the "=", like let x = 1`,
				`[line 6, column 5] syntax error: let needs a name before the "=", like let x = 1`,
			},
		},
		{
			"puts(1 2)\nputs(3 4)",
			[]string{
				`[line 1, column 8] syntax error: expected ")", got "2"`,
				`[line 2, column 8] syntax error: expected ")", got "4"`,
			},
		},
		{
			// One mistake breaks the rest of its line: one error per line.
			"puts(1 2, 3 4)",
			[]string{`[line 1, column 8] syntax error: expected ")", got "2"`},
		},
		{
			// A forgotten '{': the body and its '}' are skipped with it.
			"if (x)\n  puts(1)\n}\nlet = 2",
			[]string{
				`[line 2, column 3] syntax error: expected "{", got identifier "puts"`,
				`[line 4, column 5] syntax error: let needs a name before the "=", like let x = 1`,
			},
		},
		{
			// One ')' too many: the block after it is still the if's.
			"if (n >= 90)) {\n  1\n} elif (n > 1) {\n  2\n}\nlet = 3",
			[]string{
				`[line 1, column 13] syntax error: expected "{", got ")"`,
				`[line 6, column 5] syntax error: let needs a name before the "=", like let x = 1`,
			},
		},
		{
			// One '{' too many.
			"function f() {\n  try { {\n    1\n  } catch { 2 }\n}\nlet = 3",
			[]string{
				`[line 4, column 3] syntax error: expected ":", got "}"`,
				`[line 6, column 5] syntax error: let needs a name before the "=", like let x = 1`,
			},
		},
		{
			// An operator alone on its line doesn't take the next line.
			"let a = 1\n+\nlet b = 2",
			[]string{`[line 2, column 1] syntax error: a line can't start with +; put the + at the end of the previous line`},
		},
		{
			// A '}' in the middle of a line of a block is a stray one.
			"function f() {\n  let a = [1, 2 }]\n  let = 2\n}",
			[]string{
				`[line 2, column 16] syntax error: the "[" at line 2, column 11 is never closed`,
				`[line 3, column 7] syntax error: let needs a name before the "=", like let x = 1`,
			},
		},
		{
			// A use inside a block that is never closed is reported once.
			"function f() {\n  puts(1)\nuse \"a\"\nuse \"b\"",
			[]string{
				`[line 3, column 1] syntax error: use must be at the top level of a file, not inside a block or a layer`,
				`[line 1, column 14] syntax error: block is never closed: expected "}"`,
			},
		},
	}

	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()

		got := strings.Join(p.Errors(), "\n")
		if expected := strings.Join(tt.errors, "\n"); got != expected {
			t.Errorf("input %q\n expected:\n%s\n got:\n%s", tt.input, expected, got)
		}
	}
}
