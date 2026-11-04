package pen

import (
	"errors"
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Runner runs a parsed program: the tree-walker or the VM.
type Runner func(program *ast.Program, env *object.Environment) object.Object

// ConfigPath is the user's pen file, <config dir>/lizard/pen.liz.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lizard", "pen.liz"), nil
}

// LoadUserConfig reads the user's pen file, if there is one. A broken file
// prints a warning to warn and leaves every curve as it was: a setting
// never stops a program.
func LoadUserConfig(warn io.Writer) {
	path, err := ConfigPath()
	if err != nil {
		return
	}
	eval := func(p *ast.Program, env *object.Environment) object.Object { return evaluator.Eval(p, env) }
	if err := LoadConfig(path, eval); err != nil {
		fmt.Fprintf(warn, "lizardscript: ignoring %s: %v\n", path, err)
	}
}

// LoadConfig applies the pen file at path. A file that doesn't exist is no
// error. Either every curve in the file is valid and all are set, or none.
func LoadConfig(path string, run Runner) error {
	code, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return applyConfig(string(code), path, run)
}

// applyConfig runs the file's code and takes the variables named after the
// sources (trackpad, tablet, mouse) as curves to set on top of the current ones.
func applyConfig(code, name string, run Runner) error {
	p := parser.New(lexer.NewNamed(code, name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return errors.New(strings.Join(p.Errors(), "; "))
	}
	env := object.NewEnvironment()
	if result, ok := run(program, env).(*object.Error); ok {
		return errors.New(result.Message)
	}
	next := map[string]Curve{}
	for _, source := range Sources {
		value, defined := env.Get(source)
		if !defined {
			continue
		}
		h, isHash := value.(*object.Hash)
		if !isHash {
			return fmt.Errorf("%s must be a hash like {\"low\": 0.8}, got %s", source, value.Type())
		}
		c, err := CurveOf(source).With(h)
		if err != nil {
			return fmt.Errorf("%s: %v", source, err)
		}
		next[source] = c
	}
	for source, c := range next {
		if err := SetCurve(source, c); err != nil {
			return err
		}
	}
	return nil
}
