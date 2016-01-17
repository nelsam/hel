package types

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
)

// A GoDir is a type that represents a directory of Go files.
type GoDir interface {
	Path() string
	Packages() map[string]*ast.Package
	Import(pkg string) (*ast.Package, error)
}

// A Dir is a type that represents a directory containing Go
// packages.
type Dir struct {
	dir     string
	pkg     string
	testPkg string
	types   []*ast.TypeSpec
}

// Dir returns the directory path that d represents.
func (d Dir) Dir() string {
	return d.dir
}

// Len returns the number of types that will be returned by
// d.ExportedTypes().
func (d Dir) Len() int {
	return len(d.types)
}

// Package returns the name of d's importable package.
func (d Dir) Package() string {
	return d.pkg
}

// TestPackage returns the name of d's test package.  It may be the
// same as d.Package().
func (d Dir) TestPackage() string {
	return d.testPkg
}

// ExportedTypes returns all *ast.TypeSpecs found by d.  Interface
// types with anonymous interface types will be flattened, for ease of
// mocking by other logic.
func (d Dir) ExportedTypes() []*ast.TypeSpec {
	return d.types
}

// Filter filters d's types, removing all types that don't match any
// of the passed in matchers.
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

// Dirs is a slice of Dir values, to provide sugar for running some
// methods against multiple Dir values.
type Dirs []Dir

// Load loads a Dirs value for goDirs.
func Load(goDirs ...GoDir) Dirs {
	typeDirs := make(Dirs, 0, len(goDirs))
	for _, dir := range goDirs {
		d := Dir{
			dir: dir.Path(),
		}
		for name, pkg := range dir.Packages() {
			if d.testPkg == "" {
				// Default for packages that don't have tests yet.
				d.testPkg = name + "_test"
			}
			newTypes, testsFound := loadPkgTypeSpecs(pkg)
			if testsFound {
				// This package already has test files, so this will
				// always be the test package.
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

// Filter calls Dir.Filter for each Dir in d.
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
	var inters []*ast.InterfaceType
	for _, obj := range f.Scope.Objects {
		spec, ok := obj.Decl.(*ast.TypeSpec)
		if !ok {
			continue
		}
		inter, ok := spec.Type.(*ast.InterfaceType)
		if !ok {
			continue
		}
		inters = append(inters, inter)
		specs = append(specs, spec)
	}
	flattenAnon(inters, specs)
	return specs
}

func flattenAnon(inters []*ast.InterfaceType, withSpecs []*ast.TypeSpec) {
	for _, inter := range inters {
		flatten(inter, withSpecs)
	}
}

func flatten(inter *ast.InterfaceType, withSpecs []*ast.TypeSpec) {
	if inter.Methods == nil {
		return
	}
	methods := make([]*ast.Field, 0, len(inter.Methods.List))
	for _, method := range inter.Methods.List {
		switch src := method.Type.(type) {
		case *ast.FuncType:
			methods = append(methods, method)
		case *ast.Ident:
			methods = append(methods, findAnonMethods(src, withSpecs)...)
		case *ast.SelectorExpr:
			panic("Cannot yet handle embedded imported interfaces")
		}
	}
	inter.Methods.List = methods
}

func findAnonMethods(ident *ast.Ident, withSpecs []*ast.TypeSpec) []*ast.Field {
	var spec *ast.TypeSpec
	for idx := range withSpecs {
		if withSpecs[idx].Name.String() == ident.Name {
			spec = withSpecs[idx]
			break
		}
	}
	if spec == nil {
		// TODO: do something nicer with this error.
		panic(fmt.Errorf("Can't find anonymous type %s", ident.Name))
	}
	anon := spec.Type.(*ast.InterfaceType)
	flatten(anon, withSpecs)
	return anon.Methods.List
}
