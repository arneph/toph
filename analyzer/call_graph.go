package analyzer

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

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

	// Strongly connected components:
	sccsOk bool
	sccs   map[*ir.Func]SCC
}

// CalculateFuncCallGraph returns a new function call graph for the given
// program, program entry function, and call kind. Only calls of the given call
// kind are contained in the graph.
func CalculateFuncCallGraph(prog *ir.Program, entry *ir.Func, callKind ir.CallKind) *FuncCallGraph {
	fcg := newFuncCallGraph(entry)

	for _, caller := range prog.Funcs() {
		for callee := range findCalleesOfFunc(caller, callKind) {
			fcg.addCall(caller, callee)
		}
	}

	return fcg
}

func findCalleesOfFunc(caller *ir.Func, callKind ir.CallKind) map[*ir.Func]struct{} {
	callees := make(map[*ir.Func]struct{})
	caller.Body().WalkStmts(func(stmt *ir.Stmt, scope *ir.Scope) {
		callStmt, ok := (*stmt).(*ir.CallStmt)
		if !ok || callStmt.Kind() != callKind {
			return
		}
		callees[callStmt.Callee()] = struct{}{}
	})
	return callees
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

// SCC returns the strongly connected component of the given function in the
// function call graph.
func (fcg *FuncCallGraph) SCC(f *ir.Func) SCC {
	fcg.updateSCCs()
	return fcg.sccs[f]
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
		str += fmt.Sprintf(" (%d) -> ", fcg.sccs[caller])
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
