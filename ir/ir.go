package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// Prog represents an entire go program.
type Prog struct {
	fset *token.FileSet

	scope Scope
}

// NewProg creates a new blank program.
func NewProg(fset *token.FileSet) *Prog {
	p := new(Prog)
	p.fset = fset
	p.scope.init()

	return p
}

// Scope returns the global scope of the Prog.
func (p *Prog) Scope() *Scope {
	return &p.scope
}

func (p *Prog) String() string {
	return "prog{\n  " + strings.ReplaceAll(p.scope.String(), "\n", "\n  ") + "\n}"
}

// Scope holds the function and channel declarations of Prog, Func, etc.
type Scope struct {
	superScope *Scope

	funcs      map[*Func]string
	namedFuncs map[string]*Func

	chans      map[*Chan]string
	namedChans map[string]*Chan
}

func (s *Scope) init() {
	s.funcs = make(map[*Func]string)
	s.namedFuncs = make(map[string]*Func)

	s.chans = make(map[*Chan]string)
	s.namedChans = make(map[string]*Chan)
}

// AddFunc adds the given, unnamed function to the scope.
func (s *Scope) AddFunc(f *Func) {
	s.funcs[f] = ""
	f.body.scope.superScope = s
}

// AddNamedFunc adds the given, named function to the scope.
func (s *Scope) AddNamedFunc(f *Func, name string) {
	if _, ok := s.namedFuncs[name]; ok {
		panic("tried to add named function twice")
	}

	s.funcs[f] = name
	s.namedFuncs[name] = f
	f.body.scope.superScope = s
}

// FindNameOfFunc searches the scope and the enclosing scopes for the given
// function and returns its name.
func (s *Scope) FindNameOfFunc(f *Func) string {
	name, ok := s.funcs[f]
	if ok {
		return name
	}
	if s.superScope != nil {
		return s.superScope.FindNameOfFunc(f)
	}
	return ""
}

// FindNamedFunc searches the scope and the enclosing scopes for a function
// with the given name and returns it.
func (s *Scope) FindNamedFunc(name string) *Func {
	f, ok := s.namedFuncs[name]
	if ok {
		return f
	}
	if s.superScope != nil {
		return s.superScope.FindNamedFunc(name)
	}
	return nil
}

// AddChan adds the given, unnamed channel to the scope.
func (s *Scope) AddChan(c *Chan) {
	s.chans[c] = ""
}

// AddNamedChan adds the given, named channel to the scope.
func (s *Scope) AddNamedChan(c *Chan, name string) {
	if _, ok := s.namedChans[name]; ok {
		panic("tried to add named channel twice")
	}

	s.chans[c] = name
	s.namedChans[name] = c
}

// FindNamedChan searches the scope and the enclosing scopes for a channel with
// the given name and returns it.
func (s *Scope) FindNamedChan(name string) *Chan {
	c, ok := s.namedChans[name]
	if ok {
		return c
	}
	if s.superScope != nil {
		return s.superScope.FindNamedChan(name)
	}
	return nil
}

func (s *Scope) String() string {
	str := "scope{\n"
	for c, name := range s.chans {
		str += "  "
		if name != "" {
			str += name + " = "
		}
		str += fmt.Sprintf("%p = ", c)
		str += strings.ReplaceAll(c.String(), "\n", "\n  ")
		str += "\n"
	}
	for f, name := range s.funcs {
		str += "  "
		if name != "" {
			str += name + " = "
		}
		str += fmt.Sprintf("%p = ", f)
		str += strings.ReplaceAll(f.String(), "\n", "\n  ")
		str += "\n"
	}
	str += "}"
	return str
}

// Node represents an entity that corresponds to an ast.Node.
type Node struct {
	pos, end token.Pos
}

func (n Node) initNode(node ast.Node) {
	n.pos = node.Pos()
	n.end = node.End()
}

// Pos returns the position of the corresponding ast.Node.
func (n Node) Pos() token.Pos {
	return n.pos
}

// End returns the end of the corresponding ast.Node.
func (n Node) End() token.Pos {
	return n.end
}

// Value is the interface describing values which can be used as function
// arguments and results.
type Value interface {
	ast.Node
	fmt.Stringer

	// valueNode ensures only Chan and Func conform to the Value interface.
	valueNode()
}

