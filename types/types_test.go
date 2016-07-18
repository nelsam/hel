// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package types_test

import (
	"go/ast"
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/hel/types"
)

func TestLoad_NoTestFiles(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"foo": {
			Name: "foo",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, "type Foo interface {}"),
			},
		},
	}
	found := types.Load(mockGoDir)
	expect(found).To.Have.Len(1)
	expect(found[0].Len()).To.Equal(1)
	expect(found[0].Dir()).To.Equal("/some/path")
	expect(found[0].Package()).To.Equal("foo")
	expect(found[0].TestPackage()).To.Equal("foo_test")
}

func TestLoad_TestFilesInTestPackage(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"foo": {
			Name: "foo",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, "type Foo interface{}"),
			},
		},
		"foo_test": {
			Name: "foo_test",
			Files: map[string]*ast.File{
				"foo_test.go": parse(expect, "type Bar interface{}"),
			},
		},
	}
	found := types.Load(mockGoDir)
	expect(found).To.Have.Len(1)
	expect(found[0].Len()).To.Equal(1)
	expect(found[0].Package()).To.Equal("foo")
	expect(found[0].TestPackage()).To.Equal("foo_test")
}

func TestLoad_TestFilesInNonTestPackage(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"foo": {
			Name: "foo",
			Files: map[string]*ast.File{
				"foo.go":      parse(expect, "type Foo interface{}"),
				"foo_test.go": parse(expect, "type Bar interface{}"),
			},
		},
	}
	found := types.Load(mockGoDir)
	expect(found).To.Have.Len(1)
	expect(found[0].Len()).To.Equal(1)
	expect(found[0].Package()).To.Equal("foo")
	expect(found[0].TestPackage()).To.Equal("foo")
}

func TestFilter(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"foo": {
			Name: "foo",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, `
    type Foo interface {}
    type Bar interface {}
    type FooBar interface {}
    type BarFoo interface {}
    `),
			},
		},
	}
	found := types.Load(mockGoDir)
	expect(found).To.Have.Len(1)
	expect(found[0].Len()).To.Equal(4)

	notFiltered := found.Filter()
	expect(notFiltered).To.Have.Len(1)
	expect(notFiltered[0].Len()).To.Equal(4)

	foos := found.Filter("Foo")
	expect(foos).To.Have.Len(1)
	expect(foos[0].Len()).To.Equal(1)
	expect(foos[0].ExportedTypes()[0].Name.String()).To.Equal("Foo")

	fooPrefixes := found.Filter("Foo.*")
	expect(fooPrefixes).To.Have.Len(1)
	expect(fooPrefixes[0].Len()).To.Equal(2)
	expectNamesToMatch(expect, fooPrefixes[0].ExportedTypes(), "Foo", "FooBar")

	fooPostfixes := found.Filter(".*Foo")
	expect(fooPostfixes).To.Have.Len(1)
	expect(fooPostfixes[0].Len()).To.Equal(2)
	expectNamesToMatch(expect, fooPostfixes[0].ExportedTypes(), "Foo", "BarFoo")

	fooContainers := found.Filter("Foo.*", ".*Foo")
	expect(fooContainers).To.Have.Len(1)
	expect(fooContainers[0].Len()).To.Equal(3)
	expectNamesToMatch(expect, fooContainers[0].ExportedTypes(), "Foo", "FooBar", "BarFoo")
}

func TestLocalDependencies(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"bar": {
			Name: "bar",
			Files: map[string]*ast.File{
				"bar.go": parse(expect, `

    type Bar interface{
        Bar(Foo) Foo
    }`),
				"foo.go": parse(expect, `

    type Foo interface {
        Foo()
    }`),
			},
		},
	}

	found := types.Load(mockGoDir)

	expect(found).To.Have.Len(1)
	mockables := found[0].ExportedTypes()
	expect(mockables).To.Have.Len(2)

	var foo, bar *ast.TypeSpec
	for _, mockable := range mockables {
		switch mockable.Name.String() {
		case "Bar":
			bar = mockable
		case "Foo":
			foo = mockable
		}
	}
	expect(bar).Not.To.Be.Nil()

	dependencies := found[0].Dependencies(bar.Type.(*ast.InterfaceType))
	expect(dependencies).To.Have.Len(1)
	expect(dependencies[0]).To.Equal(foo)
}

