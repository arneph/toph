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
		return "make"
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

// Mutex returns the variable holding the newly created mutex.
func (s *MakeMutexStmt) Mutex() *Variable {
	return s.mutex
}

// SpecialOp returns the performed operation (always MakeMutex).
func (s *MakeMutexStmt) SpecialOp() SpecialOp {
	return MakeMutex
}

// CallKind returns the call kind (always Call) of the mutex creation statement.
func (s *MakeMutexStmt) CallKind() CallKind {
	return Call
}

func (s *MakeMutexStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s <- make(mutex)", s.mutex.Handle())
}

// MutexOpStmt represents a sync.(RW)Mutex operation statement.
type MutexOpStmt struct {
	mutex    *Variable
	op       MutexOp
	callKind CallKind // Call or Defer

	Node
}

// NewMutexOpStmt creates a new mutex operation statement for the given muxtex
// and with the given mutex operation.
func NewMutexOpStmt(mutex *Variable, op MutexOp, callKind CallKind, pos, end token.Pos) *MutexOpStmt {
	if op == MakeMutex {
		panic("attempted to create MutexOpStmt with MakeMutex Operation")
	}
	if callKind == Go {
		panic("attempted to create MutexOpStmt with Go CallKind")
	}

	s := new(MutexOpStmt)
	s.mutex = mutex
	s.op = op
	s.callKind = callKind
	s.pos = pos
	s.end = end

	return s
}

// Mutex returns the variable holding the mutex that is operted on.
func (s *MutexOpStmt) Mutex() *Variable {
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

// CallKind returns the call kind (regular or deferred) of the mutex operation.
func (s *MutexOpStmt) CallKind() CallKind {
	return s.callKind
}

func (s *MutexOpStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s %s %s", s.callKind, s.op, s.mutex.Handle())
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
		return "make"
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

// WaitGroup returns the variable holding the newly created wait group.
func (s *MakeWaitGroupStmt) WaitGroup() *Variable {
	return s.waitGroup
}

// SpecialOp returns the performed operation (always MakeWaitGroup).
func (s *MakeWaitGroupStmt) SpecialOp() SpecialOp {
	return MakeWaitGroup
}

// CallKind returns the call kind (always Call) of the wait group creation statement.
func (s *MakeWaitGroupStmt) CallKind() CallKind {
	return Call
}

func (s *MakeWaitGroupStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s <- make(wait group)", s.waitGroup.Handle())
}

// WaitGroupOpStmt represents a sync.WaitGroup operation statement.
type WaitGroupOpStmt struct {
	waitGroup *Variable
	op        WaitGroupOp
	delta     int      // only applicable for Add op
	callKind  CallKind // Call or Defer

	Node
}

// NewWaitGroupOpStmt creates a new wait group operation satement for the given
// wait group and with the given wait group operation.
func NewWaitGroupOpStmt(waitGroup *Variable, op WaitGroupOp, delta int, callKind CallKind, pos, end token.Pos) *WaitGroupOpStmt {
	if op == MakeWaitGroup {
		panic("attempted to create WaitGroupOpStmt with MakeWaitGroup Operation")
	}
	if callKind == Go {
		panic("attempted to create WaitGroupOpStmt with Go CallKind")
	}

	s := new(WaitGroupOpStmt)
	s.waitGroup = waitGroup
	s.op = op
	s.delta = delta
	s.callKind = callKind
	s.pos = pos
	s.end = end

	return s
}

// WaitGroup returns the variable holding the wait group that is operated on.
func (s *WaitGroupOpStmt) WaitGroup() *Variable {
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
func (s *WaitGroupOpStmt) Delta() int {
	return s.delta
}

// CallKind returns the call kind (regular or deferred) of the wait group operation.
func (s *WaitGroupOpStmt) CallKind() CallKind {
	return s.callKind
}

func (s *WaitGroupOpStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	if s.op == Add {
		fmt.Fprintf(b, "%s %s %s %d", s.callKind, s.op, s.waitGroup.Handle(), s.delta)
	}
	fmt.Fprintf(b, "%s %s %s", s.callKind, s.op, s.waitGroup.String())
}
