package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// session runs the REPL on input and returns what it wrote, without the
// prompts.
func session(input string) string {
	var out bytes.Buffer
	Start(strings.NewReader(input), &out)

	output := strings.ReplaceAll(out.String(), PROMPT, "")
	return strings.ReplaceAll(output, CONTINUE, "")
}

func TestREPL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2\n", "3\n\n"},
		{"[1, \"a\"]\n", "[1, \"a\"]\n\n"},
		{"[\"say \\\"hi\\\"\\n\"]\n", "[\"say \\\"hi\\\"\\n\"]\n\n"},
		{"{\"k\": \"v\"}\n", "{\"k\": \"v\"}\n\n"},
		{"str([1, \"a\"])\n", "\"[1, \\\"a\\\"]\"\n\n"},
		{"\"a\\\\b\"\n", "\"a\\\\b\"\n\n"},
		{"if (false) { 1 }\nelse if (true) { 2 }\n\n", "2\n\n"},
		{"let x = 1\nif (x > 3) { 1 }\nelse if (x > 0) { 2 }\nelse { 3 }\n", "2\n\n"},
		{"let x = 5\nx * 2\n", "10\n\n"},
		{"\"liz\"\n", "\"liz\"\n\n"},
		{"puts(\"hi\")\n", "hi\n\n"},
		{"if (false) { 1 }\n", "\n\n"},
		{"if (true) { 1 } else { 2 }\n3\n", "1\n3\n\n"},
		{"if (false) { 1 }\nelse { 2 }\n", "2\n\n"},
		{"let x = 5\nif (x > 10) { 1 }\nelif (x > 3) { 2 }\nelse { 3 }\n", "2\n\n"},
		{"let x = 1\nif (x > 10) { 1 }\nelif (x > 3) { 2 }\n\nx\n", "1\n\n"},
		{"if (true) {\n  1\n}\nelse {\n  2\n}\n", "1\n\n"},
		{"if (true) { 1 }\nelse\n{ 2 }\n", "1\n\n"},
		{"if (true) { 1 }\n\n2\n", "1\n2\n\n"},
		{"if (true) { 1 }\n2\n", "1\n2\n\n"},
		{"if (true) { 1 }\nexit\n", "1\n"},
		{"let y = if (false) { 1 }\nelse { 2 }\ny\n", "2\n\n"},
		{"while (false) {}\n3\n", "3\n\n"},
		{"else { 1 }\n", "[line 1, column 1] syntax error: expected an expression, got \"else\"\n\n"},
		{"\n\n7\n", "7\n\n"},
		{"function add(a, b) {\n  a + b\n}\nadd(1, 2)\n", "3\n\n"},
		{"let list = [\n1,\n2]\nlist\n", "[1, 2]\n\n"},
		{"let s = \"two\nlines\"\ns\n", "\"two\nlines\"\n\n"},
		{"1 +\n2\n", "3\n\n"},
		{"let x = 10\n-1\nx\n", "-1\n10\n\n"},
		{"let x = 10 -\n1\nx\n", "9\n\n"},
		{"let x = (10\n- 1)\nx\n", "9\n\n"},
		{"let x = 1\nx\n+ 1\n", "1\n[line 1, column 1] syntax error: expected an expression, got \"+\"\n\n"},
		{"let x = 1; x + 1\n", "2\n\n"},
		{"foo\n1\n", "[line 1, column 1] runtime error: identifier not found: foo\n1\n\n"},
		{"let = 1\n2\n", "[line 1, column 5] syntax error: let needs a name before the \"=\", like let x = 1\n2\n\n"},
		{"(1 +\n\n", "[line 3, column 1] syntax error: expected an expression, got end of input\n\n"},
		{"let x = (1\n", "\n[line 1, column 11] syntax error: the \"(\" at line 1, column 9 is never closed\n\n"},
		{"exit\n1\n", ""},
		{"1\n  quit  \n", "1\n"},
	}

	for _, tt := range tests {
		got := session(tt.input)
		if got != tt.expected {
			t.Errorf("input %q\nexpected %q\n     got %q", tt.input, tt.expected, got)
		}
	}
}

