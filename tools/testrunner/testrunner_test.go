package testrunner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRun(t *testing.T) {
	files := map[string]string{
		"pass.liz":           "use \"helper\"\nputs(double(2))",
		"pass.expected":      "4\n",
		"helper.liz":         "function double(x) { x * 2 }",
		"fail.liz":           "puts(1)",
		"fail.expected":      "2\n",
		"sub/error.liz":      "puts(\"a\")\n1 + \"b\"",
		"sub/error.expected": "a\n[line 2, column 3] runtime error: type mismatch: INTEGER + STRING\n",
		"sub/exits.liz":      "puts(\"bye\")\nos.exit(3)\nputs(\"never\")",
		"sub/exits.expected": "bye\n",
		"crlf.liz":           "puts(1)",
		"crlf.expected":      "1\r\n",
		"speed_bench.liz":    "let n = 0\nwhile (n < 100) { n += 1 }",
		"broken_bench.liz":   "nope",
		".hidden/x.liz":      "puts(1)",
		".hidden/x.expected": "wrong\n",
	}
	for _, vm := range []bool{false, true} {
		dir := writeTree(t, files)
		var out bytes.Buffer
		failed, err := Run(Options{Dir: dir, VM: vm, Out: &out})
		report := strings.ReplaceAll(out.String(), filepath.ToSlash(dir)+"/", "")
		if err != nil || failed != 1 {
			t.Errorf("vm %t: %d failed, %v\n%s", vm, failed, err, report)
		}
		for _, want := range []string{"ok    crlf.liz", "FAIL  fail.liz\n--- expected:\n2\n--- got:\n1\n", "ok    pass.liz", "ok    sub/error.liz", "ok    sub/exits.liz", "4 passed, 1 failed"} {
			if !strings.Contains(report, want) {
				t.Errorf("vm %t: the report has no %q:\n%s", vm, want, report)
			}
		}
		if strings.Contains(report, "bench") || strings.Contains(report, "helper") {
			t.Errorf("vm %t: ran more than the tests:\n%s", vm, report)
		}

		out.Reset()
		failed, _ = Run(Options{Dir: dir, VM: vm, Bench: true, Out: &out})
		report = strings.ReplaceAll(out.String(), filepath.ToSlash(dir)+"/", "")
		if failed != 2 || !strings.Contains(report, "bench speed_bench.liz") || !strings.Contains(report, "FAIL  broken_bench.liz") {
			t.Errorf("vm %t: with benchmarks %d failed:\n%s", vm, failed, report)
		}
	}
}

func TestUpdate(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.liz": "puts(\"new\")", "a.expected": "old\n"})
	var out bytes.Buffer
	if failed, _ := Run(Options{Dir: dir, Update: true, Out: &out}); failed != 0 {
		t.Fatalf("update failed:\n%s", out.String())
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "a.expected")); string(data) != "new\n" {
		t.Errorf("a.expected is %q", data)
	}
}

func TestNoTests(t *testing.T) {
	var out bytes.Buffer
	if failed, err := Run(Options{Dir: t.TempDir(), Out: &out}); failed != 0 || err != nil || !strings.HasPrefix(out.String(), "no tests in") {
		t.Errorf("got %d, %v, %q", failed, err, out.String())
	}
	if _, err := Run(Options{Dir: filepath.Join(t.TempDir(), "missing"), Out: &out}); err == nil {
		t.Error("a missing folder gave no error")
	}
}
