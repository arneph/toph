package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// ReachabilityRequirement defines the expected reachability (reachable,
// unreachable or no requirement) for a point in the program.
type ReachabilityRequirement int

const (
	// NoReachabilityRequirement indicates that no checks should be performed.
	NoReachabilityRequirement ReachabilityRequirement = iota
	// Reachable indicates that the marked point is expected to be reachable.
	Reachable
	// Unreachable indicates that the marked point is expected to be
	// unreachable.
	Unreachable
)

// IfStmt represents an if or else branch.
type IfStmt struct {
	ifBranch   Body
	elseBranch Body

	Node
}

// NewIfStmt creates a new if or else branch, embedded in the given enclosing
// scope.
func NewIfStmt(superScope *Scope, pos, end token.Pos) *IfStmt {
	s := new(IfStmt)
	s.ifBranch.init()
	s.ifBranch.scope.superScope = superScope
	s.elseBranch.init()
	s.elseBranch.scope.superScope = superScope
	s.pos = pos
	s.end = end

	return s
}

// IfBranch returns the body of the if branch.
func (s *IfStmt) IfBranch() *Body {
	return &s.ifBranch
}

// ElseBranch returns the body of the else branch.
func (s *IfStmt) ElseBranch() *Body {
	return &s.elseBranch
}

func (s *IfStmt) String() string {
	str := "if{\n"
	str += "  " + strings.ReplaceAll(s.ifBranch.String(), "\n", "\n  ") + "\n"
	str += "}else{\n"
	str += "  " + strings.ReplaceAll(s.elseBranch.String(), "\n", "\n  ") + "\n"
	str += "}"
	return str
}

// ForStmt represents a conditional loop.
type ForStmt struct {
	cond Body
	body Body

	isInfinite    bool
	minIterations int
	maxIterations int

	Node
}

// NewForStmt creates a new loop, embedded in the given enclosing scope.
func NewForStmt(superScope *Scope, pos, end token.Pos) *ForStmt {
	s := new(ForStmt)
	s.cond.init()
	s.cond.scope.superScope = superScope
	s.body.init()
	s.body.scope.superScope = superScope
	s.isInfinite = false
	s.minIterations = -1
	s.maxIterations = -1
	s.pos = pos
	s.end = end

	return s
}

// Cond returns the condition body of the conditional loop.
func (s *ForStmt) Cond() *Body {
	return &s.cond
}

// Body returns the main body of the conditional loop.
func (s *ForStmt) Body() *Body {
	return &s.body
}

// IsInfinite returns whether the loop has no condition and runs forever unless
// exited otherwise.
func (s *ForStmt) IsInfinite() bool {
	return s.isInfinite
}

// SetIsInfinite sets whether the loop has no condition and runs forver unless
// exited otherwise.
func (s *ForStmt) SetIsInfinite(isInfinite bool) {
	s.isInfinite = isInfinite
}

// HasMinIterations returns whether a lower bound on the number of loop
// iterations is known.
func (s *ForStmt) HasMinIterations() bool {
	return s.minIterations >= 0
}

// MinIterations returns the lower bound on the number of loop iterations.
// A negative value indicates that no lower bound is known.
func (s *ForStmt) MinIterations() int {
	return s.minIterations
}

// SetMinIterations sets the lower bound on the number of loop iterations.
func (s *ForStmt) SetMinIterations(minIterations int) {
	s.minIterations = minIterations
}

// HasMaxIterations returns whether an upper bound on the number of loop
// iterations is known.
func (s *ForStmt) HasMaxIterations() bool {
	return s.maxIterations >= 0
}

// MaxIterations returns the upper bound on the number of loop iterations.
// A negative value indicates that no upper bound is known.
func (s *ForStmt) MaxIterations() int {
	return s.maxIterations
}

// SetMaxIterations sets the upper bound on the number of loop iterations.
func (s *ForStmt) SetMaxIterations(maxIterations int) {
	s.maxIterations = maxIterations
}

func (s *ForStmt) String() string {
	str := "for{\n"
	str += "  cond{\n"
	str += "    " + strings.ReplaceAll(s.cond.String(), "\n", "\n    ") + "\n"
	str += "  }\n"
	str += "  body{\n"
	str += "    " + strings.ReplaceAll(s.body.String(), "\n", "\n    ") + "\n"
	str += "  }\n"
	str += "}"
	return str
}

// RangeStmt represents a loop ranging over a channel.
type RangeStmt struct {
	channel *Variable
	body    Body

	Node
}

// NewRangeStmt creates a new loop ranging over the give channel and embedded
// in the given enclosing scope.
func NewRangeStmt(channel *Variable, superScope *Scope, pos, end token.Pos) *RangeStmt {
	s := new(RangeStmt)
	s.channel = channel
	s.body.init()
	s.body.scope.superScope = superScope
	s.pos = pos
	s.end = end

	return s
}

// Channel returns the channel the range based loop is ranging over.
func (s *RangeStmt) Channel() *Variable {
	return s.channel
}

// Body returns the main body of the range based loop.
func (s *RangeStmt) Body() *Body {
	return &s.body
}

func (s *RangeStmt) String() string {
	str := fmt.Sprintf("range %s {\n", s.channel.Handle())
	str += "  " + strings.ReplaceAll(s.body.String(), "\n", "\n  ") + "\n"
	str += "}"
	return str
}

// Loop is the common interface for ForStmt and RangeStmt loops
type Loop interface {
	Stmt

	Body() *Body
}

// BranchKind represents a continue or break, undertaken in a BranchStmt.
type BranchKind int

const (
	// Continue is the BranchKind that causes a BranchStmt to return the
	// program to the start of a loop.
	Continue BranchKind = iota
	// Break is the BranchKind that causes a BranchStmt to return the program
	// to exit a loop.
	Break
)

func (k BranchKind) String() string {
	switch k {
	case Continue:
		return "continue"
	case Break:
		return "break"
	default:
		panic(fmt.Errorf("unknown branch kind: %v", k))
	}
}

// BranchStmt represents a continue or break statement in a loop.
type BranchStmt struct {
	loop Loop
	kind BranchKind

	Node
}

// NewBranchStmt creates a new continue or break statement in a loop.
func NewBranchStmt(loop Loop, kind BranchKind, pos, end token.Pos) *BranchStmt {
	b := new(BranchStmt)
	b.loop = loop
	b.kind = kind
	b.pos = pos
	b.end = end

	return b
}

// Loop returns the loop that gets continued or exited by the branch statement.
func (b *BranchStmt) Loop() Loop {
	return b.loop
}

// Kind returns what kind of operation (continue or break) the branch statement
// causes.
func (b *BranchStmt) Kind() BranchKind {
	return b.kind
}

func (b *BranchStmt) String() string {
	return b.kind.String()
}
