package builder

import (
	"fmt"
	"go/types"
	"path/filepath"
	"sort"
)

func (b *builder) Import(importPath string) (*types.Package, error) {
	return b.ImportFrom(importPath, ".", 0)
}

func (b *builder) ImportFrom(importPath, absPath string, mode types.ImportMode) (*types.Package, error) {
	if importPath == "C" {
		return nil, fmt.Errorf("cgo not supported")
	}
	if abs, err := filepath.Abs(absPath); err == nil {
		absPath = abs
	}
	buildPackage, err := b.config.BuildContext.Import(importPath, absPath, buildImportMode)
	if err != nil {
		return nil, err
	}

	pkg, ok := b.pkgs[buildPackage.Dir]
	if ok {
		typesPackage := pkg.typesPackage
		if typesPackage == nil {
			return nil, fmt.Errorf("processed imports in wrong order")
		}
		return pkg.typesPackage, nil
	}

	return b.typesSrcImporter.(types.ImporterFrom).ImportFrom(importPath, absPath, mode)
}

func (b *builder) packageProcessingOrder() ([]string, error) {
	nameOrderedPaths := make([]string, 0, len(b.pkgs))
	for absPath := range b.pkgs {
		nameOrderedPaths = append(nameOrderedPaths, absPath)
	}
	sort.Strings(nameOrderedPaths)
	topoOrderedPaths := make([]string, 0, len(b.pkgs))
	addedPaths := make(map[string]bool)

	for len(topoOrderedPaths) < len(nameOrderedPaths) {
		addedCount := 0
	pkgLoop:
		for _, absPath := range nameOrderedPaths {
			if addedPaths[absPath] {
				continue
			}
			for _, absImportPath := range b.pkgs[absPath].absImportPaths {
				if !addedPaths[absImportPath] {
					continue pkgLoop
				}
			}

			topoOrderedPaths = append(topoOrderedPaths, absPath)
			addedPaths[absPath] = true
			addedCount++
		}
		if addedCount == 0 {
			return nil, fmt.Errorf("encountered import loop")
		}
	}

	return topoOrderedPaths, nil
}
