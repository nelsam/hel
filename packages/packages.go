// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package packages

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	cwd       string
	gopathEnv = os.Getenv("GOPATH")
	gopath    = strings.Split(gopathEnv, string(os.PathListSeparator))
)

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

// Dir represents a directory containing go files.
type Dir struct {
	pkg    *packages.Package
	fsPath string
}

// Load looks for directories matching the passed in package patterns
// and returns Dir values for each directory that can be successfully
// imported and is found to match one of the patterns.
func Load(pkgPatterns ...string) []Dir {
	return load(cwd, pkgPatterns...)
}

func load(fromDir string, pkgPatterns ...string) (dirs []Dir) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedSyntax,
	}, pkgPatterns...)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		fsPath := ""
		if len(pkg.GoFiles) > 0 {
			fsPath = filepath.Dir(pkg.GoFiles[0])
		}
		dirs = append(dirs, Dir{pkg: pkg, fsPath: fsPath})
	}
	return dirs
}

// Path returns the file path to d.
func (d Dir) Path() string {
	return d.fsPath
}

// Package returns the *packages.Package for d
func (d Dir) Package() *packages.Package {
	return d.pkg
}

// Import imports path from srcDir, then loads the ast for that package.
// It ensures that the returned ast is for the package that would be
// imported by an import clause.
func (d Dir) Import(path string) (*packages.Package, error) {
	p, ok := nestedImport(d.pkg, path)
	if !ok {
		return nil, fmt.Errorf("Could not find import %s in package %s", path, d.Path())
	}
	return p, nil
}

func nestedImport(pkg *packages.Package, path string) (*packages.Package, bool) {
	if p, ok := pkg.Imports[path]; ok {
		return p, true
	}
	for _, p := range pkg.Imports {
		if subp, ok := nestedImport(p, path); ok {
			return subp, true
		}
	}
	return nil, false
}

func parsePatterns(fromDir string, pkgPatterns ...string) (packages []string) {
	for _, pkgPattern := range pkgPatterns {
		if !strings.HasSuffix(pkgPattern, "...") {
			packages = append(packages, pkgPattern)
			continue
		}
		parent := strings.TrimSuffix(pkgPattern, "...")
		parentPkg, err := build.Import(parent, fromDir, build.AllowBinary)
		if err != nil {
			panic(err)
		}
		filepath.Walk(parentPkg.Dir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			path = strings.Replace(path, parentPkg.Dir, parent, 1)
			if _, err := build.Import(path, fromDir, build.AllowBinary); err != nil {
				// This directory doesn't appear to be a go package
				return nil
			}
			packages = append(packages, path)
			return nil
		})
	}
	return
}
