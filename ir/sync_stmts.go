package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// MutexOp represents an operation performed on a sync.Mutex or sync.RWMutex.
type MutexOp int

const (
	// MakeMutex is a mutex creation operation, executed for every variable declaration.
	MakeMutex MutexOp = iota
	// Lock represents a sync.(RW)Mutex.Lock() operation.
	Lock
	// Unlock represents a sync.(RW)Mutex.Unlock() operation.
	Unlock
	// RLock represents a sync.RWMutex.RLock() operation.
	RLock
	// RUnlock represents a sync.RWMutex.RUnlock() operation.
	RUnlock
)

func (o MutexOp) String() string {
	switch o {
	case MakeMutex:
		return "make_mutex"
	case Lock:
		return "lock"
	case Unlock:
		return "unlock"
	case RLock:
		return "rlock"
	case RUnlock:
		return "runlock"
	default:
		panic(fmt.Sprintf("unknown MutexOp: %d", o))
	}
}

// MakeMutexStmt is a mutex creation statement, executed for every variable declaration.
type MakeMutexStmt struct {
	mutex *Variable

	Node
}

// NewMakeMutexStmt creates a new MakeMutexStmt for the given mutex.
func NewMakeMutexStmt(mutex *Variable, pos, end token.Pos) *MakeMutexStmt {
	s := new(MakeMutexStmt)
	s.mutex = mutex
	s.pos = pos
	s.end = end

	return s
}

// Mutex returns the lvalue holding the newly created mutex.
func (s *MakeMutexStmt) Mutex() *Variable {
	return s.mutex
}

// SpecialOp returns the performed operation (always MakeMutex).
func (s *MakeMutexStmt) SpecialOp() SpecialOp {
	return MakeMutex
}

func (s *MakeMutexStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s <- make(mutex)", s.mutex.Handle())
}

// MutexOpStmt represents a sync.(RW)Mutex operation statement.
type MutexOpStmt struct {
	mutex LValue
	op    MutexOp

	Node
}

// NewMutexOpStmt creates a new mutex operation statement for the given muxtex
// and with the given mutex operation.
func NewMutexOpStmt(mutex LValue, op MutexOp, pos, end token.Pos) *MutexOpStmt {
	if op == MakeMutex {
		panic("attempted to create MutexOpStmt with MakeMutex Operation")
	}

	s := new(MutexOpStmt)
	s.mutex = mutex
	s.op = op
	s.pos = pos
	s.end = end

	return s
}

// Mutex returns the mutex that is operted on.
func (s *MutexOpStmt) Mutex() LValue {
	return s.mutex
}

// Op returns the operation performed on the mutex.
func (s *MutexOpStmt) Op() MutexOp {
	return s.op
}

// SpecialOp returns the operation performed on the mutex.
func (s *MutexOpStmt) SpecialOp() SpecialOp {
	return s.op
}

func (s *MutexOpStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s %s", s.op, s.mutex.Handle())
}

// WaitGroupOp represents an operation performed on a wait group.
type WaitGroupOp int

const (
	// MakeWaitGroup is a wait group creation operation, executed for every variable declaration.
	MakeWaitGroup WaitGroupOp = iota
	// Add represents a sync.WaitGroup.Add operation.
	Add
	// Wait represents a sync.WaitGroup.Wait operation.
	Wait
)

func (o WaitGroupOp) String() string {
	switch o {
	case MakeWaitGroup:
		return "make_wait_group"
	case Add:
		return "add"
	case Wait:
		return "wait"
	default:
		panic(fmt.Sprintf("unknown WaitGroupOp: %d", o))
	}
}

// MakeWaitGroupStmt is a wait group creation statement, executed for every variable declaration.
type MakeWaitGroupStmt struct {
	waitGroup *Variable

	Node
}

// NewMakeWaitGroupStmt creates a new MakeWaitGroupStmt for the given wait group.
func NewMakeWaitGroupStmt(waitGroup *Variable, pos, end token.Pos) *MakeWaitGroupStmt {
	s := new(MakeWaitGroupStmt)
	s.waitGroup = waitGroup
	s.pos = pos
	s.end = end

	return s
}

// WaitGroup returns the lvalue holding the newly created wait group.
func (s *MakeWaitGroupStmt) WaitGroup() *Variable {
	return s.waitGroup
}

// SpecialOp returns the performed operation (always MakeWaitGroup).
func (s *MakeWaitGroupStmt) SpecialOp() SpecialOp {
	return MakeWaitGroup
}

func (s *MakeWaitGroupStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s <- make(wait group)", s.waitGroup.Handle())
}

// WaitGroupOpStmt represents a sync.WaitGroup operation statement.
type WaitGroupOpStmt struct {
	waitGroup LValue
	op        WaitGroupOp
	delta     RValue // only applicable for Add op

	Node
}

// NewWaitGroupOpStmt creates a new wait group operation satement for the given
// wait group and with the given wait group operation.
func NewWaitGroupOpStmt(waitGroup LValue, op WaitGroupOp, delta RValue, pos, end token.Pos) *WaitGroupOpStmt {
	if op == MakeWaitGroup {
		panic("attempted to create WaitGroupOpStmt with MakeWaitGroup Operation")
	}

	s := new(WaitGroupOpStmt)
	s.waitGroup = waitGroup
	s.op = op
	s.delta = delta
	s.pos = pos
	s.end = end

	return s
}

// WaitGroup returns the wait group that is operated on.
func (s *WaitGroupOpStmt) WaitGroup() LValue {
	return s.waitGroup
}

// Op returns the operation performed on the wait group.
func (s *WaitGroupOpStmt) Op() WaitGroupOp {
	return s.op
}

// SpecialOp returns the operation performed on the wait group.
func (s *WaitGroupOpStmt) SpecialOp() SpecialOp {
	return s.op
}

// Delta returns the argument for sync.WaitGroup.Add, if applicable.
func (s *WaitGroupOpStmt) Delta() RValue {
	return s.delta
}

func (s *WaitGroupOpStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	if s.op == Add {
		fmt.Fprintf(b, "%s %s %s", s.op, s.waitGroup.Handle(), s.delta)
		return
	}
	fmt.Fprintf(b, "%s %s", s.op, s.waitGroup.String())
}
