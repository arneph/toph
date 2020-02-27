package xta

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

func (t *Trans) Select() string {
	return t.transSelect
}

func (t *Trans) SetSelect(s string) {
	t.transSelect = s
}

func (t *Trans) Guard() string {
	return t.transGuard
}

func (t *Trans) SetGuard(g string) {
	t.transGuard = g
}

func (t *Trans) Sync() string {
	return t.transSync
}

func (t *Trans) SetSync(s string) {
	t.transSync = s
}

func (t *Trans) Update() string {
	return t.transUpdate
}

func (t *Trans) AddUpdate(u string) {
	if t.transUpdate == "" {
		t.transUpdate = u
	} else {
		t.transUpdate += ", " + u
	}
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
		s += "assign " + t.transUpdate + "; "
	}
	s += "}"
	return s
}
