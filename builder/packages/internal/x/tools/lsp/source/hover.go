// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/token"
	"go/types"
	"strings"

	"github.com/arneph/toph/builder/packages/internal/x/tools/event"
	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/protocol"
	errors "golang.org/x/xerrors"
)

type HoverInformation struct {
	// Signature is the symbol's signature.
	Signature string `json:"signature"`

	// SingleLine is a single line describing the symbol.
	// This is recommended only for use in clients that show a single line for hover.
	SingleLine string `json:"singleLine"`

	// Synopsis is a single sentence synopsis of the symbol's documentation.
	Synopsis string `json:"synopsis"`

	// FullDocumentation is the symbol's full documentation.
	FullDocumentation string `json:"fullDocumentation"`

	// ImportPath is the import path for the package containing the given symbol.
	ImportPath string

	// Link is the pkg.go.dev anchor for the given symbol.
	// For example, "go/ast#Node".
	Link string `json:"link"`

	// SymbolName is the types.Object.Name for the given symbol.
	SymbolName string

	source  interface{}
	comment *ast.CommentGroup
}

func Hover(ctx context.Context, snapshot Snapshot, fh FileHandle, position protocol.Position) (*protocol.Hover, error) {
	ident, err := Identifier(ctx, snapshot, fh, position)
	if err != nil {
		return nil, nil
	}
	h, err := HoverIdentifier(ctx, ident)
	if err != nil {
		return nil, err
	}
	rng, err := ident.Range()
	if err != nil {
		return nil, err
	}
	// See golang/go#36998: don't link to modules matching GOPRIVATE.
	if snapshot.View().IsGoPrivatePath(h.ImportPath) {
		h.Link = ""
	}
	hover, err := FormatHover(h, snapshot.View().Options())
	if err != nil {
		return nil, err
	}
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  snapshot.View().Options().PreferredContentFormat,
			Value: hover,
		},
		Range: rng,
	}, nil
}

func HoverIdentifier(ctx context.Context, i *IdentifierInfo) (*HoverInformation, error) {
	ctx, done := event.Start(ctx, "source.Hover")
	defer done()

	fset := i.Snapshot.View().Session().Cache().FileSet()
	h, err := hover(ctx, fset, i.pkg, i.Declaration)
	if err != nil {
		return nil, err
	}
	// Determine the symbol's signature.
	switch x := h.source.(type) {
	case ast.Node:
		var b strings.Builder
		if err := format.Node(&b, fset, x); err != nil {
			return nil, err
		}
		h.Signature = b.String()
	case types.Object:
		// If the variable is implicitly declared in a type switch, we need to
		// manually generate its object string.
		if typ := i.Declaration.typeSwitchImplicit; typ != nil {
			if v, ok := x.(*types.Var); ok {
				h.Signature = fmt.Sprintf("var %s %s", v.Name(), types.TypeString(typ, i.qf))
				break
			}
		}
		h.Signature = objectString(x, i.qf)
	}
	if obj := i.Declaration.obj; obj != nil {
		h.SingleLine = objectString(obj, i.qf)
	}
	h.ImportPath, h.Link, h.SymbolName = pathLinkAndSymbolName(i)

	return h, nil
}

func pathLinkAndSymbolName(i *IdentifierInfo) (string, string, string) {
	obj := i.Declaration.obj
	if obj == nil {
		return "", "", ""
	}
	switch obj := obj.(type) {
	case *types.PkgName:
		path := obj.Imported().Path()
		link := path
		if mod, version, ok := moduleAtVersion(path, i); ok {
			link = strings.Replace(path, mod, mod+"@"+version, 1)
		}
		return path, link, obj.Name()
	case *types.Builtin:
		return "builtin", fmt.Sprintf("builtin#%s", obj.Name()), obj.Name()
	}
	// Check if the identifier is test-only (and is therefore not part of a
	// package's API). This is true if the request originated in a test package,
	// and if the declaration is also found in the same test package.
	if i.pkg != nil && obj.Pkg() != nil && i.pkg.ForTest() != "" {
		if _, err := i.pkg.File(i.Declaration.MappedRange[0].URI()); err == nil {
			return "", "", ""
		}
	}
	// Don't return links for other unexported types.
	if !obj.Exported() {
		return "", "", ""
	}
	var rTypeName string
	switch obj := obj.(type) {
	case *types.Var:
		// If the object is a field, and we have an associated selector
		// composite literal, or struct, we can determine the link.
		if obj.IsField() {
			if named, ok := i.enclosing.(*types.Named); ok {
				rTypeName = named.Obj().Name()
			}
		}
	case *types.Func:
		typ, ok := obj.Type().(*types.Signature)
		if !ok {
			return "", "", ""
		}
		if r := typ.Recv(); r != nil {
			switch rtyp := deref(r.Type()).(type) {
			case *types.Struct:
				rTypeName = r.Name()
			case *types.Named:
				// If we have an unexported type, see if the enclosing type is
				// exported (we may have an interface or struct we can link
				// to). If not, don't show any link.
				if !rtyp.Obj().Exported() {
					if named, ok := i.enclosing.(*types.Named); ok && named.Obj().Exported() {
						rTypeName = named.Obj().Name()
					} else {
						return "", "", ""
					}
				} else {
					rTypeName = rtyp.Obj().Name()
				}
			}
		}
	}
	path := obj.Pkg().Path()
	link := path
	if mod, version, ok := moduleAtVersion(path, i); ok {
		link = strings.Replace(path, mod, mod+"@"+version, 1)
	}
	if rTypeName != "" {
		link = fmt.Sprintf("%s#%s.%s", link, rTypeName, obj.Name())
		symbol := fmt.Sprintf("(%s.%s).%s", obj.Pkg().Name(), rTypeName, obj.Name())
		return path, link, symbol
	}
	// For most cases, the link is "package/path#symbol".
	link = fmt.Sprintf("%s#%s", link, obj.Name())
	symbolName := fmt.Sprintf("%s.%s", obj.Pkg().Name(), obj.Name())
	return path, link, symbolName
}

