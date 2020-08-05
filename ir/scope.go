package ir

import (
	"strings"
)

// Scope holds the function and channel declarations of Program and Func.
type Scope struct {
	parent   *Scope
	children []*Scope

	variables []*Variable
}

// Parent returns the immediate super scope of the scope.
func (s *Scope) Parent() *Scope {
	return s.parent
}

// Children returns all immediate sub scopes of the scope.
func (s *Scope) Children() []*Scope {
	return s.children
}

// IsParentOf returns whether the scope is a super scope of the given scope.
func (s *Scope) IsParentOf(t *Scope) bool {
	for ; t != nil; t = t.parent {
		if t.parent == s {
			return true
		}
	}
	return false
}

func (s *Scope) addChild(t *Scope) {
	if t.parent != nil {
		panic("tired to add scope as child twice")
	}
	s.children = append(s.children, t)
	t.parent = s
}

// Variables returns all variables in the scope.
func (s *Scope) Variables() []*Variable {
	return s.variables
}

// AddVariable adds the given variable to the scope.
func (s *Scope) AddVariable(v *Variable) {
	if v == nil {
		panic("tried to add nil variable to scope")
	} else if v.scope != nil {
		panic("tried to add variable to more than one scope")
	}
	s.variables = append(s.variables, v)
	v.scope = s
}

func (s *Scope) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	b.WriteString("scope{\n")
	for _, v := range s.variables {
		v.tree(b, indent+1)
		b.WriteString("\n")
	}
	writeIndent(b, indent)
	b.WriteString("}")
}
