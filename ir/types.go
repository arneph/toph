package ir

import (
	"fmt"
)

// TypeIndex represents the index of a type.
type TypeIndex int

// Type represents the type of a variable
type Type interface {
	fmt.Stringer

	VariablePrefix() string

	xType()
}

func (t BasicType) xType()  {}
func (t StructType) xType() {}

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
	index        int
	name         string
	t            Type
	isPointer    bool
	isEmbedded   bool
	initialValue Value

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

// IsPointer returns if the value is stored directly or as a pointer (only
// relevant for structures).
func (f *Field) IsPointer() bool {
	return f.isPointer
}

// IsEmbedded returns if the field is embedded (unnamed) in the enclosing
// structure.
func (f *Field) IsEmbedded() bool {
	return f.isEmbedded
}

// InitialValue returns the value assigned to the field with its
// declaration.
func (f *Field) InitialValue() Value {
	return f.initialValue
}

// StructType returns the enclosing structure type.
func (f *Field) StructType() *StructType {
	return f.structType
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
func (t *StructType) AddField(index int, name string, fieldType Type, isPointer, isEmbedded bool, initialValue Value) *Field {
	f := new(Field)
	f.index = index
	f.name = name
	f.t = fieldType
	f.isPointer = isPointer
	f.isEmbedded = isEmbedded
	f.initialValue = initialValue
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
