package ir

import (
	"fmt"
)

// Stmt is the interface describing all statements.
type Stmt interface {
	fmt.Stringer

	stmt()
}

func (s *AssignStmt) stmt()   {}
func (s *MakeChanStmt) stmt() {}
func (s *ChanOpStmt) stmt()   {}
func (s *SelectStmt) stmt()   {}
func (s *IfStmt) stmt()       {}
func (s *ForStmt) stmt()      {}
func (s *RangeStmt) stmt()    {}
func (s *CallStmt) stmt()     {}
func (s *ReturnStmt) stmt()   {}
