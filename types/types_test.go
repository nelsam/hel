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
	mockGoDir.PathOutput.ret0 <- "/some/path"
	mockGoDir.PackagesOutput.ret0 <- map[string]*ast.Package{
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
	mockGoDir.PathOutput.ret0 <- "/some/path"
	mockGoDir.PackagesOutput.ret0 <- map[string]*ast.Package{
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
	mockGoDir.PathOutput.ret0 <- "/some/path"
	mockGoDir.PackagesOutput.ret0 <- map[string]*ast.Package{
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
	mockGoDir.PathOutput.ret0 <- "/some/path"
	mockGoDir.PackagesOutput.ret0 <- map[string]*ast.Package{
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

func TestAnonymousLocalTypes(t *testing.T) {
	expect := expect.New(t)

	mockGoDir := newMockGoDir()
	mockGoDir.PathOutput.ret0 <- "/some/path"
	mockGoDir.PackagesOutput.ret0 <- map[string]*ast.Package{
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
	mockGoDir.PathOutput.ret0 <- "/some/path"
	mockGoDir.PackagesOutput.ret0 <- map[string]*ast.Package{
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

	close(mockGoDir.ImportOutput.ret1)
	mockGoDir.ImportOutput.ret0 <- &ast.Package{
		Name: "foo",
		Files: map[string]*ast.File{
			"foo.go": parse(expect, `
    type Foo interface {
        Foo() X
    }`),
		},
	}

	found := types.Load(mockGoDir)
	expect(mockGoDir.ImportCalled).To.Have.Len(1)
	expect(<-mockGoDir.ImportInput.path).To.Equal("some/path/to/foo")
	expect(<-mockGoDir.ImportInput.pkg).To.Equal("foo")

	expect(found).To.Have.Len(1)
	typs := found[0].ExportedTypes()
	expect(typs).To.Have.Len(1)

	spec := typs[0]
	expect(spec).Not.To.Be.Nil()
	inter := spec.Type.(*ast.InterfaceType)
	expect(inter.Methods.List).To.Have.Len(2)
	read := inter.Methods.List[0]
	expect(read.Names[0].String()).To.Equal("Foo")
	f, isFunc := read.Type.(*ast.FuncType)
	expect(isFunc).To.Be.Ok()
	expect(f.Params.List).To.Have.Len(0)
	expect(f.Results.List).To.Have.Len(1)
	expr, isSelector := f.Results.List[0].Type.(*ast.SelectorExpr)
	expect(isSelector).To.Be.Ok()
	expect(expr.Sel.String()).To.Equal("foo")
	expect(expr.X.(*ast.Ident).String()).To.Equal("X")
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
