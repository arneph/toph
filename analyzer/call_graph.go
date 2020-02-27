package analyzer

import "github.com/arneph/toph/ir"

// SCC represents the strongly connected component an ir.Func belongs to.
type SCC int

// FuncNotCalled is the SCC value of all functions not reached from the entry
// function via the function call graph.
const FuncNotCalled SCC = 0

type FuncCallGraph struct {
	entry *ir.Func

	callerToCallees map[*ir.Func]map[*ir.Func]struct{}
	calleeToCallers map[*ir.Func]map[*ir.Func]struct{}

	// Strongly connected components:
	sccsOk bool
	sccs   map[*ir.Func]SCC
}

func newFuncCallGraph(entry *ir.Func) *FuncCallGraph {
	fcg := new(FuncCallGraph)
	fcg.entry = entry
	fcg.callerToCallees = make(map[*ir.Func]map[*ir.Func]struct{})
	fcg.calleeToCallers = make(map[*ir.Func]map[*ir.Func]struct{})
	fcg.sccsOk = false

	fcg.addFunc(entry)

	return fcg
}

func (fcg *FuncCallGraph) addFunc(f *ir.Func) {
	_, ok := fcg.callerToCallees[f]
	if ok {
		return
	}

	fcg.callerToCallees[f] = make(map[*ir.Func]struct{})
	fcg.calleeToCallers[f] = make(map[*ir.Func]struct{})
}

func (fcg *FuncCallGraph) addCall(caller, callee *ir.Func) {
	fcg.addFunc(caller)
	fcg.addFunc(callee)

	callees := fcg.callerToCallees[caller]
	callees[callee] = struct{}{}
	callers := fcg.calleeToCallers[callee]
	callers[caller] = struct{}{}
}

func (fcg *FuncCallGraph) updateSCCs() {
	if fcg.sccsOk {
		return
	}
	fcg.sccsOk = true
	fcg.sccs = make(map[*ir.Func]SCC)

	sccCount := 1 // FuncNotCalled has value 0

	index := 0
	indices := make(map[*ir.Func]int)
	lowLinks := make(map[*ir.Func]int)
	stack := make([]*ir.Func, 0)
	stackSet := make(map[*ir.Func]bool)

	var strongConnect func(*ir.Func)
	strongConnect = func(v *ir.Func) {
		indices[v] = index
		lowLinks[v] = index
		index++
		stack = append(stack, v)
		stackSet[v] = true

		// Consider successors of v:
		for w := range fcg.callerToCallees[v] {
			if _, ok := indices[w]; !ok {
				// Successor has not yet been visited. Recurse on it
				strongConnect(w)
				if lowLinks[v] > lowLinks[w] {
					lowLinks[v] = lowLinks[w]
				}

			} else if stackSet[w] {
				// Successor w is on stack and hence in the current SCC.
				// If w is not on stack, then (v, w) is a cross-edge in the DFS
				// tree and must be ignored
				// Note: The next line may look odd - but is correct.
				// It says w.index not w.lowlink; that is deliberate and from the original paper
				if lowLinks[v] > indices[w] {
					lowLinks[v] = indices[w]
				}
			}
		}

		// If v is a root node, pop the stack and generate a SCC
		if lowLinks[v] == indices[v] {
			scc := SCC(sccCount)
			sccCount++

			var w *ir.Func
			for v != w {
				i := len(stack) - 1
				w = stack[i]
				stack = stack[:i]
				stackSet[w] = false

				fcg.sccs[w] = scc
			}
		}
	}

	// Note: The original algorithm calls strongConnect with every node in the
	// graph that has not been visited yet. Calling strongConnect only on the
	// entry function ensures that unreachable functions are not assigned to a
	// strongly connected component (keep FuncNotCalled).
	strongConnect(fcg.entry)
}
