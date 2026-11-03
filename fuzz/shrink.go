package fuzz

import "strings"

// Shrink makes a failing program as small as it can while keep still holds
// for it: it removes statements, unwraps blocks and replaces expressions by
// a part of them or by 0, one change at a time, and keeps every change after
// which the program still fails. A change that breaks the syntax doesn't
// hold, since keep only sees programs that parse.
func Shrink(program *Node, keep func(string) bool) *Node {
	current := program.Copy()
	for {
		changed := false
		for _, candidate := range candidates(current) {
			if keep(candidate.String()) {
				current, changed = candidate, true
				break
			}
		}
		if !changed {
			return current
		}
	}
}

// candidates are the programs that are one change smaller than program,
// the biggest changes first.
func candidates(program *Node) []*Node {
	var found []*Node
	// Every path is the list of kid indexes from the root to a node.
	var paths [][]int
	var walk func(n *Node, path []int)
	walk = func(n *Node, path []int) {
		paths = append(paths, append([]int(nil), path...))
		for i, kid := range n.Kids {
			walk(kid, append(path, i))
		}
	}
	walk(program, nil)

	for _, path := range paths {
		n := at(program, path)
		switch n.Kind {
		case Program, Block, Layer, Seq:
			// Remove one statement.
			for i := range n.Kids {
				found = append(found, edit(program, path, func(n *Node) *Node {
					n.Kids = append(n.Kids[:i:i], n.Kids[i+1:]...)
					return n
				}))
			}
			// Put the statements of a statement's blocks in its place.
			for i, kid := range n.Kids {
				inner := statementsIn(kid)
				if len(inner) == 0 {
					continue
				}
				found = append(found, edit(program, path, func(n *Node) *Node {
					kids := append([]*Node(nil), n.Kids[:i]...)
					for _, s := range inner {
						kids = append(kids, s.Copy())
					}
					n.Kids = append(kids, n.Kids[i+1:]...)
					return n
				}))
			}
		case Expr:
			// Replace an expression by one of its parts, or by 0.
			for i, kid := range n.Kids {
				if kid.Kind != Expr {
					continue
				}
				found = append(found, edit(program, path, func(*Node) *Node { return kid.Copy() }))
				_ = i
			}
			if len(n.Kids) > 0 || n.Format != "0" {
				found = append(found, edit(program, path, func(*Node) *Node { return leaf("0") }))
			}
		}
		if n.Kind == Block && len(n.Kids) > 0 {
			found = append(found, edit(program, path, func(*Node) *Node { return block() }))
		}
	}
	return found
}

// statementsIn gives the statements of the blocks, layers and sequences
// right inside a statement.
func statementsIn(n *Node) []*Node {
	if n.Kind == Layer || n.Kind == Seq {
		return n.Kids
	}
	var inner []*Node
	var look func(n *Node)
	look = func(n *Node) {
		for _, kid := range n.Kids {
			if kid.Kind == Block {
				inner = append(inner, kid.Kids...)
			} else if kid.Kind == Expr {
				look(kid)
			}
		}
	}
	if n.Kind == Stmt {
		look(n)
	}
	return inner
}

func at(n *Node, path []int) *Node {
	for _, i := range path {
		n = n.Kids[i]
	}
	return n
}

// edit copies program and replaces the node at path by what change makes
// of a copy of it.
func edit(program *Node, path []int, change func(*Node) *Node) *Node {
	root := program.Copy()
	if len(path) == 0 {
		return change(root)
	}
	parent := at(root, path[:len(path)-1])
	last := path[len(path)-1]
	parent.Kids[last] = change(parent.Kids[last])
	return root
}

// ShrinkLines is Shrink for a program that is only text: it removes ranges
// of lines, halving their size down to one line, while keep still holds.
func ShrinkLines(input string, keep func(string) bool) string {
	lines := strings.Split(input, "\n")
	for size := len(lines) / 2; size >= 1; size /= 2 {
		for start := 0; start+size <= len(lines); {
			candidate := append(append([]string(nil), lines[:start]...), lines[start+size:]...)
			if keep(strings.Join(candidate, "\n")) {
				lines = candidate
			} else {
				start++
			}
		}
	}
	return strings.Join(lines, "\n")
}
