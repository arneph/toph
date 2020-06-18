package builder

import (
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) getSubstitute(typesObj types.Object) ir.RValue {
	switch typesObj := typesObj.(type) {
	case *types.Func:
		substituteFunc := b.getSubstituteFunc(typesObj)
		if substituteFunc == nil {
			return nil
		}
		return substituteFunc.FuncValue()
	default:
		return nil
	}
}

func (b *builder) getSubstituteFunc(funcType *types.Func) *ir.Func {
	var subFuncName string
	switch funcType.Pkg().Name() {
	case "time":
		if funcType.Name() == "After" {
			subFuncName = "subTimeAfter"
		}
	case "filepath":
		if funcType.Name() == "Walk" {
			subFuncName = "subFilepathWalk"
		}
	}
	for _, f := range b.program.Funcs() {
		if f.Name() == subFuncName {
			return f
		}
	}
	return nil
}

const substitutesCode = `
package subs

import (
	"math/rand"
	"path/filepath"
	"time"
)

func subTimeAfter(time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	go func() {
		ch <- time.Time{}
	}()
	return ch
}

func subFilepathWalk(root string, walkFn filepath.WalkFunc) error {
	for rand.Int() == 0 {
		walkFn("", nil, nil)
	}
	return nil
}
`
