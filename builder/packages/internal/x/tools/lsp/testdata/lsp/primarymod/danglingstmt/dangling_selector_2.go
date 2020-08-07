package danglingstmt

import "github.com/arneph/toph/builder/packages/internal/x/tools/lsp/foo"

func _() {
	foo. //@rank(" //", Foo)
	var _ = []string{foo.} //@rank("}", Foo)
}
