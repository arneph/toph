package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

type randomVariableSupplier struct {
	randomVars       []string
	selectStmts      []string
	guards           []string
	guardsUseGlobals bool
}

func (rvs *randomVariableSupplier) next(low, high int) string {
	i := len(rvs.randomVars)
	randomVar := fmt.Sprintf("r%d", i)
	selectStmt := fmt.Sprintf("%s : int[%d, %d]", randomVar, low, high)

	rvs.randomVars = append(rvs.randomVars, randomVar)
	rvs.selectStmts = append(rvs.selectStmts, selectStmt)

	return randomVar
}

func (rvs *randomVariableSupplier) addGuard(guard string, usesGlobals bool) {
	rvs.guards = append(rvs.guards, guard)
	rvs.guardsUseGlobals = rvs.guardsUseGlobals || usesGlobals
}

func (rvs *randomVariableSupplier) addToTrans(trans *uppaal.Trans) {
	for _, selectStmt := range rvs.selectStmts {
		trans.AddSelect(selectStmt)
	}
	combinedGuards := ""
	combinedGuardCount := 0
	guards := rvs.guards
	if trans.Guard() != "" {
		guards = append(guards, trans.Guard())
	}
	for _, guard := range rvs.guards {
		if combinedGuardCount == 0 {
			combinedGuards = guard
			combinedGuardCount++
		} else if combinedGuardCount == 1 {
			combinedGuards = "(" + combinedGuards + ") && (" + guard + ")"
			combinedGuardCount++
		} else {
			combinedGuards += " && (" + guard + ")"
			combinedGuardCount++
		}
	}
	trans.SetGuard(combinedGuards, rvs.guardsUseGlobals || trans.GuardUsesGlobals())
}

func (t *translator) translateRValue(v ir.RValue, typ ir.Type, rvs *randomVariableSupplier, ctx *context) (handle string, usesGlobals bool) {
	switch v := v.(type) {
	case ir.Value:
		return t.translateValue(v, typ), false
	case *ir.Variable:
		return t.translateVariable(v, ctx)
	case *ir.FieldSelection:
		return t.translateFieldSelection(v, rvs, ctx)
	case *ir.ContainerAccess:
		return t.translateContainerAccess(v, rvs, ctx)
	default:
		panic(fmt.Errorf("unexpected %T rvalue type", v))
	}
}

func (t *translator) translateLValue(v ir.LValue, rvs *randomVariableSupplier, ctx *context) (handle string, usesGlobals bool) {
	switch v := v.(type) {
	case *ir.Variable:
		return t.translateVariable(v, ctx)
	case *ir.FieldSelection:
		return t.translateFieldSelection(v, rvs, ctx)
	case *ir.ContainerAccess:
		return t.translateContainerAccess(v, rvs, ctx)
	default:
		panic(fmt.Errorf("unexpected %T lvalue type", v))
	}
}

func (t *translator) translateValue(v ir.Value, typ ir.Type) string {
	switch v {
	case ir.InitializedMutex:
		return "make_mutex()"
	case ir.InitializedWaitGroup:
		return "make_wait_group()"
	case ir.InitializedStruct:
		structType := typ.(*ir.StructType)
		return fmt.Sprintf("make_%s(true)", structType.VariablePrefix())
	case ir.InitializedArray:
		arrayType := typ.(*ir.ContainerType)
		return fmt.Sprintf("make_%s(true)", arrayType.VariablePrefix())
	}
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

func (t *translator) translateFieldSelection(fs *ir.FieldSelection, rvs *randomVariableSupplier, ctx *context) (string, bool) {
	handle, usesGlobals := t.translateLValue(fs.StructVal(), rvs, ctx)
	return fmt.Sprintf("%s_structs[%s].%s",
		fs.StructType().VariablePrefix(),
		handle,
		fs.Field().Handle()), usesGlobals
}

func (t *translator) translateContainerAccess(ca *ir.ContainerAccess, rvs *randomVariableSupplier, ctx *context) (string, bool) {
	handle, usesGlobals := t.translateLValue(ca.ContainerVal(), rvs, ctx)
	var index string
	if ca.Index() != ir.RandomIndex {
		index, _ = t.translateRValue(ca.Index(), ir.IntType, rvs, ctx)
	}
	switch ca.ContainerType().Kind() {
	case ir.Array:
		if ca.Index() == ir.RandomIndex {
			index = rvs.next(0, ca.ContainerType().Len()-1)
		}
		return fmt.Sprintf("%s_arrays[%s][%s]",
				ca.ContainerType().VariablePrefix(), handle, index),
			usesGlobals
	case ir.Slice:
		if ca.Index() == ir.RandomIndex {
			index = rvs.next(0, t.config.ContainerCapacity-1)
			rvs.addGuard(fmt.Sprintf("%s < %s_lengths[%s]",
				index,
				ca.ContainerType().VariablePrefix(),
				handle), usesGlobals)
		}
		usesGlobals = true
		return fmt.Sprintf("%s_slices[%s][%s]",
				ca.ContainerType().VariablePrefix(), handle, index),
			usesGlobals
	case ir.Map:
		if ca.Kind() != ir.Read {
			panic("expected map read access")
		}
		if ca.Index() == ir.RandomIndex {
			index = rvs.next(-1, t.config.ContainerCapacity-1)
			rvs.addGuard(fmt.Sprintf("%s < %s_lengths[%s]",
				index,
				ca.ContainerType().VariablePrefix(),
				handle), usesGlobals)
			if ca.Kind() == ir.Read {

			} else {

			}
		}
		usesGlobals = true
		return fmt.Sprintf("read_%s(%s, %s)",
				ca.ContainerType().VariablePrefix(), handle, index),
			usesGlobals
	default:
		panic("unexpected container kind")
	}
}

func (t *translator) translateSliceAppend(ca *ir.ContainerAccess, rvs *randomVariableSupplier, value string, ctx *context) string {
	if ca.Index() != ir.AppendIndex {
		panic("expected slice append")
	}
	handle, _ := t.translateLValue(ca.ContainerVal(), rvs, ctx)
	return fmt.Sprintf("append_%s(%s, %s)",
		ca.ContainerType().VariablePrefix(), handle, value)
}

func (t *translator) translateMapWriteAcces(ca *ir.ContainerAccess, rvs *randomVariableSupplier, value string, ctx *context) string {
	if ca.Kind() != ir.Write || ca.ContainerType().Kind() != ir.Map {
		panic("expected map write access")
	}
	handle, _ := t.translateLValue(ca.ContainerVal(), rvs, ctx)
	var index string
	if ca.Index() != ir.RandomIndex {
		index, _ = t.translateRValue(ca.Index(), ir.IntType, rvs, ctx)
	} else {
		index = rvs.next(0, t.config.ContainerCapacity)
		rvs.addGuard(fmt.Sprintf("%s <= %s_lengths[%s]",
			index,
			ca.ContainerType().VariablePrefix(),
			handle), true)
	}
	return fmt.Sprintf("write_%s(%s, %s, %s)",
		ca.ContainerType().VariablePrefix(), handle, index, value)
}

func (t *translator) translateCopyOfRValue(rvalueString string, typ ir.Type) string {
	switch typ := typ.(type) {
	case ir.BasicType:
		return rvalueString
	case *ir.StructType:
		return fmt.Sprintf("copy_%s(%s)", typ.VariablePrefix(), rvalueString)
	case *ir.ContainerType:
		return fmt.Sprintf("copy_%s(%s)", typ.VariablePrefix(), rvalueString)
	default:
		panic(fmt.Errorf("unexpected ir.Type: %T", typ))
	}
}
