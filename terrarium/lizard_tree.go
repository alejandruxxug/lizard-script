package terrarium

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// lizard: the language itself, as data. A program reads code as tokens or
// as a syntax tree (an editor colors and checks it), runs it in a sandbox,
// and reads the doc comments of its functions. Positions are lines and
// columns from 1, counted in characters; an end is just after the last
// character.

func lizardReflectModule() *object.Hash {
	return arg.Module("lizard", map[string]object.BuiltinFunction{
		"tokens": lizardTokens,
		"parse":  lizardParse,
		"run":    lizardRun,
		"docs":   lizardDocs,
		"check":  lizardCheck,
	})
}

// source is a text cut in lines of characters, to read the text between
// two positions.
type source [][]rune

func newSource(text string) source {
	lines := strings.Split(text, "\n")
	out := make(source, len(lines))
	for i, line := range lines {
		out[i] = []rune(line)
	}
	return out
}

// text is what is written from (line, column) up to (endLine, endColumn).
func (s source) text(line, column, endLine, endColumn int) string {
	var parts []string
	for l := line; l <= endLine && l <= len(s); l++ {
		runes := s[l-1]
		from, to := 0, len(runes)
		if l == line {
			from = min(max(column-1, 0), len(runes))
		}
		if l == endLine {
			to = min(max(endColumn-1, 0), len(runes))
		}
		parts = append(parts, string(runes[from:max(from, to)]))
	}
	return strings.Join(parts, "\n")
}

// positions puts the start and end of a span in h.
func positions(h *object.Hash, line, column, endLine, endColumn int) {
	arg.Set(h, "line", arg.Int64(int64(line)))
	arg.Set(h, "column", arg.Int64(int64(column)))
	arg.Set(h, "end_line", arg.Int64(int64(endLine)))
	arg.Set(h, "end_column", arg.Int64(int64(endColumn)))
}

// codeArgs reads the code argument and the optional name after it.
func codeArgs(args []object.Object) (code, name string, err *object.Error) {
	if err := arg.Count(args, 1, 2); err != nil {
		return "", "", err
	}
	if code, err = arg.String(args, 0, "code"); err != nil {
		return "", "", err
	}
	if len(args) == 2 {
		if name, err = arg.String(args, 1, "name"); err != nil {
			return "", "", err
		}
	}
	return code, name, nil
}

// lizard.tokens(code, options = {}) is the tokens of code, comments
// included, in order: each one {"kind", "type", "text", "value", "line",
// "column", "end_line", "end_column"}. kind groups them for coloring: name,
// keyword, number, string, color, operator, punctuation, comment or error
// (what the lexer rejected). type is the exact token, like "IDENT", "LET"
// or "+="; text is what is written; value is what it means (a string
// without quotes and escapes, 45deg in radians).
//
// With the option {"columns": true} it is one hash of those keys instead,
// each an array with one element per token: no hash per token, for a host
// that walks the tokens of a big text.
func lizardTokens(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	code, err := arg.String(args, 0, "code")
	if err != nil {
		return err
	}
	columns := false
	if len(args) == 2 {
		if columns, err = tokenOptions(args); err != nil {
			return err
		}
	}
	l := lexer.New(code)
	l.KeepComments()
	texts := newTokenTexts(code)
	if columns {
		return tokenColumns(l, texts)
	}
	out := &object.Array{}
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		h := object.NewHash()
		arg.Set(h, "kind", arg.Str(tokenKind(tok.Type)))
		arg.Set(h, "type", arg.Str(string(tok.Type)))
		arg.Set(h, "text", arg.Str(texts.of(tok)))
		arg.Set(h, "value", arg.Str(tok.Literal))
		positions(h, tok.Line, tok.Column, tok.EndLine, tok.EndColumn)
		out.Elements = append(out.Elements, h)
	}
	return out
}

