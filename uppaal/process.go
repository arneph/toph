package uppaal

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Process represents a process in Uppaal.
type Process struct {
	name string

	params []string

	decls Declarations

	states      map[string]*State
	init        string
	transitions []*Trans

	queries []Query
}

func newProcess(name string) *Process {
	p := new(Process)
	p.name = name
	p.decls.initDeclarations("Place local declarations here.")
	p.states = make(map[string]*State)
	p.transitions = nil

	return p
}

// Name returns the name of the process.
func (p *Process) Name() string {
	return p.name
}

// AddParameter adds a parameter to the process.
func (p *Process) AddParameter(param string) {
	p.params = append(p.params, param)
}

// Declarations returns the declarations that are part of the process.
func (p *Process) Declarations() *Declarations {
	return &p.decls
}

// GetStates returns all states of the process.
func (p *Process) GetStates() []*State {
	var states []*State
	for _, state := range p.states {
		states = append(states, state)
	}
	return states
}

// GetStateWithName returns the state with the given name (if present).
func (p *Process) GetStateWithName(name string) *State {
	return p.states[name]
}

// GetStatesWithType returns all states with the given state type.
func (p *Process) GetStatesWithType(t StateType) []*State {
	var filteredStates []*State
	for _, state := range p.states {
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
		if _, ok := p.states[name]; ok {
			panic("naming collision when adding state")
		}

	} else if opt == Renaming {
		baseName := name
		if baseName == "" {
			baseName = "L"
		}
		for i := 0; ; i++ {
			name = fmt.Sprintf("%s%d", baseName, i)
			if _, ok := p.states[name]; !ok {
				break
			}
		}
	}

	s := newState(name)
	p.states[name] = s
	return s
}

// InitialState returns the initial state of the process.
func (p *Process) InitialState() *State {
	return p.states[p.init]
}

// SetInitialState changes the initial state of the process to the given state.
func (p *Process) SetInitialState(s *State) {
	t := p.states[s.Name()]
	if s != t {
		panic("tried to set state as initial that is outside of process")
	}
	p.init = s.Name()
}

// GetTrans returns all transitions between the given start and end states.
func (p *Process) GetTrans(start, end *State) []*Trans {
	var trans []*Trans
	for _, t := range p.transitions {
		if t.start == start.Name() && t.end == end.Name() {
			trans = append(trans, t)
		}
	}
	return trans
}

// AddTrans adds a transition betweent the given start and end state to the
// process and returns the new transition.
func (p *Process) AddTrans(startState, endState *State) *Trans {
	index := 1
	for _, trans := range p.transitions {
		if trans.start == startState.name && trans.end == endState.name {
			index++
		}
	}
	t := newTrans(startState.Name(), endState.Name(), index)
	p.transitions = append(p.transitions, t)
	return t
}

// Queries returns all queries that are associated with the process.
func (p *Process) Queries() []Query {
	return p.queries
}

// AddQuery adds a query to be associated with the process.
func (p *Process) AddQuery(query Query) {
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
	for _, state := range p.states {
		if first {
			first = false
		} else {
			s += ",\n"
		}
		s += "    " + state.Name()
	}
	s += ";\n"
	for _, stateType := range []StateType{Committed, Urgent} {
		filteredStates := p.GetStatesWithType(stateType)
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
	s += "    " + p.init + ";\n"
	s += "trans\n"
	i := 0
	for _, transition := range p.transitions {
		s += "    " + transition.AsXTA()
		if i < len(p.transitions)-1 {
			s += ",\n"
		} else {
			s += ";\n"
		}
		i++
	}
	s += "}"
	return s
}

// AsUGI returns the ugi (file format) representation of the process.
func (p *Process) AsUGI() string {
	s := "process " + p.name + " graphinfo {\n"
	for _, state := range p.states {
		s += state.AsUGI()
	}
	for _, trans := range p.transitions {
		startState := p.states[trans.start]
		endState := p.states[trans.end]
		s += trans.AsUGI(startState.location, endState.location)
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

	stateIndices := make(map[string]int, len(p.states))
	stateCount := 0
	for name, state := range p.states {
		stateIndex := stateCount
		stateIndices[name] = stateIndex
		stateCount++
		fmt.Fprintf(b, "%s    <location id=\"id%d\" x=\"%d\" y=\"%d\">\n",
			indent, stateIndex, state.location[0], state.location[1])
		fmt.Fprintf(b, "%s        <name x=\"%d\" y=\"%d\">%s</name>\n",
			indent, state.nameLocation[0], state.nameLocation[1], state.name)
		if state.comment != "" {
			fmt.Fprintf(b, "%s    <label kind=\"comments\" x=\"%d\" y=\"%d\">",
				indent, state.commentLocation[0], state.commentLocation[1])
			b.WriteString(escapeForXML(state.comment))
			b.WriteString("</label>\n")
		}
		if state.stateType == Committed {
			b.WriteString(indent + "        <committed/>\n")
		} else if state.stateType == Urgent {
			b.WriteString(indent + "        <urgent/>\n")
		}
		b.WriteString(indent + "    </location>\n")
	}
	fmt.Fprintf(b, "%s    <init ref=\"id%d\"/>\n", indent, stateIndices[p.init])

	for _, transition := range p.transitions {
		srcIndex := stateIndices[transition.start]
		tgtIndex := stateIndices[transition.end]

		b.WriteString(indent + "    <transition>\n")
		fmt.Fprintf(b, "%s        <source ref=\"id%d\"/>\n", indent, srcIndex)
		fmt.Fprintf(b, "%s        <target ref=\"id%d\"/>\n", indent, tgtIndex)
		if transition.selectStmts != "" {
			fmt.Fprintf(b, "%s        <label kind=\"select\" x=\"%d\" y=\"%d\">",
				indent, transition.selectLocation[0], transition.selectLocation[1])
			xml.EscapeText(b, []byte(transition.selectStmts))
			b.WriteString(indent + "</label>\n")
		}
		if transition.guardExpr != "" {
			fmt.Fprintf(b, "%s        <label kind=\"guard\" x=\"%d\" y=\"%d\">",
				indent, transition.guardLocation[0], transition.guardLocation[1])
			xml.EscapeText(b, []byte(transition.guardExpr))
			b.WriteString("</label>\n")
		}
		if transition.syncStmt != "" {
			fmt.Fprintf(b, "%s        <label kind=\"synchronisation\" x=\"%d\" y=\"%d\">",
				indent, transition.syncLocation[0], transition.syncLocation[1])
			xml.EscapeText(b, []byte(transition.syncStmt))
			b.WriteString("</label>\n")
		}
		if transition.updateStmts != "" {
			fmt.Fprintf(b, "%s        <label kind=\"assignment\" x=\"%d\" y=\"%d\">",
				indent, transition.updateLocation[0], transition.updateLocation[1])
			xml.EscapeText(b, []byte(transition.updateStmts))
			b.WriteString("</label>\n")
		}
		for _, nail := range transition.nails {
			fmt.Fprintf(b, "%s        <nail x=\"%d\" y=\"%d\"/>\n", indent, nail[0], nail[1])
		}
		b.WriteString(indent + "    </transition>\n")
	}

	b.WriteString(indent + "</template>")
}
