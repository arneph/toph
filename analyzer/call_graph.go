package analyzer

import (
	"fmt"
	"go/types"

	"github.com/arneph/toph/ir"
)

// MaxCallCounts defines the maximum number of calls to a function that get counted.
const MaxCallCounts = 500

// SCC represents the strongly connected component an ir.Func belongs to.
type SCC int

// FuncNotCalled is the SCC value of all functions not reached from the entry
// function via the function call graph.
const FuncNotCalled SCC = 0

type dynamicCallInfo struct {
	signature *types.Signature
	callers   map[*ir.Func]struct{}
	callees   map[*ir.Func]struct{}
}

// FuncCallGraph represents a directed call graph of functions.
type FuncCallGraph struct {
	entry *ir.Func

	callerToCallees map[*ir.Func]map[*ir.Func]struct{}
	calleeToCallers map[*ir.Func]map[*ir.Func]struct{}

	dynamicCallInfos []dynamicCallInfo

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
	fcg.dynamicCallInfos = nil
	fcg.funcCallCounts = make(map[*ir.Func]int)
	fcg.makeChanCount = 0
	fcg.sccsOk = false

	if entry != nil {
		fcg.addFunc(entry)
	}

	return fcg
}

// ContainsEdge returns if the function call graph has an edge from the given
// caller to the given callee.
func (fcg *FuncCallGraph) ContainsEdge(caller, callee *ir.Func) bool {
	_, ok := fcg.callerToCallees[caller][callee]
	return ok
}

// AllCallees returns all callees of the given caller.
func (fcg *FuncCallGraph) AllCallees(caller *ir.Func) []*ir.Func {
	var callees []*ir.Func
	for callee := range fcg.callerToCallees[caller] {
		callees = append(callees, callee)
	}
	return callees
}

// AllCallers returns all callers of the given callee.
func (fcg *FuncCallGraph) AllCallers(callee *ir.Func) []*ir.Func {
	var callers []*ir.Func
	for caller := range fcg.calleeToCallers[callee] {
		callers = append(callers, caller)
	}
	return callers
}

// DynamicCallees returns all callees of dynamic function calls of the given
// signature.
func (fcg *FuncCallGraph) DynamicCallees(signature *types.Signature) []*ir.Func {
	if dynInfo := fcg.dynamicCallInfoForSignature(signature); dynInfo != nil {
		var callees []*ir.Func
		for callee := range dynInfo.callees {
			callees = append(callees, callee)
		}
		return callees
	}
	return nil
}

// DynamicCallers returns all callers with dynamic function calls of the given
// signature.
func (fcg *FuncCallGraph) DynamicCallers(signature *types.Signature) []*ir.Func {
	if dynInfo := fcg.dynamicCallInfoForSignature(signature); dynInfo != nil {
		var callers []*ir.Func
		for caller := range dynInfo.callers {
			callers = append(callers, caller)
		}
		return callers
	}
	return nil
}

// CallCount returns how many times a function gets called.
func (fcg *FuncCallGraph) CallCount(callee *ir.Func) int {
	return fcg.funcCallCounts[callee]
}

// MakeChanCount returns how many times a channel gets created.
func (fcg *FuncCallGraph) MakeChanCount() int {
	return fcg.makeChanCount
}

// SCCCount returns the number of strongly connected components in the
// function graph. Disconnected functions are counted in SCC 0.
func (fcg *FuncCallGraph) SCCCount() int {
	fcg.updateSCCs()
	return fcg.sccCount
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

func (fcg *FuncCallGraph) dynamicCallInfoForSignature(signature *types.Signature) *dynamicCallInfo {
	for i, info := range fcg.dynamicCallInfos {
		if types.Identical(info.signature, signature.Underlying()) {
			return &fcg.dynamicCallInfos[i]
		}
	}
	return nil
}

func (fcg *FuncCallGraph) addFunc(f *ir.Func) {
	if _, ok := fcg.callerToCallees[f]; ok {
		return
	}

	fcg.callerToCallees[f] = make(map[*ir.Func]struct{})
	fcg.calleeToCallers[f] = make(map[*ir.Func]struct{})

	if f != fcg.entry {
		if dynInfo := fcg.dynamicCallInfoForSignature(f.Signature()); dynInfo != nil {
			dynInfo.callees[f] = struct{}{}
		} else {
			fcg.dynamicCallInfos = append(fcg.dynamicCallInfos,
				dynamicCallInfo{
					signature: f.Signature().Underlying().(*types.Signature),
					callers:   map[*ir.Func]struct{}{},
					callees:   map[*ir.Func]struct{}{f: {}},
				})
		}
	}

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) addStaticCall(caller, callee *ir.Func) {
	callees := fcg.callerToCallees[caller]
	callees[callee] = struct{}{}
	callers := fcg.calleeToCallers[callee]
	callers[caller] = struct{}{}

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) addDynamicCall(caller *ir.Func, calleeSignature *types.Signature) {
	dynInfo := fcg.dynamicCallInfoForSignature(calleeSignature)
	if dynInfo == nil {
		fcg.dynamicCallInfos = append(fcg.dynamicCallInfos,
			dynamicCallInfo{
				signature: calleeSignature.Underlying().(*types.Signature),
				callers:   map[*ir.Func]struct{}{caller: {}},
				callees:   map[*ir.Func]struct{}{},
			})
	} else {
		if _, ok := dynInfo.callers[caller]; ok {
			return
		}

		dynInfo.callers[caller] = struct{}{}

		for callee := range dynInfo.callees {
			fcg.addStaticCall(caller, callee)
		}
	}

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
