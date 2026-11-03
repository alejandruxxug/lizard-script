package fuzz

import (
	"fmt"
	"strings"
)

// The generator turns a slice of bytes into a lizard-script program. Every
// choice it makes reads one byte, so the fuzzer can mutate the bytes and get
// another program; when the bytes run out every choice is 0, which is
// always the simplest option, so every input ends in a finished program.
//
// The programs are well-typed-ish: the generator remembers the type of
// every variable and function it made and mostly uses them where that type
// fits, so most programs run for a while before they fail, if they fail.
// Now and then it makes a mistake on purpose, to compare the errors too.
//
// A program is a tree of nodes (see Node), so the shrinker can take it apart.

// Kind is what a node of a generated program is.
type Kind int

const (
	Expr    Kind = iota // an expression, written inline
	Stmt                // a statement, on its own line
	Block               // { statements }, each on its own line
	Layer               // layer name and its statements, indented under it
	Program             // the statements of the whole program
	Seq                 // statements that go together, each on its own line
)

// Node is a piece of a generated program. Format is its text with a %s
// for each kid, in order; a node without kids is written as Format.
type Node struct {
	Kind   Kind
	Format string
	Kids   []*Node
}

// String writes the program the node is.
func (n *Node) String() string {
	var b strings.Builder
	n.write(&b, 0)
	return b.String()
}

