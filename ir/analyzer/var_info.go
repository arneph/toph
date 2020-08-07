package analyzer

import (
	"fmt"
	"strings"

	"github.com/arneph/toph/ir"
)

// VarInfo contains information on where variables are used.
type VarInfo struct {
	typeUsesInVars  map[ir.Type]map[*ir.Variable]struct{}
	typeUsesInFuncs map[ir.Type]map[*ir.Func]int
	totalTypeUses   map[ir.Type]int
	varUses         map[*ir.Variable]map[*ir.Func]int
	totalVarUses    map[*ir.Variable]int
}

// FindVarInfo computes and returns variable usage information for the given
// program.
func FindVarInfo(program *ir.Program) *VarInfo {
	vi := new(VarInfo)
	vi.typeUsesInVars = make(map[ir.Type]map[*ir.Variable]struct{})
	vi.typeUsesInFuncs = make(map[ir.Type]map[*ir.Func]int)
	vi.totalTypeUses = make(map[ir.Type]int)
	vi.varUses = make(map[*ir.Variable]map[*ir.Func]int)
	vi.totalVarUses = make(map[*ir.Variable]int)

	for _, t := range program.Types() {
		vi.typeUsesInVars[t] = make(map[*ir.Variable]struct{})
		vi.typeUsesInFuncs[t] = make(map[*ir.Func]int)
	}

	queue := []*ir.Scope{program.Scope()}
	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]
		queue = append(queue, scope.Children()...)
		for _, v := range scope.Variables() {
			vi.typeUsesInVars[v.Type()][v] = struct{}{}
			vi.totalTypeUses[v.Type()]++
			vi.varUses[v] = make(map[*ir.Func]int)
		}
	}

	for _, f := range program.Funcs() {
		for _, arg := range f.Args() {
			vi.addVariableUse(arg, f)
		}
		for _, result := range f.Results() {
			vi.addVariableUse(result, f)
		}
		f.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
			switch stmt := stmt.(type) {
			case *ir.AssignStmt:
				vi.addRValueUse(stmt.Source(), f)
				vi.addLValueUse(stmt.Destination(), f)
			case *ir.MakeChanStmt:
				vi.addVariableUse(stmt.Channel(), f)
			case *ir.ChanCommOpStmt:
				vi.addLValueUse(stmt.Channel(), f)
			case *ir.CloseChanStmt:
				vi.addLValueUse(stmt.Channel(), f)
			case *ir.MutexOpStmt:
				vi.addLValueUse(stmt.Mutex(), f)
			case *ir.WaitGroupOpStmt:
				vi.addLValueUse(stmt.WaitGroup(), f)
				vi.addRValueUse(stmt.Delta(), f)
			case *ir.MakeStructStmt:
				vi.addVariableUse(stmt.StructVar(), f)
			case *ir.MakeContainerStmt:
				vi.addVariableUse(stmt.ContainerVar(), f)
			case *ir.CallStmt:
				vi.addCallableUse(stmt.Callee(), f)
				for _, arg := range stmt.Args() {
					vi.addRValueUse(arg, f)
				}
				for _, result := range stmt.Results() {
					vi.addVariableUse(result, f)
				}
			case *ir.ReturnStmt:
				for _, result := range stmt.Results() {
					vi.addRValueUse(result, f)
				}
			case *ir.SelectStmt:
				for _, c := range stmt.Cases() {
					vi.addLValueUse(c.OpStmt().Channel(), f)
				}
			case *ir.ChanRangeStmt:
				vi.addLValueUse(stmt.Channel(), f)
			case *ir.ContainerRangeStmt:
				vi.addLValueUse(stmt.Container(), f)
				if stmt.CounterVar() != nil {
					vi.addVariableUse(stmt.CounterVar(), f)
				}
				if stmt.ValueVal() != nil {
					vi.addLValueUse(stmt.ValueVal(), f)
				}
			case *ir.BranchStmt, *ir.DeadEndStmt, *ir.IfStmt, *ir.SwitchStmt, *ir.ForStmt, *ir.RecoverStmt:
			default:
				panic(fmt.Errorf("unexpected ir.Stmt type: %T", stmt))
			}
		})
	}

	return vi
}

func (vi *VarInfo) addVariableUse(v *ir.Variable, f *ir.Func) {
	if v == nil {
		return
	}
	vi.varUses[v][f]++
	vi.totalVarUses[v]++
}

func (vi *VarInfo) addRValueUse(rvalue ir.RValue, f *ir.Func) {
	switch v := rvalue.(type) {
	case *ir.Variable:
		vi.addVariableUse(v, f)
	case *ir.FieldSelection:
		vi.typeUsesInFuncs[v.Type()][f]++
		vi.totalTypeUses[v.Type()]++
		vi.addLValueUse(v.StructVal(), f)
	case *ir.ContainerAccess:
		vi.typeUsesInFuncs[v.Type()][f]++
		vi.totalTypeUses[v.Type()]++
		vi.addLValueUse(v.ContainerVal(), f)
	case ir.Value:
	default:
		panic(fmt.Errorf("unexpected ir.RValue type: %T", v))
	}
}

