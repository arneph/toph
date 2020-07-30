package ir

import (
	"fmt"
	"strings"
)

// VariableIndex represents the index of a variable.
type VariableIndex int

// Variable represents a variable or constant in Go source code.
type Variable struct {
	index        VariableIndex
	name         string
	t            Type
	initialValue Value
	captured     bool
	scope        *Scope
}

func newVariable(index VariableIndex, name string, t Type, initialValue Value) *Variable {
	v := new(Variable)
	v.index = index
	v.name = name
	v.t = t
	v.initialValue = initialValue
	v.captured = false
	v.scope = nil

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

// Scope returns the Scope containing the variable (possibly nil).
func (v *Variable) Scope() *Scope {
	return v.scope
}

// Handle returns a shorthand to uniquely reference the variable.
func (v *Variable) Handle() string {
	if v.name == "" {
		return fmt.Sprintf("%s_var%d",
			v.t.VariablePrefix(), v.index)
	}
	return fmt.Sprintf("%s_var%d_%s",
		v.t.VariablePrefix(), v.index, v.name)
}

func (v *Variable) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString(fmt.Sprintf("var %s %v = %s", v.Handle(), v.t, v.initialValue))
}

func (v *Variable) String() string {
	return v.Handle()
}
