package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// MutexOp represents an operation performed on a sync.Mutex or sync.RWMutex.
type MutexOp int

const (
	// Lock represents a sync.(RW)Mutex.Lock() operation.
	Lock MutexOp = iota
	// Unlock represents a sync.(RW)Mutex.Unlock() operation.
	Unlock
	// RLock represents a sync.RWMutex.RLock() operation.
	RLock
	// RUnlock represents a sync.RWMutex.RUnlock() operation.
	RUnlock
)

func (o MutexOp) String() string {
	switch o {
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

// MutexOpStmt represents a sync.(RW)Mutex operation statement.
type MutexOpStmt struct {
	mutex LValue
	op    MutexOp

	Node
}

// NewMutexOpStmt creates a new mutex operation statement for the given muxtex
// and with the given mutex operation.
func NewMutexOpStmt(mutex LValue, op MutexOp, pos, end token.Pos) *MutexOpStmt {
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
	// Add represents a sync.WaitGroup.Add operation.
	Add WaitGroupOp = iota
	// Wait represents a sync.WaitGroup.Wait operation.
	Wait
)

func (o WaitGroupOp) String() string {
	switch o {
	case Add:
		return "add"
	case Wait:
		return "wait"
	default:
		panic(fmt.Sprintf("unknown WaitGroupOp: %d", o))
	}
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

// OnceOp represents an operation performed on a wait group.
type OnceOp int

const (
	// Do represents a sync.Once.Do operation.
	Do OnceOp = iota
)

func (o OnceOp) String() string {
	return "do"
}

// OnceDoStmt represents a sync.Once.Do call.
type OnceDoStmt struct {
	once LValue
	f    RValue

	Node
}

// NewOnceDoStmt creates a new once do statement for the given once and
// function value.
func NewOnceDoStmt(once LValue, f RValue, pos, end token.Pos) *OnceDoStmt {
	s := new(OnceDoStmt)
	s.once = once
	s.f = f
	s.pos = pos
	s.end = end

	return s
}

// Once returns the once that is called on.
func (s *OnceDoStmt) Once() LValue {
	return s.once
}

// F returns the function that may or may not get called by once.
func (s *OnceDoStmt) F() RValue {
	return s.f
}

// SpecialOp returns the operation performed on the once.
func (s *OnceDoStmt) SpecialOp() SpecialOp {
	return Do
}

func (s *OnceDoStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "once_do %s %s", s.once.Handle(), s.f.String())
}
