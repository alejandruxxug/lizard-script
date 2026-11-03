// Package format lays out lizard-script code one way, like gofmt does for
// Go: the same program always comes out the same, whoever wrote it. The
// rules are in docs/STYLE.md:
//
//   - two spaces of indentation for each block, bracket and layer body, and
//     two more for the next lines of a statement that goes on;
//   - one space around binary operators and after a comma, a ';' or a ':',
//     none inside brackets, around a '.', after a prefix operator or before
//     the '(' of a call;
//   - a block's '{' on the line of what it belongs to, and "} else {",
//     "} elif (...) {" and "} catch (e) {" on one line;
//   - a hash that spans lines has one pair per line;
//   - blank lines are kept, but never two in a row;
//   - the comments at the end of consecutive lines start in one column;
//   - no spaces at the end of a line, and one line break at the end.
//
// The formatter only changes the space between tokens, and it never joins
// or splits lines where that would change a statement (LEP 0001): every line
// break it removes or adds is one the parser reads as a space. Long lines
// are left as they are. Before giving the result back, Source checks it:
// the tokens and the syntax tree must be the ones of the input.
package format

import (
	"errors"
	"fmt"
	"lizardscript/lexer"
	"lizardscript/parser"
	"lizardscript/token"
	"strings"
)

// Source returns text laid out by the rules. Code with syntax errors can't
// be formatted: the error lists them.
func Source(text string) (string, error) {
	p := parser.New(lexer.New(text))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return "", errors.New(strings.Join(p.Errors(), "\n"))
	}

	tokens := readTokens(text)
	out := layout(tokens, collect(program))
	if err := sameProgram(text, out, program.String()); err != nil {
		return "", err
	}
	return out, nil
}

// sameProgram checks that out has the tokens and the tree of in, whose tree
// prints as tree. It can only fail through a bug of the formatter, and then
// the file is better left alone.
func sameProgram(in, out, tree string) error {
	before, after := tokenList(in), tokenList(out)
	if len(before) != len(after) {
		return fmt.Errorf("format: the layout changed the tokens (%d before, %d after)", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			return fmt.Errorf("format: the layout changed the token %q into %q", before[i], after[i])
		}
	}
	p := parser.New(lexer.New(out))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 || program.String() != tree {
		return errors.New("format: the layout would change what the program means")
	}
	return nil
}

// tokenList is each token of text as its type and literal, comments
// without the spaces at their end.
func tokenList(text string) []string {
	var list []string
	l := lexer.New(text)
	l.KeepComments()
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		literal := tok.Literal
		if tok.Type == token.COMMENT {
			literal = strings.TrimRight(literal, " \t\r")
		}
		list = append(list, string(tok.Type)+" "+literal)
	}
	return list
}
