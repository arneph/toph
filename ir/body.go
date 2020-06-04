package ir

import (
	"fmt"
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

// WalkStmts calls the given visitor function for every statement in the body,
// including statements contained in other statements, for example loops.
func (b *Body) WalkStmts(visitFunc func(stmt Stmt, scope *Scope)) {
	for i, stmt := range b.stmts {
		visitFunc(b.stmts[i], b.Scope())

		switch stmt := stmt.(type) {
		case *AssignStmt,
			*BranchStmt,
			*MakeChanStmt, *ChanOpStmt, *CloseChanStmt,
			*CallStmt, *ReturnStmt:
			continue
		case *InlinedCallStmt:
			stmt.Body().WalkStmts(visitFunc)
		case *SelectStmt:
			for _, c := range stmt.Cases() {
				c.Body().WalkStmts(visitFunc)
			}
			stmt.DefaultBody().WalkStmts(visitFunc)
		case *IfStmt:
			stmt.IfBranch().WalkStmts(visitFunc)
			stmt.ElseBranch().WalkStmts(visitFunc)
		case *ForStmt:
			stmt.Cond().WalkStmts(visitFunc)
			stmt.Body().WalkStmts(visitFunc)
		case *RangeStmt:
			stmt.Body().WalkStmts(visitFunc)
		default:
			panic(fmt.Errorf("WalkStmts encountered unknown Stmt: %T", stmt))
		}
	}
}

// AddStmt appends the given statement at the end of the body.
func (b *Body) AddStmt(stmt Stmt) {
	b.stmts = append(b.stmts, stmt)
}

// AddStmts appends the given statements at the end of the body.
func (b *Body) AddStmts(stmts ...Stmt) {
	b.stmts = append(b.stmts, stmts...)
}

// SetStmts replaces all statements in the body with the given, new statements.
func (b *Body) SetStmts(stmts []Stmt) {
	b.stmts = stmts
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
