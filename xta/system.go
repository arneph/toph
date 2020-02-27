package xta

import (
	"fmt"
	"sort"
)

type System struct {
	decls Declarations

	processes map[string]*Process
	instances map[string]*ProcessInstance
}

func NewSystem() *System {
	s := new(System)
	s.decls.initDeclarations("Place global declarations here.")
	s.processes = make(map[string]*Process)
	s.instances = make(map[string]*ProcessInstance)

	return s
}

func (s *System) Declarations() *Declarations {
	return &s.decls
}

func (s *System) AddProcess(name string, opt RenamingOption) *Process {
	if opt == NoRenaming {
		if _, ok := s.processes[name]; ok {
			panic("naming collision when adding process")
		}
		if _, ok := s.instances[name]; ok {
			panic("naming collision when adding process")
		}

	} else if opt == Renaming {
		baseName := name
		if baseName == "" {
			baseName = "Proc"
		}
		for i := 0; ; i++ {
			name = fmt.Sprintf("%s%c", baseName, rune('A'+i))
			if _, ok := s.processes[name]; ok {
				continue
			} else if _, ok := s.instances[name]; ok {
				continue
			}
			break
		}
	}

	proc := newProcess(name)
	s.processes[name] = proc
	return proc
}

func (s *System) AddProcessInstance(procName, instName string, opt RenamingOption) *ProcessInstance {
	if opt == NoRenaming {
		if _, ok := s.instances[instName]; ok {
			panic("naming collision when adding process")
		}

	} else if opt == Renaming {
		baseName := instName
		if baseName == "" {
			baseName = procName
		}
		for i := 1; ; i++ {
			instName = fmt.Sprintf("%s%d", baseName, i)
			if _, ok := s.processes[instName]; ok {
				continue
			} else if _, ok := s.instances[instName]; ok {
				continue
			}
			break
		}
	}
	if _, ok := s.processes[procName]; !ok {
		panic("tried to instantiate non-existent process")
	}

	inst := newProcessInstance(procName, instName)
	s.instances[instName] = inst
	return inst
}

func (s *System) String() string {
	str := s.decls.String() + "\n\n"
	for _, proc := range s.processes {
		str += proc.String() + "\n\n"
	}
	sortedInstances := make([]*ProcessInstance, 0, len(s.instances))
	for _, i := range s.instances {
		sortedInstances = append(sortedInstances, i)
	}
	sort.Slice(sortedInstances, func(i, j int) bool {
		return sortedInstances[i].instName < sortedInstances[j].instName
	})
	for _, inst := range sortedInstances {
		if inst.CanSkipDeclaration() {
			continue
		}
		str += inst.String() + "\n"
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