// tokenOptions reads the options of lizard.tokens: whether it gives columns.
func tokenOptions(args []object.Object) (bool, *object.Error) {
	options, err := arg.Hash(args, 1, "options")
	if err != nil {
		return false, err
	}
	columns := false
	options.Each(func(pair object.HashPair) {
		if err != nil {
			return
		}
		key := pair.Key.Inspect()
		b, ok := pair.Value.(*object.Boolean)
		switch {
		case key != "columns":
			err = arg.Errorf("unknown option %s, the only option is columns", object.Quote(key))
		case !ok:
			err = arg.Errorf("the option columns must be true or false, got %s", arg.Name(pair.Value))
		default:
			columns = b.Value
		}
	})
	return columns, err
}

// tokenColumns is the columns form of lizard.tokens: a hash of arrays. The
// strings that repeat (each kind and type, the text of a keyword, operator
// or punctuation) are one object each, and a value written as it means
// (a name, a number like 12) is the same object as its text.
func tokenColumns(l *lexer.Lexer, texts *tokenTexts) object.Object {
	names := []string{"kind", "type", "text", "value", "line", "column", "end_line", "end_column"}
	cols := make([][]object.Object, len(names))
	shared := map[string]object.Object{}
	str := func(s string) object.Object {
		if o, ok := shared[s]; ok {
			return o
		}
		o := arg.Str(s)
		shared[s] = o
		return o
	}
	// The tokens first, so each column is made at its size.
	var toks []token.Token
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		toks = append(toks, tok)
	}
	for i := range cols {
		cols[i] = make([]object.Object, len(toks))
	}
	for i, tok := range toks {
		kind := tokenKind(tok.Type)
		var text object.Object
		switch kind {
		case "keyword", "operator", "punctuation":
			text = str(texts.of(tok))
		default:
			text = arg.Str(texts.of(tok))
		}
		value := text
		if tok.Literal != text.(*object.String).Value {
			value = arg.Str(tok.Literal)
		}
		cols[0][i], cols[1][i], cols[2][i], cols[3][i] = str(kind), str(string(tok.Type)), text, value
		cols[4][i], cols[5][i] = object.IntegerOf(int64(tok.Line)), object.IntegerOf(int64(tok.Column))
		cols[6][i], cols[7][i] = object.IntegerOf(int64(tok.EndLine)), object.IntegerOf(int64(tok.EndColumn))
	}
	out := object.NewHash()
	for i, name := range names {
		arg.Set(out, name, &object.Array{Elements: cols[i]})
	}
	return out
}

// tokenTexts gives the text of each token of a code, in order. The text is
// a part of the code itself, found by walking it once from token to token;
// a code that isn't valid UTF-8 is cut in lines of characters instead (see
// source), so each wrong byte reads as one U+FFFD, as the lexer counts it.
type tokenTexts struct {
	code         string
	lines        source // when code isn't valid UTF-8
	at           int    // the byte where (line, column) is
	line, column int
}

func newTokenTexts(code string) *tokenTexts {
	t := &tokenTexts{code: code, line: 1, column: 1}
	if !utf8.ValidString(code) {
		t.lines = newSource(code)
	}
	return t
}

// of is the text of tok, which comes after the tokens asked for before it.
func (t *tokenTexts) of(tok token.Token) string {
	if t.lines != nil {
		return t.lines.text(tok.Line, tok.Column, tok.EndLine, tok.EndColumn)
	}
	from := t.offset(tok.Line, tok.Column)
	to := max(from, t.offset(tok.EndLine, tok.EndColumn))
	return t.code[from:to]
}

// offset is the byte of the code at (line, column), the end of the line
// for a column past it, the end of the code for a line past it.
func (t *tokenTexts) offset(line, column int) int {
	if line < t.line || line == t.line && column < t.column {
		t.at, t.line, t.column = 0, 1, 1
	}
	for t.at < len(t.code) && (t.line < line || t.column < column) {
		r, size := utf8.DecodeRuneInString(t.code[t.at:])
		if r == '\n' {
			if t.line == line {
				break
			}
			t.line, t.column = t.line+1, 1
		} else {
			t.column++
		}
		t.at += size
	}
	return t.at
}

