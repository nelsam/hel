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

func (d Dir) Import(path, pkg string) (*ast.Package, error) {
	newDir := Load(path)[0]
	if pkg, ok := newDir.Packages()[pkg]; ok {
		return pkg, nil
	}
	return nil, fmt.Errorf("Could not find package %s", pkg)
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
