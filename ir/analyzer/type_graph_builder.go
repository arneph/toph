package analyzer

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

// BuildTypeGraph returns a new type graph for the given program, containing
// dependecies between types, i.e. a type refering to another type by value.
// Dependencies to pointers of other types are excluded.
func BuildTypeGraph(program *ir.Program) *TypeGraph {
	tg := newTypeGraph()

	addTypesToTypeGraph(program, tg)
	addDependenciesToTypeGraph(program, tg)

	return tg
}

func addTypesToTypeGraph(program *ir.Program, tg *TypeGraph) {
	for _, t := range program.Types() {
		tg.addType(t)
	}
}

func addDependenciesToTypeGraph(program *ir.Program, tg *TypeGraph) {
	for _, t := range program.Types() {
		switch t := t.(type) {
		case ir.BasicType:
			continue
		case *ir.StructType:
			for _, f := range t.Fields() {
				if f.IsPointer() {
					continue
				}
				u := f.Type()
				tg.addDependency(t, u)
			}
		default:
			panic(fmt.Errorf("unexpected ir.Type: %T", t))
		}
	}
}
