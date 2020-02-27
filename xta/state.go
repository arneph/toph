package xta

type StateType int

const (
	Normal StateType = iota
	Urgent
	Commited
)

type State struct {
	name      string
	stateType StateType
}

func newState(name string) *State {
	s := new(State)
	s.name = name
	s.stateType = Normal

	return s
}

func (s *State) Name() string {
	return s.name
}

func (s *State) StateType() StateType {
	return s.stateType
}

func (s *State) SetStateType(t StateType) {
	s.stateType = t
}

func (s *State) String() string {
	return s.name
}
