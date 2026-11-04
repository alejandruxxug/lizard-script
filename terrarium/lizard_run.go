package terrarium

import (
	"bytes"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/engine/vm"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"strings"
)

// runOptions are the options of lizard.run.
type runOptions struct {
	globals     *object.Hash
	steps       int64
	name        string
	vm          bool
	onStatement object.Object
	onLayer     object.Object
	hookSource  *string  // on_statement only for statements of this source
	hookBodies  bool     // on_statement for the statements of function bodies too
	uses        []string // modules the code sees alive, as if it used them
}

// lizard.run(code, options = {}) runs code in a sandbox: a new top scope
// with only the builtins, the libraries and what options.globals gives,
// from a fresh start (its own steps, random numbers and modules), and the
// program that called it goes on as before. It gives {"value", "error",
// "errors", "output", "steps", "globals"}: the code's value (null when it
// failed), its error {"kind", "message", "line", "column", "source"} or
// null, every error (each syntax error of a code that doesn't parse, or the
// one error that stopped it, or none), what it printed, the steps it took
// and its top-level names after the run.
//
// Options: "globals" (a hash of names to define), "steps" (the most steps
// it may take, 0 for no limit), "name" (its source name, where its use
// statements look for modules), "engine" ("evaluator" or "vm"),
// "on_statement" (a function called with line, column and source before
// each statement), "hook_source" (a source name: on_statement is called only
// for the statements of that source, not the ones of the modules),
// "hook_functions" (false: on_statement is not called for the statements of
// function bodies, only for the ones outside every function), "on_layer"
// (called with the name, true or false and the line when a layer starts and
// ends) and "use" (a list of module paths the code sees alive before its
// first line, as if it started with use "path" for each one, so writing a
// module's variable changes the module). The hooks run as the calling
// program: their steps and output are its own.
func lizardRun(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	code, err := arg.String(args, 0, "code")
	if err != nil {
		return err
	}
	options := runOptions{globals: object.NewHash(), hookBodies: true}
	if len(args) == 2 {
		hash, err := arg.Hash(args, 1, "options")
		if err != nil {
			return err
		}
		if err := readRunOptions(hash, &options); err != nil {
			return err
		}
	}

	result := object.NewHash()
	p := parser.New(lexer.NewNamed(code, options.name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		errors := parseErrors(p.Errors())
		for _, e := range errors.Elements {
			arg.Set(e.(*object.Hash), "source", arg.Str(options.name))
		}
		fillRunResult(result, arg.Null, errors, "", 0, object.NewEnvironment())
		return result
	}
	return sandbox(program, options, result)
}

// readRunOptions checks the options of lizard.run and fills o.
func readRunOptions(hash *object.Hash, o *runOptions) *object.Error {
	var bad *object.Error
	hash.Each(func(pair object.HashPair) {
		if bad != nil {
			return
		}
		key, value := pair.Key.Inspect(), pair.Value
		wrong := func(want string) {
			bad = arg.Errorf("the option %s must be %s, got %s", key, want, arg.Name(value))
		}
		switch key {
		case "globals":
			globals, ok := value.(*object.Hash)
			if !ok {
				wrong("a HASH")
				return
			}
			o.globals = globals
		case "steps":
			n, ok := value.(*object.Integer)
			if !ok || n.Value < 0 {
				wrong("an INTEGER, 0 or more")
				return
			}
			o.steps = n.Value
		case "name":
			s, ok := value.(*object.String)
			if !ok {
				wrong("a STRING")
				return
			}
			o.name = s.Value
		case "engine":
			s, ok := value.(*object.String)
			if !ok || (s.Value != "evaluator" && s.Value != "vm") {
				wrong(`"evaluator" or "vm"`)
				return
			}
			o.vm = s.Value == "vm"
		case "hook_source":
			s, ok := value.(*object.String)
			if !ok {
				wrong("a STRING")
				return
			}
			o.hookSource = &s.Value
		case "hook_functions":
			b, ok := value.(*object.Boolean)
			if !ok {
				wrong("true or false")
				return
			}
			o.hookBodies = b.Value
		case "use":
			list, ok := value.(*object.Array)
			if !ok {
				wrong("a list of module paths")
				return
			}
			for _, e := range list.Elements {
				path, ok := e.(*object.String)
				if !ok {
					bad = arg.Errorf("the option use must be a list of module paths, got %s in it", arg.Name(e))
					return
				}
				o.uses = append(o.uses, path.Value)
			}
		case "on_statement", "on_layer":
			switch value.Type() {
			case object.FUNCTION_OBJ, object.BUILTIN_OBJ, object.NULL_OBJ:
			default:
				wrong("a function")
				return
			}
			if key == "on_statement" {
				o.onStatement = value
			} else {
				o.onLayer = value
			}
		default:
			bad = arg.Errorf("unknown option %s, the options are globals, steps, name, engine, use, on_statement, hook_source, hook_functions and on_layer",
				object.Quote(key))
		}
	})
	return bad
}

// sandbox runs program with options and fills result.
func sandbox(program *ast.Program, o runOptions, result *object.Hash) object.Object {
	outer := evaluator.SaveRunState()
	var printed bytes.Buffer
	env := object.NewEnvironment()
	globals := func(target *object.Environment) {
		o.globals.Each(func(pair object.HashPair) {
			if name, ok := pair.Key.(*object.String); ok {
				target.Define(name.Value, pair.Value)
			}
		})
	}
	globals(env)

	evaluator.Output = &printed
	evaluator.MaxSteps = int(o.steps)
	evaluator.ModulePrelude = globals
	evaluator.StartRun()
	var hookErr *object.Error
	call := func(f object.Object, values ...object.Object) {
		if hookErr != nil {
			return
		}
		outer.Outside(func() {
			if e, ok := evaluator.ApplyFunction(token.Token{}, f, values).(*object.Error); ok {
				hookErr = e
			}
		})
	}
	evaluator.OnStatement, evaluator.OnLayer = nil, nil
	evaluator.HookFunctions = o.hookBodies
	if o.onStatement != nil && o.onStatement != arg.Null {
		evaluator.OnStatement = func(node ast.Statement) {
			source := statementSource(node)
			if o.hookSource != nil && source != *o.hookSource {
				return
			}
			span := node.Range()
			call(o.onStatement, arg.Int64(int64(span.StartLine)), arg.Int64(int64(span.StartColumn)),
				arg.Str(source))
		}
	}
	if o.onLayer != nil && o.onLayer != arg.Null {
		evaluator.OnLayer = func(name string, enter bool, tok token.Token) {
			call(o.onLayer, arg.Str(name), arg.Bool(enter), arg.Int64(int64(tok.Line)))
		}
	}

	// The engine of the code, and the one its modules run on.
	run, runModule := evalProgram, evaluator.Runner(evalProgram)
	if o.vm {
		run, runModule = vm.Run, vm.RunModule
	}
	value := useModules(o, env, runModule)
	if value == nil {
		value = run(program, env)
	}
	steps := evaluator.Steps()
	outer.Restore()

	if hookErr != nil {
		// The hook is the calling program's own code: its error is the
		// error of lizard.run, at the program's call.
		return &object.Error{Message: "a hook failed: " + stripPosition(hookErr), Uncatchable: hookErr.Uncatchable}
	}
	errors := &object.Array{}
	if err, ok := value.(*object.Error); ok {
		errors.Elements, value = []object.Object{runError(err)}, arg.Null
	}
	if value == nil {
		value = arg.Null
	}
	fillRunResult(result, value, errors, printed.String(), steps, env)
	return result
}

// evalProgram runs a program on the evaluator.
func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	return evaluator.Eval(program, env)
}

