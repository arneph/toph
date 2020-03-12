package ir

import (
	"fmt"
	"strings"
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

// CallStmt represents a function call (with or without the go keyword).
type CallStmt struct {
	callee   *Func
	kind     CallKind
	args     map[int]*Variable
	captures map[string]*Variable
	results  map[int]*Variable // Variables to assign results to
}

// NewCallStmt creates a new call statement to the given callee.
func NewCallStmt(callee *Func, kind CallKind) *CallStmt {
	s := new(CallStmt)
	s.callee = callee
	s.kind = kind
	s.args = make(map[int]*Variable)
	s.captures = make(map[string]*Variable)
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
func (s *CallStmt) Args() map[int]*Variable {
	return s.args
}

// AddArg adds an argument to the function call.
func (s *CallStmt) AddArg(index int, arg *Variable) {
	s.args[index] = arg
}

// Captures returns all variables caputered by the function call.
func (s *CallStmt) Captures() map[string]*Variable {
	return s.captures
}

// GetCaptured returns the captured varibale for the capturing variable.
func (s *CallStmt) GetCaptured(capturing string) *Variable {
	return s.captures[capturing]
}

// AddCapture adds a captioning relationship to the function call.
func (s *CallStmt) AddCapture(capturing string, captured *Variable) {
	s.captures[capturing] = captured
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
	if len(s.captures) > 0 {
		str += " ["
		firstCapture := true
		for capturing, captured := range s.captures {
			if firstCapture {
				firstCapture = false
			} else {
				str += ", "
			}
			str += "'" + capturing + "' <- &(" + captured.Handle() + ")"
		}
		str += "]"
	}
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
