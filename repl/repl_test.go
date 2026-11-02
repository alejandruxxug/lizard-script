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
		{"let x = 5\nx * 2\n", "10\n\n"},
		{"\"liz\"\n", "\"liz\"\n\n"},
		{"puts(\"hi\")\n", "hi\n\n"},
		{"if (false) { 1 }\n", "\n"},
		{"\n\n7\n", "7\n\n"},
		{"function add(a, b) {\n  a + b\n}\nadd(1, 2)\n", "3\n\n"},
		{"let list = [\n1,\n2]\nlist\n", "[1, 2]\n\n"},
		{"let s = \"two\nlines\"\ns\n", "\"two\nlines\"\n\n"},
		{"1 +\n2\n", "3\n\n"},
		{"foo\n1\n", "[line 1, column 1] runtime error: identifier not found: foo\n1\n\n"},
		{"let = 1\n2\n", "[line 1, column 5] syntax error: expected identifier, got \"=\"\n2\n\n"},
		{"(1 +\n\n", "[line 3, column 1] syntax error: expected an expression, got end of input\n\n"},
		{"let x = (1\n", "\n[line 2, column 1] syntax error: expected \")\", got end of input\n\n"},
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

// TestExamples runs every program in examples/ and checks it has no errors.
func TestExamples(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "examples", "*.liz"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples found: %v", err)
	}

	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		if !Run(string(source), &out, &errOut) {
			t.Errorf("%s failed:\n%s", file, errOut.String())
		}
	}
}
