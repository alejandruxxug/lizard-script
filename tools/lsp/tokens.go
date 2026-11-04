package lsp

import (
	"encoding/json"
	"fmt"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/tools/format"
	"math"
	"strconv"
	"strings"
)

// The requests that read the tokens of a file: semantic tokens, colors and
// formatting.

// The indexes of tokenTypes and the bits of tokenModifiers.
const (
	tokKeyword = iota
	tokFunction
	tokVariable
	tokParameter
	tokProperty
	tokNamespace
	tokNumber
	tokString
	tokComment
	tokOperator
	tokColor
	tokPoint
	tokLabel
)

const (
	modDeclaration = 1 << iota
	modDefaultLibrary
	modReadonly
)

// contextual are the words that are keywords only where the parser reads
// them so: a name that is not one of the file's names is the keyword.
var contextual = map[string]bool{"layer": true, "use": true, "as": true, "try": true, "catch": true, "continue": true}

// semanticTokens marks every token by what it is: a function, a parameter,
// a module, a builtin, a color, a point. An editor colors the names with
// it, which a grammar can't tell apart.
func (s *Server) semanticTokens(params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	d, err := s.document(p.TextDocument.URI)
	if err != nil {
		return nil, err
	}
	ix := s.indexOf(d)

	names := map[pos]*occurrence{}
	for _, o := range ix.occurrences {
		if o.name != "" {
			names[pos{o.span.StartLine, o.span.StartColumn}] = o
		}
	}
	points := map[pos]bool{}
	ast.Inspect(ix.program, func(node ast.Node) bool {
		if pt, ok := node.(*ast.PointLiteral); ok {
			span := pt.Range()
			points[pos{span.StartLine, span.StartColumn}] = true
			points[pos{span.EndLine, span.EndColumn - 1}] = true
		}
		return true
	})

	var data []int
	last := Position{}
	emit := func(line, column, endColumn, kind, modifiers int) {
		start, end := d.toPosition(line, column), d.toPosition(line, endColumn)
		if end.Character <= start.Character {
			return
		}
		deltaStart := start.Character
		if start.Line == last.Line {
			deltaStart -= last.Character
		}
		data = append(data, start.Line-last.Line, deltaStart, end.Character-start.Character, kind, modifiers)
		last = start
	}

	l := lexer.New(d.text)
	l.KeepComments()
	afterLayer := false
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		kind, modifiers := tokenKind(ix, tok, names[pos{tok.Line, tok.Column}], points, afterLayer)
		afterLayer = tok.Type == token.IDENT && tok.Literal == "layer" && kind == tokKeyword
		if kind < 0 {
			continue
		}
		if tok.EndLine == tok.Line {
			emit(tok.Line, tok.Column, tok.EndColumn, kind, modifiers)
			continue
		}
		// A string over several lines is marked line by line.
		for line := tok.Line; line <= tok.EndLine; line++ {
			from, to := 1, len([]rune(d.lines[line-1]))+1
			if line == tok.Line {
				from = tok.Column
			}
			if line == tok.EndLine {
				to = tok.EndColumn
			}
			emit(line, from, to, kind, modifiers)
		}
	}
	if data == nil {
		data = []int{}
	}
	return SemanticTokens{Data: data}, nil
}

// tokenKind is the semantic token type and modifiers of tok, or -1 for a
// token left to the grammar (operators and punctuation).
func tokenKind(ix *index, tok token.Token, o *occurrence, points map[pos]bool, afterLayer bool) (int, int) {
	switch tok.Type {
	case token.COMMENT:
		return tokComment, 0
	case token.STRING:
		return tokString, 0
	case token.INT, token.FLOAT:
		return tokNumber, 0
	case token.COLOR:
		return tokColor, 0
	case token.LPAREN, token.RPAREN:
		if points[pos{tok.Line, tok.Column}] {
			return tokPoint, 0
		}
		return -1, 0
	case token.IDENT:
		if o == nil {
			switch {
			case afterLayer:
				return tokLabel, 0
			case contextual[tok.Literal]:
				return tokKeyword, 0
			}
			return tokVariable, 0
		}
		return nameKind(ix, o)
	}
	if token.LookupIdent(tok.Literal) != token.IDENT {
		return tokKeyword, 0
	}
	return -1, 0
}

