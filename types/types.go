// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package types

import (
	"fmt"
	"go/ast"
	"log"
	"regexp"
	"strings"
	"unicode"
)

// A GoDir is a type that represents a directory of Go files.
type GoDir interface {
	Path() (path string)
	Packages() (packages map[string]*ast.Package)
	Import(path, srcDir string) (name string, pkg *ast.Package, err error)
}

// A Dir is a type that represents a directory containing Go
// packages.
type Dir struct {
	dir          string
	pkg          string
	testPkg      string
	types        []*ast.TypeSpec
	dependencies map[*ast.InterfaceType][]*ast.TypeSpec
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

// Dependencies returns all interface types that typ depends on for
// method parameters or results.
func (d Dir) Dependencies(typ *ast.InterfaceType) []*ast.TypeSpec {
	return d.dependencies[typ]
}

func (d Dir) addPkg(name string, pkg *ast.Package, dir GoDir) Dir {
	if d.testPkg == "" {
		// Default for packages that don't have tests yet.
		d.testPkg = name + "_test"
	}
	newTypes, deps, testsFound := loadPkgTypeSpecs(pkg, dir)
	if testsFound {
		// This package already has test files, so this will
		// always be the test package.
		d.testPkg = name
	}
	if d.pkg == "" || !testsFound {
		d.pkg = name
	}
	for inter, typs := range deps {
		d.dependencies[inter] = append(d.dependencies[inter], typs...)
	}
	d.types = append(d.types, newTypes...)
	return d
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
			dir:          dir.Path(),
			dependencies: make(map[*ast.InterfaceType][]*ast.TypeSpec),
		}
		for name, pkg := range dir.Packages() {
			d = d.addPkg(name, pkg, dir)
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

// dependencies returns all *ast.TypeSpec values with a Type of
// *ast.InterfaceType.  It assumes that typ is pre-flattened.
func dependencies(typ *ast.InterfaceType, available []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) []*ast.TypeSpec {
	if typ.Methods == nil {
		return nil
	}
	dependencies := make(map[*ast.TypeSpec]struct{})
	for _, meth := range typ.Methods.List {
		f := meth.Type.(*ast.FuncType)
		addSpecs(dependencies, loadDependencies(f.Params, available, withImports, dir)...)
		addSpecs(dependencies, loadDependencies(f.Results, available, withImports, dir)...)
	}
	dependentSlice := make([]*ast.TypeSpec, 0, len(dependencies))
	for dependent := range dependencies {
		dependentSlice = append(dependentSlice, dependent)
	}
	return dependentSlice
}

func addSpecs(set map[*ast.TypeSpec]struct{}, values ...*ast.TypeSpec) {
	for _, value := range values {
		set[value] = struct{}{}
	}
}

func names(specs []*ast.TypeSpec) (names []string) {
	for _, spec := range specs {
		names = append(names, spec.Name.String())
	}
	return names
}

func loadDependencies(fields *ast.FieldList, available []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) (dependencies []*ast.TypeSpec) {
	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		switch src := field.Type.(type) {
		case *ast.Ident:
			for _, spec := range available {
				if spec.Name.String() == src.String() {
					if _, ok := spec.Type.(*ast.InterfaceType); ok {
						dependencies = append(dependencies, spec)
					}
					break
				}
			}
		case *ast.SelectorExpr:
			selectorName := src.X.(*ast.Ident).String()
			for _, imp := range withImports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				importName := imp.Name.String()
				pkgName, pkg, err := dir.Import(importPath, "")
				if err != nil {
					log.Printf("Error loading dependencies: %s", err)
					continue
				}
				if imp.Name == nil {
					importName = pkgName
				}
				if selectorName != importName {
					continue
				}
				d := Dir{
					dependencies: make(map[*ast.InterfaceType][]*ast.TypeSpec),
				}
				d = d.addPkg(importName, pkg, dir)
				for _, typ := range d.ExportedTypes() {
					if typ.Name.String() == src.Sel.String() {
						if _, ok := typ.Type.(*ast.InterfaceType); ok {
							dependencies = append(dependencies, typ)
						}
						break
					}
				}
			}
		case *ast.FuncType:
			dependencies = append(dependencies, loadDependencies(src.Params, available, withImports, dir)...)
			dependencies = append(dependencies, loadDependencies(src.Results, available, withImports, dir)...)
		}
	}
	return dependencies
}

func loadPkgTypeSpecs(pkg *ast.Package, dir GoDir) (specs []*ast.TypeSpec, depMap map[*ast.InterfaceType][]*ast.TypeSpec, hasTests bool) {
	depMap = make(map[*ast.InterfaceType][]*ast.TypeSpec)
	imports := make(map[*ast.TypeSpec][]*ast.ImportSpec)
	defer func() {
		for _, spec := range specs {
			inter, ok := spec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			depMap[inter] = dependencies(inter, specs, imports[spec], dir)
		}
	}()
	for name, f := range pkg.Files {
		if strings.HasSuffix(name, "_test.go") {
			hasTests = true
			continue
		}
		fileImports := f.Imports
		fileSpecs := loadFileTypeSpecs(f)
		for _, spec := range fileSpecs {
			imports[spec] = fileImports
		}

		// flattenAnon needs to be called for each file, but the
		// withSpecs parameter needs *all* specs, from *all* files.
		// So we defer the flatten call until all files are processed.
		defer func() {
			flattenAnon(fileSpecs, specs, fileImports, dir)
		}()

		specs = append(specs, fileSpecs...)
	}
	return specs, depMap, hasTests
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
			importedTypes, _ := findImportedTypes(src.X.(*ast.Ident), withImports, dir)
			methods = append(methods, findAnonMethods(src.Sel, importedTypes, nil, dir)...)
		}
	}
	inter.Methods.List = methods
}

func findImportedTypes(name *ast.Ident, withImports []*ast.ImportSpec, dir GoDir) ([]*ast.TypeSpec, map[*ast.InterfaceType][]*ast.TypeSpec) {
	importName := name.String()
	for _, imp := range withImports {
		path := strings.Trim(imp.Path.Value, `"`)
		name, pkg, err := dir.Import(path, "")
		if err != nil {
			log.Printf("Error loading import: %s", err)
			continue
		}
		if imp.Name != nil {
			name = imp.Name.String()
		}
		if name != importName {
			continue
		}
		typs, deps, _ := loadPkgTypeSpecs(pkg, dir)
		addSelector(typs, importName)
		return typs, deps
	}
	return nil, nil
}

func addSelector(typs []*ast.TypeSpec, selector string) {
	for _, typ := range typs {
		inter := typ.Type.(*ast.InterfaceType)
		for _, meth := range inter.Methods.List {
			addFuncSelectors(meth.Type.(*ast.FuncType), selector)
		}
	}
}

func addFuncSelectors(method *ast.FuncType, selector string) {
	if method.Params != nil {
		addFieldSelectors(method.Params.List, selector)
	}
	if method.Results != nil {
		addFieldSelectors(method.Results.List, selector)
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
		addFuncSelectors(src, selector)
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
