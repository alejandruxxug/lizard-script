package terrarium

import (
	"embed"
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"strings"
)

// The modules written in lizard-script. Each one is a file lib/name.liz
// that defines a variable name holding a hash of functions.
//
// A module runs in an environment of its own, so a program's variables
// (a program's own let text = ...) never reach the module's code. Each of
// its functions is handed to the program as a builtin called name.function,
// so its errors read like any builtin's: at the program's call, with the
// builtin's name in front.

//go:embed lib/*.liz
var sources embed.FS

// lizardModules are the modules in lib, in the order they are installed.
var lizardModules = []string{"json"}

// lizardModule runs lib/name.liz and returns the module it defines.
func lizardModule(name string) (*object.Hash, error) {
	file := "terrarium/" + name + ".liz"
	code, err := sources.ReadFile("lib/" + name + ".liz")
	if err != nil {
		return nil, err
	}
	p := parser.New(lexer.NewNamed(string(code), file))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, fmt.Errorf("%s: %s", file, strings.Join(p.Errors(), "; "))
	}
	env := object.NewEnvironment()
	if result, ok := evaluator.Eval(program, env).(*object.Error); ok {
		return nil, fmt.Errorf("%s: %s", file, result.Message)
	}
	value, _ := env.Get(name)
	functions, ok := value.(*object.Hash)
	if !ok {
		return nil, fmt.Errorf("%s doesn't define a hash called %s", file, name)
	}
	return asBuiltins(name, functions), nil
}

// asBuiltins makes the module hash the program sees: every function of
// functions as a builtin.
func asBuiltins(module string, functions *object.Hash) *object.Hash {
	builtins := map[string]object.BuiltinFunction{}
	functions.Each(func(pair object.HashPair) {
		name := pair.Key.Inspect()
		builtins[name] = callLizard(module+"."+name, pair.Value)
	})
	return arg.Module(module, builtins)
}

// callLizard makes the Go side of the builtin called name, which calls the
// lizard-script function f.
func callLizard(name string, f object.Object) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		result := evaluator.ApplyFunction(token.Token{}, f, args)
		err, ok := result.(*object.Error)
		if !ok {
			return result
		}
		// The evaluator puts the program's position and the builtin's name
		// back in front, so both come off here.
		prefix := fmt.Sprintf("[line %d, column %d] runtime error: ", err.Line, err.Column)
		message := strings.TrimPrefix(strings.TrimPrefix(err.Message, prefix), name+": ")
		return &object.Error{Message: message, Uncatchable: err.Uncatchable}
	}
}
