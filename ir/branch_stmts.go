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
	ifPos   token.Pos
	elsePos token.Pos
}

// NewIfStmt creates a new if or else branch, embedded in the given enclosing
// scope.
func NewIfStmt(superScope *Scope, pos, end, ifPos, elsePos token.Pos) *IfStmt {
	s := new(IfStmt)
	s.ifBranch.init()
	s.ifBranch.scope.superScope = superScope
	s.elseBranch.init()
	s.elseBranch.scope.superScope = superScope
	s.pos = pos
	s.end = end
	s.ifPos = ifPos
	s.elsePos = elsePos

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

// IfPos returns the source code position of the if branch.
func (s *IfStmt) IfPos() token.Pos {
	return s.ifPos
}

// ElsePos returns the source code position of the else branch.
func (s *IfStmt) ElsePos() token.Pos {
	return s.elsePos
}

func (s *IfStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("if{\n")
	s.ifBranch.tree(b, indent+1)
	writeIndent(b, indent)
	b.WriteString("}else{\n")
	s.elseBranch.tree(b, indent+1)
	writeIndent(b, indent)
	b.WriteString("}")
}

// SwitchCase represents a single case in a select statement.
type SwitchCase struct {
	conds          []Body
	body           Body
	isDefault      bool
	hasFallthrough bool

	pos     token.Pos
	condPos []token.Pos
	condEnd []token.Pos
}

// Conds returns the condition bodies of the switch case.
func (c *SwitchCase) Conds() []*Body {
	conds := make([]*Body, len(c.conds))
	for i := range conds {
		conds[i] = &c.conds[i]
	}
	return conds
}

// CondPos returns the source code position of the condition at the given
// index.
func (c *SwitchCase) CondPos(index int) token.Pos {
	return c.condPos[index]
}

// CondEnd returns the source code position of the end of the condition at the
// given index.
func (c *SwitchCase) CondEnd(index int) token.Pos {
	return c.condEnd[index]
}

// AddCond adds a condition body to the switch case.
func (c *SwitchCase) AddCond(pos, end token.Pos) *Body {
	var cond Body
	cond.init()
	cond.scope.superScope = c.body.scope.superScope

	i := len(c.conds)
	c.conds = append(c.conds, cond)
	c.condPos = append(c.condPos, pos)
	c.condEnd = append(c.condEnd, end)

	return &c.conds[i]
}

// Body returns the body of the switch case.
func (c *SwitchCase) Body() *Body {
	return &c.body
}

// IsDefault returns if the switch case is the default case.
func (c *SwitchCase) IsDefault() bool {
	return c.isDefault
}

// SetIsDefault sets if the switch case is the default case.
func (c *SwitchCase) SetIsDefault(isDefault bool) {
	c.isDefault = isDefault
}

// HasFallthrough returns if the switch case has a fallthrough statement at the
// end of its body, leading into the next switch case.
func (c *SwitchCase) HasFallthrough() bool {
	return c.hasFallthrough
}

// SetHasFallthrough sets if the switch case has a fallthrough statement at the
// end of its body, leading into the next switch case.
func (c *SwitchCase) SetHasFallthrough(hasFallthrough bool) {
	c.hasFallthrough = hasFallthrough
}

// Pos returns the source code position of the switch case.
func (c *SwitchCase) Pos() token.Pos {
	return c.pos
}

func (c *SwitchCase) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("case{\n")
	writeIndent(b, indent+1)
	fmt.Fprintf(b, "fefault: %b", c.isDefault)
	writeIndent(b, indent+1)
	fmt.Fprintf(b, "fallthrough: %b", c.hasFallthrough)
	writeIndent(b, indent+1)
	b.WriteString("conds{\n")
	for _, cond := range c.conds {
		writeIndent(b, indent+2)
		b.WriteString("cond{\n")
		cond.tree(b, indent+3)
		writeIndent(b, indent+2)
		b.WriteString("}\n")
	}
	writeIndent(b, indent+1)
	b.WriteString("}\n")
	c.body.tree(b, indent+1)
	writeIndent(b, indent)
	b.WriteString("}")
}

// SwitchStmt represents a switch statement.
type SwitchStmt struct {
	cases []*SwitchCase

	superScope *Scope

	Node
}

// NewSwitchStmt creates a new switch statement embedded in the given
// enclosing scope.
func NewSwitchStmt(superScope *Scope, pos, end token.Pos) *SwitchStmt {
	s := new(SwitchStmt)
	s.cases = nil
	s.superScope = superScope
	s.pos = pos
	s.end = end

	return s
}

// Cases returns all cases of the switch statement.
func (s *SwitchStmt) Cases() []*SwitchCase {
	return s.cases
}

// AddCase adds a case to the switch statement. The new case gets returned and
// the scope of its body (and conditions added later) is embedded in the
// enclosing scope of the select statement.
func (s *SwitchStmt) AddCase(pos token.Pos) *SwitchCase {
	c := new(SwitchCase)
	c.body.init()
	c.body.scope.superScope = s.superScope
	c.pos = pos

	s.cases = append(s.cases, c)

	return c
}

func (s *SwitchStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("switch{\n")
	for _, c := range s.cases {
		c.tree(b, indent+1)
		b.WriteString("\n")
	}
	writeIndent(b, indent)
	b.WriteString("}")
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

func (s *ForStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("for{\n")
	writeIndent(b, indent+1)
	b.WriteString("cond{\n")
	s.cond.tree(b, indent+2)
	b.WriteString("\n")
	writeIndent(b, indent+1)
	b.WriteString("}\n")
	s.body.tree(b, indent+1)
	b.WriteString("\n")
	writeIndent(b, indent)
	b.WriteString("\n")
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

func (s *RangeStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "range %s {\n", s.channel.Handle())
	s.body.tree(b, indent+1)
	b.WriteString("\n")
	writeIndent(b, indent)
	b.WriteString("}")
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
		panic(fmt.Errorf("unknown branch kind"))
	}
}

// BranchStmt represents a continue or break statement in a loop.
type BranchStmt struct {
	targetStmt Stmt
	kind       BranchKind

	Node
}

// NewBranchStmt creates a new continue or break statement in a loop.
func NewBranchStmt(targetStmt Stmt, kind BranchKind, pos, end token.Pos) *BranchStmt {
	b := new(BranchStmt)
	b.targetStmt = targetStmt
	b.kind = kind
	b.pos = pos
	b.end = end

	return b
}

// TargetStmt returns the statement that gets continued or exited by the branch statement.
func (b *BranchStmt) TargetStmt() Stmt {
	return b.targetStmt
}

// Kind returns what kind of operation (continue or break) the branch statement
// causes.
func (b *BranchStmt) Kind() BranchKind {
	return b.kind
}

func (b *BranchStmt) tree(bob *strings.Builder, indent int) {
	writeIndent(bob, indent)
	bob.WriteString(b.kind.String())
}