func (vi *VarInfo) addLValueUse(lvalue ir.LValue, f *ir.Func) {
	switch v := lvalue.(type) {
	case *ir.Variable:
		vi.addVariableUse(v, f)
	case *ir.FieldSelection:
		vi.typeUsesInFuncs[v.Type()][f]++
		vi.totalTypeUses[v.Type()]++
		vi.addLValueUse(v.StructVal(), f)
	case *ir.ContainerAccess:
		vi.typeUsesInFuncs[v.Type()][f]++
		vi.totalTypeUses[v.Type()]++
		vi.addLValueUse(v.ContainerVal(), f)
	default:
		panic(fmt.Errorf("unexpected ir.LValue type: %T", v))
	}
}

func (vi *VarInfo) addCallableUse(callable ir.Callable, f *ir.Func) {
	switch c := callable.(type) {
	case *ir.Variable:
		vi.addVariableUse(c, f)
	case *ir.FieldSelection:
		vi.typeUsesInFuncs[c.Type()][f]++
		vi.totalTypeUses[c.Type()]++
		vi.addLValueUse(c.StructVal(), f)
	case *ir.ContainerAccess:
		vi.typeUsesInFuncs[c.Type()][f]++
		vi.totalTypeUses[c.Type()]++
		vi.addLValueUse(c.ContainerVal(), f)
	case *ir.Func:
	default:
		panic(fmt.Errorf("unexpected ir.Callable type: %T", c))
	}
}

// VarsUsingType returns all variables using the given type.
func (vi *VarInfo) VarsUsingType(t ir.Type) []*ir.Variable {
	var users []*ir.Variable
	for v := range vi.typeUsesInVars[t] {
		users = append(users, v)
	}
	return users
}

// FuncsUsingType returns all functions using the given type.
func (vi *VarInfo) FuncsUsingType(t ir.Type) []*ir.Func {
	var users []*ir.Func
	for f := range vi.typeUsesInFuncs[t] {
		users = append(users, f)
	}
	return users
}

// TypeUsesInFunc returns how many times the given type is used in the given
// function.
func (vi *VarInfo) TypeUsesInFunc(t ir.Type, f *ir.Func) int {
	return vi.typeUsesInFuncs[t][f]
}

// TotalTypeUses returns how many times the given type is used in the program.
func (vi *VarInfo) TotalTypeUses(t ir.Type) int {
	return vi.totalTypeUses[t]
}

// FuncsUsingVar returns all functions using the given variable.
func (vi *VarInfo) FuncsUsingVar(v *ir.Variable) []*ir.Func {
	var users []*ir.Func
	for f := range vi.varUses[v] {
		users = append(users, f)
	}
	return users
}

// VarUsesInFunc returns how many times the given variable is used in the given
// function.
func (vi *VarInfo) VarUsesInFunc(v *ir.Variable, f *ir.Func) int {
	return vi.varUses[v][f]
}

// TotalVarUses returns how many times the given variable is used in the program.
func (vi *VarInfo) TotalVarUses(v *ir.Variable) int {
	return vi.totalVarUses[v]
}

func (vi *VarInfo) String() string {
	var b strings.Builder

	b.WriteString("Type Uses In Vars:\n")
	for t, uses := range vi.typeUsesInVars {
		fmt.Fprintf(&b, "%s (%d):", t.String(), vi.totalTypeUses[t])
		first := true
		for v := range uses {
			if first {
				b.WriteString(" ")
				first = false
			} else {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s", v.Handle())
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("Type Uses In Funcs:\n")
	for t, uses := range vi.typeUsesInFuncs {
		fmt.Fprintf(&b, "%s (%d):", t.String(), vi.totalTypeUses[t])
		first := true
		for f, c := range uses {
			if first {
				b.WriteString(" ")
				first = false
			} else {
				b.WriteString(", ")
			}
			if f != nil {
				fmt.Fprintf(&b, "%s (%d)", f.Handle(), c)
			} else {
				fmt.Fprintf(&b, "globals (%d)", c)
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("Var Uses:\n")
	for v, uses := range vi.varUses {
		fmt.Fprintf(&b, "%s (%d):", v.Handle(), vi.totalVarUses[v])
		first := true
		for f, c := range uses {
			if first {
				b.WriteString(" ")
				first = false
			} else {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s (%d)", f.Handle(), c)
		}
		b.WriteString("\n")
	}

	return b.String()
}
