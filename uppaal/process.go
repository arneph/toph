package uppaal

import (
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

// AsXML returns the xml (file format) representation of the process.
func (p *Process) AsXML() string {
	s := "<template>\n"
	s += "    <name>" + p.name + "</name>\n"
	if len(p.params) > 0 {
		s += "    <parameter>" + strings.Join(p.params, ", ") + "</parameter>\n"
	}
	s += "    <declaration>" + escapeForXML(p.decls.AsXTA()) + "</declaration>\n"

	stateIndices := make(map[string]int, len(p.states))
	stateCount := 0
	for name, state := range p.states {
		stateIndex := stateCount
		stateIndices[name] = stateIndex
		stateCount++
		s += fmt.Sprintf("    <location id=\"id%d\" x=\"%d\" y=\"%d\">\n",
			stateIndex, state.location[0], state.location[1])
		s += fmt.Sprintf("        <name x=\"%d\" y=\"%d\">%s</name>\n",
			state.nameLocation[0], state.nameLocation[1], state.name)
		if state.comment != "" {
			s += fmt.Sprintf("    <label kind=\"comments\" x=\"%d\" y=\"%d\">",
				state.commentLocation[0], state.commentLocation[1])
			s += escapeForXML(state.comment)
			s += "</label>\n"
		}
		if state.stateType == Committed {
			s += "        <committed/>\n"
		} else if state.stateType == Urgent {
			s += "        <urgent/>\n"
		}
		s += "    </location>\n"
	}
	s += fmt.Sprintf("    <init ref=\"id%d\"/>\n", stateIndices[p.init])

	for _, transition := range p.transitions {
		srcIndex := stateIndices[transition.start]
		tgtIndex := stateIndices[transition.end]

		s += "    <transition>\n"
		s += fmt.Sprintf("        <source ref=\"id%d\"/>\n", srcIndex)
		s += fmt.Sprintf("        <target ref=\"id%d\"/>\n", tgtIndex)
		if transition.selectStmts != "" {
			s += fmt.Sprintf("        <label kind=\"select\" x=\"%d\" y=\"%d\">",
				transition.selectLocation[0], transition.selectLocation[1])
			s += escapeForXML(transition.selectStmts)
			s += "</label>\n"
		}
		if transition.guardExpr != "" {
			s += fmt.Sprintf("        <label kind=\"guard\" x=\"%d\" y=\"%d\">",
				transition.guardLocation[0], transition.guardLocation[1])
			s += escapeForXML(transition.guardExpr)
			s += "</label>\n"
		}
		if transition.syncStmt != "" {
			s += fmt.Sprintf("        <label kind=\"synchronisation\" x=\"%d\" y=\"%d\">",
				transition.syncLocation[0], transition.syncLocation[1])
			s += escapeForXML(transition.syncStmt)
			s += "</label>\n"
		}
		if transition.updateStmts != "" {
			s += fmt.Sprintf("        <label kind=\"assignment\" x=\"%d\" y=\"%d\">",
				transition.updateLocation[0], transition.updateLocation[1])
			s += escapeForXML(transition.updateStmts)
			s += "</label>\n"
		}
		for _, nail := range transition.nails {
			s += fmt.Sprintf("        <nail x=\"%d\" y=\"%d\"/>\n", nail[0], nail[1])
		}
		s += "    </transition>\n"
	}

	s += "</template>"
	return s
}
