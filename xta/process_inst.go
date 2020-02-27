package xta

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

func (i *ProcessInstance) AddParameter(param string) {
	i.params = append(i.params, param)
}

func (i *ProcessInstance) CanSkipDeclaration() bool {
	return i.procName == i.instName && len(i.params) == 0
}

func (i *ProcessInstance) String() string {
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
