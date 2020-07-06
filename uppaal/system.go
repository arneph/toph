package uppaal

import (
	"sort"
	"strings"
)

// System represents a complete collection of global declarations, processes,
// and process instances.
type System struct {
	decls Declarations

	processes map[string]*Process
	instances map[string]*ProcessInstance

	progressMeasures []string

	queries []Query
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

// AddProgressMeasure adds a progress measure expression to the system. The
// expression has to be monotonically increasing as the system gets simulated.
func (s *System) AddProgressMeasure(measure string) {
	s.progressMeasures = append(s.progressMeasures, measure)
}

// AddQuery adds a query to be associated with the system.
func (s *System) AddQuery(query Query) {
	s.queries = append(s.queries, query)
}

// AsXTA returns the xta (file format) representation of the system.
func (s *System) AsXTA() string {
	str := s.decls.AsXTA() + "\n\n"

	for _, proc := range s.sortedProcesses() {
		str += proc.AsXTA() + "\n\n"
	}

	sortedInstances := s.sortedInstances()
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
	if len(s.progressMeasures) > 0 {
		str += "progress{\n"
		for _, measure := range s.progressMeasures {
			str += "    " + measure + ";\n"
		}
		str += "}\n"
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

	for _, query := range s.queries {
		str += query.AsQ()
	}

	sortedInstances := s.sortedInstances()
	for _, inst := range sortedInstances {
		proc := s.processes[inst.procName]

		for _, procQuery := range proc.queries {
			instQuery := procQuery.Substitute(inst.instName)
			str += instQuery.AsQ()
		}
	}

	return str
}

// AsXML returns the xml (file format) representation of the system.
func (s *System) AsXML() string {
	var str string
	str += "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"
	str += "<!DOCTYPE nta PUBLIC '-//Uppaal Team//DTD Flat System 1.1//EN' 'http://www.it.uu.se/research/group/darts/uppaal/flat-1_2.dtd'>\n"
	str += "<nta>\n"

	str += "    <declaration>"
	str += escapeForXML(s.decls.AsXTA())
	str += "    </declaration>\n"

	for _, proc := range s.sortedProcesses() {
		xml := proc.AsXML()
		str += "    " + strings.ReplaceAll(xml, "\n<", "\n    <") + "\n"
	}

	str += "    <system>\n"
	sortedInstances := s.sortedInstances()
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
	if len(s.progressMeasures) > 0 {
		str += "progress{\n"
		for _, measure := range s.progressMeasures {
			str += "    " + measure + ";\n"
		}
		str += "}\n"
	}
	str += "</system>\n"

	str += "    <queries>\n"
	for _, query := range s.queries {
		xml := query.AsXML()
		str += "        " + strings.ReplaceAll(xml, "\n<", "\n        <") + "\n"
	}
	for _, inst := range sortedInstances {
		proc := s.processes[inst.procName]

		for _, procQuery := range proc.queries {
			instQuery := procQuery.Substitute(inst.instName)
			xml := instQuery.AsXML()
			str += "        " + strings.ReplaceAll(xml, "\n<", "\n        <") + "\n"
		}
	}
	str += "    </queries>\n"

	str += "</nta>\n"
	return str
}

func (s *System) sortedProcesses() []*Process {
	sortedProcesses := make([]*Process, 0, len(s.processes))
	for _, proc := range s.processes {
		sortedProcesses = append(sortedProcesses, proc)
	}
	sort.Slice(sortedProcesses, func(i, j int) bool {
		return sortedProcesses[i].name < sortedProcesses[j].name
	})
	return sortedProcesses
}

func (s *System) sortedInstances() []*ProcessInstance {
	sortedInstances := make([]*ProcessInstance, 0, len(s.instances))
	for _, inst := range s.instances {
		sortedInstances = append(sortedInstances, inst)
	}
	sort.Slice(sortedInstances, func(i, j int) bool {
		return sortedInstances[i].instName < sortedInstances[j].instName
	})
	return sortedInstances
}
