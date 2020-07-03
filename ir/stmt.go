package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// Stmt is the interface describing all statements.
type Stmt interface {
	Pos() token.Pos
	End() token.Pos

	tree(b *strings.Builder, indent int)
	stmt()
}

func (s *AssignStmt) stmt()        {}
func (s *BranchStmt) stmt()        {}
func (s *CallStmt) stmt()          {}
func (s *ChanCommOpStmt) stmt()    {}
func (s *CloseChanStmt) stmt()     {}
func (s *DeadEndStmt) stmt()       {}
func (s *ForStmt) stmt()           {}
func (s *IfStmt) stmt()            {}
func (s *MakeChanStmt) stmt()      {}
func (s *MakeMutexStmt) stmt()     {}
func (s *MakeWaitGroupStmt) stmt() {}
func (s *MakeStructStmt) stmt()    {}
func (s *MutexOpStmt) stmt()       {}
func (s *RangeStmt) stmt()         {}
func (s *ReturnStmt) stmt()        {}
func (s *SelectStmt) stmt()        {}
func (s *SwitchStmt) stmt()        {}
func (s *WaitGroupOpStmt) stmt()   {}

// SpecialOp is the common interface for all operations representing specially
// modeled function calls.
type SpecialOp interface {
	fmt.Stringer

	specialOp()
}

func (o ChanOp) specialOp()      {}
func (o DeadEndOp) specialOp()   {}
func (o MutexOp) specialOp()     {}
func (o WaitGroupOp) specialOp() {}

// SpecialOps returns a list of all defined special operations.
func SpecialOps() []SpecialOp {
	return []SpecialOp{
		MakeChan, Close,
		DeadEnd,
		MakeMutex, Lock, Unlock, RLock, RUnlock,
		MakeWaitGroup, Add, Wait,
	}
}

// SpecialOpStmt is the common interface of all statements representing
// specially modeled function calls.
type SpecialOpStmt interface {
	Stmt

	// SpecialOp returns the performed operation.
	SpecialOp() SpecialOp
}
