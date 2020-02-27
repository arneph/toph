package ir

import "fmt"

// AssignStmt represents an assignment statement.
type AssignStmt struct {
	source      *Variable
	destination *Variable
}

// NewAssignStmt createas a new assignment statement.
func NewAssignStmt(source, destination *Variable) *AssignStmt {
	if source == nil || destination == nil {
		panic("tried to create AssignStmt with nil variable")
	}

	a := new(AssignStmt)
	a.source = source
	a.destination = destination

	return a
}

// Source returns the source variable of the assignment.
func (a *AssignStmt) Source() *Variable {
	return a.source
}

// Destination returns the destination variable of the assignment.
func (a *AssignStmt) Destination() *Variable {
	return a.destination
}

func (a *AssignStmt) String() string {
	return fmt.Sprintf("%s <- %s",
		a.destination.Handle(), a.source.Handle())
}
