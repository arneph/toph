package uppaal

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// RenamingOption indicates whether a function should resolve naming conflicts.
type RenamingOption bool

const (
	// NoRenaming indicates that a function should not resolve naming conflicts.
	NoRenaming RenamingOption = false
	// Renaming indicates that a function should resolve naming conflicts.
	Renaming RenamingOption = true
)

// Process represents a process in Uppaal.
type Process struct {
	name string

	params []string

	decls Declarations

	initialState     *State
	states           map[*State]struct{}
	stateLookup      map[string]*State
	transitions      map[*Trans]struct{}
	transitionLookup map[*State]map[*State][]*Trans

	queries []*Query
}

func newProcess(name string) *Process {
	p := new(Process)
	p.name = name
	p.decls.initDeclarations("Place local declarations here.")
	p.initialState = nil
	p.states = make(map[*State]struct{})
	p.stateLookup = make(map[string]*State)
	p.transitions = make(map[*Trans]struct{})
	p.transitionLookup = make(map[*State]map[*State][]*Trans)

	return p
}

// Name returns the name of the process.
func (p *Process) Name() string {
	return p.name
}

// Parameters returns the list of parameters of the process.
func (p *Process) Parameters() []string {
	return p.params
}

// AddParameter adds a parameter to the process.
func (p *Process) AddParameter(param string) {
	p.params = append(p.params, param)
}

// Declarations returns the declarations that are part of the process.
func (p *Process) Declarations() *Declarations {
	return &p.decls
}

// InitialState returns the initial state of the process.
func (p *Process) InitialState() *State {
	return p.initialState
}

// SetInitialState changes the initial state of the process to the given state.
func (p *Process) SetInitialState(state *State) {
	_, ok := p.states[state]
	if state != nil && !ok {
		panic("tried to set unknown state as initial state")
	}
	if p.initialState != nil {
		p.initialState.isInitial = false
	}
	p.initialState = state
	if p.initialState != nil {
		p.initialState.isInitial = true
	}
}

// States returns all states of the process.
func (p *Process) States() []*State {
	var states []*State
	for state := range p.states {
		states = append(states, state)
	}
	return states
}

// StatesWithType returns all states with the given state type.
func (p *Process) StatesWithType(t StateType) []*State {
	var filteredStates []*State
	for state := range p.states {
		if state.Type() != t {
			continue
		}
		filteredStates = append(filteredStates, state)
	}
	return filteredStates
}

// AddState adds a state with the given name (after possible renaming to avoid
// naming conflicts) to the process and returns the new state.
func (p *Process) AddState(name string, opt RenamingOption) *State {
	if opt == NoRenaming {
		if _, ok := p.stateLookup[name]; ok {
			panic("naming collision when adding state")
		}
	} else if opt == Renaming {
		baseName := name
		if baseName == "" {
			baseName = "L"
		}
		for i := 0; ; i++ {
			name = fmt.Sprintf("%s%d", baseName, i)
			if _, ok := p.stateLookup[name]; !ok {
				break
			}
		}
	}

	state := newState(name)

	p.states[state] = struct{}{}
	p.stateLookup[name] = state
	p.transitionLookup[state] = make(map[*State][]*Trans)

	return state
}

// RemoveState removes the given state from the process.
func (p *Process) RemoveState(state *State) {
	if _, ok := p.states[state]; !ok {
		panic("tried to remove unknown state")
	} else if len(state.Transitions()) > 0 {
		panic("tried to remove state with remaining transitions")
	}

	if p.initialState == state {
		p.initialState = nil
	}

	delete(p.states, state)
	delete(p.stateLookup, state.name)
	delete(p.transitionLookup, state)
}

// TransitionLookup returns a double map from start and end states to the
// transitions between them.
func (p *Process) TransitionLookup() map[*State]map[*State][]*Trans {
	return p.transitionLookup
}

// AddTransition adds a transition betweent the given start and end state to
// the process and returns the new transition.
func (p *Process) AddTransition(start, end *State) *Trans {
	if _, ok := p.states[start]; !ok {
		panic("tried to add transition with unknown start state")
	} else if _, ok := p.states[end]; !ok {
		panic("tried to add transition with unknown end state")
	}

	trans := newTrans(start, end)

	p.transitions[trans] = struct{}{}
	p.transitionLookup[start][end] = append(p.transitionLookup[start][end], trans)

	start.transitions = append(start.transitions, trans)
	if start != end {
		end.transitions = append(end.transitions, trans)
	}

	return trans
}

// RemoveTransition removes the given transition from the process.
func (p *Process) RemoveTransition(trans *Trans) {
	if _, ok := p.transitions[trans]; !ok {
		panic("tried to remove unknown transition")
	}

	delete(p.transitions, trans)
	for i, t := range p.transitionLookup[trans.start][trans.end] {
		if t == trans {
			p.transitionLookup[trans.start][trans.end] = append(p.transitionLookup[trans.start][trans.end][:i], p.transitionLookup[trans.start][trans.end][i+1:]...)
			break
		}
	}
	for _, state := range [...]*State{trans.start, trans.end} {
		for i, t := range state.transitions {
			if t == trans {
				state.transitions = append(state.transitions[:i], state.transitions[i+1:]...)
				break
			}
		}
	}
}

// Queries returns all queries that are associated with the process.
func (p *Process) Queries() []*Query {
	return p.queries
}

// AddQuery adds a query to be associated with the process.
func (p *Process) AddQuery(query *Query) {
	p.queries = append(p.queries, query)
}

