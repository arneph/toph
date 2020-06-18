package ir

import (
	"go/token"
	"go/types"
	"strings"
)

// Program represents an entire go program (consisting of functions).
type Program struct {
	funcs      []*Func
	funcLookup map[FuncIndex]*Func
	entryFunc  *Func
	funcCount  int

	scope         Scope
	variableCount int

	fset *token.FileSet
}

// NewProgram creates a new program.
func NewProgram(fset *token.FileSet) *Program {
	p := new(Program)
	p.funcs = nil
	p.funcLookup = make(map[FuncIndex]*Func)
	p.entryFunc = nil
	p.funcCount = 0
	p.scope.init()
	p.fset = fset

	return p
}

// Funcs returns all functions in the program.
func (p *Program) Funcs() []*Func {
	return p.funcs
}

// Func returns the function with the given FuncIndex.
func (p *Program) Func(index FuncIndex) *Func {
	return p.funcLookup[index]
}

// EntryFunc returns the entry function of the program.
func (p *Program) EntryFunc() *Func {
	return p.entryFunc
}

// SetEntryFunc sets the entry function of the program.
func (p *Program) SetEntryFunc(f *Func) {
	p.entryFunc = f
}

// AddOuterFunc adds a new, non-inner, empty function to the program and returns
// the new function.
func (p *Program) AddOuterFunc(name string, signature *types.Signature, pos, end token.Pos) *Func {
	fIndex := FuncIndex(p.funcCount)
	f := newOuterFunc(fIndex, name, signature, &p.scope, pos, end)
	p.funcCount++

	p.funcs = append(p.funcs, f)
	p.funcLookup[fIndex] = f

	return f
}

// AddInnerFunc adds a new, inner, empty function to the program and returns
// the new function.
func (p *Program) AddInnerFunc(signature *types.Signature, enclosingFunc *Func, enclosingScope *Scope, pos, end token.Pos) *Func {
	fIndex := FuncIndex(p.funcCount)
	f := newInnerFunc(fIndex, signature, enclosingFunc, enclosingScope, pos, end)
	p.funcCount++

	p.funcs = append(p.funcs, f)
	p.funcLookup[fIndex] = f

	return f
}

// RemoveFuncs removes the given (old) functions from the program.
func (p *Program) RemoveFuncs(oldFuncs map[*Func]bool) {
	c := 0
	for i := 0; i < len(p.funcs); i++ {
		if oldFuncs[p.funcs[i]] {
			continue
		}
		p.funcs[c] = p.funcs[i]
		c++
	}
	p.funcs = p.funcs[:c]

	for oldFunc, ok := range oldFuncs {
		if !ok {
			continue
		}
		delete(p.funcLookup, oldFunc.index)
	}

	if oldFuncs[p.entryFunc] {
		p.entryFunc = nil
	}
}

// Scope returns the global scope of the program.
func (p *Program) Scope() *Scope {
	return &p.scope
}

// NewVariable creates a new variable with the given arguments. The new
// variable is not part of any scope.
func (p *Program) NewVariable(name string, t Type, initialValue Value) *Variable {
	v := newVariable(VariableIndex(p.variableCount), name, t, initialValue)
	p.variableCount++

	return v
}

// FileSet returns the token.FileSet from which the program was built.
func (p *Program) FileSet() *token.FileSet {
	return p.fset
}

func (p *Program) String() string {
	str := "prog{\n"
	str += "\t" + strings.ReplaceAll(p.scope.String(), "\n", "\n\t") + "\n"
	str += "\tfuncs{\n"
	for _, f := range p.funcs {
		str += "\t\t" + strings.ReplaceAll(f.String(), "\n", "\n\t\t") + "\n"
	}
	str += "\t}\n"
	str += "}"
	return str
}
