package uppaal

import (
	"encoding/xml"
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
	// NoFunctionCallsWithNilVariable verifies the system is never attempting to call a nil (-1) function variable.
	NoFunctionCallsWithNilVariable
	// NoGoroutineExitWithPanic verifies the system never exits a panicking goroutine.
	NoGoroutineExitWithPanic
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
	case NoGoroutineExitWithPanic:
		return "no goroutine exit with panic"
	case ReachabilityRequirements:
		return "reachability requirements"
	default:
		panic(fmt.Errorf("unexpected query category: %d", c))
	}
}

// Query holds a query and its comment. Both can be exported to a query file.
type Query struct {
	// The '$' serves as a placeholder.
	query          string
	description    string
	sourceLocation string
	category       QueryCategory
}

// NewQuery returns a query with the given query string and comment.
func NewQuery(query, description, sourceLocation string, category QueryCategory) *Query {
	q := new(Query)
	q.query = query
	q.description = description
	q.sourceLocation = sourceLocation
	q.category = category

	return q
}

// Query returns the actual query expression.
func (q *Query) Query() string {
	return q.query
}

// Description returns a more detailed description of the query.
func (q *Query) Description() string {
	return q.description
}

// SourceLocation returns the source location associated with the query, if
// any.
func (q *Query) SourceLocation() string {
	return q.sourceLocation
}

// Category returns the QueryCategory of the query.
func (q *Query) Category() QueryCategory {
	return q.category
}

// Substitute returns a query with all placeholders in the query string
// replaced by the given replacement.
func (q *Query) Substitute(replacement string) *Query {
	s := new(Query)
	s.query = strings.ReplaceAll(q.query, "$", replacement)
	s.description = q.description
	s.sourceLocation = q.sourceLocation
	s.category = q.category

	return s
}

// AsQ returns the q (file format) representation of the query.
func (q Query) AsQ(number int) string {
	var str string
	str += "/*\n"
	str += "description: " + q.description + "\n"
	if q.sourceLocation != "" {
		str += "location: " + q.sourceLocation + "\n"
	}
	str += "category: " + q.category.String() + "\n"
	str += fmt.Sprintf("number: %d", number)
	str += "*/\n"
	str += q.query + "\n"
	return str
}

func (q Query) asXML(b *strings.Builder, number int, indent string) {
	b.WriteString(indent + "<query>\n")
	b.WriteString(indent + "    <formula>")
	xml.EscapeText(b, []byte(q.query))
	b.WriteString("</formula>\n")
	b.WriteString(indent + "    <comment>")
	b.WriteString("description: ")
	xml.EscapeText(b, []byte(q.description))
	b.WriteString("\n")
	if q.sourceLocation != "" {
		b.WriteString("location: ")
		xml.EscapeText(b, []byte(q.sourceLocation))
		b.WriteString("\n")
	}
	b.WriteString("category: ")
	xml.EscapeText(b, []byte(q.category.String()))
	b.WriteString("\n")
	fmt.Fprintf(b, "number: %d", number)
	b.WriteString("</comment>\n")
	b.WriteString(indent + "</query>")
}
