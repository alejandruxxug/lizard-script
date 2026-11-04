package terrarium

import (
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	checker "github.com/alejandruxxug/lizard-script/tools/check" // named apart from the tests' check helper
	"sync"
)

// CheckGlobals are the names the Terrarium gives a program, with the
// members of each module: the check.Globals of a program run by
// lizardscript. They are read once, from a scope Install fills, with the
// run state of the program that asks put aside (installing runs json.liz).
func CheckGlobals() map[string][]string {
	checkGlobalsOnce.Do(func() {
		state := evaluator.SaveRunState()
		defer state.Restore()
		evaluator.OnStatement, evaluator.OnLayer = nil, nil
		evaluator.MaxSteps = 0
		evaluator.StartRun()
		env := object.NewEnvironment()
		Install(env)
		checkGlobals = checker.GlobalsOf(env)
	})
	return checkGlobals
}

var (
	checkGlobalsOnce sync.Once
	checkGlobals     map[string][]string
)

// lizard.check(code, name = "", names = []) checks code without running it
// (package check): an array of {"severity", "rule", "message", "hint",
// "line", "column", "end_line", "end_column"}, in the order of the code.
// The code has the Terrarium's names, and names lists more that the host
// running it will define. name is its source name, where its use statements
// look for modules.
func lizardCheck(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 3); err != nil {
		return err
	}
	code, name, err := codeArgs(args[:min(len(args), 2)])
	if err != nil {
		return err
	}
	globals := map[string][]string{}
	for global, members := range CheckGlobals() {
		globals[global] = members
	}
	if len(args) == 3 {
		names, err := arg.Array(args, 2, "list of names")
		if err != nil {
			return err
		}
		for _, n := range names.Elements {
			s, ok := n.(*object.String)
			if !ok {
				return arg.Errorf("want names as STRING, got %s", arg.Name(n))
			}
			globals[s.Value] = nil
		}
	}

	out := &object.Array{}
	for _, d := range checker.SourceWith(name, code, globals) {
		h := object.NewHash()
		arg.Set(h, "severity", arg.Str(string(d.Severity)))
		arg.Set(h, "rule", arg.Str(d.Rule))
		arg.Set(h, "message", arg.Str(d.Message))
		hint := arg.Null
		if d.Hint != "" {
			hint = arg.Str(d.Hint)
		}
		arg.Set(h, "hint", hint)
		positions(h, d.Line, d.Column, d.EndLine, d.EndColumn)
		out.Elements = append(out.Elements, h)
	}
	return out
}