// AsXTA returns the xta (file format) representation of the process.
func (p *Process) AsXTA() string {
	s := "process " + p.name + "("
	s += strings.Join(p.params, ", ")
	s += ") {\n"
	s += p.decls.AsXTA() + "\n\n"
	s += "state\n"
	first := true
	for state := range p.states {
		if first {
			first = false
		} else {
			s += ",\n"
		}
		s += "    " + state.Name()
	}
	s += ";\n"
	for _, stateType := range []StateType{Committed, Urgent} {
		filteredStates := p.StatesWithType(stateType)
		if len(filteredStates) == 0 {
			continue
		}
		switch stateType {
		case Committed:
			s += "commit\n"
		case Urgent:
			s += "urgent\n"
		}
		first := true
		for _, state := range filteredStates {
			if first {
				first = false
			} else {
				s += ",\n"
			}
			s += "    " + state.Name()
		}
		s += ";\n"
	}
	s += "init\n"
	s += "    " + p.initialState.Name() + ";\n"
	s += "trans\n"
	i := 0
	for _, ends := range p.transitionLookup {
		for _, transitions := range ends {
			for _, trans := range transitions {
				s += "    " + trans.AsXTA()
				if i < len(p.transitions)-1 {
					s += ",\n"
				} else {
					s += ";\n"
				}
				i++
			}
		}
	}
	s += "}"
	return s
}

// AsUGI returns the ugi (file format) representation of the process.
func (p *Process) AsUGI() string {
	s := "process " + p.name + " graphinfo {\n"
	for state := range p.states {
		s += state.AsUGI()
	}
	for start, ends := range p.transitionLookup {
		for end, transitions := range ends {
			for index, trans := range transitions {
				s += trans.AsUGI(start.location, end.location, index)
			}
		}
	}
	s += "}"
	return s
}

func (p *Process) asXML(b *strings.Builder, indent string) {
	b.WriteString(indent + "<template>\n")
	b.WriteString(indent + "    <name>" + p.name + "</name>\n")
	if len(p.params) > 0 {
		b.WriteString(indent + "    <parameter>" + strings.Join(p.params, ", ") + "</parameter>\n")
	}
	b.WriteString(indent + "    <declaration>")
	xml.EscapeText(b, []byte(p.decls.AsXTA()))
	b.WriteString("</declaration>\n")

	stateIndices := make(map[*State]int, len(p.states))
	stateCount := 0
	for state := range p.states {
		stateIndex := stateCount
		stateIndices[state] = stateIndex
		stateCount++
		fmt.Fprintf(b, "%s    <location id=\"id%d\" x=\"%d\" y=\"%d\">\n",
			indent, stateIndex, state.location.X(), state.location.Y())
		fmt.Fprintf(b, "%s        <name x=\"%d\" y=\"%d\">%s</name>\n",
			indent, state.nameLocation.X(), state.nameLocation.Y(), state.name)
		if state.comment != "" {
			fmt.Fprintf(b, "%s    <label kind=\"comments\" x=\"%d\" y=\"%d\">",
				indent, state.commentLocation.X(), state.commentLocation.Y())
			xml.EscapeText(b, []byte(state.comment))
			b.WriteString("</label>\n")
		}
		if state.stateType == Committed {
			b.WriteString(indent + "        <committed/>\n")
		} else if state.stateType == Urgent {
			b.WriteString(indent + "        <urgent/>\n")
		}
		b.WriteString(indent + "    </location>\n")
	}
	fmt.Fprintf(b, "%s    <init ref=\"id%d\"/>\n", indent, stateIndices[p.initialState])

	for transition := range p.transitions {
		srcIndex := stateIndices[transition.start]
		tgtIndex := stateIndices[transition.end]

		b.WriteString(indent + "    <transition>\n")
		fmt.Fprintf(b, "%s        <source ref=\"id%d\"/>\n", indent, srcIndex)
		fmt.Fprintf(b, "%s        <target ref=\"id%d\"/>\n", indent, tgtIndex)
		if transition.selectStmts != "" {
			fmt.Fprintf(b, "%s        <label kind=\"select\" x=\"%d\" y=\"%d\">",
				indent, transition.selectLocation.X(), transition.selectLocation.Y())
			xml.EscapeText(b, []byte(transition.selectStmts))
			b.WriteString(indent + "</label>\n")
		}
		if transition.guardExpr != "" {
			fmt.Fprintf(b, "%s        <label kind=\"guard\" x=\"%d\" y=\"%d\">",
				indent, transition.guardLocation.X(), transition.guardLocation.Y())
			xml.EscapeText(b, []byte(transition.guardExpr))
			b.WriteString("</label>\n")
		}
		if transition.syncStmt != "" {
			fmt.Fprintf(b, "%s        <label kind=\"synchronisation\" x=\"%d\" y=\"%d\">",
				indent, transition.syncLocation.X(), transition.syncLocation.Y())
			xml.EscapeText(b, []byte(transition.syncStmt))
			b.WriteString("</label>\n")
		}
		if transition.updateStmts != "" {
			fmt.Fprintf(b, "%s        <label kind=\"assignment\" x=\"%d\" y=\"%d\">",
				indent, transition.updateLocation.X(), transition.updateLocation.Y())
			xml.EscapeText(b, []byte(transition.updateStmts))
			b.WriteString("</label>\n")
		}
		for _, nail := range transition.nails {
			fmt.Fprintf(b, "%s        <nail x=\"%d\" y=\"%d\"/>\n", indent, nail.X(), nail.Y())
		}
		b.WriteString(indent + "    </transition>\n")
	}

	b.WriteString(indent + "</template>")
}
