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

// GetFunc returns the function with the given name.
func (p *Program) GetFunc(name string) *Func {
	return p.funcNameLookup[name]
}

// AddFunc adds the given function to the program.
func (p *Program) AddFunc(f *Func) {
	p.funcs = append(p.funcs, f)
	p.funcIndexLookup[f.index] = f
	p.funcNameLookup[f.name] = f
}

// AddFuncs adds all the given functions to the program.
func (p *Program) AddFuncs(funcs []*Func) {
	for _, f := range funcs {
		p.AddFunc(f)
	}
}

// RemoveFuncs removes all functions from the program.
func (p *Program) RemoveFuncs() {
	p.funcs = nil
	p.funcIndexLookup = make(map[FuncIndex]*Func)
	p.funcNameLookup = make(map[string]*Func)
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
