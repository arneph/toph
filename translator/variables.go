package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

func (t *translator) translateGlobalScope() {
	addedVar := false
	for _, v := range t.program.Scope().Variables() {
		typ := t.uppaalReferenceTypeForIrType(v.Type())
		initialValue := t.translateValue(v.InitialValue(), v.Type())
		t.system.Declarations().AddVariable(v.Handle(), typ, "")
		t.system.Declarations().AddInitFuncStmt(fmt.Sprintf("%s = %s;",
			v.Handle(), initialValue))
		addedVar = true
	}
	if addedVar {
		t.system.Declarations().AddSpaceBetweenVariables()
	}
}

func (t *translator) translateScope(ctx *context) {
	addedLocalVar := false
	addedGlobalVar := false
	for _, v := range ctx.body.Scope().Variables() {
		typ := t.uppaalReferenceTypeForIrType(v.Type())
		initialValue := t.translateValue(v.InitialValue(), v.Type())
		if !v.IsCaptured() {
			ctx.proc.Declarations().AddVariable(v.Handle(), typ, "")
			ctx.proc.Declarations().AddInitFuncStmt(fmt.Sprintf("%s = %s;",
				v.Handle(), initialValue))
			addedLocalVar = true
		} else {
			t.system.Declarations().AddArray(v.Handle(), []int{t.callCount(ctx.f)}, typ)
			ctx.proc.Declarations().AddInitFuncStmt(fmt.Sprintf("%s[pid] = %s;",
				v.Handle(), initialValue))
			addedGlobalVar = true
		}
	}
	if addedLocalVar {
		ctx.proc.Declarations().AddSpaceBetweenVariables()
	}
	if addedGlobalVar {
		t.system.Declarations().AddSpaceBetweenVariables()
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

func (t *translator) translateVariable(v *ir.Variable, ctx *context) (handle string, usesGlobals bool) {
	if !v.IsCaptured() {
		return v.Handle(), v.Scope() == t.program.Scope()
	}

	f := ctx.f
	s := v.Scope()
	arg := "pid"
	for f != nil && s.IsParentOf(f.Scope()) {
		arg = "par_pid_" + t.funcToProcess[f].Name() + "[" + arg + "]"
		f = f.EnclosingFunc()
	}
	if f == nil {
		panic("attempted to translate variable not defined in function super scopes")
	}
	return v.Handle() + "[" + arg + "]", true
}
