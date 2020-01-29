package ir

import (
	"fmt"
	"go/ast"
	"strings"
)

// Stmt is the interface describing all statements.
type Stmt interface {
	ast.Node
	fmt.Stringer

	// stmtNode ensures only statement types conform to the Stmt interface.
	stmtNode()
}

func (s *CallStmt) stmtNode()    {}
func (s *GoStmt) stmtNode()      {}
func (s *SendStmt) stmtNode()    {}
func (s *ReceiveStmt) stmtNode() {}
func (s *CloseStmt) stmtNode()   {}
func (s *IfStmt) stmtNode()      {}
func (s *ForStmt) stmtNode()     {}
func (s *RangeStmt) stmtNode()   {}

// CallStmt represents a direct function call (without the go keyword).
type CallStmt struct {
	Node

	callee  *Func
	args    []Value // either Chan or Func
	results []Value // either Chan or Func
}

// NewCallStmt creates a new call statement to the given callee.
func NewCallStmt(node ast.Node, callee *Func) *CallStmt {
	s := new(CallStmt)
	s.initNode(node)
	s.callee = callee
	s.args = nil
	s.results = nil

	return s
}

// Callee returns the function called by the call statement.
func (s *CallStmt) Callee() *Func {
	return s.callee
}

func (s *CallStmt) String() string {
	return fmt.Sprintf("call %p", s.callee)
}

// GoStmt represents a go statement (using the go keyword).
type GoStmt struct {
	Node

	callee *Func
	args   []Value
}

// NewGoStmt creates a new go statement, calling the given callee.
func NewGoStmt(node ast.Node, callee *Func) *GoStmt {
	s := new(GoStmt)
	s.initNode(node)
	s.callee = callee
	s.args = nil

	return s
}

// Callee returns the function started in a new go routine by the go statement.
func (s *GoStmt) Callee() *Func {
	return s.callee
}

func (s *GoStmt) String() string {
	return fmt.Sprintf("go %p", s.callee)
}

// ChanOp represents an operation performed on a channel. This can either be a
// stand alone statement or a case in a select statement (excecpt for Close).
type ChanOp int

const (
	// Send represents a send operation on a channel.
	Send ChanOp = iota
	// Receive represents a receive operation on a channel.
	Receive
	// Close represents a close operation on a channel.
	Close
)

// ChanOpStmt represents a channel operation statement.
type ChanOpStmt struct {
	Node

	op      ChanOp
	channel *Chan
}

// SendStmt represents a send operation on a go channel.
type SendStmt struct {
	Node

	channel *Chan
}

// NewSendStmt creates a new send statement on the given channel.
func NewSendStmt(node ast.Node, channel *Chan) *SendStmt {
	s := new(SendStmt)
	s.initNode(node)
	s.channel = channel

	return s
}

// Channel returns the channel that the send statement sends on.
func (s *SendStmt) Channel() *Chan {
	return s.channel
}

func (s *SendStmt) String() string {
	return fmt.Sprintf("send %p", s.channel)
}

// ReceiveStmt represent a receive operation on a go channel.
type ReceiveStmt struct {
	Node

	channel *Chan
}

// NewReceiveStmt creates a new receive statement on the given channel.
func NewReceiveStmt(node ast.Node, channel *Chan) *ReceiveStmt {
	s := new(ReceiveStmt)
	s.initNode(node)
	s.channel = channel

	return s
}

// Channel returns the channel that the receive statement receives on.
func (s *ReceiveStmt) Channel() *Chan {
	return s.channel
}

func (s *ReceiveStmt) String() string {
	return fmt.Sprintf("receive %p", s.channel)
}

// CloseStmt represent a close operation on a channel.
type CloseStmt struct {
	Node

	channel *Chan
}

// NewCloseStmt creates a new close statement on the given channel.
func NewCloseStmt(node ast.Node, channel *Chan) *CloseStmt {
	s := new(CloseStmt)
	s.initNode(node)
	s.channel = channel

	return s
}

// Channel returns the channel that the close statement closes.
func (s *CloseStmt) Channel() *Chan {
	return s.channel
}

func (s *CloseStmt) String() string {
	return fmt.Sprintf("close %p", s.channel)
}

// IfStmt represents an if or else branch.
type IfStmt struct {
	Node

	ifBranch   Body
	elseBranch Body
}

// NewIfStmt creates a new if or else branch, embedded in the given enclosing
// scope.
func NewIfStmt(node ast.Node, superScope *Scope) *IfStmt {
	s := new(IfStmt)
	s.initNode(node)
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
	Node

	cond Body
	body Body
}

// NewForStmt creates a new loop, embedded in the given enclosing scope.
func NewForStmt(node ast.Node, superScope *Scope) *ForStmt {
	s := new(ForStmt)
	s.initNode(node)
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
	Node

	channel *Chan
	body    Body
}

// NewRangeStmt creates a new loop ranging over the give channel and embedded
// in the given enclosing scope.
func NewRangeStmt(node ast.Node, channel *Chan, superScope *Scope) *RangeStmt {
	s := new(RangeStmt)
	s.initNode(node)
	s.channel = channel
	s.body.init()
	s.body.scope.superScope = superScope

	return s
}

// Body returns the main body of the range based loop.
func (s *RangeStmt) Body() *Body {
	return &s.body
}

func (s *RangeStmt) String() string {
	str := fmt.Sprintf("range %p {\n", s.channel)
	str += "  " + strings.ReplaceAll(s.body.String(), "\n", "\n  ") + "\n"
	str += "}"
	return str
}
