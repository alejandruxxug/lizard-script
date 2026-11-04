package lsp

import (
	"encoding/base64"
	"fmt"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"math"
	"sort"
	"strconv"
	"strings"
)

// info is what the server says about a name: in a hover, a completion item
// and the signature help.
type info struct {
	kind       int         // a completion kind
	detail     string      // one line: a signature, "let size = 10", "module flower/lily2d"
	signatures []signature // when it is a function
	doc        string      // Markdown
}

// signature is one way to call a function: its label, like
// "circle(target, center, r, style)", and its parameters.
type signature struct {
	label  string
	params []string
	docs   []string // the doc of each parameter, or ""
}

// describe says what t is. ix is the index t was found in.
func describe(ix *index, t target) info {
	switch {
	case t.sym != nil && t.path == "":
		return describeSymbol(ix, t.sym)
	case t.sym != nil:
		return describeMember(ix, t)
	case t.module != nil && t.path == "":
		return info{kind: completionModule, detail: "module " + t.module.path, doc: t.module.index.header()}
	case t.module != nil:
		head, rest, _ := strings.Cut(t.path, ".")
		if sym := t.module.index.exported(head); sym != nil {
			return describe(t.module.index, target{sym: sym, path: rest})
		}
	case t.global:
		return describeGlobal(ix.globals, t.path)
	}
	return info{}
}

// describeSymbol says what a name declared in ix is.
func describeSymbol(ix *index, sym *symbol) info {
	switch sym.kind {
	case paramDecl:
		in := info{kind: completionVariable, detail: "parameter " + sym.name}
		if let := letOf(ix, sym.fn); let != nil {
			in.detail += " of " + ix.signature(let.Name.Value, sym.fn)
			in.doc = ix.docOf(let.Range().StartLine, sym.fn).params[sym.name]
		}
		return in
	case catchDecl:
		return info{kind: completionVariable, detail: "catch (" + sym.name + ")",
			doc: "The error that stopped the `try`: a hash with `message`, `line` and `column`."}
	case useDecl:
		in := info{kind: completionModule, detail: "module " + strconv.Quote(sym.use.Path.Value)}
		if sym.module != nil {
			in.doc = sym.module.index.header()
		}
		return in
	}
	line := sym.let.Range().StartLine
	if in, ok := describeAlias(ix, sym); ok {
		return in
	}
	if f, ok := sym.value().(*ast.FunctionLiteral); ok {
		sig := functionSignature(ix, sym.name, f)
		return info{kind: completionFunction, detail: "function " + sig.label, signatures: []signature{sig},
			doc: ix.docOf(line, f).markdown(f)}
	}
	in := info{kind: completionVariable, detail: "let " + sym.name, doc: ix.docOf(line, nil).summary}
	if text, ok := shortSource(ix, sym.value()); ok {
		in.detail += " = " + text
	}
	if lit := literalInfo(ix, sym.value()); lit != "" {
		in.doc = joinParts(lit, in.doc)
	}
	if _, ok := sym.value().(*ast.HashLiteral); ok {
		in.kind = completionModule
	}
	return in
}

// describeAlias says what a let that gives another name to a function is,
// like let dot = _vec.dot: the function's signatures under the new name,
// and its doc unless the let has its own.
func describeAlias(ix *index, sym *symbol) (info, bool) {
	if !isNameChain(sym.value()) {
		return info{}, false
	}
	where, t := aliased(ix, sym)
	if !t.known() || t.sym == sym {
		return info{}, false
	}
	in := describe(where, t)
	if len(in.signatures) == 0 {
		return info{}, false
	}
	for i, sig := range in.signatures {
		_, params, _ := strings.Cut(sig.label, "(")
		in.signatures[i].label = sym.name + "(" + params
	}
	in.detail = "function " + in.signatures[0].label
	if own := ix.docOf(sym.let.Range().StartLine, nil).summary; own != "" {
		in.doc = own
	}
	return in, true
}

// aliased follows the lets whose value is another name, from sym, to the
// name that is not one: the index it is in and its target. A chain that
// goes on too long (a = b, b = a) ends in nothing.
func aliased(ix *index, sym *symbol) (*index, target) {
	t := target{sym: sym}
	for steps := 0; steps < 8; steps++ {
		if t.sym == nil || t.path != "" || !isNameChain(t.sym.value()) {
			return ix, t
		}
		next := ix.targetOf(t.sym.value(), t.sym.scope)
		if next.module != nil && next.path != "" && !strings.Contains(next.path, ".") {
			if exported := next.module.index.exported(next.path); exported != nil {
				ix, next = next.module.index, target{sym: exported}
			}
		}
		if !next.known() || next.sym == t.sym {
			return ix, t
		}
		t = next
	}
	return ix, target{}
}

// isNameChain reports whether e is a name or names with dots: f, m.f.
func isNameChain(e ast.Expression) bool {
	switch e := e.(type) {
	case *ast.Identifier:
		return true
	case *ast.IndexExpression:
		_, ok := dotName(e)
		return ok && isNameChain(e.Left)
	}
	return false
}

