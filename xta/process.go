package xta

import (
	"fmt"
	"strings"
)

type Process struct {
	name string

	params []string

	decls Declarations

	states      map[string]*State
	init        string
	transitions []*Trans
}

func newProcess(name string) *Process {
	p := new(Process)
	p.name = name
	p.decls.initDeclarations("Place local declarations here.")
	p.states = make(map[string]*State)
	p.transitions = nil

	return p
}

func (p *Process) Name() string {
	return p.name
}

func (p *Process) AddParameter(param string) {
	p.params = append(p.params, param)
}

func (p *Process) Declarations() *Declarations {
	return &p.decls
}

func (p *Process) GetStates() []*State {
	var states []*State
	for _, state := range p.states {
		states = append(states, state)
	}
	return states
}

func (p *Process) GetStateWithName(name string) *State {
	return p.states[name]
}

func (p *Process) GetStatesWithType(t StateType) []*State {
	var filteredStates []*State
	for _, state := range p.states {
		if state.StateType() != t {
			continue
		}
		filteredStates = append(filteredStates, state)
	}
	return filteredStates
}

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

func (p *Process) InitialState() *State {
	return p.states[p.init]
}

func (p *Process) SetInitialState(s *State) {
	t := p.states[s.Name()]
	if s != t {
		panic("tried to set state as initial that is outside of process")
	}
	p.init = s.Name()
}

func (p *Process) GetTrans(start, end *State) *Trans {
	for _, t := range p.transitions {
		if t.start == start.Name() && t.end == end.Name() {
			return t
		}
	}
	return nil
}

func (p *Process) AddTrans(start, end *State) *Trans {
	t := newTrans(start.Name(), end.Name())
	p.transitions = append(p.transitions, t)
	return t
}

func (p *Process) String() string {
	s := "process " + p.name + "("
	s += strings.Join(p.params, ", ")
	s += ") {\n"
	s += p.decls.String() + "\n\n"
	s += "state\n"
	first := true
	for _, state := range p.states {
		if first {
			first = false
		} else {
			s += ",\n"
		}
		s += "    " + state.String()
	}
	s += ";\n"
	for _, stateType := range []StateType{Commited, Urgent} {
		filteredStates := p.GetStatesWithType(stateType)
		if len(filteredStates) == 0 {
			continue
		}
		switch stateType {
		case Commited:
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
			s += "    " + state.String()
		}
		s += ";\n"
	}
	s += "init\n"
	s += "    " + p.init + ";\n"
	s += "trans\n"
	i := 0
	for _, transition := range p.transitions {
		s += "    " + transition.String()
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
