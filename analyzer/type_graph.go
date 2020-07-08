package analyzer

import "github.com/arneph/toph/ir"

// TypeGraph represents a directed graph of types.
type TypeGraph struct {
	dependantsToDependees map[ir.Type]map[ir.Type]struct{}
	dependeesToDependants map[ir.Type]map[ir.Type]struct{}

	// Topological order:
	topologicalOrderOk bool
	topologicalOrder   []ir.Type
}

func newTypeGraph() *TypeGraph {
	tg := new(TypeGraph)
	tg.dependantsToDependees = make(map[ir.Type]map[ir.Type]struct{})
	tg.dependeesToDependants = make(map[ir.Type]map[ir.Type]struct{})
	tg.topologicalOrderOk = false

	return tg
}

// ContainsEdge returns if the type graph has an edge from the given dependant
// to the given dependee.
func (tg *TypeGraph) ContainsEdge(dependant, dependee ir.Type) bool {
	_, ok := tg.dependantsToDependees[dependant][dependee]
	return ok
}

// AllDependees returns all dependees of the given dependant.
func (tg *TypeGraph) AllDependees(dependant ir.Type) []ir.Type {
	var dependees []ir.Type
	for dependee := range tg.dependantsToDependees[dependant] {
		dependees = append(dependees, dependee)
	}
	return dependees
}

// AllDependants returns all dependants of the given dependee.
func (tg *TypeGraph) AllDependants(dependee ir.Type) []ir.Type {
	var dependants []ir.Type
	for dependant := range tg.dependeesToDependants[dependee] {
		dependants = append(dependants, dependant)
	}
	return dependants
}

// TopologicalOrder returns all types ordered such that any type later in the
// result slice only depends on types listed earlier.
func (tg *TypeGraph) TopologicalOrder() []ir.Type {
	tg.updateTopologicalOrder()
	return tg.topologicalOrder
}

func (tg *TypeGraph) addType(t ir.Type) {
	if _, ok := tg.dependantsToDependees[t]; ok {
		return
	}

	tg.dependantsToDependees[t] = make(map[ir.Type]struct{})
	tg.dependeesToDependants[t] = make(map[ir.Type]struct{})

	tg.topologicalOrderOk = false
}

func (tg *TypeGraph) addDependency(dependant, dependee ir.Type) {
	dependees := tg.dependantsToDependees[dependant]
	dependees[dependee] = struct{}{}
	dependants := tg.dependeesToDependants[dependee]
	dependants[dependant] = struct{}{}

	tg.topologicalOrderOk = false
}

func (tg *TypeGraph) updateTopologicalOrder() {
	if tg.topologicalOrderOk {
		return
	}
	tg.topologicalOrderOk = true
	tg.topologicalOrder = []ir.Type{ir.IntType, ir.FuncType, ir.ChanType, ir.MutexType, ir.WaitGroupType}

	added := map[ir.Type]bool{
		ir.IntType:       true,
		ir.FuncType:      true,
		ir.ChanType:      true,
		ir.MutexType:     true,
		ir.WaitGroupType: true,
	}

	for len(tg.topologicalOrder) < len(tg.dependantsToDependees) {
		oldCount := len(tg.topologicalOrder)
	candidateLoop:
		for dependant, dependees := range tg.dependantsToDependees {
			if added[dependant] {
				continue
			}
			for dependee := range dependees {
				if !added[dependee] {
					continue candidateLoop
				}
			}
			tg.topologicalOrder = append(tg.topologicalOrder, dependant)
			added[dependant] = true
		}
		newCount := len(tg.topologicalOrder)
		if oldCount == newCount {
			panic("internal error: type graph contains circle")
		}
	}
}