// write writes the node at the indentation of the statement it is in.
func (n *Node) write(b *strings.Builder, indent int) {
	pad := func(i int) string { return strings.Repeat(" ", i) }
	switch n.Kind {
	case Program:
		for i, kid := range n.Kids {
			if i > 0 {
				b.WriteString("\n")
			}
			kid.write(b, 0)
		}
	case Block:
		if len(n.Kids) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{")
		for _, kid := range n.Kids {
			b.WriteString("\n" + pad(indent+2))
			kid.write(b, indent+2)
		}
		b.WriteString("\n" + pad(indent) + "}")
	case Seq:
		for i, kid := range n.Kids {
			if i > 0 {
				b.WriteString("\n" + pad(indent))
			}
			kid.write(b, indent)
		}
	case Layer:
		b.WriteString(n.Format)
		for _, kid := range n.Kids {
			b.WriteString("\n" + pad(indent+2))
			kid.write(b, indent+2)
		}
	default:
		if len(n.Kids) == 0 {
			b.WriteString(n.Format)
			return
		}
		parts := strings.Split(n.Format, "%s")
		for i, part := range parts {
			b.WriteString(part)
			if i < len(n.Kids) {
				n.Kids[i].write(b, indent)
			}
		}
	}
}

// Copy makes a deep copy of the node, for the shrinker.
func (n *Node) Copy() *Node {
	c := &Node{Kind: n.Kind, Format: n.Format}
	for _, kid := range n.Kids {
		c.Kids = append(c.Kids, kid.Copy())
	}
	return c
}

func leaf(text string) *Node { return &Node{Kind: Expr, Format: text} }

func ex(format string, kids ...*Node) *Node { return &Node{Kind: Expr, Format: format, Kids: kids} }

func st(format string, kids ...*Node) *Node { return &Node{Kind: Stmt, Format: format, Kids: kids} }

func block(stmts ...*Node) *Node { return &Node{Kind: Block, Kids: stmts} }

func seq(stmts ...*Node) *Node { return &Node{Kind: Seq, Kids: stmts} }

// typ is what the generator knows of a value.
type typ int

const (
	tInt typ = iota
	tFloat
	tStr
	tBool
	tArr
	tHash
	tPoint
	tColor
	tFunc
	typeCount
)

// variable is a name in scope with its type. A tFunc takes arity integers
// (the last ones have defaults from required on) and gives an integer.
type variable struct {
	name     string
	t        typ
	arity    int
	required int
}

// generator holds the state of one generation.
type generator struct {
	data   []byte
	pos    int
	scopes [][]variable
	names  int
	stmts  int // statements made so far, to keep programs small
	loops  int // loops open in the current function, for break and continue
	inFunc bool
	depth  int // blocks open
}

// Generate builds a program from data.
func Generate(data []byte) *Node {
	g := &generator{data: data, scopes: [][]variable{nil}}
	program := &Node{Kind: Program}
	count := 1 + g.pick(12)
	for i := 0; i < count; i++ {
		program.Kids = append(program.Kids, g.statement())
	}
	// The value of the program is what its last line gives.
	program.Kids = append(program.Kids, st("%s", g.expr(typ(g.pick(int(typeCount))), 1)))
	return program
}

// pick chooses a number from 0 to n-1.
func (g *generator) pick(n int) int {
	if g.pos >= len(g.data) || n <= 1 {
		return 0
	}
	b := int(g.data[g.pos])
	g.pos++
	return b % n
}

// chance is true about once in n choices, and never when the bytes ran out.
func (g *generator) chance(n int) bool {
	return g.pos < len(g.data) && g.pick(n) == n-1
}

func (g *generator) push() { g.scopes = append(g.scopes, nil) }
func (g *generator) pop()  { g.scopes = g.scopes[:len(g.scopes)-1] }
func (g *generator) define(v variable) {
	g.scopes[len(g.scopes)-1] = append(g.scopes[len(g.scopes)-1], v)
}

func (g *generator) fresh(prefix string) string {
	g.names++
	return fmt.Sprintf("%s%d", prefix, g.names)
}

// visible gives the variables in scope of type t.
func (g *generator) visible(t typ) []variable {
	var found []variable
	for _, scope := range g.scopes {
		for _, v := range scope {
			if v.t == t {
				found = append(found, v)
			}
		}
	}
	return found
}

// statement makes one statement.
func (g *generator) statement() *Node {
	g.stmts++
	if g.stmts > 40 || g.depth > 3 {
		return g.simpleStatement()
	}
	switch g.pick(20) {
	case 0, 1:
		return g.simpleStatement()
	case 2:
		return g.assignment()
	case 3:
		return g.ifStatement()
	case 4:
		return g.whileLoop()
	case 5:
		return g.forLoop()
	case 6:
		return g.functionDecl()
	case 7:
		return g.recursiveFunction()
	case 8:
		return g.counter()
	case 9:
		return g.tryCatch()
	case 10:
		return g.layer()
	case 11:
		if g.loops > 0 {
			return g.loopJump()
		}
		return g.assignment()
	case 12:
		if g.inFunc && g.chance(2) {
			return st("return %s", g.expr(tInt, 2))
		}
		return g.puts()
	case 13:
		if g.chance(4) {
			return g.mistake()
		}
		return g.tryCatch()
	case 14:
		return g.shadow()
	case 15:
		return g.loopClosures()
	case 16:
		return g.sorted()
	case 17:
		return g.loopValue()
	case 18:
		if !g.inFunc && g.depth == 0 && g.chance(8) {
			// A return at the top level ends the program.
			return st("return %s", g.expr(typ(g.pick(int(tFunc))), 2))
		}
		return g.puts()
	default:
		return g.puts()
	}
}

// simpleStatement is a let or a puts.
func (g *generator) simpleStatement() *Node {
	if g.pick(2) == 0 {
		return g.let()
	}
	return g.puts()
}

func (g *generator) let() *Node {
	t := typ(g.pick(int(typeCount)))
	if t == tFunc {
		return g.functionValue()
	}
	value := g.expr(t, 3)
	name := g.fresh("v")
	g.define(variable{name: name, t: t})
	return st("let "+name+" = %s", value)
}

func (g *generator) puts() *Node {
	count := 1 + g.pick(3)
	args := make([]*Node, count)
	for i := range args {
		args[i] = g.expr(typ(g.pick(int(typeCount))), 3)
	}
	return st("puts("+strings.TrimSuffix(strings.Repeat("%s, ", count), ", ")+")", args...)
}

// assignment changes a variable that exists, or an element of one.
func (g *generator) assignment() *Node {
	t := typ(g.pick(int(typeCount)))
	vars := g.visible(t)
	if len(vars) == 0 {
		return g.let()
	}
	v := vars[g.pick(len(vars))]
	switch {
	case t == tInt && g.chance(2):
		ops := []string{"+=", "-=", "*="}
		return st(v.name+" "+ops[g.pick(len(ops))]+" %s", g.expr(tInt, 2))
	case t == tFloat && g.chance(3):
		return st(v.name+" /= %s", g.expr(tFloat, 2))
	case t == tStr && g.chance(2):
		return st(v.name+" += %s", g.expr(tStr, 2))
	case t == tArr && g.chance(2):
		if g.pick(2) == 0 {
			return st("push("+v.name+", %s)", g.expr(tInt, 2))
		}
		return st(v.name+"[%s] = %s", g.smallIndex(), g.expr(tInt, 2))
	case t == tHash && g.chance(2):
		// Only a and d: b and c keep their types.
		return st(v.name+"."+[]string{"a", "d"}[g.pick(2)]+" = %s", g.expr(tInt, 2))
	}
	if t == tFunc {
		return g.puts()
	}
	return st(v.name+" = %s", g.expr(t, 3))
}

// shadow lets again a name that already exists, often from itself, in the
// same scope or an inner one.
func (g *generator) shadow() *Node {
	t := typ(g.pick(int(tFunc)))
	vars := g.visible(t)
	if len(vars) == 0 {
		return g.let()
	}
	v := vars[g.pick(len(vars))]
	g.define(v)
	return st("let "+v.name+" = %s", g.expr(t, 2))
}

// body makes the statements of a block in a new scope.
func (g *generator) body(max int) []*Node {
	g.push()
	g.depth++
	var stmts []*Node
	count := g.pick(max + 1)
	for i := 0; i < count; i++ {
		stmts = append(stmts, g.statement())
	}
	g.depth--
	g.pop()
	return stmts
}

func (g *generator) ifStatement() *Node {
	format := "if %s %s"
	kids := []*Node{g.condition(), block(g.body(3)...)}
	for g.chance(3) {
		keyword := []string{" elif ", " else if "}[g.pick(2)]
		format += keyword + "%s %s"
		kids = append(kids, g.condition(), block(g.body(2)...))
	}
	if g.chance(2) {
		format += " else %s"
		kids = append(kids, block(g.body(2)...))
	}
	return st(format, kids...)
}

// condition is a bool in parentheses, or now and then any value.
func (g *generator) condition() *Node {
	if g.chance(6) {
		return ex("(%s)", g.expr(typ(g.pick(int(typeCount))), 2))
	}
	return ex("(%s)", g.expr(tBool, 2))
}

// loopBody makes the body of a loop: the loop knows it is in a loop.
func (g *generator) loopBody(first ...*Node) *Node {
	g.loops++
	stmts := append(first, g.body(4)...)
	g.loops--
	return block(stmts...)
}

func (g *generator) whileLoop() *Node {
	counter := g.fresh("w")
	limit := g.loopLimit()
	g.push()
	g.define(variable{name: counter, t: tInt})
	loop := g.loopBody(st(counter + " += 1"))
	g.pop()
	return seq(st("let "+counter+" = 0"), st("while ("+counter+" < "+limit+") %s", loop))
}

func (g *generator) forLoop() *Node {
	counter := g.fresh("i")
	limit := g.loopLimit()
	g.push()
	g.define(variable{name: counter, t: tInt})
	loop := g.loopBody()
	g.pop()
	if g.chance(4) {
		// The step at the end of the round: it runs after a continue too.
		return st("for (let "+counter+" = 0; "+counter+" < "+limit+"; "+counter+" += %s) %s", ex("1"), loop)
	}
	return st("for (let "+counter+" = 0; "+counter+" < "+limit+"; "+counter+" += 1) %s", loop)
}

// loopClosures pushes a closure in each round of a loop and calls them
// after it: each round has its own variables.
func (g *generator) loopClosures() *Node {
	fs, i, j := g.fresh("fs"), g.fresh("i"), g.fresh("j")
	limit := fmt.Sprint(1 + g.pick(4))
	g.push()
	g.define(variable{name: i, t: tInt})
	value := g.expr(tInt, 1)
	g.pop()
	loop := st("for (let "+i+" = 0; "+i+" < "+limit+"; "+i+" += 1) %s", block(
		st("let "+j+" = %s", value),
		st("push("+fs+", function() { ["+i+", "+j+"] })"),
	))
	return seq(st("let "+fs+" = []"), loop, st("puts("+fs+"[0](), "+fs+"[len("+fs+") - 1]())"))
}

// sorted sorts an array with a function, which may fail.
func (g *generator) sorted() *Node {
	order := []string{"a > b", "a < b", "b - a", "nope"}[g.pick(4)]
	return st("puts(list.sort(%s, function(a, b) { "+order+" }))", g.expr(tArr, 2))
}

// loopValue uses a loop or an if without else as a value.
func (g *generator) loopValue() *Node {
	name := g.fresh("v")
	values := []string{
		"while (false) {}",
		"for (let z = 0; z < 2; z += 1) { if (z == 1) { break } }",
		"if (false) { 1 }",
	}
	return seq(st("let "+name+" = "+values[g.pick(len(values))]), st("puts("+name+", type("+name+"))"))
}

// loopLimit is how many rounds a loop makes: a few, and now and then so
// many that the program reaches the step limit, to compare where.
func (g *generator) loopLimit() string {
	if g.chance(20) {
		return "100000"
	}
	return fmt.Sprint(g.pick(8))
}

// loopJump is a break or continue, mostly behind an if.
func (g *generator) loopJump() *Node {
	jump := []string{"break", "continue"}[g.pick(2)]
	if g.chance(4) {
		return st(jump)
	}
	return st("if %s { "+jump+" }", g.condition())
}

// parameters makes the parameters of a function: integers, the last ones
// with defaults.
func (g *generator) parameters() (names []string, list string, required int) {
	count := g.pick(4)
	required = count - g.pick(count+1)
	var parts []string
	for i := 0; i < count; i++ {
		name := g.fresh("p")
		names = append(names, name)
		if i >= required {
			// A default may use the parameters before it.
			def := fmt.Sprint(g.pick(10))
			if i > 0 && g.chance(2) {
				def = names[i-1] + " * 2"
			}
			parts = append(parts, name+" = "+def)
		} else {
			parts = append(parts, name)
		}
	}
	return names, strings.Join(parts, ", "), required
}

// functionBody makes the body of a function whose parameters are names,
// ending in an integer expression.
func (g *generator) functionBody(names []string) *Node {
	outerLoops, outerFunc := g.loops, g.inFunc
	g.loops, g.inFunc = 0, true
	g.push()
	for _, name := range names {
		g.define(variable{name: name, t: tInt})
	}
	stmts := g.body(3)
	g.push()
	stmts = append(stmts, st("%s", g.expr(tInt, 2)))
	g.pop()
	g.pop()
	g.loops, g.inFunc = outerLoops, outerFunc
	return block(stmts...)
}

func (g *generator) functionDecl() *Node {
	name := g.fresh("f")
	names, list, required := g.parameters()
	// The name is defined first, so the body can call itself (it doesn't:
	// recursion without a base case only reaches the step limit).
	body := g.functionBody(names)
	g.define(variable{name: name, t: tFunc, arity: len(names), required: required})
	return st("function "+name+"("+list+") %s", body)
}

// functionValue is a let of a function literal, often a closure of what is
// in scope.
func (g *generator) functionValue() *Node {
	name := g.fresh("g")
	names, list, required := g.parameters()
	body := g.functionBody(names)
	g.define(variable{name: name, t: tFunc, arity: len(names), required: required})
	return st("let "+name+" = %s", ex("function("+list+") %s", body))
}

// recursiveFunction counts down from a small number to a base case.
func (g *generator) recursiveFunction() *Node {
	name := g.fresh("r")
	n := g.fresh("n")
	g.push()
	g.define(variable{name: n, t: tInt})
	base := g.expr(tInt, 1)
	step := g.expr(tInt, 1)
	g.pop()
	g.define(variable{name: name, t: tFunc, arity: 1, required: 1})
	ops := []string{"+", "-", "*"}
	decl := st("function "+name+"("+n+") %s", block(
		st("if ("+n+" <= 0) %s", block(st("return %s", base))),
		st(name+"("+n+" - 1) "+ops[g.pick(len(ops))]+" %s", step),
	))
	call := st("puts("+name+"(%s))", leaf(fmt.Sprint(g.pick(12))))
	return seq(decl, call)
}

// counter makes a closure that keeps a count, and calls it.
func (g *generator) counter() *Node {
	maker, count, c := g.fresh("mk"), g.fresh("c"), g.fresh("k")
	start := fmt.Sprint(g.pick(5))
	g.define(variable{name: c, t: tFunc})
	decl := st("function "+maker+"() %s", block(
		st("let "+count+" = "+start),
		st("function() %s", block(st(count+" += 1"), st(count))),
	))
	return seq(decl, st("let "+c+" = "+maker+"()"), st("puts("+c+"(), "+c+"())"))
}

func (g *generator) tryCatch() *Node {
	var tried []*Node
	tried = append(tried, g.body(2)...)
	if g.pick(3) != 0 {
		tried = append(tried, g.mistake())
	}
	name := g.fresh("e")
	g.push()
	caught := append([]*Node{st("puts(" + name + ".message, " + name + ".line, " + name + ".column)")}, g.body(2)...)
	g.pop()
	if g.chance(5) {
		return st("try %s catch %s", block(tried...), block(g.body(1)...))
	}
	return st("try %s catch ("+name+") %s", block(tried...), block(caught...))
}

func (g *generator) layer() *Node {
	name := g.fresh("L")
	if g.chance(4) {
		name = "\"" + name + " part\""
	}
	return &Node{Kind: Layer, Format: "layer " + name, Kids: g.body(3)}
}

// mistake is a statement that fails when it runs.
func (g *generator) mistake() *Node {
	mistakes := []string{
		"error(\"boom\")",
		"puts(1 / 0)",
		"puts(nope)",
		"puts(1 + true)",
		"puts([1, 2][5])",
		"puts(\"a\" - 1)",
		"puts(5 % 0)",
		"puts(len(1))",
		"puts(math.sqrt(-1))",
		"puts({\"a\": 1}.b.c)",
		"puts((1, 2) + \"x\")",
		"puts(#fff * #000)",
		"let q = 5; q()",
		"puts(-\"a\")",
	}
	return st(mistakes[g.pick(len(mistakes))])
}

// expr makes an expression of type t, at most depth levels deep.
func (g *generator) expr(t typ, depth int) *Node {
	if vars := g.visible(t); len(vars) > 0 && g.pick(3) == 1 && t != tFunc {
		return leaf(vars[g.pick(len(vars))].name)
	}
	if depth <= 0 {
		return g.literal(t)
	}
	d := depth - 1
	switch t {
	case tInt:
		return g.intExpr(d)
	case tFloat:
		return g.floatExpr(d)
	case tStr:
		return g.strExpr(d)
	case tBool:
		return g.boolExpr(d)
	case tArr:
		switch g.pick(5) {
		case 1:
			return ex("range(0, %s)", g.smallInt())
		case 2:
			return ex("list.reverse(%s)", g.expr(tArr, d))
		case 3:
			return ex("rest(%s)", g.expr(tArr, d))
		case 4:
			return ex("[%s, %s]", g.expr(tInt, d), g.expr(tInt, d))
		}
	case tHash:
		if g.pick(4) == 2 {
			return ex("hash.merge(%s, {\"d\": %s})", g.expr(tHash, d), g.expr(tInt, d))
		}
		if g.pick(3) == 1 {
			return ex("{\"a\": %s, \"b\": %s, \"c\": %s}", g.expr(tInt, d), g.expr(tStr, d), g.expr(tArr, d))
		}
	case tPoint:
		switch g.pick(5) {
		case 1:
			return ex("(%s + %s)", g.expr(tPoint, d), g.expr(tPoint, d))
		case 2:
			return ex("(%s * %s)", g.expr(tPoint, d), g.expr(tInt, d))
		case 3:
			return ex("(-%s)", g.expr(tPoint, d))
		case 4:
			return ex("(%s, %s)", g.expr(tInt, d), g.expr(tFloat, d))
		}
	case tColor:
		switch g.pick(3) {
		case 1:
			return ex("rgb(%s, %s, %s)", g.smallInt(), g.smallInt(), g.smallInt())
		case 2:
			return ex("rgba(%s, %s, %s, %s)", g.smallInt(), g.smallInt(), g.smallInt(), g.smallInt())
		}
	case tFunc:
		return g.call(d)
	}
	return g.literal(t)
}

func (g *generator) intExpr(d int) *Node {
	ops := []string{"+", "-", "*", "//", "%", "/", "**"}
	switch g.pick(12) {
	case 1, 2, 3:
		op := ops[g.pick(len(ops))]
		switch {
		case op == "**":
			return ex("(%s ** %s)", g.expr(tInt, d), g.smallInt())
		case (op == "/" || op == "//" || op == "%") && !g.chance(4):
			return ex("(%s "+op+" %s)", g.expr(tInt, d), g.divisor())
		}
		return ex("(%s "+op+" %s)", g.expr(tInt, d), g.expr(tInt, d))
	case 4:
		return ex("len(%s)", g.expr([]typ{tArr, tStr, tHash}[g.pick(3)], d))
	case 5:
		return ex("int(%s)", g.expr(tFloat, d))
	case 6:
		return ex("%s[%s]", g.expr(tArr, d), g.smallIndex())
	case 7:
		return g.call(d)
	case 8:
		return ex("if %s { %s } else { %s }", g.condition(), g.expr(tInt, d), g.expr(tInt, d))
	case 9:
		return ex("%s."+[]string{"r", "g", "b", "a"}[g.pick(4)], g.expr(tColor, d))
	case 10:
		return ex("%s.a", g.expr(tHash, d))
	case 11:
		return ex("(-%s)", g.expr(tInt, d))
	}
	return g.literal(tInt)
}

func (g *generator) floatExpr(d int) *Node {
	ops := []string{"+", "-", "*", "/", "//", "%"}
	switch g.pick(9) {
	case 1, 2:
		op := ops[g.pick(len(ops))]
		if (op == "/" || op == "//" || op == "%") && !g.chance(4) {
			return ex("(%s "+op+" %s)", g.expr(tFloat, d), g.divisor())
		}
		return ex("(%s "+op+" %s)", g.expr(tFloat, d), g.expr([]typ{tFloat, tInt}[g.pick(2)], d))
	case 3:
		fns := []string{"sin", "cos", "abs", "floor", "round", "ceil"}
		if g.chance(5) {
			return ex("math.sqrt(math.abs(%s))", g.expr(tFloat, d))
		}
		return ex("math."+fns[g.pick(len(fns))]+"(%s)", g.expr(tFloat, d))
	case 4:
		return ex("float(%s)", g.expr(tInt, d))
	case 5:
		return ex("%s."+[]string{"x", "y"}[g.pick(2)], g.expr(tPoint, d))
	case 6:
		return ex("math.dist(%s, %s)", g.expr(tPoint, d), g.expr(tPoint, d))
	case 7:
		return ex("math.max(%s, %s)", g.expr(tFloat, d), g.expr(tInt, d))
	case 8:
		return ex("(%s * 1deg)", g.expr(tInt, d))
	}
	return g.literal(tFloat)
}

func (g *generator) strExpr(d int) *Node {
	switch g.pick(9) {
	case 1, 2:
		return ex("(%s + %s)", g.expr(tStr, d), g.expr(tStr, d))
	case 3:
		return ex("str(%s)", g.expr(typ(g.pick(int(typeCount))), d))
	case 4:
		return ex("(%s * %s)", g.expr(tStr, d), g.smallInt())
	case 5:
		return ex("type(%s)", g.expr(typ(g.pick(int(typeCount))), d))
	case 6:
		return ex("text.upper(%s)", g.expr(tStr, d))
	case 7:
		return ex("%s[%s]", g.expr(tStr, d), g.smallIndex())
	case 8:
		return ex("%s.b", g.expr(tHash, d))
	}
	return g.literal(tStr)
}

func (g *generator) boolExpr(d int) *Node {
	compare := []string{"<", ">", "<=", ">=", "==", "!="}
	switch g.pick(8) {
	case 1, 2:
		t := []typ{tInt, tFloat}[g.pick(2)]
		return ex("(%s "+compare[g.pick(len(compare))]+" %s)", g.expr(t, d), g.expr(tInt, d))
	case 3:
		return ex("(%s && %s)", g.expr(tBool, d), g.expr(tBool, d))
	case 4:
		return ex("(%s || %s)", g.expr(tBool, d), g.expr(tBool, d))
	case 5:
		return ex("(!%s)", g.expr(tBool, d))
	case 6:
		t := typ(g.pick(int(tFunc)))
		op := []string{"==", "!=", "===", "!=="}[g.pick(4)]
		return ex("(%s "+op+" %s)", g.expr(t, d), g.expr(t, d))
	case 7:
		if g.pick(2) == 0 {
			return ex("hash.has(%s, %s)", g.expr(tHash, d), g.expr(tStr, d))
		}
		return ex("same(%s, %s)", g.expr(tArr, d), g.expr(tArr, d))
	}
	return g.literal(tBool)
}

// call calls a function in scope with the right number of integers, or
// now and then with too few or too many.
func (g *generator) call(d int) *Node {
	fns := g.visible(tFunc)
	if len(fns) == 0 {
		return g.literal(tInt)
	}
	f := fns[g.pick(len(fns))]
	count := f.required + g.pick(f.arity-f.required+1)
	if g.chance(12) {
		count = f.arity + 1
	}
	args := make([]*Node, count)
	for i := range args {
		args[i] = g.expr(tInt, d)
	}
	return ex(f.name+"("+strings.TrimSuffix(strings.Repeat("%s, ", count), ", ")+")", args...)
}

// divisor is a number that is not 0.
func (g *generator) divisor() *Node {
	return leaf([]string{"3", "-2", "7", "1", "2.5"}[g.pick(5)])
}

func (g *generator) smallInt() *Node { return leaf(fmt.Sprint(g.pick(5))) }

func (g *generator) smallIndex() *Node {
	return leaf([]string{"0", "1", "2", "-1", "3"}[g.pick(5)])
}

// literal makes a literal of type t.
func (g *generator) literal(t typ) *Node {
	choose := func(options ...string) *Node { return leaf(options[g.pick(len(options))]) }
	switch t {
	case tInt:
		return choose("0", "1", "2", "3", "7", "-4", "10px", "1000", "9223372036854775807", "255")
	case tFloat:
		return choose("1.5", "0.0", "2.5e-3", "45deg", "0.25turn", "1rad", "-0.5", "1e3", "3.0", "0.1")
	case tStr:
		return choose("\"\"", "\"a\"", "\"hi there\"", "\"tab\\tquote\\\"\"", "\"\\u00e9\"", "\"\\u{1F98E}\"", "\"line\\nbreak\"", "\"x,y\"")
	case tBool:
		return choose("true", "false")
	case tArr:
		return choose("[]", "[1, 2, 3]", "[0]", "[5, -1]", "[1, 2.5, \"x\"]")
	case tHash:
		return choose("{\"a\": 0, \"b\": \"\", \"c\": []}", "{\"a\": 1, \"b\": \"t\", \"c\": [1]}", "{\"c\": [2, 3], \"b\": \"u\", \"a\": -1}")
	case tPoint:
		return choose("(1, 2)", "(0, 0)", "(1.5, -2)", "(10px, 20px)")
	case tColor:
		return choose("#fff", "#3fa34d", "#3fa34d80", "#000000", "#AbC")
	case tFunc:
		return choose("function(a) { a + 1 }", "function() { 0 }")
	}
	return leaf("null")
}
