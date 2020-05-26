package uppaal

import "fmt"

type variableInfo struct {
	name         string
	arraySize    int
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

// AddSpace adds space between variable declarations.
func (d *Declarations) AddSpace() {
	d.variables = append(d.variables, variableInfo{
		name: "",
	})
}

// AddType adds a type declaration to the list of declarations.
func (d *Declarations) AddType(_type string) {
	d.types = append(d.types, _type)
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

	d.variables[i].arraySize = -1
	d.variables[i]._type = _type
	d.variables[i].initialValue = initialValue
}

// AddArray adds an array declaration to the list of declarations.
func (d *Declarations) AddArray(name string, size int, _type string) {
	i, ok := d.variableLookup[name]
	if !ok {
		i = len(d.variables)
		d.variables = append(d.variables, variableInfo{
			name: name,
		})
		d.variableLookup[name] = i
	}

	d.variables[i].arraySize = size
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
	str := "// " + d.headerComment
	for _, _type := range d.types {
		str += "\n" + _type
	}
	if len(d.types) > 0 {
		str += "\n\n"
	}
	for _, info := range d.variables {
		if info.name == "" {
			str += "\n"
		} else if info.arraySize < 0 && info.initialValue == "" {
			str += fmt.Sprintf("\n%s %s;",
				info._type, info.name)
		} else if info.arraySize < 0 && info.initialValue != "" {
			str += fmt.Sprintf("\n%s %s = %s;",
				info._type, info.name, info.initialValue)
		} else {
			str += fmt.Sprintf("\n%s %s[%d];",
				info._type, info.name, info.arraySize)
		}
	}
	if d.RequiresInitFunc() {
		str += "\nvoid " + d.initFuncName + "() {\n"
		for _, stmt := range d.initFuncStmts {
			str += "    " + stmt + "\n"
		}
		str += "}"
	}
	for _, f := range d.funcs {
		str += "\n\n" + f
	}
	return str
}
