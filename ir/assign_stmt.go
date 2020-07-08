package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// AssignStmt represents an assignment statement.
type AssignStmt struct {
	source      RValue
	destination LValue

	requiresCopy bool

	Node
}

// NewAssignStmt createas a new assignment statement.
func NewAssignStmt(source RValue, destination LValue, requiresCopy bool, pos, end token.Pos) *AssignStmt {
	if source == nil || destination == nil {
		panic("tried to create AssignStmt with nil source or destination")
	}

	a := new(AssignStmt)
	a.source = source
	a.destination = destination
	a.requiresCopy = requiresCopy
	a.pos = pos
	a.end = end

	return a
}

// Source returns the source of the assignment.
func (a *AssignStmt) Source() RValue {
	return a.source
}

// Destination returns the destination of the assignment.
func (a *AssignStmt) Destination() LValue {
	return a.destination
}

// RequiresCopy returns if the value should be copied before being assigned.
func (a *AssignStmt) RequiresCopy() bool {
	return a.requiresCopy
}

func (a *AssignStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	switch s := a.source.(type) {
	case Value:
		fmt.Fprintf(b, "%s <- %s", a.destination.Handle(), s.String())
	case LValue:
		fmt.Fprintf(b, "%s <- %s", a.destination.Handle(), s.Handle())
	default:
		panic(fmt.Errorf("unexpected %T source type", s))
	}
	if a.requiresCopy {
		fmt.Fprintf(b, " (copy)")
	}
}