func TestImportedDependencies(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"bar": {
			Name: "bar",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, `

    import "some/path/to/foo"

    type Bar interface{
        Bar(foo.Foo) foo.Bar
    }`),
			},
		},
	}

	close(mockGoDir.ImportOutput.Err)
	pkgName := "foo"
	pkg := &ast.Package{
		Name: pkgName,
		Files: map[string]*ast.File{
			"foo.go": parse(expect, `
    type Foo interface {
        Foo()
    }

	type Bar interface {
		Bar()
	}`),
		},
	}
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName

	found := types.Load(mockGoDir)
	expect(mockGoDir.ImportCalled).To.Have.Len(2)
	expect(<-mockGoDir.ImportInput.Path).To.Equal("some/path/to/foo")

	expect(found).To.Have.Len(1)
	mockables := found[0].ExportedTypes()
	expect(mockables).To.Have.Len(1)

	dependencies := found[0].Dependencies(mockables[0].Type.(*ast.InterfaceType))
	expect(dependencies).To.Have.Len(2)

	names := make(map[string]bool)
	for _, dependent := range dependencies {
		names[dependent.Name.String()] = true
	}
	expect(names).To.Equal(map[string]bool{"Foo": true, "Bar": true})
}

func TestAliasedImportedDependencies(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"bar": {
			Name: "bar",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, `

    import baz "some/path/to/foo"

    type Bar interface{
        Bar(baz.Foo) baz.Bar
    }`),
			},
		},
	}

	close(mockGoDir.ImportOutput.Err)
	pkgName := "foo"
	pkg := &ast.Package{
		Name: pkgName,
		Files: map[string]*ast.File{
			"foo.go": parse(expect, `
    type Foo interface {
        Foo()
    }

	type Bar interface {
		Bar()
	}`),
		},
	}
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName

	found := types.Load(mockGoDir)
	expect(mockGoDir.ImportCalled).To.Have.Len(2)
	expect(<-mockGoDir.ImportInput.Path).To.Equal("some/path/to/foo")

	expect(found).To.Have.Len(1)
	mockables := found[0].ExportedTypes()
	expect(mockables).To.Have.Len(1)

	dependencies := found[0].Dependencies(mockables[0].Type.(*ast.InterfaceType))
	expect(dependencies).To.Have.Len(2)

	names := make(map[string]bool)
	for _, dependent := range dependencies {
		names[dependent.Name.String()] = true
	}
	expect(names).To.Equal(map[string]bool{"Foo": true, "Bar": true})
}

func TestAnonymousLocalTypes(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"foo": {
			Name: "foo",
			Files: map[string]*ast.File{
				"bar.go": parse(expect, `
    type Bar interface{
        Foo
        Bar()
    }`),
				"foo.go": parse(expect, `
    type Foo interface{
        Foo()
    }`),
			},
		},
	}
	found := types.Load(mockGoDir)
	expect(found).To.Have.Len(1)

	typs := found[0].ExportedTypes()
	expect(typs).To.Have.Len(2)

	spec := find(expect, typs, "Bar")
	expect(spec).Not.To.Be.Nil()
	inter := spec.Type.(*ast.InterfaceType)
	expect(inter.Methods.List).To.Have.Len(2)
	foo := inter.Methods.List[0]
	expect(foo.Names[0].String()).To.Equal("Foo")
	_, isFunc := foo.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()
}

func TestAnonymousImportedTypes(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"bar": {
			Name: "bar",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, `

    import "some/path/to/foo"

    type Bar interface{
        foo.Foo
        Bar()
    }`),
			},
		},
	}

	close(mockGoDir.ImportOutput.Err)
	pkgName := "foo"
	pkg := &ast.Package{
		Name: pkgName,
		Files: map[string]*ast.File{
			"foo.go": parse(expect, `
    type Foo interface {
        Foo(x X) Y
    }

	type X int
	type Y int`),
		},
	}
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName

	found := types.Load(mockGoDir)

	// 3 calls: 1 for the initial import, then deps imports for X and Y
	expect(mockGoDir.ImportCalled).To.Have.Len(3)
	expect(<-mockGoDir.ImportInput.Path).To.Equal("some/path/to/foo")

	expect(found).To.Have.Len(1)
	typs := found[0].ExportedTypes()
	expect(typs).To.Have.Len(1)

	spec := typs[0]
	expect(spec).Not.To.Be.Nil()
	inter := spec.Type.(*ast.InterfaceType)
	expect(inter.Methods.List).To.Have.Len(2)

	foo := inter.Methods.List[0]
	expect(foo.Names[0].String()).To.Equal("Foo")
	f, isFunc := foo.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()
	expect(f.Params.List).To.Have.Len(1)
	expect(f.Results.List).To.Have.Len(1)
	expr, isSelector := f.Params.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("foo")
	expect(expr.Sel.String()).To.Equal("X")
	expr, isSelector = f.Results.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("foo")
	expect(expr.Sel.String()).To.Equal("Y")
}

