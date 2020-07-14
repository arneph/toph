package uppaal

import (
	"fmt"
	"strings"
)

// ProcessInstance represents a process instance in System declarations.
type ProcessInstance struct {
	proc   *Process
	name   string
	params []string
}

func newProcessInstance(proc *Process, instName string) *ProcessInstance {
	i := new(ProcessInstance)
	i.proc = proc
	i.name = instName

	return i
}

// Process returns the processes the process instance instantiates.
func (i *ProcessInstance) Process() *Process {
	return i.proc
}

// Name returns the name of the process instance.
func (i *ProcessInstance) Name() string {
	return i.name
}

// Parameters returns all paramters of the the process instantiation call.
func (i *ProcessInstance) Parameters() []string {
	return i.params
}

// AddParameter adds a parameter to the process instantiation call.
func (i *ProcessInstance) AddParameter(param string) {
	i.params = append(i.params, param)
}

// CanSkipDeclaration returns whether the process needs to be explicitly
// instantiated or if it can be instantiated implicitly with the system
// statement at the end of System declarations.
func (i *ProcessInstance) CanSkipDeclaration() bool {
	return i.proc.name == i.name && len(i.params) == 0
}

// AsXTA returns the xta (file format) representation of the process instance.
func (i *ProcessInstance) AsXTA() string {
	return fmt.Sprintf("%s = %s(%s);",
		i.name,
		i.proc.name,
		strings.Join(i.Parameters(), ", "))
}
