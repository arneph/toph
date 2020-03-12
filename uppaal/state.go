package uppaal

// StateType represents whether a state is normal, urgent, or committed.
type StateType int

const (
	// Normal represents a standard state in Uppaal that a process may remain
	// in for a while.
	Normal StateType = iota
	// Committed represents a state in Uppaal that must be left immediately.
	Committed
	// Urgent represents a state in Uppaal that must be left immediately and in
	// which time is frozen.
	Urgent
)

// State represents a state that is part of a process.
type State struct {
	name string

	stateType StateType

	// All locations are absolute. AsGUI() translates to relative coordinates.
	location     Location
	nameLocation Location
}

func newState(name string) *State {
	s := new(State)
	s.name = name

	s.stateType = Normal

	s.location = Location{}
	s.nameLocation = Location{}

	return s
}

// Name returns the name of the state.
func (s *State) Name() string {
	return s.name
}

// Type returns the type of the state.
func (s *State) Type() StateType {
	return s.stateType
}

// SetType sets the type of the state.
func (s *State) SetType(t StateType) {
	s.stateType = t
}

// Location returns the location of the state.
func (s *State) Location() Location {
	return s.location
}

// SetLocation sets the location of the state.
func (s *State) SetLocation(location Location) {
	s.location = location
}

// NameLocation returns the location of the name label of the state.
func (s *State) NameLocation() Location {
	return s.nameLocation
}

// SetNameLocation sets the location of the name label of the state.
func (s *State) SetNameLocation(nameLocation Location) {
	s.nameLocation = nameLocation
}

// SetLocationAndResetNameLocation sets the location of the state and sets the
// location of the name label of the state to the default, below the state.
func (s *State) SetLocationAndResetNameLocation(location Location) {
	s.location = location
	s.nameLocation = location.Add(Location{4, 16})
}

// AsUGI returns the ugi (file format) representation of the state.
func (s *State) AsUGI() string {
	relNameLocation := s.nameLocation.Sub(s.location)

	str := "location " + s.name + " " + s.location.AsUGI() + ";\n"
	str += "locationName " + s.name + relNameLocation.AsUGI() + ";\n"
	return str
}
