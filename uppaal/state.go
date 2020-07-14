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
	name      string
	isInitial bool
	stateType StateType
	comment   string

	transitions []*Trans

	// All locations are absolute. AsGUI() translates to relative coordinates.
	location        Location
	nameLocation    Location
	commentLocation Location
}

func newState(name string) *State {
	s := new(State)
	s.name = name
	s.isInitial = false
	s.stateType = Normal
	s.comment = ""
	s.transitions = nil
	s.location = Location{}
	s.nameLocation = Location{}
	s.commentLocation = Location{}

	return s
}

// Name returns the name of the state.
func (s *State) Name() string {
	return s.name
}

// IsInitialState returns if the state is the initial state within its process.
func (s *State) IsInitialState() bool {
	return s.isInitial
}

// Type returns the type of the state.
func (s *State) Type() StateType {
	return s.stateType
}

// SetType sets the type of the state.
func (s *State) SetType(t StateType) {
	s.stateType = t
}

// Comment returns the comment for the state.
func (s *State) Comment() string {
	return s.comment
}

// SetComment sets the comment for the state.
func (s *State) SetComment(comment string) {
	s.comment = comment
}

// Transitions returns all transitions beginning and/or ending at the state.
func (s *State) Transitions() []*Trans {
	return s.transitions
}

// IncomingTransitions returns all transitions ending in the state.
func (s *State) IncomingTransitions() []*Trans {
	var incomingTransitions []*Trans
	for _, t := range s.transitions {
		if t.end == s {
			incomingTransitions = append(incomingTransitions, t)
		}
	}
	return incomingTransitions
}

// OutgoingTransitions returns all transitions starting from the state.
func (s *State) OutgoingTransitions() []*Trans {
	var outgoingTransitions []*Trans
	for _, t := range s.transitions {
		if t.start == s {
			outgoingTransitions = append(outgoingTransitions, t)
		}
	}
	return outgoingTransitions
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

// CommentLocation returns the location of the comment label of the state.
func (s *State) CommentLocation() Location {
	return s.commentLocation
}

// SetCommentLocation sets the location of the comment label of the state.
func (s *State) SetCommentLocation(commentLocation Location) {
	s.commentLocation = commentLocation
}

// SetLocationAndResetNameAndCommentLocation sets the location of the state and sets the
// location of the name label of the state to the default, below the state.
func (s *State) SetLocationAndResetNameAndCommentLocation(location Location) {
	s.location = location
	s.nameLocation = location.Add(Location{4, 16})
	s.commentLocation = location.Add(Location{4, 34})
}

// AsUGI returns the ugi (file format) representation of the state.
func (s *State) AsUGI() string {
	relNameLocation := s.nameLocation.Sub(s.location)

	str := "location " + s.name + " " + s.location.AsUGI() + ";\n"
	str += "locationName " + s.name + relNameLocation.AsUGI() + ";\n"
	return str
}
