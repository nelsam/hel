package types

import (
	"go/ast"
	"regexp"
	"strings"
)

type GoDir interface {
	Path() string
	Packages() map[string]*ast.Package
}

type TypeDir struct {
	dir     string
	testPkg string
	types   []*ast.TypeSpec
}

func (t TypeDir) Dir() string {
	return t.dir
}

func (t TypeDir) Len() int {
	return len(t.types)
}

func (t TypeDir) TestPackage() string {
	return t.testPkg
}

func (t TypeDir) ExportedTypes() []*ast.TypeSpec {
	return t.types
}

func (t TypeDir) Filter(matchers ...*regexp.Regexp) TypeDir {
	oldTypes := t.ExportedTypes()
	t.types = make([]*ast.TypeSpec, 0, t.Len())
	for _, typ := range oldTypes {
		for _, matcher := range matchers {
			if !matcher.MatchString(typ.Name.String()) {
				continue
			}
			t.types = append(t.types, typ)
			break
		}
	}
	return t
}

type TypeDirs []TypeDir

func (t TypeDirs) Filter(patterns ...string) (dirs TypeDirs) {
	if len(patterns) == 0 {
		return t
	}
	matchers := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		matchers = append(matchers, regexp.MustCompile("^"+pattern+"$"))
	}
	for _, typeDir := range t {
		typeDir = typeDir.Filter(matchers...)
		if typeDir.Len() > 0 {
			dirs = append(dirs, typeDir)
		}
	}
	return dirs
}

func Load(dirs ...GoDir) TypeDirs {
	types := make(TypeDirs, 0, len(dirs))
	for _, dir := range dirs {
		t := TypeDir{
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
