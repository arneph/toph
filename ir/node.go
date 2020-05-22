package ir

import "go/token"

// Node represents a IR entity with a corresponding AST node.
type Node struct {
	pos token.Pos
	end token.Pos
}

// Pos returns the start of the represented AST node.
func (n Node) Pos() token.Pos {
	return n.pos
}

// End returns the end of the represented AST node.
func (n Node) End() token.Pos {
	return n.end
}
