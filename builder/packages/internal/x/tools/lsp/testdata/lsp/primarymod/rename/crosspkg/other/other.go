package other

import "github.com/arneph/toph/builder/packages/internal/x/tools/lsp/rename/crosspkg"

func Other() {
	crosspkg.Bar
	crosspkg.Foo() //@rename("Foo", "Flamingo")
}
