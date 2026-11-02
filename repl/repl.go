// Package repl runs lizard-script code: interactively, one input at a time
// (the Read-Eval-Print Loop), or a whole source file at once.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"lizardscript/token"
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
// line.
func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	env := object.NewEnvironment()
	evaluator.Output = out

	var input strings.Builder
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
		if isIncomplete(input.String()) && strings.TrimSpace(line) != "" {
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

	result := evaluator.Eval(program, env)
	switch result := result.(type) {
	case nil, *object.Null:
	case *object.Error:
		fmt.Fprintln(out, result.Message)
	case *object.String:
		fmt.Fprintln(out, "\""+result.Value+"\"")
	default:
		fmt.Fprintln(out, result.Inspect())
	}
}

// Run runs a whole program. puts writes to out, and syntax and runtime
// errors go to errOut. It reports whether the program ran without errors.
func Run(source string, out, errOut io.Writer) bool {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		printErrors(errOut, p.Errors())
		return false
	}

	evaluator.Output = out
	if err, ok := evaluator.Eval(program, object.NewEnvironment()).(*object.Error); ok {
		fmt.Fprintln(errOut, err.Message)
		return false
	}
	return true
}

func printErrors(out io.Writer, errors []string) {
	for _, message := range errors {
		fmt.Fprintln(out, message)
	}
}

// isIncomplete reports whether input looks like the start of something
// longer: it has an open '(', '[' or '{', an open string, or it ends with an
// operator or a ','. It only decides when to keep reading. The parser still
// judges what is read.
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

	if depth > 0 {
		return true
	}
	switch last.Type {
	case token.ASSIGN, token.PLUS, token.MINUS, token.ASTERISK, token.SLASH,
		token.DOUBLE_SLASH, token.POW, token.MODULE, token.EQ, token.NOT_EQ,
		token.EXACT, token.NOT_EXACT, token.LT, token.GT, token.LTE, token.GTE,
		token.AND, token.OR, token.COMMA, token.BANG:
		return true
	}
	return false
}
