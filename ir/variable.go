package ir

import "fmt"

// Type represents the type of a variable
type Type int

const (
	// FuncType is the type of a function variable.
	FuncType Type = iota
	// ChanType is the type of a channel variable.
	ChanType
)

// VariablePrefix returns the variable prefix for the given type.
func (t Type) VariablePrefix() string {
	switch t {
	case FuncType:
		return "fid"
	case ChanType:
		return "cid"
	default:
		panic(fmt.Errorf("unknown Type: %d", t))
	}
}

func (t Type) String() string {
	switch t {
	case FuncType:
		return "Func"
	case ChanType:
		return "Chan"
	default:
		panic(fmt.Errorf("unknown Type: %d", t))
	}
}

// Value represents the value of a variable
type Value int

func (v Value) String() string {
	return fmt.Sprintf("%d", v)
}

// VariableIndex represents the index of a variable.
type VariableIndex int

var variableCount int

// Variable represents a variable or constant in Go source code.
type Variable struct {
	index        VariableIndex
	name         string
	t            Type
	initialValue Value
	captured     bool
}

// NewVariable creates a new variable with the given arguments.
func NewVariable(name string, t Type, initialValue Value) *Variable {
	v := new(Variable)
	v.index = VariableIndex(variableCount)
	v.name = name
	v.t = t
	v.initialValue = initialValue
	v.captured = false

	variableCount++

	return v
}

// Name returns the name of the variable.
func (v *Variable) Name() string {
	return v.name
}

// Type returns the type of the variable.
func (v *Variable) Type() Type {
	return v.t
}

// InitialValue returns the value assigned to the variable with its
// declaration.
func (v *Variable) InitialValue() Value {
	return v.initialValue
}

// IsCaptured returns whether the variable gets captured by any inner
// functions.
func (v *Variable) IsCaptured() bool {
	return v.captured
}

// SetCaptured sets whether the variabled gets captured by any inner functions.
func (v *Variable) SetCaptured(captured bool) {
	v.captured = captured
}

func (v *Variable) String() string {
	return fmt.Sprintf("var %s %v = %d", v.Handle(), v.t, v.initialValue)
}

// Handle returns a shorthand to reference the variable.
func (v *Variable) Handle() string {
	if v.name == "" {
		return fmt.Sprintf("%s_var%d",
			v.t.VariablePrefix(), v.index)
	}
	return fmt.Sprintf("%s_var%d_%s",
		v.t.VariablePrefix(), v.index, v.name)
}

// RValue represents a value that can be assigned to a variable, either an
// ir.Value or an ir.Variable.
type RValue interface {
	fmt.Stringer

	rvalue()
}

func (v Value) rvalue()     {}
func (v *Variable) rvalue() {}
