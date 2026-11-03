package lsp

import (
	"encoding/json"
	"lizardscript/ast"
	"lizardscript/lexer"
	"lizardscript/token"
	"strings"
)

// documentSymbols are the outline of a file: its functions, its top-level
// lets and its layers, as a tree (what a function or a layer holds is
// inside it).
func (s *Server) documentSymbols(params json.RawMessage) (any, error) {
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
	return ix.outline(ix.program.Statements, true), nil
}

// outline is the symbols of a list of statements. Lets count at the top
// (top is true) and in layers; inside a function only functions and layers
// do.
func (ix *index) outline(list []ast.Statement, top bool) []DocumentSymbol {
	out := []DocumentSymbol{}
	for _, st := range list {
		switch st := st.(type) {
		case *ast.LetStatement:
			if f, ok := st.Value.(*ast.FunctionLiteral); ok {
				sym := ix.documentSymbol(st.Name.Value, symbolFunction, st, st.Name.Range())
				sym.Detail = ix.signature("", f)
				if f.Body != nil {
					sym.Children = ix.outline(f.Body.Statements, false)
				}
				out = append(out, sym)
			} else if top {
				kind := symbolVariable
				if _, ok := st.Value.(*ast.HashLiteral); ok {
					kind = symbolModule
				}
				out = append(out, ix.documentSymbol(st.Name.Value, kind, st, st.Name.Range()))
			}
		case *ast.LayerStatement:
			sym := ix.documentSymbol(st.Name, symbolNamespace, st, ix.layerName(st))
			sym.Detail = "layer"
			sym.Children = ix.outline(st.Body.Statements, true)
			out = append(out, sym)
		case *ast.UseStatement:
			if top && st.Name != nil {
				sym := ix.documentSymbol(st.Name.Value, symbolModule, st, st.Name.Range())
				sym.Detail = "use " + st.Path.String()
				out = append(out, sym)
			}
		case *ast.ExpressionStatement:
			// The functions and layers in the blocks of an if, a loop or a try.
			for _, b := range blocksOf(st.Expression) {
				out = append(out, ix.outline(b.Statements, false)...)
			}
		case *ast.TryStatement:
			out = append(out, ix.outline(st.Body.Statements, false)...)
			out = append(out, ix.outline(st.Catch.Statements, false)...)
		}
	}
	return out
}

// blocksOf are the blocks of an if, a while or a for.
func blocksOf(e ast.Expression) []*ast.BlockStatement {
	var out []*ast.BlockStatement
	switch e := e.(type) {
	case *ast.IfExpression:
		out = append(out, e.Consequence)
		if e.Alternative != nil {
			out = append(out, e.Alternative)
		}
	case *ast.WhileExpression:
		out = append(out, e.Body)
	case *ast.ForExpression:
		out = append(out, e.Body)
	}
	return out
}

func (ix *index) documentSymbol(name string, kind int, node ast.Node, selection *ast.Span) DocumentSymbol {
	if name == "" {
		name = "(no name)"
	}
	return DocumentSymbol{Name: name, Kind: kind, Range: spanRange(ix.lines, node.Range()), SelectionRange: spanRange(ix.lines, selection)}
}

// layerName is the span of the name of a layer: the token after "layer".
func (ix *index) layerName(st *ast.LayerStatement) *ast.Span {
	span := st.Range()
	l := lexer.New(ix.source(span))
	l.NextToken() // layer
	name := l.NextToken()
	if name.Type == token.EOF {
		return span
	}
	return &ast.Span{
		StartLine: span.StartLine + name.Line - 1, StartColumn: column(span, name.Line, name.Column),
		EndLine: span.StartLine + name.EndLine - 1, EndColumn: column(span, name.EndLine, name.EndColumn),
	}
}

// column is a column of a text that starts at span, in the source.
func column(span *ast.Span, line, col int) int {
	if line == 1 {
		return span.StartColumn + col - 1
	}
	return col
}

// workspaceSymbols are the top-level functions, lets and layers of every
// file of the workspace whose name has the query's letters in order.
func (s *Server) workspaceSymbols(params json.RawMessage) (any, error) {
	var p struct {
		Query string `json:"query"`
	}
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	out := []SymbolInformation{}
	for _, ix := range s.workspaceIndexes() {
		uri := s.uriOf(ix)
		var walk func(list []DocumentSymbol, container string)
		walk = func(list []DocumentSymbol, container string) {
			for _, sym := range list {
				if fuzzy(sym.Name, p.Query) {
					out = append(out, SymbolInformation{Name: sym.Name, Kind: sym.Kind, ContainerName: container,
						Location: Location{URI: uri, Range: sym.SelectionRange}})
				}
				walk(sym.Children, sym.Name)
			}
		}
		walk(ix.outline(ix.program.Statements, true), "")
	}
	return out, nil
}

// fuzzy reports whether the letters of query are in name, in order, any
// case.
func fuzzy(name, query string) bool {
	name, query = strings.ToLower(name), strings.ToLower(query)
	for _, c := range query {
		i := strings.IndexRune(name, c)
		if i < 0 {
			return false
		}
		name = name[i+1:]
	}
	return true
}

// foldingRanges are the parts of a file that fold: blocks, layers, hashes,
// arrays and calls that span lines, and runs of comment lines.
func (s *Server) foldingRanges(params json.RawMessage) (any, error) {
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
	out := []FoldingRange{}
	ast.Inspect(ix.program, func(node ast.Node) bool {
		if b, ok := node.(*ast.BlockStatement); ok && b.Token.Type == token.IDENT {
			return true // a layer's body, which folds with the layer
		}
		switch node.(type) {
		case *ast.BlockStatement, *ast.LayerStatement, *ast.HashLiteral, *ast.ArrayLiteral, *ast.CallExpression:
			span := node.Range()
			end := span.EndLine
			if _, ok := node.(*ast.LayerStatement); !ok {
				end-- // the line of the closing bracket stays in view
			}
			if end > span.StartLine {
				out = append(out, FoldingRange{StartLine: span.StartLine - 1, EndLine: end - 1})
			}
		}
		return true
	})
	start := -1
	for i, line := range append(d.lines, "") {
		if strings.HasPrefix(strings.TrimSpace(line), "-~>") {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 && i-1 > start {
			out = append(out, FoldingRange{StartLine: start, EndLine: i - 1, Kind: "comment"})
		}
		start = -1
	}
	return out, nil
}
