package ir

import (
	"fmt"
)

// Value represents the value of a variable
type Value int64

func (v Value) String() string {
	return fmt.Sprintf("%d", v)
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

// RValue represents a value that can be assigned from.
type RValue interface {
	fmt.Stringer

	rvalue()
}

func (v Value) rvalue()            {}
func (v *Variable) rvalue()        {}
func (fs *FieldSelection) rvalue() {}

// LValue represents a storage location that can be assigned to.
type LValue interface {
	fmt.Stringer

	Name() string
	Type() Type
	Handle() string

	lvalue()
}

func (v *Variable) lvalue()        {}
func (fs *FieldSelection) lvalue() {}
