// Package repl runs lizard-script code: interactively, one input at a time
// (the Read-Eval-Print Loop), or a whole source file at once.
package repl

import (
	"bufio"
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/engine/vm"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"io"
	"path/filepath"
	"strings"
)

const (
	PROMPT      = "🦎 >> "
	CONTINUE    = "🦎 .. "
	maxLineSize = 1 << 20
)

// Start runs the REPL until in ends or the user types exit. Variables live
// on from one input to the next. An input that is clearly not finished, like
// one with an open '{', keeps reading lines until it is, or until an empty
// line. An input that ends with an if waits for one more line, because an
// elif or else there continues it: the input runs once that line, or an
// empty one, shows it doesn't. The Terrarium is there, but for window and
// pen.
func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	env := object.NewEnvironment()
	evaluator.Output = out
	evaluator.ResetSteps()
	evaluator.ResetMath()
	evaluator.ResetModules()
	evaluator.ModulePrelude = terrarium.InstallCore
	terrarium.InstallCore(env)

	var input strings.Builder
	waitingForElse := false
	for {
		if input.Len() == 0 {
			fmt.Fprint(out, PROMPT)
		} else {
			fmt.Fprint(out, CONTINUE)
		}

		if !scanner.Scan() {
			if input.Len() > 0 {
				fmt.Fprintln(out)
				evalInput(input.String(), env, out)
			}
			fmt.Fprintln(out)
			return
		}
		line := scanner.Text()
		blank := strings.TrimSpace(line) == ""

		// The if before this line is complete, so it runs first, and then
		// the line is handled as a new input.
		if waitingForElse && !continuesIf(line) {
			evalInput(input.String(), env, out)
			input.Reset()
		}
		waitingForElse = false

		if input.Len() == 0 {
			switch strings.TrimSpace(line) {
			case "":
				continue
			case "exit", "quit":
				return
			}
		}

		input.WriteString(line)
		input.WriteString("\n")
		if !blank && isIncomplete(input.String()) {
			continue
		}
		if !blank && endsWithOpenIf(input.String()) {
			waitingForElse = true
			continue
		}

		evalInput(input.String(), env, out)
		input.Reset()
	}
}

// evalInput runs one input of the REPL and shows its value. null is not
// shown, so a let or a puts prints nothing extra.
func evalInput(input string, env *object.Environment, out io.Writer) {
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		printErrors(out, p.Errors())
		return
	}

	evaluator.ResetSteps()
	result := evaluator.Eval(program, env)
	switch result := result.(type) {
	case nil, *object.Null:
	case *object.Error:
		PrintError(out, result, "")
	case *object.String:
		fmt.Fprintln(out, quoteForDisplay(result.Value))
	default:
		fmt.Fprintln(out, result.Inspect())
	}
}

// quoteForDisplay shows a string result the way it is written in code, so
// it can be pasted back. Only the quote and the backslash are escaped: a new
// line stays a new line, which a string literal can hold, so a string over
// several lines still reads as several lines.
func quoteForDisplay(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `"`, `\"`)
	return `"` + text + `"`
}

// Run runs a whole program with the Terrarium installed. puts writes to out,
// and syntax and runtime errors go to errOut. It reports whether the program
// ran without errors.
func Run(source string, out, errOut io.Writer) bool {
	return RunFile("", source, out, errOut, false)
}

// RunVM is Run on the bytecode VM (package vm) instead of the evaluator.
// The output is the same.
func RunVM(source string, out, errOut io.Writer) bool {
	return RunFile("", source, out, errOut, true)
}

