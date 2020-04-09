package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

func (t *translator) translateGlobalScope() {
	addedVar := false
	for _, v := range t.program.Scope().Variables() {
		initialValue := fmt.Sprintf("%d", v.InitialValue())
		t.system.Declarations().AddVariable(v.Handle(), "int", initialValue)
	}
	if addedVar {
		t.system.Declarations().AddSpace()
	}
}

func (t *translator) translateScope(ctx *context) {
	addedLocalVar := false
	addedGlobalVar := false
	for _, v := range ctx.body.Scope().Variables() {
		initialValue := fmt.Sprintf("%d", v.InitialValue())
		if !v.IsCaptured() {
			ctx.proc.Declarations().AddVariable(v.Handle(), "int", initialValue)
			addedLocalVar = true
		} else {
			t.system.Declarations().AddArray(v.Handle(), t.callCount(ctx.f), "int")
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
	return fmt.Sprintf("res_%s_%d_%v", proc.Name(), index, res)
}

func (t *translator) translateResult(f *ir.Func, index int, pidStr string) string {
	name := t.translateResultName(f, index)
	return fmt.Sprintf("%s[%s]", name, pidStr)
}

func (t *translator) translateRValue(v ir.RValue, ctx *context) string {
	switch v := v.(type) {
	case ir.Value:
		return v.String()
	case *ir.Variable:
		return t.translateVariable(v, ctx)
	default:
		panic(fmt.Errorf("unexpected %T rvalue type", v))
	}
}

func (t *translator) translateVariable(v *ir.Variable, ctx *context) string {
	if !v.IsCaptured() {
		return v.Handle()
	}

	w, s := ctx.body.Scope().GetVariable(v.Name())
	if v != w {
		panic(fmt.Errorf("scope returned unexpected variable: %v, expected: %v", w, v))
	}

	f := ctx.f
	arg := "pid"
	for s.IsSuperScopeOf(f.Scope()) {
		arg = "par_pid_" + t.funcToProcess[f].Name() + "[" + arg + "]"
		f = f.EnclosingFunc()
	}
	return v.Handle() + "[" + arg + "]"
}
