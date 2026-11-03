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

// moduleValues are what some programs of TestModulesOnBothEngines must
// give, so the engines can't agree on a wrong one.
var moduleValues = map[string]string{
	"use \"ring_a\"\nuse \"ring_b\"\n[seen, seen_in_hash, from_a(), hash_later()]": "[1, 1, 22, 2]",
	"use \"early_a\"":                                                     "early_a.liz defines it further down, but it is still loading",
	"use \"checks\"\n\ntwice(\"a\")":                                      "line 3 column 6",
	"use \"checks\"\n[twice(4), inside()]":                                "[8, [2, 36]]",
	"use \"checks\"\ntry { twice(1.5) } catch (e) { [e.line, e.column] }": "[2, 12]",
	"use \"checks\"\nfunction go() { twice(\"b\") }\ngo()":                "line 2 column 22",
}

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
		"ring_a.liz":  "let first = 1\nuse \"ring_b\"\nlet later = 2\nfunction from_a() { return from_b() + later }",
		"ring_b.liz":  "use \"ring_a\"\nuse \"ring_a\" as a\nlet seen = first\nlet seen_in_hash = a.first\nfunction from_b() { return later * 10 }\nfunction hash_later() { return a.later }",
		"early_a.liz": "use \"early_b\"\nlet late = 5",
		"early_b.liz": "use \"early_a\"\nlet x = late + 1",
		"checks.liz":  "function _need(v) {\n  if (type(v) != \"INTEGER\") { error(\"twice: want an INTEGER\", {\"at\": \"caller\"}) }\n}\nfunction twice(v) { _need(v); return v * 2 }\nfunction inside() { try { _need(\"x\") } catch (e) { return [e.line, e.column] } }",
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
		"use \"ring_a\"\nuse \"ring_b\"\n[seen, seen_in_hash, from_a(), hash_later()]",
		"use \"early_a\"",
		"use \"checks\"\n\ntwice(\"a\")",
		"use \"checks\"\n[twice(4), inside()]",
		"use \"checks\"\ntry { twice(1.5) } catch (e) { [e.line, e.column] }",
		"use \"checks\"\nfunction go() { twice(\"b\") }\ngo()",
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
		if want, ok := moduleValues[input]; ok && !strings.Contains(compiled.value, want) {
			t.Errorf("%q gave %s on the VM, want %s", input, compiled.value, want)
		}
	}
}
