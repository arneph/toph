package builder

import (
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) getSubstitute(funcType *types.Func) *ir.Func {
	funcSub, ok := b.addedSubstitutes[funcType]
	if ok {
		return funcSub
	}

	switch funcType.Pkg().Name() {
	case "time":
		if funcType.Name() == "After" {
			funcSub = b.makeTimeAfterSubstitute()
		}
	}
	if funcSub != nil {
		b.addedSubstitutes[funcType] = funcSub
	}
	return funcSub
}

func (b *builder) makeTimeAfterSubstitute() *ir.Func {
	timeAfter := ir.NewOuterFunc("time_after", b.program.Scope())
	timeAfter.AddResultType(0, ir.ChanType)

	b.program.AddFunc(timeAfter)

	chVar := ir.NewVariable("ch", ir.ChanType, -1)
	chVar.SetCaptured(true)
	timeAfter.Scope().AddVariable(chVar)
	timeAfter.Body().AddStmt(ir.NewMakeChanStmt(chVar, 1))

	timeAfterHelper := ir.NewInnerFunc(timeAfter, timeAfter.Scope())
	timeAfterHelper.Body().AddStmt(ir.NewChanOpStmt(chVar, ir.Send))
	timeAfter.Body().AddStmt(ir.NewCallStmt(timeAfterHelper, ir.Go))

	b.program.AddFunc(timeAfterHelper)

	returnStmt := ir.NewReturnStmt()
	returnStmt.AddResult(0, chVar)
	timeAfter.Body().AddStmt(returnStmt)

	return timeAfter
}