func (c *Chan) valueNode() {}
func (f *Func) valueNode() {}

// Chan represents a go channel.
type Chan struct {
	Node
}

// NewChan creates a new channel.
func NewChan(node ast.Node) *Chan {
	c := new(Chan)
	c.initNode(node)

	return c
}

func (c *Chan) String() string {
	return "chan{}"
}

// Body represents a function, if statement, or for statement body.
type Body struct {
	scope *Scope
	stmts []Stmt
}

func (b *Body) init() {
	b.scope = new(Scope)
	b.scope.init()
	b.stmts = nil
}

func (b *Body) initWithScope(s *Scope) {
	b.scope = s
	b.stmts = nil
}

// Scope returns the scope corresponding to the body.
func (b *Body) Scope() *Scope {
	return b.scope
}

// Stmts returns the statements inside the body.
func (b *Body) Stmts() []Stmt {
	return b.stmts
}

// AddStmt appends the given statement at the end of the body.
func (b *Body) AddStmt(stmt Stmt) {
	b.stmts = append(b.stmts, stmt)
}

// AddStmts appends the given statements at the end of the body.
func (b *Body) AddStmts(stmts ...Stmt) {
	b.stmts = append(b.stmts, stmts...)
}

func (b *Body) String() string {
	s := b.scope.String() + "\n"
	s += "stmts{\n"
	for _, stmt := range b.stmts {
		s += "  " + strings.ReplaceAll(stmt.String(), "\n", "\n  ") + "\n"
	}
	s += "}"
	return s
}

// Func represents go functions and function literals.
type Func struct {
	Node
	args     []Value
	results  []Value
	scope    Scope
	body     Body
	deferred []Body
}

// NewFunc creates a new blank function.
func NewFunc(node ast.Node) *Func {
	f := new(Func)
	f.initNode(node)
	f.args = nil
	f.results = nil
	f.scope.init()
	f.body.initWithScope(&f.scope)
	f.deferred = nil

	return f
}

// AddArg adds a named argument to the function.
func (f *Func) AddArg(arg Value, name string) {
	switch arg := arg.(type) {
	case *Func:
		f.body.scope.AddNamedFunc(arg, name)
	case *Chan:
		f.body.scope.AddNamedChan(arg, name)
	}
	f.args = append(f.args, arg)
}

// AddResult adds a named result to the function.
func (f *Func) AddResult(result Value, name string) {
	switch result := result.(type) {
	case *Func:
		f.body.scope.AddNamedFunc(result, name)
	case *Chan:
		f.body.scope.AddNamedChan(result, name)
	}
	f.results = append(f.results, result)
}

// Scope returns the function scope.
func (f *Func) Scope() *Scope {
	return &f.scope
}

// Body returns the function body.
func (f *Func) Body() *Body {
	return &f.body
}

// DeferredCount returns the number of deferred bodies, correspoding to defer
// calls in go.
func (f *Func) DeferredCount() int {
	return len(f.deferred)
}

// DeferredAt returns the deferred body at the given index, corresponding to a
// defer call in go. Indices correspond to order of execution, e.g.
// f.DeferredAt(0) executes before f.DeferredAt(1).
func (f *Func) DeferredAt(index int) *Body {
	index = len(f.deferred) - 1 - index
	return &f.deferred[index]
}

// AddDeferred adds a new deferred body to the function and returns it.
// The new deferred body executes before all the deferred bodies the function
// already had.
func (f *Func) AddDeferred() *Body {
	index := len(f.deferred)
	f.deferred = append(f.deferred, Body{})
	f.deferred[index].initWithScope(&f.scope)
	return &f.deferred[index]
}

func (f *Func) String() string {
	s := "func{\n"
	s += "  args: "
	sep := ""
	for _, arg := range f.args {
		s += fmt.Sprintf("%s%p", sep, arg)
		sep = ", "
	}
	s += "\n"
	s += "  results: "
	for _, result := range f.results {
		s += fmt.Sprintf("%s%p", sep, result)
		sep = ", "
	}
	s += "\n"
	s += "  " + strings.ReplaceAll(f.body.String(), "\n", "\n  ") + "\n"
	s += "}"
	return s
}
