package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateMakeStructStmt(stmt *ir.MakeStructStmt, ctx *context) {
	handle, usesGlobals := t.translateVariable(stmt.StructVar(), ctx)
	name := stmt.StructVar().Name()

	made := ctx.proc.AddState("made_"+name+"_", uppaal.Renaming)
	made.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	made.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTransition(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_%s(%t)",
		handle,
		stmt.StructType().VariablePrefix(),
		stmt.InitialzeFields()),
		usesGlobals)
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = made
	ctx.addLocation(made.Location())
}

func (t *translator) translateMakeContainerStmt(stmt *ir.MakeContainerStmt, ctx *context) {
	var rvs randomVariableSupplier
	handle, usesGlobals := t.translateVariable(stmt.ContainerVar(), ctx)
	name := stmt.ContainerVar().Name()

	made := ctx.proc.AddState("made_"+name+"_", uppaal.Renaming)
	made.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	made.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTransition(ctx.currentState, made)
	switch stmt.ContainerType().Kind() {
	case ir.Array:
		make.AddUpdate(fmt.Sprintf("%s = make_%s(%t)",
			handle,
			stmt.ContainerType().VariablePrefix(),
			stmt.InitializeElements()),
			usesGlobals)
	case ir.Slice:
		lenHandle, lenUsesGlobals := t.translateRValue(stmt.ContainerLen(), &rvs, ctx)
		make.AddUpdate(fmt.Sprintf("%s = make_%s(%s, %t)",
			handle,
			stmt.ContainerType().VariablePrefix(),
			lenHandle,
			stmt.InitializeElements()),
			usesGlobals || lenUsesGlobals)
		rvs.addToTrans(make)
	case ir.Map:
		make.AddUpdate(fmt.Sprintf("%s = make_%s()",
			handle,
			stmt.ContainerType().VariablePrefix()),
			usesGlobals)
	default:
		panic("unexpected container kind")
	}
	make.SetSelectLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	make.SetGuardLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 80}))

	ctx.currentState = made
	ctx.addLocation(made.Location())
}

func (t *translator) translateCopySliceStmt(stmt *ir.CopySliceStmt, ctx *context) {
	var rvs randomVariableSupplier
	dstHandle, dstUsesGlobals := t.translateLValue(stmt.DestinationVal(), &rvs, ctx)
	srcHandle, srcUsesGlobals := t.translateLValue(stmt.SourceVal(), &rvs, ctx)

	copied := ctx.proc.AddState("copied_"+stmt.SliceType().VariablePrefix()+"_", uppaal.Renaming)
	copied.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	copied.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	copy := ctx.proc.AddTransition(ctx.currentState, copied)
	copy.AddUpdate(fmt.Sprintf("copy_between_%s(%s, %s)",
		stmt.SliceType().VariablePrefix(),
		dstHandle, srcHandle),
		dstUsesGlobals || srcUsesGlobals)
	rvs.addToTrans(copy)
	copy.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	copy.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	copy.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))

	ctx.currentState = copied
	ctx.addLocation(copied.Location())
}

func (t *translator) translateDeleteMapEntryStmt(stmt *ir.DeleteMapEntryStmt, ctx *context) {
	var rvs randomVariableSupplier
	handle, _ := t.translateLValue(stmt.MapVal(), &rvs, ctx)
	name := stmt.MapVal().Name()
	index := rvs.next(-1, t.config.ContainerCapacity-1)
	rvs.addGuard(fmt.Sprintf("%s < %s_lengths[%s]",
		index, stmt.MapType().VariablePrefix(), handle), true)

	deleted := ctx.proc.AddState("deleted_entry_"+name+"_", uppaal.Renaming)
	deleted.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	deleted.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	delete := ctx.proc.AddTransition(ctx.currentState, deleted)
	delete.AddUpdate(fmt.Sprintf("delete_%s(%s, %s)", stmt.MapType().VariablePrefix(), handle, index), true)
	rvs.addToTrans(delete)
	delete.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	delete.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	delete.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))

	ctx.currentState = deleted
	ctx.addLocation(deleted.Location())
}
