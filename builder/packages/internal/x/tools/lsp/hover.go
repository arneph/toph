// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"

	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/mod"
	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/protocol"
	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/source"
)

func (s *Server) hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	snapshot, fh, ok, err := s.beginFileRequest(ctx, params.TextDocument.URI, source.UnknownKind)
	if !ok {
		return nil, err
	}
	switch fh.Kind() {
	case source.Mod:
		return mod.Hover(ctx, snapshot, fh, params.Position)
	case source.Go:
		return source.Hover(ctx, snapshot, fh, params.Position)
	}
	return nil, nil
}
