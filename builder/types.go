package builder

import (
	"fmt"
	"go/types"

	"github.com/arneph/toph/ir"
)

type typeInfo struct {
	irType ir.Type
}

func (b *builder) typesTypeToIrType(typesType types.Type) ir.Type {
	switch underlyingTypesType := typesType.Underlying().(type) {
	case *types.Chan:
		return ir.ChanType

	case *types.Signature:
		return ir.FuncType

	case *types.Struct:
		if typesType.String() == "sync.Mutex" {
			return ir.MutexType
		} else if typesType.String() == "sync.RWMutex" {
			return ir.MutexType
		} else if typesType.String() == "sync.WaitGroup" {
			return ir.WaitGroupType
		} else if typesType.String() == "testing.T" {
			return nil
		}
		info, ok := b.types.At(typesType).(*typeInfo)
		if ok {
			return info.irType
		}
		if !shouldModelType(typesType, nil) {
			info = new(typeInfo)
			info.irType = nil
			b.types.Set(typesType, info)
			return nil
		}
		return b.typesStructToIrType(typesType, underlyingTypesType)

	case *types.Pointer:
		elementTypesType := underlyingTypesType.Elem()
		switch elementTypesType.(type) {
		case *types.Pointer:
			return nil
		default:
			return b.typesTypeToIrType(elementTypesType)
		}

	default:
		return nil
	}
}

func (b *builder) typesStructToIrType(typesType types.Type, typesStruct *types.Struct) ir.Type {
	name := ""
	if typesNamed, ok := typesType.(*types.Named); ok {
		name = typesNamed.Obj().Name()
	}
	irStructType := b.program.AddStructType(name)
	info := new(typeInfo)
	info.irType = irStructType
	b.types.Set(typesType, info)

	for i := 0; i < typesStruct.NumFields(); i++ {
		fieldTypesVar := typesStruct.Field(i)
		fieldTypesType := fieldTypesVar.Type()
		fieldIrType := b.typesTypeToIrType(fieldTypesType)
		if fieldIrType == nil {
			continue
		}
		isPointer := b.isPointer(fieldTypesType)
		isEmbedded := fieldTypesVar.Embedded()
		initialValue := b.initialValueForIrType(fieldIrType)
		irField := irStructType.AddField(i, fieldTypesVar.Name(), fieldIrType, isPointer, isEmbedded, initialValue)
		b.fields[fieldTypesVar] = irField
	}
	return irStructType
}

func shouldModelType(typesType types.Type, seen []types.Type) bool {
	for _, seen := range seen {
		if seen == typesType {
			return false
		}
	}

	switch typesType := typesType.Underlying().(type) {
	case *types.Chan:
		return true
	case *types.Signature:
		return true
	case *types.Pointer:
		if i := len(seen) - 1; i >= 0 {
			_, ok := seen[i].(*types.Pointer)
			if ok {
				return false
			}
		}
		return shouldModelType(typesType.Elem(), append(seen, typesType))
	case *types.Struct:
		for i := 0; i < typesType.NumFields(); i++ {
			typesVar := typesType.Field(i)
			if shouldModelType(typesVar.Type(), append(seen, typesType)) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (b *builder) isPointer(typesType types.Type) bool {
	switch typesType.Underlying().(type) {
	case *types.Pointer:
		irType := b.typesTypeToIrType(typesType)
		_, ok := irType.(*ir.StructType)
		return ok
	default:
		return false
	}
}

func (b *builder) initialValueForIrType(irType ir.Type) ir.Value {
	switch irType := irType.(type) {
	case ir.BasicType:
		switch irType {
		case ir.IntType:
			return 0
		case ir.FuncType:
			return -1
		case ir.ChanType:
			return -1
		case ir.MutexType:
			return -1
		case ir.WaitGroupType:
			return -1
		default:
			panic(fmt.Errorf("unexpected ir.BaseType: %d", irType))
		}
	case *ir.StructType:
		return -1
	default:
		panic(fmt.Errorf("unexpected ir.Type: %T", irType))
	}
}
