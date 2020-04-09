package ir

import (
	"fmt"
	"strings"
)

// CallKind represents whether a call is synchronous or asynchronous
// (go statement). Multiple call kinds can be used in a bit map.
type CallKind int

const (
	// Call is the CallKind of a synchronous call.
	Call CallKind = 1 << iota
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

// CallStmt represents a function call (with or without the go keyword).
type CallStmt struct {
	callee  *Func
	kind    CallKind
	args    map[int]RValue
	results map[int]*Variable // Variables to assign results to
}

// NewCallStmt creates a new call statement to the given callee.
func NewCallStmt(callee *Func, kind CallKind) *CallStmt {
	s := new(CallStmt)
	s.callee = callee
	s.kind = kind
	s.args = make(map[int]RValue)
	s.results = make(map[int]*Variable)

	return s
}

// Callee returns the function called by the call statement.
func (s *CallStmt) Callee() *Func {
	return s.callee
}

// SetCallee sets the function called by the call statement.
func (s *CallStmt) SetCallee(callee *Func) {
	s.callee = callee
}

// Kind returns the kind of function call statement.
func (s *CallStmt) Kind() CallKind {
	return s.kind
}

// Args returns all results processed by the function call.
func (s *CallStmt) Args() map[int]RValue {
	return s.args
}

// AddArg adds an argument to the function call.
func (s *CallStmt) AddArg(index int, arg RValue) {
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
	results map[int]RValue
}

// NewReturnStmt creates a new return statement.
func NewReturnStmt() *ReturnStmt {
	s := new(ReturnStmt)
	s.results = make(map[int]RValue)

	return s
}

// Results returns the variables holding the returned values.
func (s *ReturnStmt) Results() map[int]RValue {
	return s.results
}

// AddResult adds a result to the return statement.
func (s *ReturnStmt) AddResult(index int, result RValue) {
	if result == nil {
		panic("tried to add nil result to ReturnStmt")
	}
	v, ok := result.(*Variable)
	if ok && v == nil {
		panic("tried to add nil result to ReturnStmt")
	}
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
		switch result := result.(type) {
		case Value:
			str += fmt.Sprintf("%d: %s", i, result)
		case *Variable:
			str += fmt.Sprintf("%d: %s", i, result.Handle())
		default:
			panic(fmt.Errorf("unexpected %T rvalue type", result))
		}
	}
	return str
}

// InlinedCallStmt represents an inlined function call.
type InlinedCallStmt struct {
	calleeName string
	body       Body
}

// NewInlinedCallStmt returns a new inlined call to the given function,
// embedded in the given enclosing scope.
func NewInlinedCallStmt(calleeName string, superScope *Scope) *InlinedCallStmt {
	s := new(InlinedCallStmt)
	s.calleeName = calleeName
	s.body.init()
	s.body.scope.superScope = superScope

	return s
}

// CalleeName returns the name of the inlined function of the inlined call.
func (s *InlinedCallStmt) CalleeName() string {
	return s.calleeName
}

// Body returns the body of the inlined function call.
func (s *InlinedCallStmt) Body() *Body {
	return &s.body
}

func (s *InlinedCallStmt) String() string {
	str := fmt.Sprintf("inlined call %v {\n", s.calleeName)
	str += "  " + strings.ReplaceAll(s.body.String(), "\n", "\n  ") + "\n"
	str += "}"
	return str
}
