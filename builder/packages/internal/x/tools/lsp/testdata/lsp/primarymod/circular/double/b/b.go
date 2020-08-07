package b

import (
	_ "github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/double/one" //@diag("_ \"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/double/one\"", "compiler", "import cycle not allowed", "error"),diag("\"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/double/one\"", "compiler", "could not import github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/double/one (no package for import github.com/arneph/toph/builder/packages/internal/x/tools/lsp/circular/double/one)", "error")
)