// RunFile is Run for the program in the file name, on the VM when onVM is
// set. The name is where its use statements look for modules (next to it)
// and what os.program_dir() tells; it doesn't change any message of the
// program's own, and an error from a module shows the module's file in
// front. Modules get the Terrarium too, and io.error writes to errOut.
func RunFile(name, source string, out, errOut io.Writer, onVM bool) bool {
	p := parser.New(lexer.NewNamed(source, name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		printErrors(errOut, p.Errors())
		return false
	}

	evaluator.Output = out
	evaluator.ResetSteps()
	evaluator.ResetMath()
	evaluator.ResetModules()
	evaluator.ModulePrelude = terrarium.Install
	terrarium.Stderr = errOut
	terrarium.ProgramFile = name
	env := object.NewEnvironment()
	terrarium.Install(env)
	var result object.Object
	if onVM {
		result = vm.Run(program, env)
	} else {
		result = evaluator.Eval(program, env)
	}
	if err, ok := result.(*object.Error); ok {
		PrintError(errOut, err, name)
		return false
	}
	return true
}

// PrintError writes err the way the command line shows it: its message,
// and for an error from another file than main (a module) that file in
// front of each line, from main's folder: "draw/shapes.liz: [line 2,
// column 5] runtime error: ...". A standard module shows its own name,
// like "std:/lily2d/shapes.liz".
func PrintError(out io.Writer, err *object.Error, main string) {
	if err.Source == "" || err.Source == main {
		fmt.Fprintln(out, err.Message)
		return
	}
	source := err.Source
	if rel, rerr := filepath.Rel(filepath.Dir(main), source); rerr == nil && !evaluator.IsStandardModule(source) {
		source = filepath.ToSlash(rel)
	}
	for _, line := range strings.Split(err.Message, "\n") {
		fmt.Fprintln(out, source+": "+line)
	}
}

func printErrors(out io.Writer, errors []string) {
	for _, message := range errors {
		fmt.Fprintln(out, message)
	}
}

// isIncomplete reports whether input looks like the start of something
// longer. A statement ends at the end of its line, the same rule as in a
// file (lexer.ContinuesLine): the input goes on while a '(', '[' or '{' or a
// string is open, or when it ends with an operator, a ',' or an opening
// bracket. It also goes on after a keyword that needs more after it, like
// else. It only decides when to keep reading: the parser still judges what
// is read.
func isIncomplete(input string) bool {
	l := lexer.New(input)
	depth := 0
	var last token.Token

	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		switch tok.Type {
		case token.LPAREN, token.LBRACKET, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACKET, token.RBRACE:
			depth--
		case token.ILLEGAL:
			if strings.HasPrefix(tok.Literal, "\"") {
				return true
			}
		}
		last = tok
	}

	if depth > 0 || lexer.ContinuesLine(last.Type) {
		return true
	}
	switch last.Type {
	case token.LET, token.FUNCTION, token.IF, token.ELIF, token.ELSE, token.WHILE, token.FOR:
		return true
	}
	return false
}

// continuesIf reports whether line starts with elif or else, so it belongs
// to the if before it.
func continuesIf(line string) bool {
	first := lexer.New(line).NextToken()
	return first.Type == token.ELIF || first.Type == token.ELSE
}

// endsWithOpenIf reports whether input is a complete program whose last
// token is the '}' of an if that has no else yet, so an elif or else on the
// next line could still continue it.
func endsWithOpenIf(input string) bool {
	l := lexer.New(input)
	var last token.Token
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		last = tok
	}
	if last.Type != token.RBRACE {
		return false
	}

	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 || len(program.Statements) == 0 {
		return false
	}
	return isOpenIf(lastExpression(program.Statements[len(program.Statements)-1]))
}

// lastExpression finds the expression that statement ends with, the one an
// else on the next line would attach to. It is nil if statement doesn't end
// with an expression.
func lastExpression(statement ast.Statement) ast.Expression {
	var expression ast.Expression
	switch statement := statement.(type) {
	case *ast.ExpressionStatement:
		expression = statement.Expression
	case *ast.LetStatement:
		expression = statement.Value
	case *ast.ReturnStatement:
		expression = statement.ReturnValue
	}

	for {
		switch e := expression.(type) {
		case *ast.AssignExpression:
			expression = e.Value
		case *ast.InfixExpression:
			expression = e.Right
		case *ast.PrefixExpression:
			expression = e.Right
		default:
			return expression
		}
	}
}

// isOpenIf reports whether expression is an if whose chain of elifs doesn't
// end with an else.
func isOpenIf(expression ast.Expression) bool {
	ifExpression, ok := expression.(*ast.IfExpression)
	if !ok {
		return false
	}
	for ifExpression.Alternative != nil {
		// An elif or an "else if" is an if alone in the else block, with
		// the elif or if token. A plain else block has the '{' token.
		alternative := ifExpression.Alternative
		if alternative.Token.Type != token.ELIF && alternative.Token.Type != token.IF {
			return false
		}
		statement := alternative.Statements[0].(*ast.ExpressionStatement)
		ifExpression = statement.Expression.(*ast.IfExpression)
	}
	return true
}
