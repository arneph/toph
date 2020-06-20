package ir

import (
	"strings"
)

// Scope holds the function and channel declarations of Program and Func.
type Scope struct {
	superScope *Scope

	variables      []*Variable
	variableLookup map[string]*Variable
}

func (s *Scope) init() {
	s.variableLookup = make(map[string]*Variable)
}

// SuperScope returns the immediate super scope of the scope.
func (s *Scope) SuperScope() *Scope {
	return s.superScope
}

// IsSuperScopeOf returns whether the scope is a super scope of the given scope.
func (s *Scope) IsSuperScopeOf(t *Scope) bool {
	for ; t != nil; t = t.superScope {
		if t.superScope == s {
			return true
		}
	}
	return false
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
	s.variableLookup[v.name] = v
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
