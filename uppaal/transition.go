package uppaal

import "fmt"

// Trans represents a transition between two states in a process.
type Trans struct {
	start, end *State

	selectStmts       string
	guardExpr         string
	guardUsesGlobals  bool
	syncStmt          string
	updateStmts       string
	updateUsesGlobals bool

	// All locations are absolute. AsUGI() translates to relative coordinates.
	nails          []Location
	selectLocation Location
	guardLocation  Location
	syncLocation   Location
	updateLocation Location
}

func newTrans(start, end *State) *Trans {
	t := new(Trans)
	t.start = start
	t.end = end

	t.selectStmts = ""
	t.guardExpr = ""
	t.guardUsesGlobals = false
	t.syncStmt = ""
	t.updateStmts = ""
	t.updateUsesGlobals = false

	t.nails = nil
	t.selectLocation = Location{}

	return t
}

// Start returns the start state of the transition.
func (t *Trans) Start() *State {
	return t.start
}

// End returns the end state of the transition.
func (t *Trans) End() *State {
	return t.end
}

// Select returns all select statements of the transition.
func (t *Trans) Select() string {
	return t.selectStmts
}

// AddSelect adds the select statement to the transition.
func (t *Trans) AddSelect(selectStmt string) {
	if t.selectStmts == "" {
		t.selectStmts = selectStmt
	} else {
		t.selectStmts += ", " + selectStmt
	}
}

// Guard returns the guard of the transition that has to be fulfilled to enable
// the transition.
func (t *Trans) Guard() string {
	return t.guardExpr
}

// SetGuard sets the guard of the transition that has to be fulfilled to enable
// the transition.
func (t *Trans) SetGuard(guardExpr string, usesGlobals bool) {
	t.guardExpr = guardExpr
	t.guardUsesGlobals = usesGlobals
}

// GuardUsesGlobals returns if the guard expression uses global variables.
func (t *Trans) GuardUsesGlobals() bool {
	return t.guardUsesGlobals
}

// Sync returns the sync statement (on a Uppaal channel) of the transition.
func (t *Trans) Sync() string {
	return t.syncStmt
}

// SetSync sets the sync statement (on a Uppaal channel) of the transition.
func (t *Trans) SetSync(syncStmt string) {
	t.syncStmt = syncStmt
}

// Update returns all update statements that are executed when the transition
// gets taken.
func (t *Trans) Update() string {
	return t.updateStmts
}

// AddUpdate adds an update statement to the transition that gets executed when
// the transition gets taken.
func (t *Trans) AddUpdate(updateStmt string, usesGlobals bool) {
	if t.updateStmts == "" {
		t.updateStmts = updateStmt
	} else {
		t.updateStmts += ", " + updateStmt
	}
	t.updateUsesGlobals = t.updateUsesGlobals || usesGlobals
}

// UpdateUsesGlobals returns if any update statement uses global variables.
func (t *Trans) UpdateUsesGlobals() bool {
	return t.updateUsesGlobals
}

// UsesGlobals returns if the transition uses global variables.
func (t *Trans) UsesGlobals() bool {
	return t.guardUsesGlobals || t.syncStmt != "" || t.updateUsesGlobals
}

// NailLocations returns the locations of all nails used by the transition in
// order.
func (t *Trans) NailLocations() []Location {
	return t.nails
}

// AddNail adds the location of a nail to the transition.
func (t *Trans) AddNail(nail Location) {
	t.nails = append(t.nails, nail)
}

// SelectLocation returns the location of the select label.
func (t *Trans) SelectLocation() Location {
	return t.selectLocation
}

// SetSelectLocation sets the location of the select label.
func (t *Trans) SetSelectLocation(selectLocation Location) {
	t.selectLocation = selectLocation
}

// GuardLocation returns the location of the guard label.
func (t *Trans) GuardLocation() Location {
	return t.guardLocation
}

// SetGuardLocation sets the location of the guard label.
func (t *Trans) SetGuardLocation(guardLocation Location) {
	t.guardLocation = guardLocation
}

// SyncLocation returns the  location of the sync label.
func (t *Trans) SyncLocation() Location {
	return t.syncLocation
}

// SetSyncLocation sets the location of the sync label.
func (t *Trans) SetSyncLocation(syncLocation Location) {
	t.syncLocation = syncLocation
}

// UpdateLocation returns the location of the update label.
func (t *Trans) UpdateLocation() Location {
	return t.updateLocation
}

// SetUpdateLocation sets the location of the update label.
func (t *Trans) SetUpdateLocation(updateLocation Location) {
	t.updateLocation = updateLocation
}

// AsXTA returns the xta (file format) representation of the transition.
func (t *Trans) AsXTA() string {
	s := t.start.Name() + " -> " + t.end.Name()
	s += " { "
	if t.selectStmts != "" {
		s += "select " + t.selectStmts + "; "
	}
	if t.guardExpr != "" {
		s += "guard " + t.guardExpr + "; "
	}
	if t.syncStmt != "" {
		s += "sync " + t.syncStmt + "; "
	}
	if t.updateStmts != "" {
		s += "assign " + t.updateStmts + "; "
	}
	s += "}"
	return s
}

// AsUGI returns the ugi (file format) representation of the transition, given
// the locations of the start and end state of the transition. The locations
// are necessary to compute relative locations.
func (t *Trans) AsUGI(startLocation, endLocation Location, index int) string {
	id := fmt.Sprintf("%s %s %d", t.start.name, t.end.name, index)
	var s string
	for _, nail := range t.nails {
		p := absoluteToTransRelative(nail, startLocation, endLocation)
		s += "trans " + id + " " + p.AsUGI() + ";\n"
	}
	if t.selectStmts != "" {
		p := absoluteToTransRelative(t.selectLocation, startLocation, endLocation)
		s += "select " + id + " " + p.AsUGI() + ";\n"
	}
	if t.guardExpr != "" {
		p := absoluteToTransRelative(t.guardLocation, startLocation, endLocation)
		s += "guard " + id + " " + p.AsUGI() + ";\n"
	}
	if t.syncStmt != "" {
		p := absoluteToTransRelative(t.syncLocation, startLocation, endLocation)
		s += "sync " + id + " " + p.AsUGI() + ";\n"
	}
	if t.updateStmts != "" {
		p := absoluteToTransRelative(t.updateLocation, startLocation, endLocation)
		s += "assign " + id + " " + p.AsUGI() + ";\n"
	}
	return s
}
