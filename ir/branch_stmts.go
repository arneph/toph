package ir

import (
	"fmt"
	"strings"
)

// IfStmt represents an if or else branch.
type IfStmt struct {
	ifBranch   Body
	elseBranch Body
}

// NewIfStmt creates a new if or else branch, embedded in the given enclosing
// scope.
func NewIfStmt(superScope *Scope) *IfStmt {
	s := new(IfStmt)
	s.ifBranch.init()
	s.ifBranch.scope.superScope = superScope
	s.elseBranch.init()
	s.elseBranch.scope.superScope = superScope

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
	inc  Body
}

// NewForStmt creates a new loop, embedded in the given enclosing scope.
func NewForStmt(superScope *Scope) *ForStmt {
	s := new(ForStmt)
	s.cond.init()
	s.cond.scope.superScope = superScope
	s.body.init()
	s.body.scope.superScope = superScope

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
}

// NewRangeStmt creates a new loop ranging over the give channel and embedded
// in the given enclosing scope.
func NewRangeStmt(channel *Variable, superScope *Scope) *RangeStmt {
	s := new(RangeStmt)
	s.channel = channel
	s.body.init()
	s.body.scope.superScope = superScope

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
