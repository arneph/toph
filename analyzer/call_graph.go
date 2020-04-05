package analyzer

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

// MaxCallCounts defines the maximum number of calls to a function that get counted.
const MaxCallCounts = 100

// SCC represents the strongly connected component an ir.Func belongs to.
type SCC int

// FuncNotCalled is the SCC value of all functions not reached from the entry
// function via the function call graph.
const FuncNotCalled SCC = 0

// FuncCallGraph represents a directed call graph of functions.
type FuncCallGraph struct {
	entry *ir.Func

	callerToCallees map[*ir.Func]map[*ir.Func]struct{}
	calleeToCallers map[*ir.Func]map[*ir.Func]struct{}

	funcCallCounts map[*ir.Func]int
	makeChanCount  int

	// Strongly connected components:
	sccsOk     bool
	sccCount   int
	funcToSCCs map[*ir.Func]SCC
	sccToFuncs map[SCC][]*ir.Func
}

func newFuncCallGraph(entry *ir.Func) *FuncCallGraph {
	fcg := new(FuncCallGraph)
	fcg.entry = entry
	fcg.callerToCallees = make(map[*ir.Func]map[*ir.Func]struct{})
	fcg.calleeToCallers = make(map[*ir.Func]map[*ir.Func]struct{})
	fcg.funcCallCounts = make(map[*ir.Func]int)
	fcg.makeChanCount = 0
	fcg.sccsOk = false

	fcg.addFunc(entry)

	return fcg
}

// ContainsEdge returns if the function call graph has an edge from the given
// caller to the given callee.
func (fcg *FuncCallGraph) ContainsEdge(caller, callee *ir.Func) bool {
	_, ok := fcg.callerToCallees[caller][callee]
	return ok
}

// Callees returns all callees of the given caller.
func (fcg *FuncCallGraph) Callees(caller *ir.Func) []*ir.Func {
	var callees []*ir.Func
	for callee := range fcg.callerToCallees[caller] {
		callees = append(callees, callee)
	}
	return callees
}

// Callers returns all callers of the given callee.
func (fcg *FuncCallGraph) Callers(callee *ir.Func) []*ir.Func {
	var callers []*ir.Func
	for caller := range fcg.calleeToCallers[callee] {
		callers = append(callers, caller)
	}
	return callers
}

// CallCount returns how many times a function gets called.
func (fcg *FuncCallGraph) CallCount(callee *ir.Func) int {
	return fcg.funcCallCounts[callee]
}

// MakeChanCount returns how many times a channel gets created.
func (fcg *FuncCallGraph) MakeChanCount() int {
	return fcg.makeChanCount
}

// SCCOfFunc returns the strongly connected component of the given function in the
// function call graph.
func (fcg *FuncCallGraph) SCCOfFunc(f *ir.Func) SCC {
	fcg.updateSCCs()
	return fcg.funcToSCCs[f]
}

// FuncsInSCC returns all functions that are part of the given strongly
// connected component in the function call graph.
func (fcg *FuncCallGraph) FuncsInSCC(scc SCC) []*ir.Func {
	fcg.updateSCCs()
	return fcg.sccToFuncs[scc]
}

func (fcg *FuncCallGraph) addFunc(f *ir.Func) {
	_, ok := fcg.callerToCallees[f]
	if ok {
		return
	}

	fcg.callerToCallees[f] = make(map[*ir.Func]struct{})
	fcg.calleeToCallers[f] = make(map[*ir.Func]struct{})

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) addCall(caller, callee *ir.Func) {
	fcg.addFunc(caller)
	fcg.addFunc(callee)

	callees := fcg.callerToCallees[caller]
	callees[callee] = struct{}{}
	callers := fcg.calleeToCallers[callee]
	callers[caller] = struct{}{}

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) addCallCount(callee *ir.Func, count int) {
	fcg.funcCallCounts[callee] += count
	if fcg.funcCallCounts[callee] > MaxCallCounts {
		fcg.funcCallCounts[callee] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addMakeChanCallCount(count int) {
	fcg.makeChanCount += count
	if fcg.makeChanCount > MaxCallCounts {
		fcg.makeChanCount = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) updateSCCs() {
	if fcg.sccsOk {
		return
	}
	fcg.sccsOk = true
	fcg.funcToSCCs = make(map[*ir.Func]SCC)
	fcg.sccToFuncs = make(map[SCC][]*ir.Func)

	fcg.sccCount = 1 // FuncNotCalled has value 0

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
			scc := SCC(fcg.sccCount)
			fcg.sccCount++

			var w *ir.Func
			for v != w {
				i := len(stack) - 1
				w = stack[i]
				stack = stack[:i]
				stackSet[w] = false

				fcg.funcToSCCs[w] = scc
				fcg.sccToFuncs[scc] = append(fcg.sccToFuncs[scc], w)
			}
		}
	}

	for f := range fcg.callerToCallees {
		if _, ok := indices[f]; !ok {
			strongConnect(f)
		}
	}
}

func (fcg *FuncCallGraph) String() string {
	fcg.updateSCCs()
	str := ""
	for caller, callees := range fcg.callerToCallees {
		str += caller.Name()
		str += fmt.Sprintf(" (%d) -> ", fcg.funcToSCCs[caller])
		firstCallee := true
		for callee := range callees {
			if firstCallee {
				firstCallee = false
			} else {
				str += ", "
			}
			str += callee.Name()
		}
		str += "\n"
	}
	return str
}
