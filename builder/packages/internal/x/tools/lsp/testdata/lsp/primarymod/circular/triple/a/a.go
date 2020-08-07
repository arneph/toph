package a

import (
	_ "github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/triple/b" //@diag("_ \"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/triple/b\"", "compiler", "import cycle not allowed", "error")
)
