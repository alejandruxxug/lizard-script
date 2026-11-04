package check

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"sort"
	"strings"
)

// The rules that look at one place of the tree are checked during the
// second walk; the ones that need every name (is it used? is it ever
// assigned?) are checked by finish.

// call is a call whose function may be known: a name, or m.name of a
// module used with as.
type call struct {
	node *ast.CallExpression
	f    found  // what the name, or the m of m.name, means
	name string // the name, or "m.name"
	dot  string // the name after the dot, or ""
}

// memberRead is a.name where a may be a module.
type memberRead struct {
	node   *ast.IndexExpression
	f      found
	module string // a
	member string // name
}

// call notes e, in the second walk, if it calls a name or a module's.
func (c *checker) call(e *ast.CallExpression, s *scope) {
	if c.declaring {
		return
	}
	switch fn := e.Function.(type) {
	case *ast.Identifier:
		c.calls = append(c.calls, call{node: e, f: c.resolve(fn.Value, startOf(fn), s), name: fn.Value})
	case *ast.IndexExpression:
		left, ok := fn.Left.(*ast.Identifier)
		member, dot := isDot(fn)
		if ok && dot {
			c.calls = append(c.calls, call{node: e, f: c.resolve(left.Value, startOf(left), s),
				name: left.Value + "." + member, dot: member})
		}
	}
}

// memberRead notes a.name, in the second walk, when a is a name.
func (c *checker) memberRead(e *ast.IndexExpression, s *scope) {
	left, ok := e.Left.(*ast.Identifier)
	member, dot := isDot(e)
	if c.declaring || !ok || !dot {
		return
	}
	c.reads = append(c.reads, memberRead{node: e, f: c.resolve(left.Value, startOf(left), s), module: left.Value, member: member})
}

// fieldSet notes a.name = ... and a["name"] = ...: a member a program adds
// to a module is one it may read later.
func (c *checker) fieldSet(target ast.Expression) {
	index, ok := target.(*ast.IndexExpression)
	if !ok {
		return
	}
	left, ok := index.Left.(*ast.Identifier)
	key, isString := index.Index.(*ast.StringLiteral)
	if ok && isString {
		c.setFields[left.Value+"."+key.Value] = true
	}
}

// unreachable reports the statements of list that come after a return, a
// break or a continue. A return alone at the end of its line followed by an
// expression is worse: the value meant to be returned is a statement of its
// own (LEP 0001).
func (c *checker) unreachable(list []ast.Statement) {
	for i, st := range list[:max(len(list)-1, 0)] {
		var word string
		switch st := st.(type) {
		case *ast.ReturnStatement:
			word = "return"
			next, ok := list[i+1].(*ast.ExpressionStatement)
			if st.ReturnValue == nil && ok && next.Range().StartLine > st.Range().EndLine {
				c.add(Error, RuleReturn, st, "put the value on the line of the return",
					"this return gives null: the value on the next line is a statement of its own")
				return
			}
		case *ast.BreakStatement:
			word = "break"
		case *ast.ContinueStatement:
			word = "continue"
		default:
			continue
		}
		first, last := list[i+1].Range(), list[len(list)-1].Range()
		span := &ast.Program{Span: ast.Span{StartLine: first.StartLine, StartColumn: first.StartColumn,
			EndLine: last.EndLine, EndColumn: last.EndColumn}}
		c.add(Warning, RuleUnreachable, span, "", "this code never runs: it comes after a %s", word)
		return
	}
}

// literals reports a comparison of two literals, whose value is always the
// same.
func (c *checker) literals(e *ast.InfixExpression) {
	switch e.Operator {
	case "==", "!=", "===", "!==":
	default:
		return
	}
	if c.declaring || !isLiteral(e.Left) || !isLiteral(e.Right) {
		return
	}
	value := evaluator.Eval(e, object.NewEnvironment())
	c.add(Warning, RuleLiterals, e, "", "both sides of %s are literals, so it is always %s", e.Operator, value.Inspect())
}

func isLiteral(e ast.Expression) bool {
	switch e := e.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.Boolean, *ast.ColorLiteral:
		return true
	case *ast.PrefixExpression:
		return e.Operator == "-" && isLiteral(e.Right)
	}
	return false
}

// selfAssignment reports x = x and a[i] = a[i].
func (c *checker) selfAssignment(e *ast.AssignExpression) {
	if !c.declaring && e.Operator == "" && e.Left.String() == e.Value.String() {
		c.add(Warning, RuleSelfAssign, e, "", "%s is assigned to itself", e.Left.String())
	}
}

// finish applies the rules that need every name and every use of it. With
// syntax errors parts of the tree are missing, so they are left out.
func (c *checker) finish() {
	if c.broken {
		return
	}
	c.arguments()
	c.members()
	for _, d := range c.decls {
		c.unused(d)
		c.shadows(d)
	}
	for _, m := range c.modules {
		if m.exports != nil && len(m.exports) > 0 && !m.used {
			c.add(Warning, RuleUnusedModule, m.use, "", "none of the names of %q are used", m.path)
		}
	}
}

// arguments checks the calls of functions whose parameters are known.
func (c *checker) arguments() {
	for _, call := range c.calls {
		if fn := c.knownFunction(call); fn != nil {
			countArguments(c, call, fn)
		}
	}
}

