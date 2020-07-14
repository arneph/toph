package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

func (t *translator) translateRValue(v ir.RValue, typ ir.Type, ctx *context) (handle string, usesGlobals bool) {
	switch v := v.(type) {
	case ir.Value:
		return t.translateValue(v, typ), false
	case *ir.Variable:
		return t.translateVariable(v, ctx)
	case *ir.FieldSelection:
		return t.translateFieldSelection(v, ctx), true
	default:
		panic(fmt.Errorf("unexpected %T rvalue type", v))
	}
}

func (t *translator) translateLValue(v ir.LValue, ctx *context) (handle string, usesGlobals bool) {
	switch v := v.(type) {
	case *ir.Variable:
		return t.translateVariable(v, ctx)
	case *ir.FieldSelection:
		return t.translateFieldSelection(v, ctx), true
	default:
		panic(fmt.Errorf("unexpected %T lvalue type", v))
	}
}

func (t *translator) translateValue(v ir.Value, typ ir.Type) string {
	if typ == ir.FuncType {
		irFuncIndex := ir.FuncIndex(v)
		if irFuncIndex == -1 {
			return "make_fid(-1, -1)"
		}
		irFunc := t.program.Func(irFuncIndex)
		parentPid := "-1"
		if irFunc.EnclosingFunc() != nil {
			parentPid = "pid"
		}
		return fmt.Sprintf("make_fid(%s, %s)", v.String(), parentPid)
	}
	return v.String()
}

func (t *translator) translateFieldSelection(fs *ir.FieldSelection, ctx *context) string {
	handle, _ := t.translateLValue(fs.StructVal(), ctx)
	return fmt.Sprintf("%s_array[%s].%s",
		fs.StructType().VariablePrefix(),
		handle,
		fs.Field().Handle())
}

func (t *translator) translateCopyOfRValue(rvalueString string, typ ir.Type) string {
	switch typ := typ.(type) {
	case ir.BasicType:
		return rvalueString
	case *ir.StructType:
		return fmt.Sprintf("copy_%s(%s)", typ.VariablePrefix(), rvalueString)
	default:
		panic(fmt.Errorf("unexpected ir.Type: %T", typ))
	}
}