func moduleAtVersion(path string, i *IdentifierInfo) (string, string, bool) {
	if strings.ToLower(i.Snapshot.View().Options().LinkTarget) != "pkg.go.dev" {
		return "", "", false
	}
	impPkg, err := i.pkg.GetImport(path)
	if err != nil {
		return "", "", false
	}
	if impPkg.Module() == nil {
		return "", "", false
	}
	version, modpath := impPkg.Module().Version, impPkg.Module().Path
	if modpath == "" || version == "" {
		return "", "", false
	}
	return modpath, version, true
}

// objectString is a wrapper around the types.ObjectString function.
// It handles adding more information to the object string.
func objectString(obj types.Object, qf types.Qualifier) string {
	str := types.ObjectString(obj, qf)
	switch obj := obj.(type) {
	case *types.Const:
		str = fmt.Sprintf("%s = %s", str, obj.Val())
	}
	return str
}

func hover(ctx context.Context, fset *token.FileSet, pkg Package, d Declaration) (*HoverInformation, error) {
	_, done := event.Start(ctx, "source.hover")
	defer done()

	return hoverInfo(pkg, d.obj, d.node)
}

func hoverInfo(pkg Package, obj types.Object, node ast.Node) (*HoverInformation, error) {
	var info *HoverInformation

	switch node := node.(type) {
	case *ast.Ident:
		// The package declaration.
		for _, f := range pkg.GetSyntax() {
			if f.Name == node {
				info = &HoverInformation{comment: f.Doc}
			}
		}
	case *ast.ImportSpec:
		// Try to find the package documentation for an imported package.
		if pkgName, ok := obj.(*types.PkgName); ok {
			imp, err := pkg.GetImport(pkgName.Imported().Path())
			if err != nil {
				return nil, err
			}
			// Assume that only one file will contain package documentation,
			// so pick the first file that has a doc comment.
			for _, file := range imp.GetSyntax() {
				if file.Doc != nil {
					info = &HoverInformation{source: obj, comment: file.Doc}
					break
				}
			}
		}
		info = &HoverInformation{source: node}
	case *ast.GenDecl:
		switch obj := obj.(type) {
		case *types.TypeName, *types.Var, *types.Const, *types.Func:
			var err error
			info, err = formatGenDecl(node, obj, obj.Type())
			if err != nil {
				return nil, err
			}
		}
	case *ast.TypeSpec:
		if obj.Parent() == types.Universe {
			if obj.Name() == "error" {
				info = &HoverInformation{source: node}
			} else {
				info = &HoverInformation{source: node.Name} // comments not needed for builtins
			}
		}
	case *ast.FuncDecl:
		switch obj.(type) {
		case *types.Func:
			info = &HoverInformation{source: obj, comment: node.Doc}
		case *types.Builtin:
			info = &HoverInformation{source: node.Type, comment: node.Doc}
		}
	}

	if info == nil {
		info = &HoverInformation{source: obj}
	}

	if info.comment != nil {
		info.FullDocumentation = info.comment.Text()
		info.Synopsis = doc.Synopsis(info.FullDocumentation)
	}

	return info, nil
}

