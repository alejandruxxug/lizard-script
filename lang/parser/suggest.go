package parser

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
)

// Messages that teach: when a mistake looks like a common slip, the error
// says what was probably meant.

// keywords are the words a misspelled one is compared with: the reserved
// words and the names that work as words at the start of a statement.
var keywords = []string{
	"function", "let", "return", "if", "else", "elif", "while", "for", "break",
	"continue", "layer", "try", "catch", "use",
}

// slips are words from other languages, and their lizard-script word.
var slips = map[string]string{
	"elseif": "elif", "elsif": "elif",
	"func": "function", "fn": "function", "fun": "function", "def": "function",
	"var": "let", "const": "let", "val": "let",
}

// suggestKeyword returns the keyword that word was probably meant to be, or
// "" when it doesn't look like one. A word counts when it is a known slip,
// or one letter away from a keyword (two for the longer ones), where two
// letters swapped count as one. A keyword of three letters only counts with
// a letter changed or swapped, so a name like "form" or "user" doesn't.
func suggestKeyword(word string) string {
	if keyword, ok := slips[word]; ok {
		return keyword
	}
	if len(word) < 3 {
		return ""
	}
	for _, keyword := range keywords {
		limit := 1
		if len(keyword) > 5 {
			limit = 2
		}
		if len(keyword) <= 3 && len(word) != len(keyword) {
			continue
		}
		if word != keyword && editDistance(word, keyword) <= limit {
			return keyword
		}
	}
	return ""
}

// editDistance counts the letters to add, remove, change or swap with the
// next one to turn a into b.
func editDistance(a, b string) int {
	rows := make([][]int, len(a)+1)
	for i := range rows {
		rows[i] = make([]int, len(b)+1)
		rows[i][0] = i
	}
	for j := range len(b) + 1 {
		rows[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+1)
			}
		}
	}
	return rows[len(a)][len(b)]
}

// misspelledKeyword turns the error of a broken statement that starts with a
// word like "fucntion" into "did you mean" at that word, when the statement
// gave one error, on the word's line. errors is how many errors there were
// before the statement.
func (p *Parser) misspelledKeyword(first token.Token, errors int) {
	if first.Type != token.IDENT || len(p.errors) != errors+1 || p.errorLine != first.Line {
		return
	}
	if keyword := suggestKeyword(first.Literal); keyword != "" {
		p.errors[errors] = at(first, fmt.Sprintf("syntax error: %q is not a keyword; did you mean %q?",
			first.Literal, keyword))
	}
}

// isReserved reports whether tok is a reserved word, which can't be a name.
func isReserved(tok token.Token) bool {
	return tok.Type != token.IDENT && token.LookupIdent(tok.Literal) == tok.Type
}

// expectName moves to the name after keyword, like the x of "let x". When it
// is missing, the error says why: a reserved word can't be a name, and a
// let needs one before its '='.
func (p *Parser) expectName(keyword string) bool {
	switch {
	case p.peekTokenIs(token.IDENT):
		p.nextToken()
		return true
	case isReserved(p.peekToken):
		p.errorAt(p.peekToken, "%q is a reserved word, it can't be a name", p.peekToken.Literal)
	case keyword == "let" && p.peekTokenIs(token.ASSIGN):
		p.errorAt(p.peekToken, "let needs a name before the \"=\", like let x = 1")
	case keyword == "let":
		p.errorAt(p.peekToken, "expected a name after let, got %s", describeToken(p.peekToken))
	default:
		return p.expectPeek(token.IDENT)
	}
	return false
}

// assignInCondition reports, as an error, a condition that is a plain
// assignment, like the x = 1 of "if (x = 1)", which was surely meant as ==.
func (p *Parser) assignInCondition(condition ast.Expression) bool {
	assign, ok := condition.(*ast.AssignExpression)
	if !ok || assign.Operator != "" {
		return false
	}
	p.errorAt(assign.Token, "\"=\" in a condition gives %s a value; did you mean \"==\"?", assign.Left.String())
	return true
}
