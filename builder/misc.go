package builder

import (
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func astTypeToIrType(t ast.Expr) (ir.Type, bool) {
	switch t.(type) {
	case *ast.FuncType:
		return ir.FuncType, true
	case *ast.ChanType:
		return ir.ChanType, true
	default:
		return ir.Type(-1), false
	}
}

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
