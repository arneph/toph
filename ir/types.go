package ir

import (
	"fmt"
)

// TypeIndex represents the index of a type.
type TypeIndex int

// Type represents the type of a variable
type Type interface {
	fmt.Stringer

	UninitializedValue() Value
	InitializedValue() Value
	VariablePrefix() string

	xType()
}

func (t BasicType) xType()     {}
func (t StructType) xType()    {}
func (t ContainerType) xType() {}

// BasicType is an atomic types.
type BasicType int

const (
	// IntType is the type of an integer variable (used only internally).
	IntType BasicType = iota
	// FuncType is the type of a function variable.
	FuncType
	// ChanType is the type of a channel variable.
	ChanType
	// MutexType is the type of a (rw)mutex variable.
	MutexType
	// WaitGroupType is the type of a wait group variable.
	WaitGroupType
)

// UninitializedValue returns the Uppaal zero value for the given type.
func (t BasicType) UninitializedValue() Value {
	switch t {
	case IntType:
		return Value{0, IntType}
	case FuncType, ChanType, MutexType, WaitGroupType:
		return Value{-1, t}
	default:
		panic(fmt.Errorf("unknown Type: %d", t))
	}
}

// InitializedValue returns the Go zero value for the given type.
func (t BasicType) InitializedValue() Value {
	switch t {
	case IntType:
		return Value{0, IntType}
	case FuncType, ChanType:
		return Value{-1, t}
	case MutexType:
		return InitializedMutex
	case WaitGroupType:
		return InitializedWaitGroup
	default:
		panic(fmt.Errorf("unknown Type: %d", t))
	}
}

// VariablePrefix returns the variable prefix for the given type.
func (t BasicType) VariablePrefix() string {
	switch t {
	case IntType:
		return "num"
	case FuncType:
		return "fid"
	case ChanType:
		return "cid"
	case MutexType:
		return "mid"
	case WaitGroupType:
		return "wid"
	default:
		panic(fmt.Errorf("unknown Type: %d", t))
	}
}

func (t BasicType) String() string {
	switch t {
	case IntType:
		return "Integer"
	case FuncType:
		return "Func"
	case ChanType:
		return "Chan"
	case MutexType:
		return "Mutex"
	case WaitGroupType:
		return "WaitGroup"
	default:
		panic(fmt.Errorf("unknown Type: %d", t))
	}
}

// Field represents a field inside a StructType
type Field struct {
	index      int
	name       string
	t          Type
	isPointer  bool
	isEmbedded bool

	structType *StructType
}

// Index returns the position of the field inside its StructType.
func (f *Field) Index() int {
	return f.index
}

// Name returns the name of the field.
func (f *Field) Name() string {
	return f.name
}

// Type returns the type of the field.
func (f *Field) Type() Type {
	return f.t
}

// IsPointer returns if the value is stored directly or as a pointer.
func (f *Field) IsPointer() bool {
	return f.isPointer
}

// IsEmbedded returns if the field is embedded (unnamed) in the enclosing
// structure.
func (f *Field) IsEmbedded() bool {
	return f.isEmbedded
}

// StructType returns the enclosing structure type.
func (f *Field) StructType() *StructType {
	return f.structType
}

// RequiresDeepCopy returns if copying the field requires copying its field
// deeply.
func (f *Field) RequiresDeepCopy() bool {
	_, ok := f.t.(BasicType)
	return !ok && !f.isPointer
}

// Handle returns a shorthand to uniquely reference the field.
func (f *Field) Handle() string {
	if f.name == "" {
		return fmt.Sprintf("%s_field%d",
			f.t.VariablePrefix(), f.index)
	}
	return fmt.Sprintf("%s_%s",
		f.t.VariablePrefix(), f.name)
}

func (f *Field) String() string {
	return fmt.Sprintf("%s %s", f.name, f.t)
}

// StructType represents a defined structure.
type StructType struct {
	index TypeIndex
	name  string

	fields []*Field
}

func newStructType(index TypeIndex, name string) *StructType {
	t := new(StructType)
	t.index = index
	t.name = name
	t.fields = nil

	return t
}

// Name returns the name of the struct (might be empty).
func (t *StructType) Name() string {
	return t.name
}

// Fields returns the list of fields belonging to the struct.
func (t *StructType) Fields() []*Field {
	return t.fields
}

// AddField adds a new field with the given index, name, and type to the struct.
func (t *StructType) AddField(index int, name string, fieldType Type, isPointer, isEmbedded bool) *Field {
	f := new(Field)
	f.index = index
	f.name = name
	f.t = fieldType
	f.isPointer = isPointer
	f.isEmbedded = isEmbedded
	f.structType = t

	t.fields = append(t.fields, f)

	return f
}

