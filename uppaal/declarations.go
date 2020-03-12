package uppaal

import "strings"

// Declarations stores all declarations and the corresponding init function of
// a Process.
type Declarations struct {
	headerComment string
	variableDecls string
	initFuncName  string
	initFuncBody  string
}

func (d *Declarations) initDeclarations(comment string) {
	d.headerComment = comment
	d.variableDecls = ""
	d.initFuncName = "initialize"
	d.initFuncBody = ""
}

// AddVariableDeclaration adds a variable declaration to the list of declarations.
func (d *Declarations) AddVariableDeclaration(decl string) {
	d.variableDecls += "\n" + decl
}

// RequiresInitFunc returns whether the declarations require an initialization
// function.
func (d *Declarations) RequiresInitFunc() bool {
	return len(d.initFuncBody) > 0
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
	d.initFuncBody += "\n    "
	d.initFuncBody += strings.ReplaceAll(stmt, "\n", "\n    ")
}

// AsXTA returns the xta (file format) representation of the declarations.
func (d *Declarations) AsXTA() string {
	str := "// " + d.headerComment
	str += d.variableDecls
	if d.RequiresInitFunc() {
		str += "\nvoid " + d.initFuncName + "() {" + d.initFuncBody + "\n}"
	}
	return str
}