func formatGenDecl(node *ast.GenDecl, obj types.Object, typ types.Type) (*HoverInformation, error) {
	if _, ok := typ.(*types.Named); ok {
		switch typ.Underlying().(type) {
		case *types.Interface, *types.Struct:
			return formatGenDecl(node, obj, typ.Underlying())
		}
	}
	var spec ast.Spec
	for _, s := range node.Specs {
		if s.Pos() <= obj.Pos() && obj.Pos() <= s.End() {
			spec = s
			break
		}
	}
	if spec == nil {
		return nil, errors.Errorf("no spec for node %v at position %v", node, obj.Pos())
	}

	// If we have a field or method.
	switch obj.(type) {
	case *types.Var, *types.Const, *types.Func:
		return formatVar(spec, obj, node), nil
	}
	// Handle types.
	switch spec := spec.(type) {
	case *ast.TypeSpec:
		if len(node.Specs) > 1 {
			// If multiple types are declared in the same block.
			return &HoverInformation{source: spec.Type, comment: spec.Doc}, nil
		} else {
			return &HoverInformation{source: spec, comment: node.Doc}, nil
		}
	case *ast.ValueSpec:
		return &HoverInformation{source: spec, comment: spec.Doc}, nil
	case *ast.ImportSpec:
		return &HoverInformation{source: spec, comment: spec.Doc}, nil
	}
	return nil, errors.Errorf("unable to format spec %v (%T)", spec, spec)
}

func formatVar(node ast.Spec, obj types.Object, decl *ast.GenDecl) *HoverInformation {
	var fieldList *ast.FieldList
	switch spec := node.(type) {
	case *ast.TypeSpec:
		switch t := spec.Type.(type) {
		case *ast.StructType:
			fieldList = t.Fields
		case *ast.InterfaceType:
			fieldList = t.Methods
		}
	case *ast.ValueSpec:
		comment := spec.Doc
		if comment == nil {
			comment = decl.Doc
		}
		if comment == nil {
			comment = spec.Comment
		}
		return &HoverInformation{source: obj, comment: comment}
	}
	// If we have a struct or interface declaration,
	// we need to match the object to the corresponding field or method.
	if fieldList != nil {
		for i := 0; i < len(fieldList.List); i++ {
			field := fieldList.List[i]
			if field.Pos() <= obj.Pos() && obj.Pos() <= field.End() {
				if field.Doc.Text() != "" {
					return &HoverInformation{source: obj, comment: field.Doc}
				}
				return &HoverInformation{source: obj, comment: field.Comment}
			}
		}
	}
	return &HoverInformation{source: obj, comment: decl.Doc}
}

func FormatHover(h *HoverInformation, options Options) (string, error) {
	signature := h.Signature
	if signature != "" && options.PreferredContentFormat == protocol.Markdown {
		signature = fmt.Sprintf("```go\n%s\n```", signature)
	}

	switch options.HoverKind {
	case SingleLine:
		return h.SingleLine, nil
	case NoDocumentation:
		return signature, nil
	case Structured:
		b, err := json.Marshal(h)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	link := formatLink(h, options)
	switch options.HoverKind {
	case SynopsisDocumentation:
		doc := formatDoc(h.Synopsis, options)
		return formatHover(options, signature, link, doc), nil
	case FullDocumentation:
		doc := formatDoc(h.FullDocumentation, options)
		return formatHover(options, signature, link, doc), nil
	}
	return "", errors.Errorf("no hover for %v", h.source)
}

func formatLink(h *HoverInformation, options Options) string {
	if !options.LinksInHover || options.LinkTarget == "" || h.Link == "" {
		return ""
	}
	plainLink := fmt.Sprintf("https://%s/%s", options.LinkTarget, h.Link)
	switch options.PreferredContentFormat {
	case protocol.Markdown:
		return fmt.Sprintf("[`%s` on %s](%s)", h.SymbolName, options.LinkTarget, plainLink)
	case protocol.PlainText:
		return ""
	default:
		return plainLink
	}
}

func formatDoc(doc string, options Options) string {
	if options.PreferredContentFormat == protocol.Markdown {
		return CommentToMarkdown(doc)
	}
	return doc
}

func formatHover(options Options, x ...string) string {
	var b strings.Builder
	for i, el := range x {
		if el != "" {
			b.WriteString(el)

			// Don't write out final newline.
			if i == len(x) {
				continue
			}
			// If any elements of the remainder of the list are non-empty,
			// write a newline.
			if anyNonEmpty(x[i+1:]) {
				if options.PreferredContentFormat == protocol.Markdown {
					b.WriteString("\n\n")
				} else {
					b.WriteRune('\n')
				}
			}
		}
	}
	return b.String()
}

func anyNonEmpty(x []string) bool {
	for _, el := range x {
		if el != "" {
			return true
		}
	}
	return false
}
