package ir

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

// FuncIndex represents the index of a function.
type FuncIndex int

// Func represents go functions and funvction literals.
type Func struct {
	index FuncIndex
	name  string

	signature *types.Signature

	args        map[int]*Variable
	resultTypes map[int]Type
	results     map[int]*Variable

	enclosingFunc *Func
	body          Body

	Node
}

func newOuterFunc(index FuncIndex, name string, signature *types.Signature, globalScope *Scope, pos, end token.Pos) *Func {
	if name == "" {
		name = fmt.Sprintf("func%d", index)
	}

	f := new(Func)
	f.index = index
	f.name = name

	f.signature = signature

	f.args = make(map[int]*Variable)
	f.resultTypes = make(map[int]Type)
	f.results = make(map[int]*Variable)

	f.enclosingFunc = nil
	f.body.init()
	f.body.scope.superScope = globalScope

	f.pos = pos
	f.end = end

	return f
}

func newInnerFunc(index FuncIndex, signature *types.Signature, enclosingFunc *Func, enclosingScope *Scope, pos, end token.Pos) *Func {
	f := new(Func)
	f.index = index
	f.name = fmt.Sprintf("%s_func%d", enclosingFunc.name, index)

	f.signature = signature

	f.args = make(map[int]*Variable)
	f.resultTypes = make(map[int]Type)
	f.results = make(map[int]*Variable)

	f.enclosingFunc = enclosingFunc
	f.body.init()
	f.body.scope.superScope = enclosingScope

	f.pos = pos
	f.end = end

	return f
}

// FuncValue returns a Value representing the function.
func (f *Func) FuncValue() Value {
	return Value(f.index)
}

// Name returns the name of the function.
func (f *Func) Name() string {
	return f.name
}

// Signature returns the full type of the function, including argument and
// result types that are not otherwise included in the IR.
func (f *Func) Signature() *types.Signature {
	return f.signature
}

// Args returns an map from index to argument variables of the function.
func (f *Func) Args() map[int]*Variable {
	return f.args
}

// AddArg adds an argument to the function.
func (f *Func) AddArg(index int, arg *Variable) {
	f.args[index] = arg
	f.body.scope.AddVariable(arg)
}

// DefineArg sets an existing variable of the function as an argument.
func (f *Func) DefineArg(index int, arg *Variable) {
	f.args[index] = arg
}

// ResultTypes returns a map from index to result types of the function.
func (f *Func) ResultTypes() map[int]Type {
	return f.resultTypes
}

// Results returns a map from index to result variables of the function.
func (f *Func) Results() map[int]*Variable {
	return f.results
}

// AddResultType adds a result type to the function.
func (f *Func) AddResultType(index int, resultType Type) {
	f.resultTypes[index] = resultType
}

// AddResult adds a result variable to the function and its scope.
func (f *Func) AddResult(index int, result *Variable) {
	f.resultTypes[index] = result.Type()
	f.results[index] = result
	f.body.scope.AddVariable(result)
}

// DefineResult sets an existing variable of the function as a result variable.
func (f *Func) DefineResult(index int, result *Variable) {
	f.resultTypes[index] = result.Type()
	f.results[index] = result
}

// EnclosingFunc returns the function within which the function is defined or
// nil if the function is defined at package scope.
func (f *Func) EnclosingFunc() *Func {
	return f.enclosingFunc
}

// Scope returns the function scope.
func (f *Func) Scope() *Scope {
	return f.body.scope
}

// Body returns the function body.
func (f *Func) Body() *Body {
	return &f.body
}

func (f *Func) String() string {
	s := "func{\n"
	s += fmt.Sprintf("\tindex: %d\n", f.index)
	s += "\tname: " + f.Name() + "\n"
	s += "\targs: "
	firstArg := true
	for _, arg := range f.args {
		if firstArg {
			firstArg = false
		} else {
			s += ", "
		}
		s += arg.Handle()
	}
	s += "\n"
	s += "\tresults: "
	firstResult := true
	for i, resultType := range f.resultTypes {
		result, ok := f.results[i]
		if firstResult {
			firstResult = false
		} else {
			s += ", "
		}
		if ok {
			s += result.Handle()
		} else {
			s += resultType.String()
		}
	}
	s += "\n"
	if f.enclosingFunc != nil {
		s += fmt.Sprintf("\tenclosing func index: %d\n", f.enclosingFunc.index)
	}
	s += "\t" + strings.ReplaceAll(f.body.String(), "\n", "\n\t") + "\n"
	s += "}"
	return s
}