// useModules makes env use the modules of the option use, in order, the
// way use statements at the top of the code would: found from the code's
// source name and run on its engine. It gives the error of the first one
// that fails, or nil.
func useModules(o runOptions, env *object.Environment, run evaluator.Runner) object.Object {
	for _, path := range o.uses {
		tok := token.Token{Type: token.IDENT, Literal: "use", Line: 1, Column: 1, Source: o.name}
		node := &ast.UseStatement{Token: tok, Path: &ast.StringLiteral{Token: tok, Value: path}}
		if err, ok := evaluator.Use(node, env, run).(*object.Error); ok {
			return err
		}
	}
	return nil
}

// fillRunResult puts the parts of the result of lizard.run in h: its
// error is the first of errors, or null.
func fillRunResult(h *object.Hash, value object.Object, errors *object.Array, output string, steps int, env *object.Environment) {
	globals := object.NewHash()
	for _, name := range env.Names() {
		v, _ := env.Get(name)
		arg.Set(globals, name, v)
	}
	first := object.Object(arg.Null)
	if len(errors.Elements) != 0 {
		first = errors.Elements[0]
	}
	arg.Set(h, "value", value)
	arg.Set(h, "error", first)
	arg.Set(h, "errors", errors)
	arg.Set(h, "output", arg.Str(output))
	arg.Set(h, "steps", arg.Int64(int64(steps)))
	arg.Set(h, "globals", globals)
}

// runError is the hash of the error that stopped the code.
func runError(err *object.Error) *object.Hash {
	h := object.NewHash()
	kind := "runtime"
	if m := positionPrefix.FindStringSubmatch(err.Message); m != nil {
		kind = m[3]
	}
	arg.Set(h, "kind", arg.Str(kind))
	arg.Set(h, "message", arg.Str(stripPosition(err)))
	arg.Set(h, "line", arg.Int64(int64(err.Line)))
	arg.Set(h, "column", arg.Int64(int64(err.Column)))
	arg.Set(h, "source", arg.Str(err.Source))
	return h
}

// stripPosition is the message of err without the position and kind in
// front of its first line.
func stripPosition(err *object.Error) string {
	if m := positionPrefix.FindStringSubmatch(err.Message); m != nil {
		return strings.TrimPrefix(err.Message, m[0])
	}
	return err.Message
}

// statementSource is the source name of a statement's first token.
func statementSource(node ast.Statement) string {
	switch n := node.(type) {
	case *ast.LetStatement:
		return n.Token.Source
	case *ast.ReturnStatement:
		return n.Token.Source
	case *ast.BreakStatement:
		return n.Token.Source
	case *ast.ContinueStatement:
		return n.Token.Source
	case *ast.TryStatement:
		return n.Token.Source
	case *ast.ExpressionStatement:
		return n.Token.Source
	case *ast.BlockStatement:
		return n.Token.Source
	case *ast.LayerStatement:
		return n.Token.Source
	case *ast.UseStatement:
		return n.Token.Source
	}
	return ""
}
