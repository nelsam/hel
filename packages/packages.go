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

type Dir struct {
	path string
}

func Load(pkgPatterns ...string) []Dir {
	return load(cwd, pkgPatterns...)
}

func load(fromDir string, pkgPatterns ...string) (dirs []Dir) {
	pkgPatterns = parsePatterns(fromDir, pkgPatterns...)
	for _, pkgPattern := range pkgPatterns {
		pkg, err := build.Import(pkgPattern, fromDir, build.AllowBinary)
		if err != nil {
			panic(err)
		}
		dirs = append(dirs, Dir{path: pkg.Dir})
	}
	return dirs
}

func (d Dir) Path() string {
	return d.path
}

func (d Dir) Packages() map[string]*ast.Package {
	packages, err := parser.ParseDir(token.NewFileSet(), d.Path(), nil, 0)
	if err != nil {
		panic(err)
	}
	return packages
}

func (d Dir) Import(path string) (string, *ast.Package, error) {
	pkgInfo, err := build.Import(path, d.Path(), 0)
	if err != nil {
		return "", nil, err
	}
	newDir := load(d.Path(), path)[0]
	if pkg, ok := newDir.Packages()[pkgInfo.Name]; ok {
		return pkgInfo.Name, pkg, nil
	}
	return "", nil, fmt.Errorf("Could not find package %s", path)
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
