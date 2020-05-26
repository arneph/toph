package uppaal

import "strings"

// Query holds a query and its comment. Both can be exported to a query file.
type Query struct {
	// The '$' serves as a placeholder.
	query   string
	comment string
}

// MakeQuery returns a query with the given query string and comment.
func MakeQuery(query, comment string) Query {
	return Query{
		query:   query,
		comment: comment,
	}
}

// Substitute returns a query with all placeholders in the query string
// replaced by the given replacement.
func (q Query) Substitute(replacement string) Query {
	return Query{
		query:   strings.ReplaceAll(q.query, "$", replacement),
		comment: q.comment,
	}
}

// AsQ returns the q (file format) representation of the query.
func (q Query) AsQ() string {
	var str string
	if q.comment != "" {
		str += "/*\n"
		str += q.comment + "\n"
		str += "*/\n"
	}
	str += q.query + "\n"
	return str
}

// AsXML returns the xml (file format) representation of the query.
func (q Query) AsXML() string {
	str := "<query>\n"
	str += "    <formula>" + escapeForXML(q.query) + "</formula>\n"
	str += "    <comment>" + escapeForXML(q.comment) + "</comment>\n"
	str += "</query>\n"
	return str
}
