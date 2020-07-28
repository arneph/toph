package uppaal

import (
	"fmt"
	"strings"
)

type variableInfo struct {
	name         string
	dimensions   []int
	_type        string
	initialValue string
}

// Declarations stores all declarations and the corresponding init function of
// a Process.
type Declarations struct {
	headerComment  string
	types          []string
	variables      []variableInfo
	variableLookup map[string]int
	funcs          []string
	initFuncName   string
	initFuncStmts  []string
}

func (d *Declarations) initDeclarations(comment string) {
	d.headerComment = comment
	d.variables = []variableInfo{}
	d.variableLookup = make(map[string]int)
	d.initFuncName = "initialize"
	d.initFuncStmts = []string{}
}

// AddSpaceBetweenTypes adds space between variable declarations.
func (d *Declarations) AddSpaceBetweenTypes() {
	d.types = append(d.types, "")
}

// AddType adds a type declaration to the list of declarations.
func (d *Declarations) AddType(_type string) {
	d.types = append(d.types, _type)
}

// AddSpaceBetweenVariables adds space between variable declarations.
func (d *Declarations) AddSpaceBetweenVariables() {
	d.variables = append(d.variables, variableInfo{
		name: "",
	})
}

// AddVariable adds a variable declaration to the list of declarations.
func (d *Declarations) AddVariable(name, _type, initialValue string) {
	i, ok := d.variableLookup[name]
	if !ok {
		i = len(d.variables)
		d.variables = append(d.variables, variableInfo{
			name: name,
		})
		d.variableLookup[name] = i
	}

	d.variables[i].dimensions = nil
	d.variables[i]._type = _type
	d.variables[i].initialValue = initialValue
}

// AddArray adds an array declaration to the list of declarations.
func (d *Declarations) AddArray(name string, dimensions []int, _type string) {
	i, ok := d.variableLookup[name]
	if !ok {
		i = len(d.variables)
		d.variables = append(d.variables, variableInfo{
			name: name,
		})
		d.variableLookup[name] = i
	}

	d.variables[i].dimensions = dimensions
	d.variables[i]._type = _type
	d.variables[i].initialValue = ""
}

// AddFunc adds a function declaration to the list of declarations.
func (d *Declarations) AddFunc(f string) {
	d.funcs = append(d.funcs, f)
}

// RequiresInitFunc returns whether the declarations require an initialization
// function.
func (d *Declarations) RequiresInitFunc() bool {
	return len(d.initFuncStmts) > 0
}

// InitFuncName returns the name of the initialization function for the
// declarations.
func (d *Declarations) InitFuncName() string {
	return d.initFuncName
}

// SetInitFuncName sets the name of the initialization function for the
// declarations.
func (d *Declarations) SetInitFuncName(n string) {
	d.initFuncName = n
}

// AddInitFuncStmt adds a statement to the initialization function.
func (d *Declarations) AddInitFuncStmt(stmt string) {
	d.initFuncStmts = append(d.initFuncStmts, stmt)
}

// AsXTA returns the xta (file format) representation of the declarations.
func (d *Declarations) AsXTA() string {
	var b strings.Builder
	b.WriteString("// " + d.headerComment)
	for _, t := range d.types {
		b.WriteString("\n" + t)
	}
	for _, info := range d.variables {
		b.WriteString("\n")
		if info.name == "" {
			continue
		}
		fmt.Fprintf(&b, "%s %s", info._type, info.name)
		for _, dim := range info.dimensions {
			fmt.Fprintf(&b, "[%d]", dim)
		}
		if info.initialValue != "" {
			fmt.Fprintf(&b, " = %s", info.initialValue)
		}
		b.WriteString(";")
	}
	for _, f := range d.funcs {
		b.WriteString("\n" + f + "\n")
	}
	if d.RequiresInitFunc() {
		fmt.Fprintf(&b, "\nvoid %s() {\n", d.initFuncName)
		for _, stmt := range d.initFuncStmts {
			b.WriteString("    " + stmt + "\n")
		}
		b.WriteString("}")
	}
	return b.String()
}
