package ir

import "strings"

// Program represents an entire go program (consisting of functions).
type Program struct {
	funcs           []*Func
	funcIndexLookup map[FuncIndex]*Func
	funcNameLookup  map[string]*Func

	scope Scope
}

// NewProgram creates a new program.
func NewProgram() *Program {
	p := new(Program)
	p.funcs = nil
	p.funcIndexLookup = make(map[FuncIndex]*Func)
	p.funcNameLookup = make(map[string]*Func)
	p.scope.init()

	return p
}

// Funcs returns all functions in the program.
func (p *Program) Funcs() []*Func {
	return p.funcs
}

// GetNamedFunc returns the function with the given name.
func (p *Program) GetNamedFunc(name string) *Func {
	return p.funcNameLookup[name]
}

// AddNamedFunc adds the given, named function to the program.
func (p *Program) AddNamedFunc(name string, f *Func) {
	p.funcs = append(p.funcs, f)
	p.funcIndexLookup[f.index] = f
	p.funcNameLookup[name] = f
}

// AddUnnamedFunc adds the given, unnamed function to the program.
func (p *Program) AddUnnamedFunc(f *Func) {
	p.funcs = append(p.funcs, f)
	p.funcIndexLookup[f.index] = f
}

// Scope returns the global scope of the program.
func (p *Program) Scope() *Scope {
	return &p.scope
}

func (p *Program) String() string {
	str := "prog{\n"
	str += "  " + strings.ReplaceAll(p.scope.String(), "\n", "\n  ") + "\n"
	str += "  funcs{\n"
	for _, f := range p.funcs {
		str += "    " + strings.ReplaceAll(f.String(), "\n", "\n    ") + "\n"
	}
	str += "  }\n"
	str += "}"
	return str
}
