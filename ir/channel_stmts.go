package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// MakeChanStmt represents a make(chan ...) call.
type MakeChanStmt struct {
	channel    *Variable
	bufferSize int

	Node
}

// NewMakeChanStmt creates a new MakeChanStmt for the given channel and buffer
// size.
func NewMakeChanStmt(channel *Variable, bufferSize int, pos, end token.Pos) *MakeChanStmt {
	s := new(MakeChanStmt)
	s.channel = channel
	s.bufferSize = bufferSize
	s.pos = pos
	s.end = end

	return s
}

// Channel returns the variable holding the newly made channel.
func (s *MakeChanStmt) Channel() *Variable {
	return s.channel
}

// BufferSize returns the buffer size of the newly made channel.
func (s *MakeChanStmt) BufferSize() int {
	return s.bufferSize
}

func (s *MakeChanStmt) String() string {
	return fmt.Sprintf("%s <- make(chan, %d)", s.channel.Handle(), s.bufferSize)
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

func (o ChanOp) String() string {
	switch o {
	case Send:
		return "send"
	case Receive:
		return "receive"
	case Close:
		return "close"
	default:
		panic(fmt.Sprintf("unknown ChanOp: %d", o))
	}
}

// ChanOpStmt represents a channel operation statement.
type ChanOpStmt struct {
	channel *Variable
	op      ChanOp

	Node
}

// NewChanOpStmt creates a new channel operation statement for the given
// channel and with the given channel operation.
func NewChanOpStmt(channel *Variable, op ChanOp, pos, end token.Pos) *ChanOpStmt {
	s := new(ChanOpStmt)
	s.channel = channel
	s.op = op
	s.pos = pos
	s.end = end

	return s
}

// Channel returns the variable holding the channel that is operated on.
func (s *ChanOpStmt) Channel() *Variable {
	return s.channel
}

// Op returns the operation performed on the channel.
func (s *ChanOpStmt) Op() ChanOp {
	return s.op
}

func (s *ChanOpStmt) String() string {
	return fmt.Sprintf("%v %s", s.op, s.channel.Handle())
}

// SelectCase represents a single case in a select statement.
type SelectCase struct {
	op   *ChanOpStmt
	body Body

	reachReq ReachabilityRequirement

	pos token.Pos
}

// OpStmt returns the operation of the select case.
func (c *SelectCase) OpStmt() *ChanOpStmt {
	return c.op
}

// Body returns the body of the select case.
func (c *SelectCase) Body() *Body {
	return &c.body
}

// ReachReq returns the reachability requirement of the select case.
func (c *SelectCase) ReachReq() ReachabilityRequirement {
	return c.reachReq
}

// SetReachReq sets the reachability requirement of the select case.
func (c *SelectCase) SetReachReq(reachReq ReachabilityRequirement) {
	c.reachReq = reachReq
}

// Pos returns the source code position of the select case.
func (c *SelectCase) Pos() token.Pos {
	return c.pos
}

func (c *SelectCase) String() string {
	str := "case " + c.op.String() + " {\n"
	str += "  " + strings.ReplaceAll(c.body.String(), "\n", "\n  ") + "\n"
	str += "}"
	return str
}

// SelectStmt represents a select statement.
type SelectStmt struct {
	cases []*SelectCase

	hasDefault  bool
	defaultBody Body

	superScope *Scope

	Node
	defaultPos token.Pos
}

// NewSelectStmt creates a new select statement, embedded in the given
// enclosing scope.
func NewSelectStmt(superScope *Scope, pos, end token.Pos) *SelectStmt {
	s := new(SelectStmt)
	s.cases = nil
	s.hasDefault = false
	s.defaultBody.init()
	s.defaultBody.scope.superScope = superScope
	s.superScope = superScope
	s.pos = pos
	s.end = end

	return s
}

// Cases returns all cases of the select statement. This does not include the
// default case, if present.
func (s *SelectStmt) Cases() []*SelectCase {
	return s.cases
}

// AddCase adds a case for the given channel op to the select statement. The
// new SelectCase gets returned and the scope of its body is embeded in the
// enclosing scope of the select statement.
func (s *SelectStmt) AddCase(op *ChanOpStmt, pos token.Pos) *SelectCase {
	c := new(SelectCase)
	c.op = op
	c.body.init()
	c.body.scope.superScope = s.superScope
	c.pos = pos

	s.cases = append(s.cases, c)

	return c
}

// HasDefault returns whether the select statement has a default case.
func (s *SelectStmt) HasDefault() bool {
	return s.hasDefault
}

// SetHasDefault sets whether the select statement has a default case.
func (s *SelectStmt) SetHasDefault(hasDefault bool) {
	s.hasDefault = hasDefault
}

// DefaultBody returns the body of the default case of the select statement.
func (s *SelectStmt) DefaultBody() *Body {
	return &s.defaultBody
}

// DefaultPos returns the source code position of the default case.
func (s *SelectStmt) DefaultPos() token.Pos {
	return s.defaultPos
}

// SetDefaultPos sets the source code position of the default case.
func (s *SelectStmt) SetDefaultPos(defaultPos token.Pos) {
	s.defaultPos = defaultPos
}

func (s *SelectStmt) String() string {
	str := "select{\n"
	for _, c := range s.cases {
		str += "  " + strings.ReplaceAll(c.String(), "\n", "\n  ") + "\n"
	}
	if s.hasDefault {
		str += "default {\n"
		str += "  " + strings.ReplaceAll(s.defaultBody.String(), "\n", "\n  ") + "\n"
		str += "}\n"
	}
	str += "}"
	return str
}
