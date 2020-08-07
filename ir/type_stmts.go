package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// MakeStructStmt represents a new(T) call, a struct literal, or a struct
// variable initialization if the struct is stored by value.
type MakeStructStmt struct {
	structVar        *Variable
	initializeFields bool

	Node
}

// NewMakeStructStmt creates a new MakeStructStmt for the given structure value.
func NewMakeStructStmt(structVar *Variable, initializeFields bool, pos, end token.Pos) *MakeStructStmt {
	s := new(MakeStructStmt)
	s.structVar = structVar
	s.initializeFields = initializeFields
	s.pos = pos
	s.end = end

	return s
}

// StructVar returns the variable holding the newly allocated structure.
func (s *MakeStructStmt) StructVar() *Variable {
	return s.structVar
}

// StructType returns the type of the newly allocated structure.
func (s *MakeStructStmt) StructType() *StructType {
	return s.structVar.Type().(*StructType)
}

// InitialzeFields returns if the statement initializes the fields of the new
// struct instance.
func (s *MakeStructStmt) InitialzeFields() bool {
	return s.initializeFields
}

func (s *MakeStructStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	arg2 := "initialized"
	if !s.initializeFields {
		arg2 = "uninitialized"
	}
	fmt.Fprintf(b, "%s <- make(%s, %s)", s.structVar.Handle(), s.structVar.Type(), arg2)
}

// MakeContainerStmt represents a make([]T), make(map[T]U) call or a container literal.
type MakeContainerStmt struct {
	containerVar       *Variable
	containerLen       int
	initializeElements bool

	Node
}

// NewMakeContainerStmt creates a new MakeContainerStmt for the given container value.
func NewMakeContainerStmt(containerVar *Variable, containerLen int, initializeElements bool, pos, end token.Pos) *MakeContainerStmt {
	s := new(MakeContainerStmt)
	s.containerVar = containerVar
	s.containerLen = containerLen
	s.initializeElements = initializeElements
	s.pos = pos
	s.end = end

	return s
}

// ContainerVar returns the lvalue holding the newly allocated container.
func (s *MakeContainerStmt) ContainerVar() *Variable {
	return s.containerVar
}

// ContainerLen returns the length of the newly allocated container.
func (s *MakeContainerStmt) ContainerLen() int {
	return s.containerLen
}

// ContainerType returns the type of the newly allocated structure.
func (s *MakeContainerStmt) ContainerType() *ContainerType {
	return s.containerVar.Type().(*ContainerType)
}

// InitializeElements returns if the statement initializes the elements of the
// new container instance.
func (s *MakeContainerStmt) InitializeElements() bool {
	return s.initializeElements
}

func (s *MakeContainerStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	arg2 := "initialized"
	if !s.initializeElements {
		arg2 = "uninitialized"
	}
	fmt.Fprintf(b, "%s <- make(%s, %s)", s.containerVar.Handle(), s.containerVar.Type(), arg2)
}
