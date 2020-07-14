package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateAssignStmt(stmt *ir.AssignStmt, ctx *context) {
	sourceHandle, sourceUsesGlobals := t.translateRValue(stmt.Source(), stmt.Destination().Type(), ctx)
	destination, destinationUsesGlobals := t.translateLValue(stmt.Destination(), ctx)

	if stmt.RequiresCopy() {
		sourceHandle = t.translateCopyOfRValue(sourceHandle, stmt.Destination().Type())
	}

	assigned := ctx.proc.AddState("assigned_"+stmt.Destination().Handle()+"_", uppaal.Renaming)
	assigned.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	assigned.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	assign := ctx.proc.AddTransition(ctx.currentState, assigned)
	assign.AddUpdate(fmt.Sprintf("%s = %s", destination, sourceHandle),
		sourceUsesGlobals || destinationUsesGlobals)
	assign.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = assigned
	ctx.addLocation(assigned.Location())
}
