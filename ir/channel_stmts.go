package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// ChanOp represents an operation performed on a channel.
// This can either be a make or close channel statement.
type ChanOp int

const (
	// MakeChan represents a make channel operation.
	MakeChan ChanOp = iota
	// Close represents a close channel operation.
	Close
)

func (o ChanOp) String() string {
	switch o {
	case MakeChan:
		return "make_chan"
	case Close:
		return "close"
	default:
		panic(fmt.Sprintf("unknown ChanOp: %d", o))
	}
}

// MakeChanStmt represents a make(chan ...) call.
type MakeChanStmt struct {
	channel    LValue
	bufferSize int

	Node
}

// NewMakeChanStmt creates a new MakeChanStmt for the given channel and buffer
// size.
func NewMakeChanStmt(channel LValue, bufferSize int, pos, end token.Pos) *MakeChanStmt {
	s := new(MakeChanStmt)
	s.channel = channel
	s.bufferSize = bufferSize
	s.pos = pos
	s.end = end

	return s
}

// Channel returns the lvalue holding the newly made channel.
func (s *MakeChanStmt) Channel() LValue {
	return s.channel
}

// BufferSize returns the buffer size of the newly made channel.
func (s *MakeChanStmt) BufferSize() int {
	return s.bufferSize
}

// SpecialOp returns the performed operation (always MakeChan).
func (s *MakeChanStmt) SpecialOp() SpecialOp {
	return MakeChan
}

func (s *MakeChanStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s <- make(chan, %d)", s.channel.Handle(), s.bufferSize)
}

// ChanCommOp represents an communication operation performed on a channel.
// This can either be a stand alone statement or a case in a select statement.
type ChanCommOp int

const (
	// Send represents a send operation on a channel.
	Send ChanCommOp = iota
	// Receive represents a receive operation on a channel.
	Receive
)

func (o ChanCommOp) String() string {
	switch o {
	case Send:
		return "send"
	case Receive:
		return "receive"
	default:
		panic(fmt.Sprintf("unknown ChanCommOp: %d", o))
	}
}

// ChanCommOpStmt represents a channel operation statement.
type ChanCommOpStmt struct {
	channel LValue
	op      ChanCommOp

	Node
}

// NewChanCommOpStmt creates a new channel operation statement for the given
// channel and with the given channel operation.
func NewChanCommOpStmt(channel LValue, op ChanCommOp, pos, end token.Pos) *ChanCommOpStmt {
	s := new(ChanCommOpStmt)
	s.channel = channel
	s.op = op
	s.pos = pos
	s.end = end

	return s
}

// Channel returns the lvalue holding the channel that is operated on.
func (s *ChanCommOpStmt) Channel() LValue {
	return s.channel
}

// Op returns the operation performed on the channel.
func (s *ChanCommOpStmt) Op() ChanCommOp {
	return s.op
}

func (s *ChanCommOpStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%v %s", s.op, s.channel.Handle())
}

// CloseChanStmt represents a channel close statement.
type CloseChanStmt struct {
	channel LValue

	Node
}

// NewCloseChanStmt creates a new channel close statement for the given
// channel.
func NewCloseChanStmt(channel LValue, pos, end token.Pos) *CloseChanStmt {
	s := new(CloseChanStmt)
	s.channel = channel
	s.pos = pos
	s.end = end

	return s
}

// Channel returns the lvalue holding the channel to be closed.
func (s *CloseChanStmt) Channel() LValue {
	return s.channel
}

// SpecialOp returns the performed operation (always Close).
func (s *CloseChanStmt) SpecialOp() SpecialOp {
	return Close
}

func (s *CloseChanStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "close %s", s.channel.Handle())
}

// SelectCase represents a single case in a select statement.
type SelectCase struct {
	opStmt *ChanCommOpStmt
	body   Body

	reachReq ReachabilityRequirement

	pos token.Pos
}

// OpStmt returns the operation of the select case.
func (c *SelectCase) OpStmt() *ChanCommOpStmt {
	return c.opStmt
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

func (c *SelectCase) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("case ")
	c.opStmt.tree(b, 0)
	b.WriteString(" {\n")
	c.body.tree(b, indent+1)
	b.WriteString("\n")
	writeIndent(b, indent)
	b.WriteString("}")
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
func (s *SelectStmt) AddCase(op *ChanCommOpStmt, pos token.Pos) *SelectCase {
	c := new(SelectCase)
	c.opStmt = op
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

func (s *SelectStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("select{\n")
	for _, c := range s.cases {
		c.tree(b, indent+1)
		b.WriteString("\n")
	}
	if s.hasDefault {
		writeIndent(b, indent+1)
		b.WriteString("default{\n")
		s.defaultBody.tree(b, indent+2)
		b.WriteString("\n")
		writeIndent(b, indent+1)
		b.WriteString("}\n")
	}
	writeIndent(b, indent)
	b.WriteString("}")
}
