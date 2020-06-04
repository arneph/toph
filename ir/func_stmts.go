package ir

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

// Callable is the common interface for receivers of function and go calls.
type Callable interface {
	callable()
}

func (v *Variable) callable() {}
func (f *Func) callable()     {}

// CallKind represents whether a call is synchronous or asynchronous
// (go statement). Multiple call kinds can be used in a bit map.
type CallKind int

const (
	// Call is the CallKind of a regular synchronous call.
	Call CallKind = 1 << iota
	// Defer is the CallKind of a deferred synchronous call.
	Defer
	// Go is the CallKind of an asynchronous call (go statement).
	Go
)

func (k CallKind) String() string {
	switch k {
	case Call:
		return "call"
	case Defer:
		return "call"
	case Go:
		return "go"
	default:
		panic(fmt.Errorf("unknown CallKind: %d", k))
	}
}

// CallStmt represents a function call (with or without the go keyword).
type CallStmt struct {
	callee          Callable
	calleeSignature *types.Signature
	callKind        CallKind
	args            map[int]RValue
	results         map[int]*Variable // Variables to assign results to

	Node
}

// NewCallStmt creates a new call statement to the given callee.
func NewCallStmt(callee Callable, calleeSignature *types.Signature, callKind CallKind, pos, end token.Pos) *CallStmt {
	if callee == nil {
		panic("tried to create CallStmt with nil callee")
	}

	s := new(CallStmt)
	s.callee = callee
	s.calleeSignature = calleeSignature
	s.callKind = callKind
	s.args = make(map[int]RValue)
	s.results = make(map[int]*Variable)
	s.pos = pos
	s.end = end

	return s
}

// Callee returns the function or function variable called by the call
// statement.
func (s *CallStmt) Callee() Callable {
	return s.callee
}

// IsStaticCall returns whether the call statement represents a static function
// call.
func (s *CallStmt) IsStaticCall() bool {
	_, ok := s.callee.(*Func)
	return ok
}

// IsDynamicCall returns whether the call statement represents a dynamic
// function call, involving a function variable.
func (s *CallStmt) IsDynamicCall() bool {
	_, ok := s.callee.(*Variable)
	return ok
}

// CalleeSignature returns the full callee type, including argument and result
// types that are otherwise not represented in the IR.
func (s *CallStmt) CalleeSignature() *types.Signature {
	return s.calleeSignature
}

// CallKind returns the kind of function call statement.
func (s *CallStmt) CallKind() CallKind {
	return s.callKind
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
	switch callee := s.callee.(type) {
	case *Func:
		str += fmt.Sprintf("%s %v (static)", s.callKind, callee.FuncValue())
	case *Variable:
		str += fmt.Sprintf("%s %v (dynamic)", s.callKind, callee.Handle())
	default:
		panic(fmt.Errorf("unexpected callee type: %T", callee))
	}
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

	Node
}

// NewReturnStmt creates a new return statement.
func NewReturnStmt(pos, end token.Pos) *ReturnStmt {
	s := new(ReturnStmt)
	s.results = make(map[int]RValue)
	s.pos = pos
	s.end = end

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

	Node
}

// NewInlinedCallStmt returns a new inlined call to the given function,
// embedded in the given enclosing scope.
func NewInlinedCallStmt(calleeName string, superScope *Scope, pos, end token.Pos) *InlinedCallStmt {
	s := new(InlinedCallStmt)
	s.calleeName = calleeName
	s.body.init()
	s.body.scope.superScope = superScope
	s.pos = pos
	s.end = end

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
