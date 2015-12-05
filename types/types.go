package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/nelsam/hel/packages"
)

type Types struct {
	dir     string
	testPkg string
	types   []*ast.TypeSpec
}

func (t Types) Dir() string {
	return t.dir
}

func (t Types) Len() int {
	return len(t.types)
}

func (t Types) TestPackage() string {
	return t.testPkg
}

func (t Types) ExportedTypes() []*ast.TypeSpec {
	return t.types
}

func Load(packages ...packages.Package) []Types {
	types := make([]Types, 0, len(packages))
	for _, pkg := range packages {
		t := Types{
			dir: pkg.Path,
		}
		pkgs, err := parser.ParseDir(token.NewFileSet(), pkg.Path, nil, 0)
		if err != nil {
			panic(err)
		}
		for name, pkg := range pkgs {
			if strings.HasSuffix(name, "_test") {
				t.testPkg = name
				continue
			}
			newTypes, testsFound := loadPkgTypeSpecs(pkg)
			t.types = append(t.types, newTypes...)
			if testsFound && t.testPkg == "" {
				t.testPkg = name
			}
		}
		types = append(types, t)
	}
	return types
}

func loadPkgTypeSpecs(pkg *ast.Package) (specs []*ast.TypeSpec, hasTests bool) {
	for name, f := range pkg.Files {
		if strings.HasSuffix(name, "_test.go") {
			hasTests = true
			continue
		}
		specs = append(specs, loadFileTypeSpecs(f)...)
	}
	return specs, hasTests
}

func loadFileTypeSpecs(f *ast.File) (specs []*ast.TypeSpec) {
	for _, obj := range f.Scope.Objects {
		spec, ok := obj.Decl.(*ast.TypeSpec)
		if !ok {
			continue
		}
		if _, ok = spec.Type.(*ast.InterfaceType); !ok {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}
