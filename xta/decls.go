package xta

import "strings"

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

func (d *Declarations) AddVariableDeclaration(decl string) {
	d.variableDecls += "\n" + decl
}

func (d *Declarations) RequiresInitFunc() bool {
	return len(d.initFuncBody) > 0
}

func (d *Declarations) InitFuncName() string {
	return d.initFuncName
}

func (d *Declarations) SetInitFuncName(n string) {
	d.initFuncName = n
}

func (d *Declarations) AddInitFuncInstr(instr string) {
	d.initFuncBody += "\n    "
	d.initFuncBody += strings.ReplaceAll(instr, "\n", "\n    ")
}

func (d *Declarations) String() string {
	str := "// " + d.headerComment +
		d.variableDecls
	if d.RequiresInitFunc() {
		str += "\nvoid " + d.initFuncName + "() {" + d.initFuncBody + "\n}"
	}
	return str
}