// knownFunction is the function a call surely calls, or nil: the one
// declaration the name can mean, a function that is never assigned.
func (c *checker) knownFunction(call call) *ast.FunctionLiteral {
	f := call.f
	if call.dot != "" {
		if len(f.decls) != 1 || f.decls[0].kind != useDecl || f.decls[0].assigned || c.setFields[call.name] {
			return nil
		}
		if exports := f.decls[0].module.exports; exports != nil && exports[call.dot] != nil {
			return exports[call.dot].function
		}
		return nil
	}
	if len(f.decls) == 1 {
		d := f.decls[0]
		if d.assigned || len(d.scope.decls[d.name]) > 1 {
			return nil
		}
		return d.function
	}
	if len(f.decls) == 0 && f.module != nil {
		if e := f.module.exports[call.name]; !e.assigned {
			return e.function
		}
	}
	return nil
}

func countArguments(c *checker, call call, fn *ast.FunctionLiteral) {
	required := len(fn.Parameters)
	for _, d := range fn.Defaults {
		if d != nil {
			required--
		}
	}
	got := len(call.node.Arguments)
	if got >= required && got <= len(fn.Parameters) {
		return
	}
	want := fmt.Sprint(required)
	if required != len(fn.Parameters) {
		want = fmt.Sprintf("%d to %d", required, len(fn.Parameters))
	}
	c.add(Error, RuleArguments, call.node, "", "wrong number of arguments to %s: want %s, got %d", call.name, want, got)
}

// members checks the a.name reads of the modules that are known: the
// globals that are hashes (math, fs, ...) and modules used with as.
func (c *checker) members() {
	called := map[ast.Node]bool{}
	for _, call := range c.calls {
		called[call.node.Function] = true
	}
	for _, read := range c.reads {
		names, private := c.moduleMembers(read)
		if names == nil || c.setFields[read.module+"."+read.member] {
			continue
		}
		severity := Warning
		if called[read.node] {
			severity = Error
		}
		if private[read.member] {
			c.add(severity, RuleMember, read.node, "", "%s.%s is private to its module", read.module, read.member)
			continue
		}
		if !contains(names, read.member) {
			hint := ""
			if best := closest(read.member, names); best != "" {
				hint = "did you mean `" + read.module + "." + best + "`?"
			}
			c.add(severity, RuleMember, read.node, hint, "%s has no %s", read.module, read.member)
		}
	}
}

// moduleMembers are the names a.name can read when a is a module, or nil
// when a is not one, or not one whose names are known. The names that
// start with _ are in private: a module keeps them.
func (c *checker) moduleMembers(read memberRead) ([]string, map[string]bool) {
	f := read.f
	if len(f.decls) == 0 && f.module == nil && f.global {
		return c.globals[read.module], nil
	}
	if len(f.decls) != 1 || f.decls[0].kind != useDecl || f.decls[0].assigned {
		return nil, nil
	}
	exports := f.decls[0].module.exports
	if exports == nil {
		return nil, nil
	}
	names := []string{}
	for name := range exports {
		names = append(names, name)
	}
	return names, map[string]bool{read.member: !object.Exported(read.member)}
}

// unused reports a let inside a block or a function, or a parameter, that
// is never read, and a use ... as whose name isn't. A name that starts
// with _ is unused on purpose. The program's own lets are not reported:
// another file may use them.
func (c *checker) unused(d *decl) {
	if d.used || strings.HasPrefix(d.name, "_") {
		return
	}
	switch {
	case d.kind == letDecl && d.scope.parent != nil:
		c.add(Warning, RuleUnused, d.node, "start its name with _ if that is on purpose", "%s is never used", d.name)
	case d.kind == paramDecl:
		c.add(Warning, RuleUnused, d.node, "start its name with _ if that is on purpose", "the parameter %s is never used", d.name)
	case d.kind == useDecl:
		c.add(Warning, RuleUnusedModule, d.node, "", "%s is never used", d.name)
	}
}

// shadows reports a let that hides a name of a scope around it.
func (c *checker) shadows(d *decl) {
	if d.kind != letDecl || strings.HasPrefix(d.name, "_") {
		return
	}
	if o := outerDecl(d); o != nil {
		what := ""
		if o.kind == paramDecl {
			what = "parameter "
		}
		c.add(Warning, RuleShadow, d.node, "", "let %s hides the %s%s of line %d", d.name, what, d.name, o.node.Token.Line)
	}
}

// suggestName is the hint for a name that isn't defined: the closest of
// the names that could be meant at scope s.
func (c *checker) suggestName(name string, s *scope) string {
	if name == "continue" {
		return "continue is a statement only when it is alone, at the end of its line or before a ;"
	}
	for _, m := range c.modules {
		if m.private[name] {
			return fmt.Sprintf("%s is private to the module %q: a name that starts with _ stays inside it", name, m.path)
		}
	}
	if best := closest(name, c.visibleNames(s)); best != "" {
		return "did you mean `" + best + "`?"
	}
	return ""
}

// closest is the name of names nearest to name, if one is near enough to
// be a typo: one edit away for a short name, up to three for a long one.
func closest(name string, names []string) string {
	limit := 3
	switch n := len(name); {
	case n <= 2:
		return ""
	case n <= 4:
		limit = 1
	case n <= 8:
		limit = 2
	}
	sort.Strings(names)
	best, bestDistance := "", limit+1
	for _, candidate := range names {
		if candidate == name {
			continue
		}
		if d := distance(name, candidate); d < bestDistance {
			best, bestDistance = candidate, d
		}
	}
	return best
}

// distance is how many characters to add, take out, change or swap with
// the next one to turn a into b.
func distance(a, b string) int {
	rows := make([][]int, len(a)+1)
	for i := range rows {
		rows[i] = make([]int, len(b)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
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

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
