package types

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
	"unicode"
)

// A GoDir is a type that represents a directory of Go files.
type GoDir interface {
	Path() string
	Packages() map[string]*ast.Package
	Import(path, pkg string) (*ast.Package, error)
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
			newTypes, testsFound := loadPkgTypeSpecs(pkg, dir)
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

func loadPkgTypeSpecs(pkg *ast.Package, dir GoDir) (specs []*ast.TypeSpec, hasTests bool) {
	for name, f := range pkg.Files {
		if strings.HasSuffix(name, "_test.go") {
			hasTests = true
			continue
		}
		fileImports := f.Imports
		fileSpecs := loadFileTypeSpecs(f)

		// flattenAnon needs to be called for each file, but the
		// withSpecs parameter needs *all* specs, from *all* files.
		// So we defer the flatten call until all files are processed.
		defer func() {
			flattenAnon(fileSpecs, specs, fileImports, dir)
		}()

		specs = append(specs, fileSpecs...)
	}
	return specs, hasTests
}

func loadFileTypeSpecs(f *ast.File) (specs []*ast.TypeSpec) {
	for _, obj := range f.Scope.Objects {
		spec, ok := obj.Decl.(*ast.TypeSpec)
		if !ok {
			continue
		}
		if _, ok := spec.Type.(*ast.InterfaceType); !ok {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func flattenAnon(specs, withSpecs []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) {
	for _, spec := range specs {
		inter := spec.Type.(*ast.InterfaceType)
		flatten(inter, withSpecs, withImports, dir)
	}
}

func flatten(inter *ast.InterfaceType, withSpecs []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) {
	if inter.Methods == nil {
		return
	}
	methods := make([]*ast.Field, 0, len(inter.Methods.List))
	for _, method := range inter.Methods.List {
		switch src := method.Type.(type) {
		case *ast.FuncType:
			methods = append(methods, method)
		case *ast.Ident:
			methods = append(methods, findAnonMethods(src, withSpecs, withImports, dir)...)
		case *ast.SelectorExpr:
			importedTypes := findImportedTypes(src.X.(*ast.Ident), withImports, dir)
			methods = append(methods, findAnonMethods(src.Sel, importedTypes, nil, dir)...)
		}
	}
	inter.Methods.List = methods
}

func findImportedTypes(name *ast.Ident, withImports []*ast.ImportSpec, dir GoDir) []*ast.TypeSpec {
	importName := name.String()
	for _, imp := range withImports {
		path := strings.Trim(imp.Path.Value, `"`)
		if pkg, err := dir.Import(path, importName); err == nil {
			typs, _ := loadPkgTypeSpecs(pkg, dir)
			addSelector(typs, importName)
			return typs
		}
	}
	return nil
}

func addSelector(typs []*ast.TypeSpec, selector string) {
	for _, typ := range typs {
		inter := typ.Type.(*ast.InterfaceType)
		for _, meth := range inter.Methods.List {
			method := meth.Type.(*ast.FuncType)
			if method.Params != nil {
				addFieldSelectors(method.Params.List, selector)
			}
			if method.Results != nil {
				addFieldSelectors(method.Results.List, selector)
			}
		}
	}
}

func addFieldSelectors(fields []*ast.Field, selector string) {
	for idx, field := range fields {
		fields[idx] = addFieldSelector(field, selector)
	}
}

func addFieldSelector(field *ast.Field, selector string) *ast.Field {
	switch src := field.Type.(type) {
	case *ast.Ident:
		if !unicode.IsUpper(rune(src.String()[0])) {
			return field
		}
		return &ast.Field{
			Type: &ast.SelectorExpr{
				X:   &ast.Ident{Name: selector},
				Sel: src,
			},
		}
	case *ast.FuncType:
		addFieldSelectors(src.Params.List, selector)
		addFieldSelectors(src.Results.List, selector)
	}
	return field
}

func findAnonMethods(ident *ast.Ident, withSpecs []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) []*ast.Field {
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
	flatten(anon, withSpecs, withImports, dir)
	return anon.Methods.List
}
