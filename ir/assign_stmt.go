package ir

import "fmt"

// AssignStmt represents an assignment statement.
type AssignStmt struct {
	source      RValue
	destination *Variable
}

// NewAssignStmt createas a new assignment statement.
func NewAssignStmt(source RValue, destination *Variable) *AssignStmt {
	if source == nil || destination == nil {
		panic("tried to create AssignStmt with nil source or destination")
	}

	a := new(AssignStmt)
	a.source = source
	a.destination = destination

	return a
}

// Source returns the source variable of the assignment.
func (a *AssignStmt) Source() RValue {
	return a.source
}

// Destination returns the destination variable of the assignment.
func (a *AssignStmt) Destination() *Variable {
	return a.destination
}

func (a *AssignStmt) String() string {
	switch s := a.source.(type) {
	case Value:
		return fmt.Sprintf("%s <- %s",
			a.destination.Handle(), s.String())
	case *Variable:
		return fmt.Sprintf("%s <- %s",
			a.destination.Handle(), s.Handle())
	default:
		panic(fmt.Errorf("unexpected %T source type", s))
	}
}
