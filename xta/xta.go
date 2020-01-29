package xta

import "fmt"

type System struct {
	processes map[string]*Process
	instances []*ProcessInstance
}

func NewSystem() *System {
	s := new(System)
	s.processes = make(map[string]*Process)
	s.instances = nil

	return s
}

func (s *System) AddProcess(name string) *Process {
	baseName := name
	if baseName == "" {
		baseName = "Proc"
	}
	for i := 0; ; i++ {
		name = fmt.Sprintf("%s%c", baseName, rune('A'+i))
		if _, ok := s.processes[name]; !ok {
			break
		}
	}

	proc := newProcess(name)
	s.processes[name] = proc
	return proc
}

func (s *System) String() string {
	str := ""
	for _, proc := range s.processes {
		str += proc.String() + "\n\n"
	}
	return str
}

const Start = "start"
const End = "end"

type Process struct {
	name string

	states      map[string]*State
	transitions []*Trans
}

func newProcess(name string) *Process {
	p := new(Process)
	p.name = name
	p.states = make(map[string]*State)
	p.states[Start] = newState(Start)
	p.states[End] = newState(End)
	p.transitions = nil

	return p
}

func (p *Process) GetState(name string) *State {
	return p.states[name]
}

func (p *Process) AddState(name string) *State {
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

	s := newState(name)
	p.states[name] = s
	return s
}

func (p *Process) AddTrans(start, end *State) *Trans {
	t := newTrans(start.Name(), end.Name())
	p.transitions = append(p.transitions, t)
	return t
}

func (p *Process) String() string {
	s := "process " + p.name + "() {\n"
	s += "state\n"
	i := 0
	for _, state := range p.states {
		s += "    " + state.String()
		if i < len(p.states)-1 {
			s += ",\n"
		} else {
			s += ";\n"
		}
		i++
	}
	s += "init\n"
	s += "    " + Start + ";\n"
	s += "trans\n"
	i = 0
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

type State struct {
	name string
}

func newState(name string) *State {
	s := new(State)
	s.name = name

	return s
}

func (s *State) Name() string {
	return s.name
}

func (s *State) String() string {
	return s.name
}

type Trans struct {
	start, end  string
	transSelect string
	transGuard  string
	transSync   string
	transUpdate string
}

func newTrans(start, end string) *Trans {
	t := new(Trans)
	t.start = start
	t.end = end
	t.transSelect = ""
	t.transGuard = ""
	t.transSync = ""
	t.transUpdate = ""

	return t
}

func (t *Trans) String() string {
	s := t.start + " -> " + t.end
	s += " { "
	if t.transSelect != "" {
		s += "select " + t.transSelect + "; "
	}
	if t.transGuard != "" {
		s += "guard " + t.transGuard + "; "
	}
	if t.transSync != "" {
		s += "sync " + t.transSync + "; "
	}
	if t.transUpdate != "" {
		s += "update " + t.transUpdate + "; "
	}
	s += "}"
	return s
}

type ProcessInstance struct {
	procName string
	instName string
}

func newProcessInstance(procName, instName string) *ProcessInstance {
	i := new(ProcessInstance)
	i.procName = procName
	i.instName = instName

	return i
}
