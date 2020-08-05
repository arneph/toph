package ir

import (
	"fmt"
	"math"
)

// Value represents the value of a variable
type Value struct {
	v int64
	t Type
}

var (
	// InitializedMutex is a placeholder initial value for mutexes.
	InitializedMutex = Value{math.MinInt64 + 0, MutexType}
	// InitializedWaitGroup is a placeholder initial value for wait groups.
	InitializedWaitGroup = Value{math.MinInt64 + 1, WaitGroupType}

	initializedStruct int64 = math.MinInt64 + 2
	initializedArray  int64 = math.MinInt64 + 3

	// AppendIndex is the value used to indicate a slice append.
	AppendIndex = Value{math.MinInt64 + 4, IntType}
	// RandomIndex is fallback value for container accesses.
	RandomIndex = Value{math.MinInt64 + 5, IntType}

	// Nil represents an untyped nil
	Nil = Value{math.MinInt64 + 6, nil}
)

// InitializedStruct returns a placeholder initial value for structure types.
func InitializedStruct(t *StructType) Value {
	return Value{initializedStruct, t}
}

// IsInitializedStruct returns if the value is a placeholder initial value for a struct.
func (v Value) IsInitializedStruct() bool {
	return v.v == initializedStruct
}

// InitializedArray returns a placeholder initial value for array types.
func InitializedArray(t *ContainerType) Value {
	return Value{initializedArray, t}
}

// IsInitializedArray returns if the value is a placeholder initial value for an array.
func (v Value) IsInitializedArray() bool {
	return v.v == initializedArray
}

// MakeValue creates a new value for the given int value and type.
func MakeValue(v int64, t Type) Value {
	return Value{v, t}
}

// Value returns the underlying int value of the value.
func (v Value) Value() int64 {
	return v.v
}

// Type returns the type of the value.
func (v Value) Type() Type {
	return v.t
}

func (v Value) String() string {
	switch v.v {
	case InitializedMutex.v:
		return "initialized mutex"
	case InitializedWaitGroup.v:
		return "initialized wait group"
	case initializedStruct:
		return "initialized struct"
	case initializedArray:
		return "initialied array"
	case AppendIndex.v:
		return "append index"
	case RandomIndex.v:
		return "random index"
	default:
		return fmt.Sprintf("%d", v.v)
	}
}

// FieldSelection represents a read or write access to a structure field.
type FieldSelection struct {
	structVal LValue
	field     *Field
}

// NewFieldSelection creates a new field selection for the given structure
// variable and field.
func NewFieldSelection(structVal LValue, field *Field) *FieldSelection {
	if structVal == nil {
		panic("attempted to create FieldSelection with nil struct variable")
	} else if field == nil {
		panic("attempted to create FieldSelection with nil field")
	} else if field.structType != structVal.Type() {
		panic("attempted to create FieldSelection for field not in struct variable type")
	}

	fs := new(FieldSelection)
	fs.structVal = structVal
	fs.field = field

	return fs
}

// StructVal returns the lvalue holding the accessed structure.
func (fs *FieldSelection) StructVal() LValue {
	return fs.structVal
}

// StructType returns the type of the accessed structure.
func (fs *FieldSelection) StructType() *StructType {
	return fs.field.structType
}

// Field returns the field of the accessed structure.
func (fs *FieldSelection) Field() *Field {
	return fs.field
}

// Name returns the qualified name of the field.
func (fs *FieldSelection) Name() string {
	return fs.structVal.Name() + "_" + fs.field.name
}

// Type returns the type of accessed field and the field selection overall.
func (fs *FieldSelection) Type() Type {
	return fs.field.t
}

// Handle returns a shorthand to uniquely reference the field selection.
func (fs *FieldSelection) Handle() string {
	return fs.structVal.Handle() + "_" + fs.field.Handle()
}

func (fs *FieldSelection) String() string {
	return fs.Handle()
}

// AccessKind defines if a container access is a read or a write.
type AccessKind int

const (
	// Read defines a read access of a container
	Read AccessKind = iota
	// Write defines a write access of a container
	Write
)

// ContainerAccess represents a read or write access to a container.
type ContainerAccess struct {
	containerVal LValue
	index        RValue
	kind         AccessKind
}

// NewContainerAccess creates a new container access for the given container
// variable.
func NewContainerAccess(containerVal LValue, index RValue) *ContainerAccess {
	if containerVal == nil {
		panic("attempted to create ContainerAccess with nil struct variable")
	}

	ca := new(ContainerAccess)
	ca.containerVal = containerVal
	ca.index = index
	ca.kind = Read

	return ca
}

// ContainerVal returns the lvalue holding the accessed container.
func (ca *ContainerAccess) ContainerVal() LValue {
	return ca.containerVal
}

// ContainerType returns the type of the accessed container.
func (ca *ContainerAccess) ContainerType() *ContainerType {
	return ca.containerVal.Type().(*ContainerType)
}

// Index returns the accessed index value.
func (ca *ContainerAccess) Index() RValue {
	return ca.index
}

// Kind returns the access kind of the container access (read or write).
func (ca *ContainerAccess) Kind() AccessKind {
	return ca.kind
}

// SetKind rets the access kind of the container access (read or write).
func (ca *ContainerAccess) SetKind(kind AccessKind) {
	ca.kind = kind
}

// IsSliceAppend returns if the container access is a slice append. This is a
// convenience method.
func (ca *ContainerAccess) IsSliceAppend() bool {
	return ca.Index() == AppendIndex
}

// IsMapRead returns if the container access is a map read. This is a
// convenience method.
func (ca *ContainerAccess) IsMapRead() bool {
	return ca.ContainerType().Kind() == Map && ca.Kind() == Read
}

// IsMapWrite returns if the container access is a map write. This is a
// convenience method.
func (ca *ContainerAccess) IsMapWrite() bool {
	return ca.ContainerType().Kind() == Map && ca.Kind() == Write
}

// Name returns the qualified name of the field.
func (ca *ContainerAccess) Name() string {
	return ca.containerVal.Name() + "_elem"
}

// Type returns the element type of the accessed container.
func (ca *ContainerAccess) Type() Type {
	return ca.ContainerType().ElementType()
}

// Handle returns a shorthand to uniquely reference the container access.
func (ca *ContainerAccess) Handle() string {
	return ca.containerVal.Handle() + "_elem"
}

func (ca *ContainerAccess) String() string {
	return ca.Handle()
}

// RValue represents a value that can be assigned from.
type RValue interface {
	fmt.Stringer

	Type() Type

	rvalue()
}

func (v Value) rvalue()             {}
func (v *Variable) rvalue()         {}
func (fs *FieldSelection) rvalue()  {}
func (ca *ContainerAccess) rvalue() {}

// LValue represents a storage location that can be assigned to.
type LValue interface {
	fmt.Stringer

	Name() string
	Type() Type
	Handle() string

	lvalue()
}

func (v *Variable) lvalue()         {}
func (fs *FieldSelection) lvalue()  {}
func (ca *ContainerAccess) lvalue() {}