// FindEmbeddedFieldOfType is used to resolve receivers of methods of embedded fields.
func (t *StructType) FindEmbeddedFieldOfType(embeddedFieldType Type) (embeddedFieldsPath []*Field, ok bool) {
	for _, field := range t.fields {
		if !field.isEmbedded {
			continue
		}
		if field.Type() == embeddedFieldType {
			return []*Field{field}, true
		}
	}
	for _, field := range t.fields {
		if !field.isEmbedded {
			continue
		}
		if structType, ok := field.Type().(*StructType); ok {
			subPath, ok := structType.FindEmbeddedFieldOfType(embeddedFieldType)
			if ok {
				return append([]*Field{field}, subPath...), true
			}
		}
	}
	return nil, false
}

// UninitializedValue returns the Uppaal zeor value for the structure.
func (t *StructType) UninitializedValue() Value {
	return Value{-1, t}
}

// InitializedValue returns the Go zero value for the structure.
func (t *StructType) InitializedValue() Value {
	return InitializedStruct(t)
}

// VariablePrefix returns the variable prefix for the given type.
func (t *StructType) VariablePrefix() string {
	if t.name != "" {
		return fmt.Sprintf("s%02d_%s", t.index, t.name)
	}
	return fmt.Sprintf("s%02d", t.index)
}

func (t *StructType) String() string {
	if t.name != "" {
		return fmt.Sprintf("Struct{%d, %s}", t.index, t.name)
	}
	return fmt.Sprintf("Struct{%d}", t.index)
}

// ContainerKind defines if a container is an array, slice, or map.
type ContainerKind int

const (
	// Array is the ContainerKind for an array.
	Array ContainerKind = iota
	// Slice is the ContainerKind for a slice.
	Slice
	// Map is the ContainerKind for a map.
	Map
)

// ContainerType represents an array, slice, or map. Map keys are not modelled.
// Arrays, slices are ordered, maps are unordered.
type ContainerType struct {
	index         TypeIndex
	kind          ContainerKind
	length        int
	elementType   Type
	holdsPointers bool
}

func newContainerType(index TypeIndex, kind ContainerKind, len int, elementType Type, holdsPointers bool) *ContainerType {
	t := new(ContainerType)
	t.index = index
	t.kind = kind
	t.length = len
	t.elementType = elementType
	t.holdsPointers = holdsPointers

	return t
}

// Kind returns if the container represents an array, slice or map.
func (t *ContainerType) Kind() ContainerKind {
	return t.kind
}

// Len returns the static length of the container type (only applicable for
// arrays).
func (t *ContainerType) Len() int {
	return t.length
}

// ElementType returns the type of elements stored in the container.
func (t *ContainerType) ElementType() Type {
	return t.elementType
}

// HoldsPointers returns if values are stored directly or as pointers.
func (t *ContainerType) HoldsPointers() bool {
	return t.holdsPointers
}

// RequiresDeepCopies returns if copying the container requires copying its
// elements deeply.
func (t *ContainerType) RequiresDeepCopies() bool {
	_, ok := t.elementType.(BasicType)
	return !ok && !t.holdsPointers
}

// UninitializedValue returns the Uppaal zero value for the container type.
func (t *ContainerType) UninitializedValue() Value {
	return Value{-1, t}
}

// InitializedValue returns the Go zero value for the container type.
func (t *ContainerType) InitializedValue() Value {
	switch t.kind {
	case Array:
		return InitializedArray(t)
	case Slice, Map:
		return Value{-1, t}
	default:
		panic("unexpected container kind")
	}
}

// VariablePrefix returns the variable prefix for the given type.
func (t *ContainerType) VariablePrefix() string {
	switch t.kind {
	case Array:
		return fmt.Sprintf("a%02d", t.index)
	case Slice:
		return fmt.Sprintf("b%02d", t.index)
	case Map:
		return fmt.Sprintf("m%02d", t.index)
	default:
		panic("unexpected container kind")
	}
}

func (t *ContainerType) String() string {
	switch t.kind {
	case Array:
		return fmt.Sprintf("Array{%d, %s}", t.index, t.elementType.String())
	case Slice:
		return fmt.Sprintf("Slice{%d, %s}", t.index, t.elementType.String())
	case Map:
		return fmt.Sprintf("Map{%d, %s}", t.index, t.elementType.String())
	default:
		panic("unexpected container kind")
	}
}