func TestAnonymousAliasedImportedTypes(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"bar": {
			Name: "bar",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, `

    import baz "some/path/to/foo"

    type Bar interface{
        baz.Foo
        Bar()
    }`),
			},
		},
	}

	close(mockGoDir.ImportOutput.Err)
	pkgName := "foo"
	pkg := &ast.Package{
		Name: pkgName,
		Files: map[string]*ast.File{
			"foo.go": parse(expect, `
    type Foo interface {
        Foo(x X) Y
    }

	type X int
	type Y int`),
		},
	}
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName

	found := types.Load(mockGoDir)

	// 3 calls: 1 for the initial import, then deps imports for X and Y
	expect(mockGoDir.ImportCalled).To.Have.Len(3)
	expect(<-mockGoDir.ImportInput.Path).To.Equal("some/path/to/foo")

	expect(found).To.Have.Len(1)
	typs := found[0].ExportedTypes()
	expect(typs).To.Have.Len(1)

	spec := typs[0]
	expect(spec).Not.To.Be.Nil()
	inter := spec.Type.(*ast.InterfaceType)
	expect(inter.Methods.List).To.Have.Len(2)

	foo := inter.Methods.List[0]
	expect(foo.Names[0].String()).To.Equal("Foo")
	f, isFunc := foo.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()
	expect(f.Params.List).To.Have.Len(1)
	expect(f.Results.List).To.Have.Len(1)
	expr, isSelector := f.Params.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("baz")
	expect(expr.Sel.String()).To.Equal("X")
	expr, isSelector = f.Results.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("baz")
	expect(expr.Sel.String()).To.Equal("Y")
}

func TestAnonymousImportedTypes_Recursion(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.Path <- "/some/path"
	mockGoDir.PackagesOutput.Packages <- map[string]*ast.Package{
		"bar": {
			Name: "bar",
			Files: map[string]*ast.File{
				"foo.go": parse(expect, `

    import "some/path/to/foo"

    type Bar interface{
        foo.Foo
        Bar()
    }`),
			},
		},
	}

	close(mockGoDir.ImportOutput.Err)
	pkgName := "foo"
	pkg := &ast.Package{
		Name: pkgName,
		Files: map[string]*ast.File{
			"foo.go": parse(expect, `
    type Foo interface {
        Foo(func(X) Y) func(Y) X
    }`),
		},
	}
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName
	mockGoDir.ImportOutput.Pkg <- pkg
	mockGoDir.ImportOutput.Name <- pkgName

	found := types.Load(mockGoDir)

	// One call for the initial import, four more for dependency checking
	expect(mockGoDir.ImportCalled).To.Have.Len(5)
	expect(<-mockGoDir.ImportInput.Path).To.Equal("some/path/to/foo")

	expect(found).To.Have.Len(1)
	typs := found[0].ExportedTypes()
	expect(typs).To.Have.Len(1)

	spec := typs[0]
	expect(spec).Not.To.Be.Nil()
	inter := spec.Type.(*ast.InterfaceType)
	expect(inter.Methods.List).To.Have.Len(2)

	foo := inter.Methods.List[0]
	expect(foo.Names[0].String()).To.Equal("Foo")
	f, isFunc := foo.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()
	expect(f.Params.List).To.Have.Len(1)
	expect(f.Results.List).To.Have.Len(1)

	input := f.Params.List[0]
	in, isFunc := input.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()

	expr, isSelector := in.Params.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("foo")
	expect(expr.Sel.String()).To.Equal("X")
	expr, isSelector = in.Results.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("foo")
	expect(expr.Sel.String()).To.Equal("Y")

	output := f.Params.List[0]
	out, isFunc := output.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()

	expr, isSelector = out.Params.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("foo")
	expect(expr.Sel.String()).To.Equal("X")
	expr, isSelector = out.Results.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.X.(*ast.Ident).String()).To.Equal("foo")
	expect(expr.Sel.String()).To.Equal("Y")
}

func expectNamesToMatch(expect func(interface{}) *expect.Expect, list []*ast.TypeSpec, names ...string) {
	listNames := make(map[string]struct{}, len(list))
	for _, spec := range list {
		listNames[spec.Name.String()] = struct{}{}
	}
	expectedNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		expectedNames[name] = struct{}{}
	}
	expect(listNames).To.Equal(expectedNames)
}

func find(expect func(interface{}) *expect.Expect, typs []*ast.TypeSpec, name string) *ast.TypeSpec {
	for _, typ := range typs {
		if typ.Name.String() == name {
			return typ
		}
	}
	return nil
}
