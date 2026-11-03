package vm

import (
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	lizardast "lizardscript/ast"
)

// TestModulesOnBothEngines runs programs that use modules on both engines
// and compares everything they show, hooks and steps included.
func TestModulesOnBothEngines(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"shapes.liz":  "let sides = 3\nfunction area(w, h) { return w * h }\nlet _hidden = 1",
		"counter.liz": "let count = 0\nfunction bump() { count += 1; return count }",
		"layered.liz": "puts(\"loading\")\nlayer art\n  let made = 1\nfunction twice(f) { return [f(), f()] }",
		"cycle.liz":   "use \"cycle\"",
		"fails.liz":   "let a = 1\nfunction boom() { return a + \"x\" }\nlet b = boom()",
		"loops.liz":   "let total = 0\nfor (let i = 0; i < 50; i += 1) { total += i }",
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	programs := []string{
		"use \"shapes\"\narea(2, sides)",
		"use \"shapes\" as s\n[s.sides, keys(s), s.area(1, 2)]",
		"use \"counter\"\nbump()\ncount = 10\nbump()\n[count, bump()]",
		"use \"counter\"\nlet count = 5\nfunction f() { return count }\nbump()\n[f(), count]",
		"use \"counter\"\nfunction read() { return count }\nlet before = read()\nbump()\n[before, read()]",
		"use \"layered\"\nuse \"layered\"\ntwice(function() { made += 1; return made })",
		"use \"cycle\"",
		"use \"fails\"",
		"use \"loops\"\ntotal",
		"use \"missing\"",
		"use \"shapes\"\n_hidden",
		"puts(1)\nuse \"counter\" as c\nlet x = c.bump()\nuse \"counter\"\n[x, count]",
	}
	saved := evaluator.MaxSteps
	defer func() { evaluator.MaxSteps = saved }()
	for _, input := range programs {
		p := parser.New(lexer.NewNamed(input, filepath.ToSlash(dir)+"/main.liz"))
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("%q: %v", input, p.Errors())
		}
		fresh := func(run func(*lizardast.Program, *object.Environment) object.Object) func(*lizardast.Program, *object.Environment) object.Object {
			return func(program *lizardast.Program, env *object.Environment) object.Object {
				evaluator.ResetModules()
				return run(program, env)
			}
		}
		tree := runEngine(program, fresh(func(p *lizardast.Program, env *object.Environment) object.Object {
			return evaluator.Eval(p, env)
		}), true)
		compiled := runEngine(program, fresh(Run), true)
		// Each run parses the modules again, so their statements are new
		// nodes: compare the hooks without the addresses.
		addresses := regexp.MustCompile(`0x[0-9a-f]+`)
		if a, b := addresses.ReplaceAllString(tree.String(), "_"), addresses.ReplaceAllString(compiled.String(), "_"); a != b {
			t.Errorf("the engines differ on %q\n--- evaluator:\n%s\n--- vm:\n%s", input, tree, compiled)
		}
		if strings.Contains(input, "area(2") && tree.value != "INTEGER 6" {
			t.Errorf("%q gave %s", input, tree.value)
		}
	}
}
