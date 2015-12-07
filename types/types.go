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

type Dir struct {
	dir     string
	pkg     string
	testPkg string
	types   []*ast.TypeSpec
}

func (d Dir) Dir() string {
	return d.dir
}

func (d Dir) Len() int {
	return len(d.types)
}

func (d Dir) Package() string {
	return d.pkg
}

func (d Dir) TestPackage() string {
	return d.testPkg
}

func (d Dir) ExportedTypes() []*ast.TypeSpec {
	return d.types
}

func (d Dir) Filter(matchers ...*regexp.Regexp) Dir {
	oldTypes := d.ExportedTypes()
	d.types = make([]*ast.TypeSpec, 0, d.Len())
	for _, typ := range oldTypes {
		for _, matcher := range matchers {
			if !matcher.MatchString(typ.Name.String()) {
				continue
			}
			d.types = append(d.types, typ)
			break
		}
	}
	return d
}

type Dirs []Dir

func (d Dirs) Filter(patterns ...string) (dirs Dirs) {
	if len(patterns) == 0 {
		return d
	}
	matchers := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		matchers = append(matchers, regexp.MustCompile("^"+pattern+"$"))
	}
	for _, dir := range d {
		dir = dir.Filter(matchers...)
		if dir.Len() > 0 {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func Load(goDirs ...GoDir) Dirs {
	typeDirs := make(Dirs, 0, len(goDirs))
	for _, dir := range goDirs {
		d := Dir{
			dir: dir.Path(),
		}
		for name, pkg := range dir.Packages() {
			if d.testPkg == "" {
				// This will get overridden if we later find pre-existing test
				// files in one of the packages.  As such, don't worry about
				// test packages getting an extra "_test", since test packages
				// will be made up of only test files.
				d.testPkg = name + "_test"
			}
			newTypes, testsFound := loadPkgTypeSpecs(pkg)
			if testsFound {
				d.testPkg = name
			}
			if d.pkg == "" || !testsFound {
				d.pkg = name
			}
			d.types = append(d.types, newTypes...)
		}
		typeDirs = append(typeDirs, d)
	}
	return typeDirs
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
