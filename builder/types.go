package builder

import (
	"go/types"

	"github.com/arneph/toph/ir"
)

func typesTypeToIrType(t types.Type) (ir.Type, bool) {
	switch t.Underlying().(type) {
	case *types.Chan:
		return ir.ChanType, true
	case *types.Signature:
		return ir.FuncType, true
	default:
		return ir.Type(-1), false
	}
}
