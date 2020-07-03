package ir

import (
	"fmt"
	"go/token"
	"strings"
)

// MakeStructStmt represents a new(T) call, a struct literal, or a struct
// variable initialization if the struct is stored by value.
type MakeStructStmt struct {
	structVal LValue

	Node
}

// NewMakeStructStmt creates a new MakeStructStmt for the given structure value.
func NewMakeStructStmt(structVal LValue, pos, end token.Pos) *MakeStructStmt {
	s := new(MakeStructStmt)
	s.structVal = structVal
	s.pos = pos
	s.end = end

	return s
}

// StructVal returns the lvalue holding the newly allocated structure.
func (s *MakeStructStmt) StructVal() LValue {
	return s.structVal
}

// StructType returns the type of the newly allocated structure.
func (s *MakeStructStmt) StructType() *StructType {
	return s.structVal.Type().(*StructType)
}

func (s *MakeStructStmt) tree(b *strings.Builder, indent int) {
	writeIndent(b, indent)
	fmt.Fprintf(b, "%s <- make(%s)", s.structVal.Handle(), s.structVal.Type())
}
