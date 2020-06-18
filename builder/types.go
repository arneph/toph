package builder

import (
	"go/types"

	"github.com/arneph/toph/ir"
)

func typesTypeToIrType(typesType types.Type) (ir.Type, ir.Value, bool) {
	switch typesType.Underlying().(type) {
	case *types.Chan:
		return ir.ChanType, -1, true
	case *types.Signature:
		return ir.FuncType, -1, true
	default:
		if typesType.String() == "sync.Mutex" {
			return ir.MutexType, -1, true
		} else if typesType.String() == "sync.RWMutex" {
			return ir.MutexType, -1, true
		} else if typesType.String() == "sync.WaitGroup" {
			return ir.WaitGroupType, -1, true
		}
		return ir.Type(-1), 0, false
	}
}
