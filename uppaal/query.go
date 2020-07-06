package uppaal

import (
	"fmt"
	"strings"
)

// QueryCategory identifies what system property a query is for.
type QueryCategory int

const (
	// ResourceBoundUnreached verifies the system can never run out of resources.
	ResourceBoundUnreached QueryCategory = iota
	// ChannelSafety verifies the system never performs disallowed channel operations.
	ChannelSafety
	// MutexSafety verifies the system never performs disallowed mutex operations.
	MutexSafety
	// WaitGroupSafety verifies the system never performs disallowed wait group operations.
	WaitGroupSafety
	// NoChannelRelatedDeadlocks verifies the system is never stuck waiting on a channel operation.
	NoChannelRelatedDeadlocks
	// NoMutexRelatedDeadlocks verifies the system is never stuck waiting on a mutex operation.
	NoMutexRelatedDeadlocks
	// NoWaitGroupRelatedDeadlocks verifies the system is never stuck waiting on a wait group operation.
	NoWaitGroupRelatedDeadlocks
	// NoFunctionCallsWithNilVariable verifies the system is attempting to call a nil (-1) function variable.
	NoFunctionCallsWithNilVariable
	// ReachabilityRequirements are user generated and verify that a certain state is or is not reachable.
	ReachabilityRequirements
)

func (c QueryCategory) String() string {
	switch c {
	case ResourceBoundUnreached:
		return "resource bound unreached"
	case ChannelSafety:
		return "channel safety"
	case MutexSafety:
		return "mutex safety"
	case WaitGroupSafety:
		return "wait group safety"
	case NoChannelRelatedDeadlocks:
		return "no channel related deadlocks"
	case NoMutexRelatedDeadlocks:
		return "no mutex related deadlocks"
	case NoWaitGroupRelatedDeadlocks:
		return "no wait group related deadlocks"
	case NoFunctionCallsWithNilVariable:
		return "no function calls with nil variable"
	case ReachabilityRequirements:
		return "reachability requirements"
	default:
		panic(fmt.Errorf("unexpected query category: %d", c))
	}
}

// Query holds a query and its comment. Both can be exported to a query file.
type Query struct {
	// The '$' serves as a placeholder.
	query       string
	description string
	category    QueryCategory
}

// MakeQuery returns a query with the given query string and comment.
func MakeQuery(query, description string, category QueryCategory) Query {
	return Query{
		query:       query,
		description: description,
		category:    category,
	}
}

// Substitute returns a query with all placeholders in the query string
// replaced by the given replacement.
func (q Query) Substitute(replacement string) Query {
	return Query{
		query:       strings.ReplaceAll(q.query, "$", replacement),
		description: q.description,
		category:    q.category,
	}
}

// AsQ returns the q (file format) representation of the query.
func (q Query) AsQ() string {
	var str string
	str += "/*\n"
	str += "description: " + escapeForXML(q.description) + "\n"
	str += "category: " + q.category.String()
	str += "*/\n"
	str += q.query + "\n"
	return str
}

// AsXML returns the xml (file format) representation of the query.
func (q Query) AsXML() string {
	comment := "description: " + escapeForXML(q.description) + "\n"
	comment += "category: " + q.category.String()
	str := "<query>\n"
	str += "    <formula>" + escapeForXML(q.query) + "</formula>\n"
	str += "    <comment>" + comment + "</comment>\n"
	str += "</query>\n"
	return str
}
