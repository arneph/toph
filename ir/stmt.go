package ir

import (
	"fmt"
	"go/token"
)

// Stmt is the interface describing all statements.
type Stmt interface {
	fmt.Stringer

	Pos() token.Pos
	End() token.Pos

	stmt()
}

func (s *AssignStmt) stmt()      {}
func (s *BranchStmt) stmt()      {}
func (s *CallStmt) stmt()        {}
func (s *ChanOpStmt) stmt()      {}
func (s *ForStmt) stmt()         {}
func (s *IfStmt) stmt()          {}
func (s *InlinedCallStmt) stmt() {}
func (s *MakeChanStmt) stmt()    {}
func (s *RangeStmt) stmt()       {}
func (s *ReturnStmt) stmt()      {}
func (s *SelectStmt) stmt()      {}
