package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateAssignStmt(stmt *ir.AssignStmt, ctx *context) {
	var rvs randomVariableSupplier
	sourceHandle, sourceUsesGlobals := t.translateRValue(stmt.Source(), &rvs, ctx)
	if stmt.RequiresCopy() {
		sourceHandle = t.translateCopyOfRValue(sourceHandle, stmt.Destination().Type())
	}

	assigned := ctx.proc.AddState("assigned_"+stmt.Destination().Handle()+"_", uppaal.Renaming)
	assigned.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	assigned.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	assign := ctx.proc.AddTransition(ctx.currentState, assigned)

	irContainerAccess, ok := stmt.Destination().(*ir.ContainerAccess)
	if ok && irContainerAccess.IsSliceAppend() {
		sliceAppend := t.translateSliceAppend(irContainerAccess, &rvs, sourceHandle, ctx)

		assign.AddUpdate(sliceAppend, true)
	} else if ok && irContainerAccess.IsMapWrite() {
		mapWrite := t.translateMapWriteAcces(irContainerAccess, &rvs, sourceHandle, ctx)

		assign.AddUpdate(mapWrite, true)
	} else {
		destination, destinationUsesGlobals := t.translateLValue(stmt.Destination(), &rvs, ctx)

		assign.AddUpdate(fmt.Sprintf("%s = %s", destination, sourceHandle),
			sourceUsesGlobals || destinationUsesGlobals)
	}
	rvs.addToTrans(assign)
	assign.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	assign.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	assign.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))

	ctx.currentState = assigned
	ctx.addLocation(assigned.Location())
}
