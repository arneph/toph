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

func (v *Variable) callable()        {}
func (fs *FieldSelection) callable() {}
func (f *Func) callable()            {}

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
		return "defer"
	case Go:
		return "go"
	default:
		panic(fmt.Errorf("unknown CallKind: %d", k))
	}
}

// CallStmt represents a function call (with or without the go keyword).
type CallStmt struct {
	callee             Callable
	calleeSignature    *types.Signature
	callKind           CallKind
	args               map[int]RValue
	argRequiresCopy    map[int]bool
	results            map[int]*Variable
	resultRequiresCopy map[int]bool

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
	s.argRequiresCopy = make(map[int]bool)
	s.results = make(map[int]*Variable)
	s.resultRequiresCopy = make(map[int]bool)
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
	return !s.IsStaticCall()
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

// ArgRequiresCopy returns if the argument at the given index requires a copy.
func (s *CallStmt) ArgRequiresCopy(index int) bool {
	return s.argRequiresCopy[index]
}

// AddArg adds an argument to the function call.
func (s *CallStmt) AddArg(index int, arg RValue, requiresCopy bool) {
	s.args[index] = arg
	s.argRequiresCopy[index] = requiresCopy
}

// Results returns all results processed by the function call.
func (s *CallStmt) Results() map[int]*Variable {
	return s.results
}

// ResultRequiresCopy returns if the result at the given index requires a copy.
func (s *CallStmt) ResultRequiresCopy(index int) bool {
	return s.resultRequiresCopy[index]
}

// AddResult adds an variable to store a function call result in.
func (s *CallStmt) AddResult(index int, result *Variable, requiresCopy bool) {
	s.results[index] = result
	s.resultRequiresCopy[index] = requiresCopy
}

func (s *CallStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	if len(s.results) > 0 {
		firstResult := true
		for i, result := range s.results {
			if firstResult {
				firstResult = false
			} else {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%d: %v", i, result)
			if s.resultRequiresCopy[i] {
				fmt.Fprintf(b, " (copy)")
			}
		}
		b.WriteString(" <- ")
	}
	switch callee := s.callee.(type) {
	case *Func:
		fmt.Fprintf(b, "%s %v (static)", s.callKind, callee.FuncValue())
	case LValue:
		fmt.Fprintf(b, "%s %v (dynamic)", s.callKind, callee.Handle())
	default:
		panic(fmt.Errorf("unexpected callee type: %T", callee))
	}
	b.WriteString("(")
	firstArg := true
	for i, arg := range s.args {
		if firstArg {
			firstArg = false
		} else {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d: %v", i, arg)
		if s.argRequiresCopy[i] {
			fmt.Fprintf(b, " (copy)")
		}
	}
	b.WriteString(")")
}

// ReturnStmt represents a return statement inside a function.
type ReturnStmt struct {
	results map[int]RValue
	isPanic bool

	Node
}

// NewReturnStmt creates a new return statement.
func NewReturnStmt(isPanic bool, pos, end token.Pos) *ReturnStmt {
	s := new(ReturnStmt)
	s.results = make(map[int]RValue)
	s.isPanic = isPanic
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

// IsPanic returns if the statement represents a regular return stmt or a call
// to panic.
func (s *ReturnStmt) IsPanic() bool {
	return s.isPanic
}

func (s *ReturnStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("return")
	if s.isPanic {
		b.WriteString(" (panic)")
	}
	firstResult := true
	for i, result := range s.results {
		if firstResult {
			firstResult = false
			b.WriteString(" ")
		} else {
			b.WriteString(", ")
		}
		switch result := result.(type) {
		case Value:
			fmt.Fprintf(b, "%d: %s", i, result)
		case LValue:
			fmt.Fprintf(b, "%d: %s", i, result.Handle())
		default:
			panic(fmt.Errorf("unexpected %T rvalue type", result))
		}
	}
}

// RecoverStmt represents a call to the builtin recover function.
type RecoverStmt struct {
	Node
}

// NewRecoverStmt creates a new RecoverStmt.
func NewRecoverStmt(pos, end token.Pos) *RecoverStmt {
	s := new(RecoverStmt)
	s.pos = pos
	s.end = end

	return s
}

func (s *RecoverStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("recover")
}
