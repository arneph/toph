package ir

import (
	"fmt"
	"strings"
)

// FuncIndex represents the index of a function.
type FuncIndex int

var funcCount int

// Func represents go functions and funvction literals.
type Func struct {
	index       FuncIndex
	name        string
	args        map[int]*Variable
	captures    map[string]*Variable
	resultTypes map[int]Type
	results     map[int]*Variable
	scope       Scope
	body        Body
}

// NewFunc creates a new blank function.
func NewFunc(name string, superScope *Scope) *Func {
	if strings.HasSuffix(name, "_func") {
		name = fmt.Sprintf("%s%d", name, funcCount)
	}

	f := new(Func)
	f.index = FuncIndex(funcCount)
	f.name = name
	f.args = make(map[int]*Variable)
	f.captures = make(map[string]*Variable)
	f.resultTypes = make(map[int]Type)
	f.results = make(map[int]*Variable)
	f.scope.init()
	f.scope.superScope = superScope
	f.body.initWithScope(&f.scope)

	funcCount++

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

// Args returns an map from index to argument variables of the function.
func (f *Func) Args() map[int]*Variable {
	return f.args
}

// AddArg adds an argument to the function.
func (f *Func) AddArg(index int, arg *Variable) {
	f.args[index] = arg
	f.scope.AddVariable(arg)
}

// DefineArg sets an existing variable of the function as an argument.
func (f *Func) DefineArg(index int, arg *Variable) {
	f.args[index] = arg
}

// Captures returns all variables caputered by the function.
func (f *Func) Captures() map[string]*Variable {
	return f.captures
}

// GetCapturer returns the capturing varibale for the captured variable.
func (f *Func) GetCapturer(capturing string) *Variable {
	return f.captures[capturing]
}

// AddCapture adds a capturer to the function and its scope.
func (f *Func) AddCapture(capturing string, capturer *Variable) {
	f.captures[capturing] = capturer
	f.scope.AddVariable(capturer)
}

// DefineCapture sets an existing variable of the function as a capturer.
func (f *Func) DefineCapture(capturing string, capturer *Variable) {
	f.captures[capturing] = capturer
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
	f.scope.AddVariable(result)
}

// DefineResult sets an existing variable of the function as a result variable.
func (f *Func) DefineResult(index int, result *Variable) {
	f.resultTypes[index] = result.Type()
	f.results[index] = result
}

// Scope returns the function scope.
func (f *Func) Scope() *Scope {
	return &f.scope
}

// Body returns the function body.
func (f *Func) Body() *Body {
	return &f.body
}

func (f *Func) String() string {
	s := "func{\n"
	s += "  index: " + fmt.Sprintf("%d", f.index) + "\n"
	s += "  args: "
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
	s += "  captures: "
	firstCapture := true
	for capturing, capturer := range f.captures {
		if firstCapture {
			firstCapture = false
		} else {
			s += ", "
		}
		s += capturer.Handle() + " <- *'" + capturing + "'"
	}
	s += "\n"
	s += "  results: "
	firstResult := true
	for _, result := range f.results {
		if firstResult {
			firstResult = false
		} else {
			s += ", "
		}
		s += result.Handle()
	}
	s += "\n"
	s += "  " + strings.ReplaceAll(f.body.String(), "\n", "\n  ") + "\n"
	s += "}"
	return s
}
