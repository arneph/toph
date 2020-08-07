package nodisk

import (
	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/foo"
)

func _() {
	foo.Foo() //@complete("F", IntFoo, StructFoo, Foo)
}
