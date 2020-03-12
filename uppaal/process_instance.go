package uppaal

// ProcessInstance represents a process instance in System declarations.
type ProcessInstance struct {
	procName string
	params   []string

	instName string
}

func newProcessInstance(procName, instName string) *ProcessInstance {
	i := new(ProcessInstance)
	i.procName = procName
	i.instName = instName

	return i
}

// AddParameter adds a parameter to the process instantiation call.
func (i *ProcessInstance) AddParameter(param string) {
	i.params = append(i.params, param)
}

// CanSkipDeclaration returns whether the process needs to be explicitly
// instantiated or if it can be instantiated implicitly with the system
// statement at the end of System declarations.
func (i *ProcessInstance) CanSkipDeclaration() bool {
	return i.procName == i.instName && len(i.params) == 0
}

// AsXTA returns the xta (file format) representation of the process instance.
func (i *ProcessInstance) AsXTA() string {
	str := i.instName + " = " + i.procName + "("
	first := true
	for _, param := range i.params {
		if first {
			first = false
		} else {
			str += ", "
		}
		str += param
	}
	str += ");"
	return str
}
