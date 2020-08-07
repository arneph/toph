package other

import (
	references "github.com/arneph/toph/builder/packages/internal/x/tools/lsp/references"
)

func GetXes() []references.X {
	return []references.X{
		{
			Y: 1, //@mark(GetXesY, "Y"),refs("Y", typeXY, GetXesY, anotherXY)
		},
	}
}

func _() {
	references.Q = "hello" //@mark(assignExpQ, "Q")
	bob := func(_ string) {}
	bob(references.Q) //@mark(bobExpQ, "Q")
}
