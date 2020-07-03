package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

func (t *translator) translateGlobalScope() {
	addedVar := false
	for _, v := range t.program.Scope().Variables() {
		var typ, initialValue string
		switch v.Type() {
		case ir.FuncType:
			typ = "fid"
		default:
			typ = "int"
		}
		initialValue = t.translateValue(v.InitialValue(), v.Type())
		t.system.Declarations().AddVariable(v.Handle(), typ, initialValue)
	}
	if addedVar {
		t.system.Declarations().AddSpace()
	}
}

func (t *translator) translateScope(ctx *context) {
	addedLocalVar := false
	addedGlobalVar := false
	for _, v := range ctx.body.Scope().Variables() {
		var typ, initialValue string
		switch v.Type() {
		case ir.FuncType:
			typ = "fid"
		default:
			typ = "int"
		}
		initialValue = t.translateValue(v.InitialValue(), v.Type())
		if !v.IsCaptured() {
			ctx.proc.Declarations().AddVariable(v.Handle(), typ, "")
			ctx.proc.Declarations().AddInitFuncStmt(v.Handle() + " = " + initialValue + ";")
			addedLocalVar = true
		} else {
			t.system.Declarations().AddArray(v.Handle(), t.callCount(ctx.f), typ)
			ctx.proc.Declarations().AddInitFuncStmt(v.Handle() + "[pid] = " + initialValue + ";")
			addedGlobalVar = true
		}
	}
	if addedLocalVar {
		ctx.proc.Declarations().AddSpace()
	}
	if addedGlobalVar {
		t.system.Declarations().AddSpace()
	}
}

func (t *translator) translateArgName(v *ir.Variable) string {
	return fmt.Sprintf("arg_%s", v.Handle())
}

func (t *translator) translateArg(v *ir.Variable, pidStr string) string {
	name := t.translateArgName(v)
	return fmt.Sprintf("%s[%s]", name, pidStr)
}

func (t *translator) translateResultName(f *ir.Func, index int) string {
	proc := t.funcToProcess[f]
	res := f.ResultTypes()[index]
	return fmt.Sprintf("res_%s_%s_%d", res.VariablePrefix(), proc.Name(), index)
}

func (t *translator) translateResult(f *ir.Func, index int, pidStr string) string {
	name := t.translateResultName(f, index)
	return fmt.Sprintf("%s[%s]", name, pidStr)
}

func (t *translator) translateVariable(v *ir.Variable, ctx *context) string {
	if !v.IsCaptured() {
		return v.Handle()
	}

	f := ctx.f
	s := v.Scope()
	arg := "pid"
	for f != nil && s.IsSuperScopeOf(f.Scope()) {
		arg = "par_pid_" + t.funcToProcess[f].Name() + "[" + arg + "]"
		f = f.EnclosingFunc()
	}
	if f == nil {
		panic("attempted to translate variable not defined in function super scopes")
	}
	return v.Handle() + "[" + arg + "]"
}