// nameKind is the semantic token of a name, from what it means.
func nameKind(ix *index, o *occurrence) (int, int) {
	modifiers := 0
	if o.decl {
		modifiers |= modDeclaration
	}
	t := o.target
	switch {
	case t.global:
		modifiers |= modDefaultLibrary
		head, _, isMember := strings.Cut(t.path, ".")
		if !isMember && len(ix.globals[head]) > 0 {
			return tokNamespace, modifiers
		}
		if t.path == "null" {
			return tokKeyword, 0
		}
		if doc, ok := builtinDocs[t.path]; ok && len(doc.Signatures) == 0 {
			return tokProperty, modifiers | modReadonly // math.pi
		}
		return tokFunction, modifiers
	case t.sym != nil && t.path == "" && t.sym.kind == paramDecl, t.sym != nil && t.path == "" && t.sym.kind == catchDecl:
		return tokParameter, modifiers
	case t.sym != nil && t.path == "" && t.sym.kind == useDecl:
		return tokNamespace, modifiers
	}
	switch valueOf(t).(type) {
	case *ast.FunctionLiteral:
		return tokFunction, modifiers
	case *ast.HashLiteral:
		if t.path == "" && t.module == nil {
			return tokVariable, modifiers
		}
	}
	if o.dot {
		return tokProperty, modifiers
	}
	return tokVariable, modifiers
}

// documentColors are the colors written in a file, for the editor to show
// a swatch and a color picker at each.
func (s *Server) documentColors(params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	d, err := s.document(p.TextDocument.URI)
	if err != nil {
		return nil, err
	}
	out := []ColorInformation{}
	l := lexer.New(d.text)
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if tok.Type != token.COLOR {
			continue
		}
		if c, ok := parseColor(tok.Literal); ok {
			out = append(out, ColorInformation{Range: d.toRange(tok.Line, tok.Column, tok.EndLine, tok.EndColumn), Color: c})
		}
	}
	return out, nil
}

// colorPresentations are how a color picked in the editor is written back:
// as a literal, #rrggbb (or #rrggbbaa with alpha), or as rgba(...).
func (s *Server) colorPresentations(params json.RawMessage) (any, error) {
	var p colorPresentationParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	r, g, b, a := channel(p.Color.Red), channel(p.Color.Green), channel(p.Color.Blue), channel(p.Color.Alpha)
	literal := fmt.Sprintf("#%02x%02x%02x", r, g, b)
	call := fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	if a != 255 {
		literal += fmt.Sprintf("%02x", a)
		call = fmt.Sprintf("rgba(%d, %d, %d, %d)", r, g, b, a)
	}
	out := []ColorPresentation{}
	for _, label := range []string{literal, call} {
		out = append(out, ColorPresentation{Label: label, TextEdit: &TextEdit{Range: p.Range, NewText: label}})
	}
	return out, nil
}

// channel is a channel from 0 to 1 as 0 to 255.
func channel(v float64) int {
	return int(math.Round(math.Max(0, math.Min(1, v)) * 255))
}

// parseColor reads a color literal: #rgb, #rrggbb or #rrggbbaa.
func parseColor(literal string) (Color, bool) {
	hex := strings.TrimPrefix(literal, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) == 6 {
		hex += "ff"
	}
	if len(hex) != 8 {
		return Color{}, false
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return Color{}, false
	}
	part := func(shift uint) float64 { return float64(value>>shift&0xff) / 255 }
	return Color{Red: part(24), Green: part(16), Blue: part(8), Alpha: part(0)}, true
}

// formatting lays the file out with package format. A file with syntax
// errors is left as it is: its diagnostics say why.
func (s *Server) formatting(params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	d, err := s.document(p.TextDocument.URI)
	if err != nil {
		return nil, err
	}
	formatted, err := format.Source(d.text)
	if err != nil || formatted == d.text {
		return []TextEdit{}, nil
	}
	return []TextEdit{{Range: Range{End: d.end()}, NewText: formatted}}, nil
}
