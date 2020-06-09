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

// GetVariable does name resolution for the given name and returns the
// associated variable and its scope.
func (s *Scope) GetVariable(name string) (*Variable, *Scope) {
	v, ok := s.variableLookup[name]
	if ok {
		return v, s
	}
	if s.superScope != nil {
		return s.superScope.GetVariable(name)
	}
	return nil, nil
}

// AddVariable adds the given variable to the scope.
func (s *Scope) AddVariable(v *Variable) {
	if v == nil {
		panic("tried to add nil variable to scope")
	}
	s.variables = append(s.variables, v)
	s.variableLookup[v.name] = v
}

func (s *Scope) String() string {
	str := "scope{\n"
	for _, v := range s.variables {
		str += "\t" + strings.ReplaceAll(v.String(), "\n", "\n\t") + "\n"
	}
	str += "}"
	return str
}
