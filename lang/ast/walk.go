package ast

// Children returns the nodes directly inside node, in the order they are
// written. A nil child (an else that isn't there, a missing default) is
// left out.
func Children(node Node) []Node {
	var children []Node
	add := func(nodes ...Node) {
		for _, n := range nodes {
			if n != nil && !isNilNode(n) {
				children = append(children, n)
			}
		}
	}
	switch n := node.(type) {
	case *Program:
		for _, s := range n.Statements {
			add(s)
		}
	case *BlockStatement:
		for _, s := range n.Statements {
			add(s)
		}
	case *LetStatement:
		add(n.Name, n.Value)
	case *ReturnStatement:
		add(n.ReturnValue)
	case *ExpressionStatement:
		add(n.Expression)
	case *TryStatement:
		add(n.Body, n.Name, n.Catch)
	case *LayerStatement:
		add(n.Body)
	case *UseStatement:
		add(n.Path, n.Name)
	case *PointLiteral:
		for _, c := range n.Coords {
			add(c)
		}
	case *PrefixExpression:
		add(n.Right)
	case *InfixExpression:
		add(n.Left, n.Right)
	case *AssignExpression:
		add(n.Left, n.Value)
	case *IfExpression:
		add(n.Condition, n.Consequence, n.Alternative)
	case *WhileExpression:
		add(n.Condition, n.Body)
	case *ForExpression:
		add(n.Init, n.Condition, n.Update, n.Body)
	case *FunctionLiteral:
		for i, p := range n.Parameters {
			add(p)
			if i < len(n.Defaults) {
				add(n.Defaults[i])
			}
		}
		add(n.Body)
	case *CallExpression:
		add(n.Function)
		for _, a := range n.Arguments {
			add(a)
		}
	case *ArrayLiteral:
		for _, e := range n.Elements {
			add(e)
		}
	case *IndexExpression:
		add(n.Left, n.Index)
	case *HashLiteral:
		for _, pair := range n.Pairs {
			add(pair.Key, pair.Value)
		}
	}
	return children
}

// Inspect visits node and everything inside it, parents before their
// children. When visit returns false the children of that node are skipped.
func Inspect(node Node, visit func(Node) bool) {
	if !visit(node) {
		return
	}
	for _, child := range Children(node) {
		Inspect(child, visit)
	}
}

// isNilNode reports whether n is an interface holding a nil pointer, like a
// *BlockStatement Alternative that isn't there.
func isNilNode(n Node) bool {
	switch n := n.(type) {
	case *BlockStatement:
		return n == nil
	case *Identifier:
		return n == nil
	case *StringLiteral:
		return n == nil
	}
	return false
}
