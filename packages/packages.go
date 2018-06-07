// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package packages

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var (
	cwd       string
	gopathEnv = os.Getenv("GOPATH")
	gopath    []string
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
	path string
}

// Load looks for directories matching the passed in package patterns
// and returns Dir values for each directory that can be successfully
// imported and is found to match one of the patterns.
func Load(pkgPatterns ...string) (dirs []Dir) {
	pkgPatterns = parsePatterns(pkgPatterns...)
	for _, pkgPattern := range pkgPatterns {
		pkg, err := build.Import(pkgPattern, cwd, build.AllowBinary)
		if err != nil {
			panic(err)
		}
		dirs = append(dirs, Dir{path: pkg.Dir})
	}
	return
}

// Path returns the file path to d.
func (d Dir) Path() string {
	return d.path
}

// Packages returns the AST for all packages in d.
func (d Dir) Packages() map[string]*ast.Package {
	packages, err := parser.ParseDir(token.NewFileSet(), d.Path(), nil, 0)
	if err != nil {
		panic(err)
	}
	return packages
}

// Import imports path from srcDir, then loads the ast for that package.
// It ensures that the returned ast is for the package that would be
// imported by an import clause.
func (d Dir) Import(path, srcDir string) (string, *ast.Package, error) {
	pkgInfo, err := build.Import(path, srcDir, 0)
	if err != nil {
		return "", nil, err
	}
	newDir := Load(path)[0]
	if pkg, ok := newDir.Packages()[pkgInfo.Name]; ok {
		return pkgInfo.Name, pkg, nil
	}
	return "", nil, fmt.Errorf("Could not find package %s", path)
}

func parsePatterns(pkgPatterns ...string) (packages []string) {
	for _, pkgPattern := range pkgPatterns {
		if !strings.HasSuffix(pkgPattern, "...") {
			packages = append(packages, pkgPattern)
			continue
		}
		parent := strings.TrimSuffix(pkgPattern, "...")
		parentPkg, err := build.Import(parent, cwd, build.AllowBinary)
		if err != nil {
			panic(err)
		}
		filepath.Walk(parentPkg.Dir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			path = strings.Replace(path, parentPkg.Dir, parent, 1)
			if _, err := build.Import(path, cwd, build.AllowBinary); err != nil {
				// This directory doesn't appear to be a go package
				return nil
			}
			packages = append(packages, path)
			return nil
		})
	}
	return
}
