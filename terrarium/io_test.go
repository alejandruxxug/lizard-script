package terrarium

import (
	"bytes"
	"lizardscript/evaluator"
	"strings"
	"testing"
)

func TestIO(t *testing.T) {
	savedIn, savedErr, savedOut := Stdin, Stderr, evaluator.Output
	defer func() { Stdin, Stderr, evaluator.Output = savedIn, savedErr, savedOut }()
	for _, e := range engines {
		var out, errOut bytes.Buffer
		Stdin = strings.NewReader("one\r\ntwo\n\nlast")
		Stderr, evaluator.Output = &errOut, &out
		got := runOn(t, e.run, `let lines = []
let line = io.read_line()
while (line != null) { lines = lines + [line]; line = io.read_line() }
io.write("no break"); puts("!"); io.error("to stderr")
[lines, io.read_line()]`)
		if got.Inspect() != `[["one", "two", "", "last"], null]` {
			t.Errorf("%s: got %s", e.name, got.Inspect())
		}
		if out.String() != "no break!\n" || errOut.String() != "to stderr" {
			t.Errorf("%s: wrote %q and %q", e.name, out.String(), errOut.String())
		}
	}
	check(t, []testCase{
		{"keys", `keys(io)`, `["error", "read_line", "write"]`, false},
		{"write a number", `io.write(1)`, "want a STRING text, got INTEGER", true},
		{"error takes one", `io.error()`, "want 1, got 0", true},
		{"read_line takes nothing", `io.read_line(1)`, "want 0, got 1", true},
	})
}
