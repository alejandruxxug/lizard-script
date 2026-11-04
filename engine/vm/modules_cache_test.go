package vm

import (
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"os"
	"path/filepath"
	"testing"
)

// TestModulesAreCompiledOnce checks that a module used by two runs is
// compiled once, and again when its file changes.
func TestModulesAreCompiledOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sides.liz")
	write := func(source string) {
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func() string {
		program := parser.New(lexer.NewNamed("use \"sides\"\nsides", filepath.ToSlash(dir)+"/main.liz")).ParseProgram()
		evaluator.ResetModules()
		return Run(program, object.NewEnvironment()).Inspect()
	}
	clear(moduleUnits)
	write("let sides = 3")
	if got := run(); got != "3" || len(moduleUnits) != 1 {
		t.Fatalf("got %s with %d units, want 3 with 1", got, len(moduleUnits))
	}
	if got := run(); got != "3" || len(moduleUnits) != 1 {
		t.Errorf("the second run gave %s with %d units, want 3 with 1", got, len(moduleUnits))
	}
	write("let sides = 4")
	if got := run(); got != "4" || len(moduleUnits) != 2 {
		t.Errorf("a changed file gave %s with %d units, want 4 with 2", got, len(moduleUnits))
	}
}
