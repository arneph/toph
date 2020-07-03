package analyzer

import (
	"fmt"
	"go/types"
	"strings"

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

	isCallerCounts map[*ir.Func]int
	isCalleeCounts map[*ir.Func]int

	callerToSpecialOpCounts map[*ir.Func]map[ir.SpecialOp]int
	totalSpecialOpCounts    map[ir.SpecialOp]int

	callerToStructAllocations map[*ir.Func]map[*ir.StructType]int
	totalStructAllocations    map[*ir.StructType]int

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
	fcg.isCallerCounts = make(map[*ir.Func]int)
	fcg.isCalleeCounts = make(map[*ir.Func]int)
	fcg.callerToSpecialOpCounts = make(map[*ir.Func]map[ir.SpecialOp]int)
	fcg.totalSpecialOpCounts = make(map[ir.SpecialOp]int)
	fcg.callerToStructAllocations = make(map[*ir.Func]map[*ir.StructType]int)
	fcg.totalStructAllocations = make(map[*ir.StructType]int)
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

// CallerCount returns how many times a function calls others.
func (fcg *FuncCallGraph) CallerCount(callee *ir.Func) int {
	return fcg.isCallerCounts[callee]
}

// CalleeCount returns how many times a function gets called.
func (fcg *FuncCallGraph) CalleeCount(callee *ir.Func) int {
	return fcg.isCalleeCounts[callee]
}

// SpecialOpCount returns how many times the caller calls an ir.SpecialOpStmt
// with the given ir.SpecialOp
func (fcg *FuncCallGraph) SpecialOpCount(caller *ir.Func, op ir.SpecialOp) int {
	return fcg.callerToSpecialOpCounts[caller][op]
}

// TotalSpecialOpCount returns how many times a ir.SpecialOpStmt with the given
// ir.SpecialOp gets called.
func (fcg *FuncCallGraph) TotalSpecialOpCount(op ir.SpecialOp) int {
	return fcg.totalSpecialOpCounts[op]
}

// StructAllocations returns how many times the caller allocates an instance
// of the given struct type.
func (fcg *FuncCallGraph) StructAllocations(caller *ir.Func, structType *ir.StructType) int {
	return fcg.callerToStructAllocations[caller][structType]
}

// TotalStructAllocations returns how many times an instance of the given
// struct type gets created.
func (fcg *FuncCallGraph) TotalStructAllocations(structType *ir.StructType) int {
	return fcg.totalStructAllocations[structType]
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

	if f != fcg.entry && f.Signature() != nil {
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

	fcg.callerToSpecialOpCounts[f] = make(map[ir.SpecialOp]int)
	fcg.callerToStructAllocations[f] = make(map[*ir.StructType]int)

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

func (fcg *FuncCallGraph) addCallerCount(caller *ir.Func, count int) {
	fcg.isCallerCounts[caller] += count
	if fcg.isCallerCounts[caller] > MaxCallCounts {
		fcg.isCallerCounts[caller] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addCalleeCount(callee *ir.Func, count int) {
	fcg.isCalleeCounts[callee] += count
	if fcg.isCalleeCounts[callee] > MaxCallCounts {
		fcg.isCalleeCounts[callee] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addSpecialOpCount(caller *ir.Func, op ir.SpecialOp, count int) {
	fcg.callerToSpecialOpCounts[caller][op] += count
	if fcg.callerToSpecialOpCounts[caller][op] > MaxCallCounts {
		fcg.callerToSpecialOpCounts[caller][op] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addTotalSpecialOpCount(op ir.SpecialOp, count int) {
	fcg.totalSpecialOpCounts[op] += count
	if fcg.totalSpecialOpCounts[op] > MaxCallCounts {
		fcg.totalSpecialOpCounts[op] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addStructAllocations(caller *ir.Func, structType *ir.StructType, count int) {
	fcg.callerToStructAllocations[caller][structType] += count
	if fcg.callerToStructAllocations[caller][structType] > MaxCallCounts {
		fcg.callerToStructAllocations[caller][structType] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addTotalStructAllocations(structType *ir.StructType, count int) {
	fcg.totalStructAllocations[structType] += count
	if fcg.totalStructAllocations[structType] > MaxCallCounts {
		fcg.totalStructAllocations[structType] = MaxCallCounts
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

	var b strings.Builder

	b.WriteString("Graph with SCCs:\n")
	for caller, callees := range fcg.callerToCallees {
		fmt.Fprintf(&b, "%s (%d)\n", caller.Handle(), fcg.funcToSCCs[caller])
		i := 0
		for callee := range callees {
			fmt.Fprintf(&b, "\t-> %s\n", callee.Handle())
			i++
			if i == 10 {
				break
			}
		}
		if i < len(callees) {
			fmt.Fprintf(&b, "\t... (%d more)\n", len(callees)-i)
		}
	}
	b.WriteString("\n")

	b.WriteString("Dynamic Call Info:\n")
	for _, info := range fcg.dynamicCallInfos {
		b.WriteString(info.signature.String() + "\n")
		b.WriteString("\tcallers: ")
		firstCaller := true
		for caller := range info.callers {
			if firstCaller {
				firstCaller = false
			} else {
				b.WriteString(", ")
			}
			b.WriteString(caller.Handle())
		}
		b.WriteString("\n")
		b.WriteString("\tcallees: ")
		firstCallee := true
		for callee := range info.callees {
			if firstCallee {
				firstCallee = false
			} else {
				b.WriteString(", ")
			}
			b.WriteString(callee.Handle())
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("Call Counts:\n")
	for f := range fcg.callerToCallees {
		b.WriteString(f.Handle() + "\n")
		fmt.Fprintf(&b, "\tcaller: %d\n", fcg.isCallerCounts[f])
		fmt.Fprintf(&b, "\tcallee: %d\n", fcg.isCalleeCounts[f])
		fmt.Fprintf(&b, "\tspecial ops:\n")
		for op, count := range fcg.callerToSpecialOpCounts[f] {
			if count == 0 {
				continue
			}
			fmt.Fprintf(&b, "\t\t%s: %d\n", op, count)
		}
	}
	b.WriteString("\n")

	return b.String()
}
