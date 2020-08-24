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
	containerLen       RValue
	initializeElements bool

	Node
}

// NewMakeContainerStmt creates a new MakeContainerStmt for the given container value.
func NewMakeContainerStmt(containerVar *Variable, containerLen RValue, initializeElements bool, pos, end token.Pos) *MakeContainerStmt {
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
func (s *MakeContainerStmt) ContainerLen() RValue {
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

// CopySliceStmt represents a copy(dst, src []T) call.
type CopySliceStmt struct {
	dstVal LValue
	srcVal LValue

	Node
}

// NewCopySliceStmt creats a new CopySliceStmt for the given source and
// destination slice values.
func NewCopySliceStmt(dstVal, srcVal LValue, pos, end token.Pos) *CopySliceStmt {
	if dstVal.Type() != srcVal.Type() {
		panic("attempted to create slice copy stmt between different slice types")
	}
	s := new(CopySliceStmt)
	s.dstVal = dstVal
	s.srcVal = srcVal
	s.pos = pos
	s.end = end

	return s
}

// DestinationVal returns the lvalue holding the slice to copy to.
func (s *CopySliceStmt) DestinationVal() LValue {
	return s.dstVal
}

// SourceVal returns the lvalue holding the slice to copy from.
func (s *CopySliceStmt) SourceVal() LValue {
	return s.srcVal
}

// SliceType returns the type of the slice to copy from and to.
func (s *CopySliceStmt) SliceType() *ContainerType {
	return s.dstVal.Type().(*ContainerType)
}

func (s *CopySliceStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "copy(%s, %s)", s.dstVal.Handle(), s.srcVal.Handle())
}

// DeleteMapEntryStmt represents a delete(map[K]V, k) call.
type DeleteMapEntryStmt struct {
	mapVal LValue

	Node
}

// NewDeleteMapEntryStmt creates a new DeleteMapEntryStmt for the given map value.
func NewDeleteMapEntryStmt(mapVal LValue, pos, end token.Pos) *DeleteMapEntryStmt {
	s := new(DeleteMapEntryStmt)
	s.mapVal = mapVal
	s.pos = pos
	s.end = end

	return s
}

// MapVal returns the lvalue holding the map to delete an entry from.
func (s *DeleteMapEntryStmt) MapVal() LValue {
	return s.mapVal
}

// MapType returns the type of the map to delete an entry from.
func (s *DeleteMapEntryStmt) MapType() *ContainerType {
	return s.mapVal.Type().(*ContainerType)
}

func (s *DeleteMapEntryStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "delete(%s)", s.mapVal.Handle())
}