// tokenKind is the group of a token type, for lizard.tokens.
func tokenKind(t token.TokenType) string {
	switch t {
	case token.IDENT:
		return "name"
	case token.INT, token.FLOAT:
		return "number"
	case token.STRING:
		return "string"
	case token.COLOR:
		return "color"
	case token.COMMENT:
		return "comment"
	case token.ILLEGAL:
		return "error"
	case token.COMMA, token.SEMICOLON, token.COLON, token.DOT,
		token.LPAREN, token.RPAREN, token.LBRACE, token.RBRACE, token.LBRACKET, token.RBRACKET:
		return "punctuation"
	}
	if token.LookupIdent(strings.ToLower(string(t))) == t {
		return "keyword"
	}
	return "operator"
}

// lizard.parse(code, name = "") is the syntax tree of code:
// {"program": node, "errors": [...]}. Each node is a hash with its "kind"
// and its positions, and the fields of that kind (see TERRARIUM.md). Each
// error is {"kind": "syntax" or "lexical", "message", "line", "column"}.
// With errors the program has only the statements that parsed. The name is
// the source name of the tokens; it is in no message.
func lizardParse(args ...object.Object) object.Object {
	code, name, err := codeArgs(args)
	if err != nil {
		return err
	}
	p := parser.New(lexer.NewNamed(code, name))
	program := p.ParseProgram()
	out := object.NewHash()
	arg.Set(out, "program", treeNode(program))
	arg.Set(out, "errors", parseErrors(p.Errors()))
	return out
}

// positionPrefix is the "[line L, column C] kind error: " of a message.
var positionPrefix = regexp.MustCompile(`^\[line (\d+), column (\d+)\] (\w+) error: `)

// parseErrors turns the parser's messages into hashes.
func parseErrors(messages []string) *object.Array {
	out := &object.Array{}
	for _, message := range messages {
		h := object.NewHash()
		line, column, kind := 0, 0, "syntax"
		if m := positionPrefix.FindStringSubmatch(message); m != nil {
			line, _ = strconv.Atoi(m[1])
			column, _ = strconv.Atoi(m[2])
			kind = m[3]
			message = message[len(m[0]):]
		}
		arg.Set(h, "kind", arg.Str(kind))
		arg.Set(h, "message", arg.Str(message))
		arg.Set(h, "line", arg.Int64(int64(line)))
		arg.Set(h, "column", arg.Int64(int64(column)))
		out.Elements = append(out.Elements, h)
	}
	return out
}

