package format

import (
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
)

// What the layout needs to know from the syntax tree, by the position of a
// token: the tokens alone can't tell a prefix '-' from a binary one, or a
// block's '{' from a hash's.

// pos is where a token starts.
type pos struct{ line, column int }

// before reports whether a comes before b in the source.
func (a pos) before(b pos) bool {
	return a.line < b.line || a.line == b.line && a.column < b.column
}

func startOf(span *ast.Span) pos { return pos{span.StartLine, span.StartColumn} }

// lastChar is where the last character of span is, the '}' of a block or a
// hash.
func lastChar(span *ast.Span) pos { return pos{span.EndLine, span.EndColumn - 1} }

func tokenPos(tok token.Token) pos { return pos{tok.Line, tok.Column} }

// layerBody is a layer whose body is not empty: it is indented like a
// block from the token after its name to the end of its last statement.
type layerBody struct {
	column int // of the word layer
	end    pos // just after the last statement of the body
}

// facts are the positions the layout looks up.
type facts struct {
	prefix     map[pos]bool // prefix operators
	calls      map[pos]bool // the '(' of calls
	indexes    map[pos]bool // the '[' of indexes
	blocks     map[pos]bool // the '{' of blocks
	hashes     map[pos]bool // the '{' of hashes
	statements map[pos]bool // the first token of each statement
	breaks     map[pos]bool // tokens that start a line: pairs of a hash over several lines, its '}', layers
	joins      map[pos]bool // tokens that go on the line before: the '}' of an empty hash
	tryEnds    map[pos]bool // the '}' of a try's body: the catch after it goes on its line
	layers     map[pos]layerBody
}

// collect reads the facts of program.
func collect(program *ast.Program) *facts {
	f := &facts{
		prefix: map[pos]bool{}, calls: map[pos]bool{}, indexes: map[pos]bool{},
		blocks: map[pos]bool{}, hashes: map[pos]bool{}, statements: map[pos]bool{},
		breaks: map[pos]bool{}, joins: map[pos]bool{}, tryEnds: map[pos]bool{}, layers: map[pos]layerBody{},
	}
	ast.Inspect(program, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Program:
			f.addStatements(n.Statements)
		case *ast.BlockStatement:
			if n.Token.Type == token.LBRACE {
				f.blocks[tokenPos(n.Token)] = true
			}
			f.addStatements(n.Statements)
		case *ast.PrefixExpression:
			f.prefix[tokenPos(n.Token)] = true
		case *ast.CallExpression:
			f.calls[tokenPos(n.Token)] = true
		case *ast.IndexExpression:
			if n.Token.Type == token.LBRACKET {
				f.indexes[tokenPos(n.Token)] = true
			}
		case *ast.HashLiteral:
			f.addHash(n)
		case *ast.TryStatement:
			f.tryEnds[lastChar(n.Body.Range())] = true
		case *ast.LayerStatement:
			f.breaks[tokenPos(n.Token)] = true
			if body := n.Body.Statements; len(body) > 0 {
				last := body[len(body)-1].Range()
				f.layers[tokenPos(n.Token)] = layerBody{column: n.Token.Column, end: pos{last.EndLine, last.EndColumn}}
			}
		}
		return true
	})
	return f
}

func (f *facts) addStatements(statements []ast.Statement) {
	for _, s := range statements {
		f.statements[startOf(s.Range())] = true
	}
}

// addHash notes a hash's '{', and for one that spans lines the breaks
// before each pair and before its '}'.
func (f *facts) addHash(h *ast.HashLiteral) {
	f.hashes[tokenPos(h.Token)] = true
	span := h.Range()
	if span.StartLine == span.EndLine {
		return
	}
	if len(h.Pairs) == 0 {
		f.joins[lastChar(span)] = true
		return
	}
	for _, pair := range h.Pairs {
		f.breaks[startOf(pair.Key.Range())] = true
	}
	f.breaks[lastChar(span)] = true
}