// describeMember says what the member t.path of a hash is.
func describeMember(ix *index, t target) info {
	value := valueOf(t)
	names := splitPath(t.path)
	name := names[len(names)-1]
	line := 0
	if key := keyOf(t); key != nil {
		line = key.Range().StartLine
	}
	if f, ok := value.(*ast.FunctionLiteral); ok {
		sig := functionSignature(ix, name, f)
		return info{kind: completionFunction, detail: "function " + sig.label, signatures: []signature{sig},
			doc: ix.docOf(line, f).markdown(f)}
	}
	in := info{kind: completionProperty, detail: t.sym.name + "." + t.path}
	if line > 0 {
		in.doc = ix.docOf(line, nil).summary
	}
	if text, ok := shortSource(ix, value); ok {
		in.detail += ": " + text
	}
	if lit := literalInfo(ix, value); lit != "" {
		in.doc = joinParts(lit, in.doc)
	}
	return in
}

// describeGlobal says what a builtin, a library's member or a module of
// the host is, from the documentation (builtin_docs.go).
func describeGlobal(globals map[string][]string, path string) info {
	head, member, isMember := strings.Cut(path, ".")
	if !isMember {
		if members := globals[head]; len(members) > 0 {
			in := info{kind: completionModule, detail: "module " + head, doc: moduleDocs[head]}
			if in.doc == "" {
				in.doc = "The " + head + " library."
			}
			return in
		}
	}
	doc, ok := builtinDocs[path]
	if !ok && isMember && hasMember(globals[head], member) {
		shared, found := builtinDocs["*."+member]
		doc, ok = builtinDoc{Doc: shared.Doc}, found
		for _, sig := range shared.Signatures {
			doc.Signatures = append(doc.Signatures, head+"."+sig)
		}
	}
	if !ok {
		return info{kind: completionVariable, detail: path}
	}
	in := info{kind: completionConstant, detail: path, doc: doc.Doc}
	for _, label := range doc.Signatures {
		in.signatures = append(in.signatures, parseSignature(label))
	}
	if len(in.signatures) > 0 {
		in.kind = completionFunction
		labels := make([]string, len(in.signatures))
		for i, sig := range in.signatures {
			labels[i] = sig.label
		}
		in.detail = strings.Join(labels, "  ·  ")
	}
	return in
}

func hasMember(members []string, name string) bool {
	for _, m := range members {
		if m == name {
			return true
		}
	}
	return false
}

// functionSignature is the signature of f declared as name, with the doc
// of each parameter.
func functionSignature(ix *index, name string, f *ast.FunctionLiteral) signature {
	sig := parseSignature(ix.signature(name, f))
	line := 0
	if let := letOf(ix, f); let != nil {
		line = let.Range().StartLine
	}
	doc := ix.docOf(line, f)
	for i, p := range f.Parameters {
		if i < len(sig.docs) {
			sig.docs[i] = doc.params[p.Value]
		}
	}
	return sig
}

