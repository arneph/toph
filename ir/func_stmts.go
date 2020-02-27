package ir

import (
	"fmt"
)

// CallKind represents whether a call is synchronous or asynchronous
// (go statement).
type CallKind int

const (
	// Call is the CallKind of a synchronous call.
	Call CallKind = iota
	// Go is the CallKind of an asynchronous call (go statement).
	Go
)

func (k CallKind) String() string {
	switch k {
	case Call:
		return "call"
	case Go:
		return "go"
	default:
		panic(fmt.Errorf("unknown CallKind: %d", k))
	}
}

// CallStmt represents a direct function call (without the go keyword).
type CallStmt struct {
	callee  *Func
	kind    CallKind
	args    map[int]*Variable
	results map[int]*Variable // Variables to assign results to
}

// NewCallStmt creates a new call statement to the given callee.
func NewCallStmt(callee *Func, kind CallKind) *CallStmt {
	s := new(CallStmt)
	s.callee = callee
	s.kind = kind
	s.args = make(map[int]*Variable)
	s.results = make(map[int]*Variable)

	return s
}

// Callee returns the function called by the call statement.
func (s *CallStmt) Callee() *Func {
	return s.callee
}

// Kind returns the kind of function call statement.
func (s *CallStmt) Kind() CallKind {
	return s.kind
}

// Args returns all results processed by the function call.
func (s *CallStmt) Args() map[int]*Variable {
	return s.args
}

// AddArg adds an argument to the function call.
func (s *CallStmt) AddArg(index int, arg *Variable) {
	s.args[index] = arg
}

// Results returns all results processed by the function call.
func (s *CallStmt) Results() map[int]*Variable {
	return s.results
}

// AddResult adds a variable to store a function call result in.
func (s *CallStmt) AddResult(index int, result *Variable) {
	s.results[index] = result
}

func (s *CallStmt) String() string {
	var str string
	if len(s.results) > 0 {
		firstResult := false
		for i, result := range s.results {
			if firstResult {
				firstResult = false
			} else {
				str += ", "
			}
			str += fmt.Sprintf("%d: %v", i, result)
		}
		str += " <- "
	}
	str += fmt.Sprintf("%s %v", s.kind, s.callee.FuncValue())
	str += "("
	firstArg := true
	for i, arg := range s.args {
		if firstArg {
			firstArg = false
		} else {
			str += ", "
		}
		str += fmt.Sprintf("%d: %v", i, arg)
	}
	str += ")"
	return str
}

// ReturnStmt represents a return statement inside a function.
type ReturnStmt struct {
	results map[int]*Variable
}

// NewReturnStmt creates a new return statement.
func NewReturnStmt() *ReturnStmt {
	s := new(ReturnStmt)
	s.results = make(map[int]*Variable)

	return s
}

// Results returns the variables holding the returned values.
func (s *ReturnStmt) Results() map[int]*Variable {
	return s.results
}

// AddResult adds a result to the return statement.
func (s *ReturnStmt) AddResult(index int, result *Variable) {
	s.results[index] = result
}

func (s *ReturnStmt) String() string {
	str := "return"
	firstResult := true
	for i, result := range s.results {
		if firstResult {
			firstResult = false
			str += " "
		} else {
			str += ", "
		}
		str += fmt.Sprintf("%d: %s", i, result.Handle())
	}
	return str
}
