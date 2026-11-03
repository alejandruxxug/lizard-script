package lsp

import (
	"encoding/json"
	"lizardscript/ast"
	"lizardscript/lexer"
	"lizardscript/token"
	"strings"
)

// hover says what the name or the literal under the cursor is: a function's
// signatures and doc comment, a builtin's documentation, a let's value, or
// a literal's type and worth (90deg = 1.5708 rad, a color's channels).
func (s *Server) hover(params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	_, ix, at, err := s.at(p)
	if err != nil {
		return nil, err
	}
	if o := ix.occurrenceAt(at); o != nil && o.target.known() {
		in := describe(ix, o.target)
		r := spanRange(ix.lines, o.span)
		return Hover{Contents: MarkupContent{Kind: "markdown", Value: hoverText(in)}, Range: &r}, nil
	}
	if lit := literalAt(ix.program, at); lit != nil {
		r := spanRange(ix.lines, lit.Range())
		return Hover{Contents: MarkupContent{Kind: "markdown", Value: literalInfo(ix, lit)}, Range: &r}, nil
	}
	return nil, nil
}

// hoverText is the Markdown of a hover: the signatures, or the detail, as
// code, then the doc.
func hoverText(in info) string {
	lines := []string{in.detail}
	if len(in.signatures) > 0 {
		lines = nil
		for _, sig := range in.signatures {
			lines = append(lines, sig.label)
		}
	}
	return joinParts("```lizard\n"+strings.Join(lines, "\n")+"\n```", in.doc)
}

// literalAt is the innermost literal of program at p, or nil.
func literalAt(program *ast.Program, p pos) ast.Expression {
	var found ast.Expression
	ast.Inspect(program, func(node ast.Node) bool {
		if node == nil || !inside(p, node.Range()) {
			return node == program
		}
		switch n := node.(type) {
		case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.ColorLiteral, *ast.PointLiteral, *ast.StringLiteral, *ast.Boolean:
			found = n.(ast.Expression)
		}
		return true
	})
	return found
}

// signatureHelp shows the signature of the call the cursor is in, with the
// parameter it is at.
func (s *Server) signatureHelp(params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	d, ix, at, err := s.at(p)
	if err != nil {
		return nil, err
	}
	chain, argument, head, ok := callAt(d.text[:d.offset(p.Position)])
	if !ok {
		return nil, nil
	}
	t := ix.resolve(chain[0], head, ix.scopeAt(at))
	for _, name := range chain[1:] {
		t = t.member(name)
	}
	in := describe(ix, t)
	if len(in.signatures) == 0 {
		return nil, nil
	}
	help := SignatureHelp{ActiveParameter: argument, ActiveSignature: -1}
	for i, sig := range in.signatures {
		help.Signatures = append(help.Signatures, signatureInformation(sig, in.doc))
		if help.ActiveSignature < 0 && len(sig.params) > argument {
			help.ActiveSignature = i // the first one with that many parameters
		}
	}
	if help.ActiveSignature < 0 {
		help.ActiveSignature = len(in.signatures) - 1
	}
	return help, nil
}

// signatureInformation is sig for the protocol: each parameter marked by
// where it is in the label.
func signatureInformation(sig signature, doc string) SignatureInformation {
	info := SignatureInformation{Label: sig.label, Parameters: []ParameterInformation{}}
	if doc != "" {
		info.Documentation = &MarkupContent{Kind: "markdown", Value: doc}
	}
	from := strings.IndexByte(sig.label, '(') + 1
	for i, param := range sig.params {
		start := from + strings.Index(sig.label[from:], param)
		end := start + len(param)
		parameter := ParameterInformation{Label: [2]int{utf16Len(sig.label[:start]), utf16Len(sig.label[:end])}}
		if sig.docs[i] != "" {
			parameter.Documentation = &MarkupContent{Kind: "markdown", Value: sig.docs[i]}
		}
		info.Parameters = append(info.Parameters, parameter)
		from = end
	}
	return info
}

// callAt finds the call that the end of text is inside: the names of what
// is called (["lily2d", "circle"] for lily2d.circle(...)), how many
// arguments come before the cursor and where the first name is. It reads
// tokens, not the tree, because the code of a call being written doesn't
// parse yet.
func callAt(text string) (chain []string, argument int, head pos, ok bool) {
	var tokens []token.Token
	l := lexer.New(text)
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		tokens = append(tokens, tok)
	}
	depth, commas := 0, 0
	for i := len(tokens) - 1; i >= 0; i-- {
		switch tokens[i].Type {
		case token.RPAREN, token.RBRACKET, token.RBRACE:
			depth++
		case token.LBRACKET, token.LBRACE:
			if depth == 0 {
				return nil, 0, pos{}, false // inside an array or a hash, not the call's arguments
			}
			depth--
		case token.COMMA:
			if depth == 0 {
				commas++
			}
		case token.LPAREN:
			if depth > 0 {
				depth--
				continue
			}
			if chain, head = calledNames(tokens[:i]); chain != nil {
				return chain, commas, head, true
			}
			commas = 0 // a point or a group: the call may be around it
		}
	}
	return nil, 0, pos{}, false
}

// calledNames reads a.b.c at the end of tokens.
func calledNames(tokens []token.Token) ([]string, pos) {
	var chain []string
	var head pos
	for i := len(tokens) - 1; i >= 0; i -= 2 {
		if tokens[i].Type != token.IDENT {
			break
		}
		chain = append([]string{tokens[i].Literal}, chain...)
		head = pos{tokens[i].Line, tokens[i].Column}
		if i == 0 || tokens[i-1].Type != token.DOT {
			break
		}
	}
	return chain, head
}