// parseSignature reads a label like "text.find(s, sub, [from])" into its
// parameters. A comma inside brackets or a default value doesn't cut one.
func parseSignature(label string) signature {
	sig := signature{label: label}
	open := strings.IndexByte(label, '(')
	if open < 0 || !strings.HasSuffix(label, ")") {
		return sig
	}
	inner := label[open+1 : len(label)-1]
	depth, start := 0, 0
	for i, c := range inner {
		switch c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				sig.params = append(sig.params, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	if strings.TrimSpace(inner) != "" {
		sig.params = append(sig.params, strings.TrimSpace(inner[start:]))
	}
	sig.docs = make([]string, len(sig.params))
	return sig
}

// letOf is the let in ix whose value is f, or nil.
func letOf(ix *index, f *ast.FunctionLiteral) *ast.LetStatement {
	if f == nil {
		return nil
	}
	for _, sym := range ix.symbols {
		if sym.let != nil && sym.let.Value == f {
			return sym.let
		}
	}
	return nil
}

// keyOf is the key of the hash literal pair t.path names, or nil.
func keyOf(t target) ast.Expression {
	names := splitPath(t.path)
	if len(names) == 0 {
		return nil
	}
	parent := t
	parent.path = strings.Join(names[:len(names)-1], ".")
	var holder ast.Expression
	if parent.path == "" && parent.sym != nil {
		holder = parent.sym.value()
	} else if parent.path == "" {
		return nil
	} else {
		holder = valueOf(parent)
	}
	if pair := pairOf(holder, names[len(names)-1]); pair != nil {
		return pair.Key
	}
	return nil
}

// header is the doc comment at the top of a module: its first -~> lines,
// before the first line that is not a comment.
func (ix *index) header() string {
	var lines []string
	for _, line := range ix.lines {
		text := strings.TrimSpace(line)
		if !strings.HasPrefix(text, "-~>") {
			break
		}
		lines = append(lines, strings.TrimPrefix(strings.TrimPrefix(text, "-~>"), " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// shortSource is the text of e when it fits on one short line.
func shortSource(ix *index, e ast.Expression) (string, bool) {
	if e == nil {
		return "", false
	}
	text := ix.source(e.Range())
	if text == "" || strings.Contains(text, "\n") || len(text) > 60 {
		return "", false
	}
	return text, true
}

// literalInfo says what the literal e is worth: its type, a unit converted
// (90deg = 1.5708 rad), a color's channels with a swatch, a point's
// coordinates. It is "" when e is not a literal.
func literalInfo(ix *index, e ast.Expression) string {
	switch e := e.(type) {
	case *ast.IntegerLiteral:
		if text := ix.source(e.Range()); strings.HasSuffix(text, "px") {
			return fmt.Sprintf("INTEGER `%s = %d`: a length in pixels", text, e.Value)
		}
		return fmt.Sprintf("INTEGER `%d`", e.Value)
	case *ast.FloatLiteral:
		return "FLOAT " + floatInfo(ix.source(e.Range()), e.Value)
	case *ast.ColorLiteral:
		return colorInfo(e)
	case *ast.PointLiteral:
		return pointInfo(ix, e)
	case *ast.StringLiteral:
		return fmt.Sprintf("STRING of %d characters", len([]rune(e.Value)))
	case *ast.Boolean:
		return "BOOLEAN `" + e.Token.Literal + "`"
	case *ast.FunctionLiteral:
		return "FUNCTION " + ix.signature("function", e)
	case *ast.ArrayLiteral:
		return fmt.Sprintf("ARRAY of %d elements", len(e.Elements))
	case *ast.HashLiteral:
		return fmt.Sprintf("HASH of %d pairs", len(e.Pairs))
	}
	return ""
}

// floatInfo writes a float, written as literal, with its unit converted:
// "`90deg = 1.5708 rad`".
func floatInfo(literal string, value float64) string {
	degrees := shortFloat(value * 180 / math.Pi)
	switch {
	case strings.HasSuffix(literal, "deg"), strings.HasSuffix(literal, "turn"):
		return fmt.Sprintf("`%s = %s rad`", literal, shortFloat(value))
	case strings.HasSuffix(literal, "rad"):
		return fmt.Sprintf("`%s = %s rad` (%s°)", literal, shortFloat(value), degrees)
	case strings.HasSuffix(literal, "px"):
		return fmt.Sprintf("`%s = %s`: a length in pixels", literal, shortFloat(value))
	case strings.ContainsAny(literal, "eE"):
		return fmt.Sprintf("`%s = %s`", literal, strconv.FormatFloat(value, 'f', -1, 64))
	}
	return "`" + literal + "`"
}

// shortFloat writes v with at most 4 decimals.
func shortFloat(v float64) string {
	return strconv.FormatFloat(math.Round(v*10000)/10000, 'f', -1, 64)
}

// colorInfo writes a color's channels and a swatch of it.
func colorInfo(e *ast.ColorLiteral) string {
	hex := fmt.Sprintf("#%02x%02x%02x", e.R, e.G, e.B)
	channels := fmt.Sprintf("rgb(%d, %d, %d)", e.R, e.G, e.B)
	if e.A != 255 {
		hex += fmt.Sprintf("%02x", e.A)
		channels = fmt.Sprintf("rgba(%d, %d, %d, %d)", e.R, e.G, e.B, e.A)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="48" height="16"><rect width="48" height="16" rx="3" fill="#%02x%02x%02x" fill-opacity="%.3f" stroke="#888"/></svg>`,
		e.R, e.G, e.B, float64(e.A)/255)
	swatch := "![" + hex + "](data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)) + ")"
	return fmt.Sprintf("COLOR `%s` = `%s`\n\n%s", hex, channels, swatch)
}

// pointInfo writes a point's coordinates.
func pointInfo(ix *index, e *ast.PointLiteral) string {
	axes := []string{"x", "y", "z"}
	parts := make([]string, len(e.Coords))
	for i, c := range e.Coords {
		parts[i] = axes[min(i, 2)] + " = " + numberText(ix, c)
	}
	return fmt.Sprintf("POINT %dD: `%s`", len(e.Coords), strings.Join(parts, "`, `"))
}

// numberText writes a coordinate: a number's value, or the code.
func numberText(ix *index, e ast.Expression) string {
	switch e := e.(type) {
	case *ast.IntegerLiteral:
		return strconv.FormatInt(e.Value, 10)
	case *ast.FloatLiteral:
		return shortFloat(e.Value)
	case *ast.PrefixExpression:
		if e.Operator == "-" {
			return "-" + numberText(ix, e.Right)
		}
	}
	return ix.source(e.Range())
}

func joinParts(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n\n")
}

// globalNames are the builtins and the host's names, sorted.
func globalNames(globals map[string][]string) []string {
	names := make([]string, 0, len(globals))
	for name := range globals {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
