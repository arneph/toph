package uppaal

import (
	"encoding/xml"
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

	queries []*Query
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

// Processes returns all processes in the system.
func (s *System) Processes() []*Process {
	processes := make([]*Process, 0, len(s.processes))
	for _, proc := range s.processes {
		processes = append(processes, proc)
	}
	return processes
}

// AddProcess adds a process with the given name to the system and returns the
// new process.
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

// ProcessInstances returns all processes in the system.
func (s *System) ProcessInstances() []*ProcessInstance {
	instances := make([]*ProcessInstance, 0, len(s.instances))
	for _, inst := range s.instances {
		instances = append(instances, inst)
	}
	return instances
}

// AddProcessInstance adds an instance of a process with the given name (after
// possible renaming to avoid naming conflicts) to the system and returns the
// new instance.
func (s *System) AddProcessInstance(proc *Process, name string) *ProcessInstance {
	if _, ok := s.instances[name]; ok {
		panic("naming collision when adding process instance")
	}
	if _, ok := s.processes[proc.Name()]; !ok {
		panic("tried to instantiate non-existent process")
	}

	inst := newProcessInstance(proc, name)
	s.instances[name] = inst
	return inst
}

// ProgressMeasures returns all progress measures of the system.
func (s *System) ProgressMeasures() []string {
	return s.progressMeasures
}

// AddProgressMeasure adds a progress measure expression to the system. The
// expression has to be monotonically increasing as the system gets simulated.
func (s *System) AddProgressMeasure(measure string) {
	s.progressMeasures = append(s.progressMeasures, measure)
}

// Queries returns all system queries (excluding process specific queries).
func (s *System) Queries() []*Query {
	return s.queries
}

// AddQuery adds a query to be associated with the system.
func (s *System) AddQuery(query *Query) {
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
			str += inst.Name()
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

	queryNumber := 1
	for _, query := range s.queries {
		str += query.AsQ(queryNumber)
		queryNumber++
	}

	sortedInstances := s.sortedInstances()
	for _, inst := range sortedInstances {
		proc := s.processes[inst.Process().Name()]

		for _, procQuery := range proc.queries {
			instQuery := procQuery.Substitute(inst.Name())
			str += instQuery.AsQ(queryNumber)
			queryNumber++
		}
	}

	return str
}

// AsXML returns the xml (file format) representation of the system.
func (s *System) AsXML() string {
	var b strings.Builder

	b.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	b.WriteString("<!DOCTYPE nta PUBLIC '-//Uppaal Team//DTD Flat System 1.1//EN' 'http://www.it.uu.se/research/group/darts/uppaal/flat-1_2.dtd'>\n")
	b.WriteString("<nta>\n")

	b.WriteString("    <declaration>")
	xml.EscapeText(&b, []byte(s.decls.AsXTA()))
	b.WriteString("    </declaration>\n")

	for _, proc := range s.sortedProcesses() {
		proc.asXML(&b, "    ")
		b.WriteString("\n")
	}

	b.WriteString("    <system>\n")
	sortedInstances := s.sortedInstances()
	for _, inst := range sortedInstances {
		if inst.CanSkipDeclaration() {
			continue
		}
		b.WriteString(inst.AsXTA() + "\n")
	}
	if len(sortedInstances) > 0 {
		b.WriteString("system ")
		first := true
		for _, inst := range sortedInstances {
			if first {
				first = false
			} else {
				b.WriteString(", ")
			}
			b.WriteString(inst.Name())
		}
		b.WriteString(";\n")
	}
	if len(s.progressMeasures) > 0 {
		b.WriteString("progress{\n")
		for _, measure := range s.progressMeasures {
			b.WriteString("    " + measure + ";\n")
		}
		b.WriteString("}\n")
	}
	b.WriteString("</system>\n")

	b.WriteString("    <queries>\n")
	queryNumber := 1
	for _, query := range s.queries {
		query.asXML(&b, queryNumber, "        ")
		b.WriteString("\n")
		queryNumber++
	}
	for _, inst := range sortedInstances {
		for _, procQuery := range inst.Process().Queries() {
			instQuery := procQuery.Substitute(inst.Name())
			instQuery.asXML(&b, queryNumber, "        ")
			b.WriteString("\n")
			queryNumber++
		}
	}
	b.WriteString("    </queries>\n")

	b.WriteString("</nta>\n")
	return b.String()
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
		return sortedInstances[i].Name() < sortedInstances[j].Name()
	})
	return sortedInstances
}
