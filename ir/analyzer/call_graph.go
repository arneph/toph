package analyzer

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/arneph/toph/ir"

	gv "github.com/awalterschulze/gographviz"
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

	callerToTypeAllocations map[*ir.Func]map[ir.Type]int
	totalTypeAllocations    map[ir.Type]int

	canPanicInternally map[*ir.Func]bool
	canPanicExternally map[*ir.Func]bool
	canRecover         map[*ir.Func]bool

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
	fcg.callerToTypeAllocations = make(map[*ir.Func]map[ir.Type]int)
	fcg.totalTypeAllocations = make(map[ir.Type]int)
	fcg.canPanicInternally = make(map[*ir.Func]bool)
	fcg.canPanicExternally = make(map[*ir.Func]bool)
	fcg.canRecover = make(map[*ir.Func]bool)
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

// TypeAllocations returns how many times the caller allocates an instance
// of the given type.
func (fcg *FuncCallGraph) TypeAllocations(caller *ir.Func, irType ir.Type) int {
	return fcg.callerToTypeAllocations[caller][irType]
}

// TotalTypeAllocations returns how many times an instance of the given
// type gets created.
func (fcg *FuncCallGraph) TotalTypeAllocations(irType ir.Type) int {
	return fcg.totalTypeAllocations[irType]
}

// CanPanic returns if the given function can panic.
func (fcg *FuncCallGraph) CanPanic(f *ir.Func) bool {
	return fcg.canPanicInternally[f] || fcg.canPanicExternally[f]
}

// CanRecover returns if the given function can recover from a panic (in a different function).
func (fcg *FuncCallGraph) CanRecover(f *ir.Func) bool {
	return fcg.canRecover[f]
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
		wantRecv := info.signature.Recv()
		haveRecv := signature.Recv()
		if (wantRecv == nil) != (haveRecv == nil) {
			continue
		} else if wantRecv != nil &&
			!types.Identical(wantRecv.Type(), haveRecv.Type()) {
			continue
		}
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

	fcg.callerToSpecialOpCounts[f] = make(map[ir.SpecialOp]int)
	fcg.callerToTypeAllocations[f] = make(map[ir.Type]int)

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) addStaticCall(caller, callee *ir.Func) {
	callees := fcg.callerToCallees[caller]
	callees[callee] = struct{}{}
	callers := fcg.calleeToCallers[callee]
	callers[caller] = struct{}{}

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) addDynamicCaller(caller *ir.Func, calleeSignature *types.Signature) {
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

func (fcg *FuncCallGraph) addDynamicCallee(callee *ir.Func) {
	calleeSignature := callee.Signature()
	dynInfo := fcg.dynamicCallInfoForSignature(calleeSignature)
	if dynInfo == nil {
		fcg.dynamicCallInfos = append(fcg.dynamicCallInfos,
			dynamicCallInfo{
				signature: calleeSignature.Underlying().(*types.Signature),
				callers:   map[*ir.Func]struct{}{},
				callees:   map[*ir.Func]struct{}{callee: {}},
			})
	} else {
		if _, ok := dynInfo.callees[callee]; ok {
			return
		}

		dynInfo.callees[callee] = struct{}{}

		for caller := range dynInfo.callers {
			fcg.addStaticCall(caller, callee)
		}
	}

	fcg.sccsOk = false
}

func (fcg *FuncCallGraph) zeroCalleeCounts(callee *ir.Func) {
	for caller := range fcg.calleeToCallers[callee] {
		delete(fcg.callerToCallees[caller], callee)
	}
	fcg.calleeToCallers[callee] = make(map[*ir.Func]struct{})

	calleeSignature := callee.Signature()
	if dynInfo := fcg.dynamicCallInfoForSignature(calleeSignature); dynInfo != nil {
		delete(dynInfo.callees, callee)
	}

	fcg.isCalleeCounts[callee] = 0

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

func (fcg *FuncCallGraph) addTypeAllocations(caller *ir.Func, irType ir.Type, count int) {
	fcg.callerToTypeAllocations[caller][irType] += count
	if fcg.callerToTypeAllocations[caller][irType] > MaxCallCounts {
		fcg.callerToTypeAllocations[caller][irType] = MaxCallCounts
	}
}

func (fcg *FuncCallGraph) addTotalTypeAllocations(irType ir.Type, count int) {
	fcg.totalTypeAllocations[irType] += count
	if fcg.totalTypeAllocations[irType] > MaxCallCounts {
		fcg.totalTypeAllocations[irType] = MaxCallCounts
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

// Graph returns a graphviz graph representation of the function call graph.
func (fcg *FuncCallGraph) Graph() (*gv.Graph, error) {
	g := gv.NewGraph()
	if err := g.SetName("fcg"); err != nil {
		return nil, err
	}
	if err := g.SetDir(true); err != nil {
		return nil, err
	}
	for f := range fcg.callerToCallees {
		if err := g.AddNode("fcg", f.Handle(), nil); err != nil {
			return nil, err
		}
	}
	for caller, callees := range fcg.callerToCallees {
		for callee := range callees {
			if err := g.AddEdge(caller.Handle(), callee.Handle(), true, nil); err != nil {
				return nil, err
			}
		}
	}
	return g, nil
}

func (fcg *FuncCallGraph) String() string {
	fcg.updateSCCs()

	var b strings.Builder

	b.WriteString("Call Graph (with SCCs):\n")
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
		fmt.Fprintf(&b, "\ttype allocations:\n")
		for t, count := range fcg.callerToTypeAllocations[f] {
			if count == 0 {
				continue
			}
			fmt.Fprintf(&b, "\t\t%s: %d\n", t, count)
		}
		fmt.Fprintf(&b, "\tcan panic internally: %t\n", fcg.canPanicInternally[f])
		fmt.Fprintf(&b, "\tcan panic externally: %t\n", fcg.canPanicExternally[f])
		fmt.Fprintf(&b, "\tcan recover: %t\n", fcg.canRecover[f])
	}
	b.WriteString("\n")

	b.WriteString("Special Ops Counts:\n")
	for op, count := range fcg.totalSpecialOpCounts {
		fmt.Fprintf(&b, "%s: %d\n", op, count)
	}
	b.WriteString("\n")

	b.WriteString("Type Allocations Counts:\n")
	for t, count := range fcg.totalTypeAllocations {
		fmt.Fprintf(&b, "%s: %d\n", t, count)
	}
	b.WriteString("\n")

	return b.String()
}