// treeNode is the hash of a node of the tree, or null for no node.
func treeNode(node ast.Node) object.Object {
	if node == nil || isNil(node) {
		return arg.Null
	}
	h := object.NewHash()
	set := func(key string, value object.Object) { arg.Set(h, key, value) }
	kind := func(k string) { set("kind", arg.Str(k)) }

	switch n := node.(type) {
	case *ast.Program:
		kind("program")
		set("statements", statementList(n.Statements))
	case *ast.LetStatement:
		kind("let")
		set("name", treeNode(n.Name))
		set("value", treeNode(n.Value))
		// function name() {} is a let; it says how it was written.
		literal, isFunction := n.Value.(*ast.FunctionLiteral)
		set("declaration", arg.Bool(isFunction && literal.Token.Line == n.Token.Line && literal.Token.Column == n.Token.Column))
	case *ast.ReturnStatement:
		kind("return")
		set("value", treeNode(n.ReturnValue))
	case *ast.BreakStatement:
		kind("break")
	case *ast.ContinueStatement:
		kind("continue")
	case *ast.TryStatement:
		kind("try")
		set("body", treeNode(n.Body))
		set("error_name", treeNode(n.Name))
		set("catch", treeNode(n.Catch))
	case *ast.ExpressionStatement:
		kind("expression")
		set("expression", treeNode(n.Expression))
	case *ast.BlockStatement:
		kind("block")
		set("statements", statementList(n.Statements))
	case *ast.LayerStatement:
		kind("layer")
		set("name", arg.Str(n.Name))
		set("body", treeNode(n.Body))
	case *ast.UseStatement:
		kind("use")
		set("path", treeNode(n.Path))
		set("as", treeNode(n.Name))
	case *ast.Identifier:
		kind("name")
		set("value", arg.Str(n.Value))
	case *ast.IntegerLiteral:
		kind("integer")
		set("value", arg.Int64(n.Value))
	case *ast.FloatLiteral:
		kind("float")
		set("value", arg.Float(n.Value))
	case *ast.StringLiteral:
		kind("string")
		set("value", arg.Str(n.Value))
	case *ast.ColorLiteral:
		kind("color")
		set("value", &object.Color{R: n.R, G: n.G, B: n.B, A: n.A})
	case *ast.Boolean:
		kind("boolean")
		set("value", arg.Bool(n.Value))
	case *ast.PointLiteral:
		kind("point")
		set("coords", expressionList(n.Coords))
	case *ast.PrefixExpression:
		kind("prefix")
		set("operator", arg.Str(n.Operator))
		set("right", treeNode(n.Right))
	case *ast.InfixExpression:
		kind("infix")
		set("operator", arg.Str(n.Operator))
		set("left", treeNode(n.Left))
		set("right", treeNode(n.Right))
	case *ast.AssignExpression:
		kind("assign")
		set("operator", arg.Str(n.Operator+"="))
		set("target", treeNode(n.Left))
		set("value", treeNode(n.Value))
	case *ast.IfExpression:
		kind("if")
		set("condition", treeNode(n.Condition))
		set("consequence", treeNode(n.Consequence))
		set("alternative", treeNode(n.Alternative))
	case *ast.WhileExpression:
		kind("while")
		set("condition", treeNode(n.Condition))
		set("body", treeNode(n.Body))
	case *ast.ForExpression:
		kind("for")
		set("init", treeNode(n.Init))
		set("condition", treeNode(n.Condition))
		set("update", treeNode(n.Update))
		set("body", treeNode(n.Body))
	case *ast.FunctionLiteral:
		kind("function")
		params := &object.Array{}
		defaults := &object.Array{}
		for i, p := range n.Parameters {
			params.Elements = append(params.Elements, treeNode(p))
			var value ast.Node
			if i < len(n.Defaults) && n.Defaults[i] != nil {
				value = n.Defaults[i]
			}
			defaults.Elements = append(defaults.Elements, treeNode(value))
		}
		set("params", params)
		set("defaults", defaults)
		set("body", treeNode(n.Body))
	case *ast.CallExpression:
		kind("call")
		set("callee", treeNode(n.Function))
		set("arguments", expressionList(n.Arguments))
	case *ast.ArrayLiteral:
		kind("array")
		set("elements", expressionList(n.Elements))
	case *ast.IndexExpression:
		if n.Token.Type == token.DOT {
			kind("dot")
			set("left", treeNode(n.Left))
			set("name", treeNode(n.Index))
		} else {
			kind("index")
			set("left", treeNode(n.Left))
			set("index", treeNode(n.Index))
		}
	case *ast.HashLiteral:
		kind("hash")
		pairs := &object.Array{}
		for _, pair := range n.Pairs {
			p := object.NewHash()
			arg.Set(p, "key", treeNode(pair.Key))
			arg.Set(p, "value", treeNode(pair.Value))
			pairs.Elements = append(pairs.Elements, p)
		}
		set("pairs", pairs)
	default:
		kind(strings.ToLower(strings.TrimPrefix(fmt.Sprintf("%T", node), "*ast.")))
	}
	span := node.Range()
	positions(h, span.StartLine, span.StartColumn, span.EndLine, span.EndColumn)
	return h
}

func statementList(statements []ast.Statement) *object.Array {
	out := &object.Array{Elements: make([]object.Object, len(statements))}
	for i, s := range statements {
		out.Elements[i] = treeNode(s)
	}
	return out
}

func expressionList(expressions []ast.Expression) *object.Array {
	out := &object.Array{Elements: make([]object.Object, len(expressions))}
	for i, e := range expressions {
		out.Elements[i] = treeNode(e)
	}
	return out
}

// isNil reports whether node holds a nil pointer, like a missing else.
func isNil(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.BlockStatement:
		return n == nil
	case *ast.Identifier:
		return n == nil
	case *ast.StringLiteral:
		return n == nil
	}
	return false
}
