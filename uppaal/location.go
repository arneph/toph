package uppaal

import "fmt"

// Location represents the 2D coordinates of a state or transition nail.
type Location [2]int

// Add returns a new location with the given offset from the existing location
// added.
func (l Location) Add(offset Location) Location {
	return Location{l[0] + offset[0], l[1] + offset[1]}
}

// Sub returns a new location with the given offset from the existing location
// subtracted.
func (l Location) Sub(offset Location) Location {
	return Location{l[0] - offset[0], l[1] - offset[1]}
}

// Min returns the minimum x and y coordinates of the given locations as a new
// location.
func Min(locs ...Location) Location {
	minX := locs[0][0]
	minY := locs[0][1]
	for _, l := range locs {
		if minX > l[0] {
			minX = l[0]
		}
		if minY > l[1] {
			minY = l[1]
		}
	}
	return Location{minX, minY}
}

// Max returns the maximum x and y coordinates of the given locations as a new
// location.
func Max(locs ...Location) Location {
	maxX := locs[0][0]
	maxY := locs[0][1]
	for _, l := range locs {
		if maxX < l[0] {
			maxX = l[0]
		}
		if maxY < l[1] {
			maxY = l[1]
		}
	}
	return Location{maxX, maxY}
}

// AsUGI returns the ugi (file format) representation of the point.
func (l Location) AsUGI() string {
	return fmt.Sprintf("(%d,%d)", l[0], l[1])
}

func absoluteToTransRelative(absolute, start, end Location) (relative Location) {
	relative[0] = absolute[0] - (start[0]+end[0])/2
	relative[1] = absolute[1] - (start[1]+end[1])/2
	return
}
