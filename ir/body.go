package ir

import (
	"strings"
)

// Body represents a function or branch instruction body.
type Body struct {
	scope *Scope
	stmts []Stmt
}

func (b *Body) init() {
	b.scope = new(Scope)
	b.scope.init()
	b.stmts = nil
}

func (b *Body) initWithScope(s *Scope) {
	b.scope = s
	b.stmts = nil
}

// Scope returns the scope corresponding to the body.
func (b *Body) Scope() *Scope {
	return b.scope
}

// Stmts returns the statements inside the body.
func (b *Body) Stmts() []Stmt {
	return b.stmts
}

// AddStmt appends the given statement at the end of the body.
func (b *Body) AddStmt(stmt Stmt) {
	b.stmts = append(b.stmts, stmt)
}

// AddStmts appends the given statements at the end of the body.
func (b *Body) AddStmts(stmts ...Stmt) {
	b.stmts = append(b.stmts, stmts...)
}

func (b *Body) String() string {
	s := b.scope.String() + "\n"
	s += "stmts{\n"
	for _, stmt := range b.stmts {
		s += "  " + strings.ReplaceAll(stmt.String(), "\n", "\n  ") + "\n"
	}
	s += "}"
	return s
}
