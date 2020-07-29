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
			var loopFinder func(curr string, prior []string) []string
			loopFinder = func(curr string, prior []string) []string {
				for i, p := range prior {
					if p == curr {
						return prior[i:]
					}
				}
				for _, absImportPath := range b.pkgs[curr].absImportPaths {
					if addedPaths[absImportPath] {
						continue
					}
					loop := loopFinder(absImportPath, append(prior, curr))
					if loop != nil {
						return loop
					}
				}
				return nil
			}
			for _, absPath := range nameOrderedPaths {
				loop := loopFinder(absPath, nil)
				loopStr := strings.Join(loop, "\n")
				if loop != nil {
					return nil, fmt.Errorf("encountered import loop:\n%s", loopStr)
				}
			}
			return nil, fmt.Errorf("encountered import loop")
		}
	}

	return topoOrderedPaths, nil
}
