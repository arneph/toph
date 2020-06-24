package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// DeadEndOp represents the operation performed by DeadEndStmt.
type DeadEndOp struct{}

// DeadEnd is the sole instance of DeadEndOp.
var DeadEnd DeadEndOp

func (o DeadEndOp) String() string {
	return "dead_end"
}

// DeadEndStmt is a statement that stalls the current function and models
// functions such as os.Exit.
type DeadEndStmt struct {
	Node
}

// NewDeadEndStmt creates a new DeadEndStmt.
func NewDeadEndStmt(pos, end token.Pos) *DeadEndStmt {
	s := new(DeadEndStmt)
	s.pos = pos
	s.end = end

	return s
}

// SpecialOp returns the performed operation (always DeadEnd).
func (s *DeadEndStmt) SpecialOp() SpecialOp {
	return DeadEnd
}

func (s *DeadEndStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s", DeadEnd.String())
}
