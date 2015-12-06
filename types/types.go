package types

import (
	"go/ast"
	"strings"
)

type GoDir interface {
	Path() string
	Packages() map[string]*ast.Package
}

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

func Load(dirs ...GoDir) []Types {
	types := make([]Types, 0, len(dirs))
	for _, dir := range dirs {
		t := Types{
			dir: dir.Path(),
		}
		for name, pkg := range dir.Packages() {
			if t.testPkg == "" {
				// This will get overridden if we later find pre-existing test
				// files in one of the packages.  As such, don't worry about
				// test packages getting an extra "_test", since test packages
				// will be made up of only test files.
				t.testPkg = name + "_test"
			}
			newTypes, testsFound := loadPkgTypeSpecs(pkg)
			if testsFound {
				t.testPkg = name
			}
			t.types = append(t.types, newTypes...)
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