func TestIsIncomplete(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1 + 2", false},
		{"function f() {", true},
		{"function f() { 1 }", false},
		{"[1, 2", true},
		{"f(1,", true},
		{"\"open", true},
		{"\"closed\"", false},
		{"x =", true},
		{"a &&", true},
		{"}", false},
		{"1 -~> comment {", false},
		{"if (a) { 1 } else", true},
		{"let", true},
		{"x +=", true},
		{"x /=", true},
		{"x += 1", false},
		{"try { 1 } catch (e) {", true},
		// The same rule as in a file: a statement ends at the end of its
		// line unless the line ends with an operator, a ',' or an opening
		// bracket, or a bracket is still open.
		{"let x = 10", false},
		{"let x = 10 -", true},
		{"a ||", true},
		{"a ==", true},
		{"a **", true},
		{"a.", true},
		{"!", true},
		{"(", true},
		{"f(a\n+ b", true},
		{"f(a\n+ b)", false},
		{"let h = {\"a\": 1", true},
		{"1 + -~> more on the next line", true},
		{"return", false},
	}

	for _, tt := range tests {
		if got := isIncomplete(tt.input); got != tt.expected {
			t.Errorf("isIncomplete(%q) = %t, want %t", tt.input, got, tt.expected)
		}
	}
}

func TestRun(t *testing.T) {
	var out, errOut bytes.Buffer
	if !Run("puts(1 + 1)", &out, &errOut) || out.String() != "2\n" || errOut.Len() != 0 {
		t.Errorf("unexpected result: out %q, errors %q", out.String(), errOut.String())
	}

	out.Reset()
	if Run("puts(1)\n1 + true\nputs(2)", &out, &errOut) {
		t.Error("a runtime error should make Run fail")
	}
	if out.String() != "1\n" || errOut.String() != "[line 2, column 3] runtime error: type mismatch: INTEGER + BOOLEAN\n" {
		t.Errorf("unexpected result: out %q, errors %q", out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if Run("puts(1)\nlet = 2", &out, &errOut) {
		t.Error("a syntax error should make Run fail")
	}
	if out.Len() != 0 || !strings.Contains(errOut.String(), "syntax error") {
		t.Errorf("a program with syntax errors must not run: out %q, errors %q", out.String(), errOut.String())
	}
}

func TestEndsWithOpenIf(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"if (a) { 1 }", true},
		{"if (a) { 1 } elif (b) { 2 }", true},
		{"if (a) { 1 } else { 2 }", false},
		{"if (a) { 1 } else if (b) { 2 }", true},
		{"if (a) { 1 } else if (b) { 2 } else { 3 }", false},
		{"if (a) { 1 } elif (b) { 2 } else { 3 }", false},
		{"if (a) { if (b) { 1 } }", true},
		{"if (a) { 1 } else { if (b) { 2 } }", false},
		{"let x = if (a) { 1 }", true},
		{"x = if (a) { 1 }", true},
		{"return if (a) { 1 }", true},
		{"1 + if (a) { 1 }", true},
		{"if (a) { 1 };", false},
		{"while (a) { if (b) { 1 } }", false},
		{"function f() { if (a) { 1 } }", false},
		{"{\"a\": if (a) { 1 }}", false},
		{"let = if (a) { 1 }", false},
		{"1", false},
	}

	for _, tt := range tests {
		if got := endsWithOpenIf(tt.input); got != tt.expected {
			t.Errorf("endsWithOpenIf(%q) = %t, want %t", tt.input, got, tt.expected)
		}
	}
}

// TestExamples runs every program in examples/ and checks it has no errors,
// and that the ones in examples/errors/ do fail.
func TestExamples(t *testing.T) {
	for _, dir := range []string{"examples", filepath.Join("examples", "errors")} {
		shouldSucceed := dir == "examples"
		files, err := filepath.Glob(filepath.Join("..", "..", dir, "*.liz"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no examples found in %s: %v", dir, err)
		}

		for _, file := range files {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var out, errOut bytes.Buffer
			if Run(string(source), &out, &errOut) != shouldSucceed {
				t.Errorf("%s: expected success to be %t. Errors:\n%s", file, shouldSucceed, errOut.String())
			}
		}
	}
}
