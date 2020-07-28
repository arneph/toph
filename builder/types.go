package builder

import (
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
		underlyingElementTypesType := elementTypesType.Underlying()
		switch underlyingElementTypesType.(type) {
		case *types.Pointer:
			return nil
		default:
			elementIrType := b.typesTypeToIrType(elementTypesType)
			switch elementIrType := elementIrType.(type) {
			case *ir.StructType:
				return elementIrType
			case *ir.ContainerType:
				if elementIrType.Kind() == ir.Array {
					return elementIrType
				}
				return nil
			default:
				return nil
			}
		}
	case *types.Array, *types.Slice, *types.Map:
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
		return b.typesContainerToIrType(typesType)
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
		irField := irStructType.AddField(i, fieldTypesVar.Name(), fieldIrType, isPointer, isEmbedded)
		b.fields[fieldTypesVar] = irField
	}
	return irStructType
}

func (b *builder) typesContainerToIrType(typesType types.Type) ir.Type {
	var kind ir.ContainerKind
	var len int
	var elementTypesType types.Type
	switch underlyingTypesType := typesType.Underlying().(type) {
	case *types.Array:
		kind = ir.Array
		len = int(underlyingTypesType.Len())
		elementTypesType = underlyingTypesType.Elem()
	case *types.Slice:
		kind = ir.Slice
		elementTypesType = underlyingTypesType.Elem()
	case *types.Map:
		kind = ir.Map
		elementTypesType = underlyingTypesType.Elem()
	}
	elementIrType := b.typesTypeToIrType(elementTypesType)
	if elementIrType == nil {
		return nil
	}
	isPointer := b.isPointer(elementTypesType)
	irContainerType := b.program.AddContainerType(kind, len, elementIrType, isPointer)
	info := new(typeInfo)
	info.irType = irContainerType
	b.types.Set(typesType, info)

	return irContainerType
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
		switch elementType := typesType.Elem().Underlying().(type) {
		case *types.Struct:
			return shouldModelType(elementType, append(seen, typesType))
		case *types.Array:
			return shouldModelType(elementType, append(seen, typesType))
		default:
			return false
		}
	case *types.Struct:
		for i := 0; i < typesType.NumFields(); i++ {
			typesVar := typesType.Field(i)
			if shouldModelType(typesVar.Type(), append(seen, typesType)) {
				return true
			}
		}
		return false
	case *types.Array:
		return shouldModelType(typesType.Elem(), append(seen, typesType))
	case *types.Slice:
		return shouldModelType(typesType.Elem(), append(seen, typesType))
	case *types.Map:
		return shouldModelType(typesType.Elem(), append(seen, typesType))
	default:
		return false
	}
}

func (b *builder) isPointer(typesType types.Type) bool {
	switch typesType.Underlying().(type) {
	case *types.Pointer:
		irType := b.typesTypeToIrType(typesType)
		switch irType := irType.(type) {
		case *ir.StructType:
			return true
		case *ir.ContainerType:
			return irType.Kind() == ir.Array
		default:
			return false
		}
	case *types.Array:
		return false
	case *types.Slice:
		return true
	case *types.Map:
		return true
	default:
		return false
	}
}
