package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateMakeStructStmt(stmt *ir.MakeStructStmt, ctx *context) {
	handle := t.translateLValue(stmt.StructVal(), ctx)
	name := stmt.StructVal().Name()

	made := ctx.proc.AddState("made_"+name+"_", uppaal.Renaming)
	made.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	made.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTrans(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_%s()", handle, stmt.StructType().VariablePrefix()))
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = made
	ctx.addLocation(made.Location())
}
