package uppaal

import (
	"sort"
)

// System represents a complete collection of global declarations, processes,
// and process instances.
type System struct {
	decls Declarations

	processes map[string]*Process
	instances map[string]*ProcessInstance
}

// NewSystem creates a new system.
func NewSystem() *System {
	s := new(System)
	s.decls.initDeclarations("Place global declarations here.")
	s.processes = make(map[string]*Process)
	s.instances = make(map[string]*ProcessInstance)

	return s
}

// Declarations returns all global declarations of the system.
func (s *System) Declarations() *Declarations {
	return &s.decls
}

// AddProcess adds a process with the given name (after possible renaming to
// avoid naming conflicts) to the system and returns the new process.
func (s *System) AddProcess(name string) *Process {
	if _, ok := s.processes[name]; ok {
		panic("naming collision when adding process")
	}
	if _, ok := s.instances[name]; ok {
		panic("naming collision when adding process")
	}

	proc := newProcess(name)
	s.processes[name] = proc
	return proc
}

// AddProcessInstance adds an instance of a process with the given name (after
// possible renaming to avoid naming conflicts) to the system and returns the
// new instance.
func (s *System) AddProcessInstance(procName, instName string) *ProcessInstance {
	if _, ok := s.instances[instName]; ok {
		panic("naming collision when adding process instance")
	}
	if _, ok := s.processes[procName]; !ok {
		panic("tried to instantiate non-existent process")
	}

	inst := newProcessInstance(procName, instName)
	s.instances[instName] = inst
	return inst
}

// AsXTA returns the xta (file format) representation of the system.
func (s *System) AsXTA() string {
	str := s.decls.AsXTA() + "\n\n"

	sortedProcesses := make([]*Process, 0, len(s.processes))
	for _, proc := range s.processes {
		sortedProcesses = append(sortedProcesses, proc)
	}
	sort.Slice(sortedProcesses, func(i, j int) bool {
		return sortedProcesses[i].name < sortedProcesses[j].name
	})
	for _, proc := range sortedProcesses {
		str += proc.AsXTA() + "\n\n"
	}

	sortedInstances := make([]*ProcessInstance, 0, len(s.instances))
	for _, inst := range s.instances {
		sortedInstances = append(sortedInstances, inst)
	}
	sort.Slice(sortedInstances, func(i, j int) bool {
		return sortedInstances[i].instName < sortedInstances[j].instName
	})
	for _, inst := range sortedInstances {
		if inst.CanSkipDeclaration() {
			continue
		}
		str += inst.AsXTA() + "\n"
	}
	if len(sortedInstances) > 0 {
		str += "system "
		first := true
		for _, inst := range sortedInstances {
			if first {
				first = false
			} else {
				str += ", "
			}
			str += inst.instName
		}
		str += ";\n"
	}

	return str
}

// AsUGI returns the gui (file format) representation of the system.
func (s *System) AsUGI() string {
	var str string
	for _, proc := range s.processes {
		str += proc.AsUGI() + "\n\n"
	}
	return str
}

// AsQ returns the q (file format) representation of the system.
func (s *System) AsQ() string {
	var str string

	sortedInstances := make([]*ProcessInstance, 0, len(s.instances))
	for _, inst := range s.instances {
		sortedInstances = append(sortedInstances, inst)
	}
	sort.Slice(sortedInstances, func(i, j int) bool {
		return sortedInstances[i].instName < sortedInstances[j].instName
	})
	for _, inst := range sortedInstances {
		proc := s.processes[inst.procName]

		for _, procQuery := range proc.queries {
			instQuery := procQuery.Substitute(inst.instName)
			str += instQuery.AsQ()
		}
	}

	return str
}
