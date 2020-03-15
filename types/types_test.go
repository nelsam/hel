// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package types_test

import (
	"go/ast"
	"testing"

	"github.com/nelsam/hel/v2/pers"
	"github.com/nelsam/hel/v2/types"
	"github.com/poy/onpar"
	"github.com/poy/onpar/expect"
	"github.com/poy/onpar/matchers"
	"golang.org/x/tools/go/packages"
)

type expectation = expect.Expectation

var (
	equal        = matchers.Equal
	not          = matchers.Not
	haveOccurred = matchers.HaveOccurred
	haveLen      = matchers.HaveLen
	beNil        = matchers.BeNil
	beTrue       = matchers.BeTrue
)

func TestTypes(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) (expectation, *mockGoDir) {
		return expect.New(t), newMockGoDir()
	})

	o.Spec("Load_EmptyInterface", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "foo",
			Syntax: []*ast.File{
				parse(expect, "type Foo interface {}"),
			},
		})
		found := types.Load(mockGoDir)
		expect(found).To(haveLen(1))
		expect(found[0].Len()).To(equal(1))
		expect(found[0].Dir()).To(equal("/some/path"))
		expect(found[0].Package()).To(equal("foo"))
	})

	o.Spec("Filter", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "foo",
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface {}
    type Bar interface {}
    type FooBar interface {}
    type BarFoo interface {}
    `),
			},
		})
		found := types.Load(mockGoDir)
		expect(found).To(haveLen(1))
		expect(found[0].Len()).To(equal(4))

		notFiltered := found.Filter()
		expect(notFiltered).To(haveLen(1))
		expect(notFiltered[0].Len()).To(equal(4))

		foos := found.Filter("Foo")
		expect(foos).To(haveLen(1))
		expect(foos[0].Len()).To(equal(1))
		expect(foos[0].ExportedTypes()[0].Name.String()).To(equal("Foo"))

		fooPrefixes := found.Filter("Foo.*")
		expect(fooPrefixes).To(haveLen(1))
		expect(fooPrefixes[0].Len()).To(equal(2))
		expectNamesToMatch(expect, fooPrefixes[0].ExportedTypes(), "Foo", "FooBar")

		fooPostfixes := found.Filter(".*Foo")
		expect(fooPostfixes).To(haveLen(1))
		expect(fooPostfixes[0].Len()).To(equal(2))
		expectNamesToMatch(expect, fooPostfixes[0].ExportedTypes(), "Foo", "BarFoo")

		fooContainers := found.Filter("Foo.*", ".*Foo")
		expect(fooContainers).To(haveLen(1))
		expect(fooContainers[0].Len()).To(equal(3))
		expectNamesToMatch(expect, fooContainers[0].ExportedTypes(), "Foo", "FooBar", "BarFoo")
	})

	o.Spec("LocalDependencies", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "bar",
			Syntax: []*ast.File{
				parse(expect, `

    type Bar interface{
        Bar(Foo) Foo
    }`),
				parse(expect, `

    type Foo interface {
        Foo()
    }`),
			},
		})

		found := types.Load(mockGoDir)

		expect(found).To(haveLen(1))
		mockables := found[0].ExportedTypes()
		expect(mockables).To(haveLen(2))

		var foo, bar *ast.TypeSpec
		for _, mockable := range mockables {
			switch mockable.Name.String() {
			case "Bar":
				bar = mockable
			case "Foo":
				foo = mockable
			}
		}
		if bar == nil {
			t.Fatal("expected to find a Bar type")
		}

		expect(found).To(haveLen(1))
		dependencies := found[0].Dependencies(bar.Type.(*ast.InterfaceType))
		expect(dependencies).To(haveLen(1))
		expect(dependencies[0].Type).To(equal(foo))
		expect(dependencies[0].PkgName).To(equal(""))
		expect(dependencies[0].PkgPath).To(equal(""))
	})

	o.Spec("ImportedDependencies", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "bar",
			Syntax: []*ast.File{
				parse(expect, `

    import "some/path/to/foo"

    type Bar interface{
        Bar(foo.Foo) foo.Bar
    }`),
			},
		})

		pkgName := "foo"
		pkg := &packages.Package{
			Name: pkgName,
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface {
        Foo()
    }

	type Bar interface {
		Bar()
	}`),
			},
		}
		done, err := pers.ConsistentlyReturn(mockGoDir.ImportOutput, pkg, nil)
		expect(err).To(not(haveOccurred()))
		defer done()

		found := types.Load(mockGoDir)
		expect(found).To(haveLen(1))
		expect(mockGoDir.ImportCalled).To(haveLen(2))

		expect(<-mockGoDir.ImportInput.Path).To(equal("some/path/to/foo"))

		mockables := found[0].ExportedTypes()
		expect(mockables).To(haveLen(1))
		if mockables[0] == nil {
			t.Fatal("expected mockables[0] to be non-nil")
		}

		dependencies := found[0].Dependencies(mockables[0].Type.(*ast.InterfaceType))
		expect(dependencies).To(haveLen(2))

		names := make(map[string]bool)
		for _, dependent := range dependencies {
			expect(dependent.PkgName).To(equal("foo"))
			expect(dependent.PkgPath).To(equal("some/path/to/foo"))
			names[dependent.Type.Name.String()] = true
		}
		expect(names).To(equal(map[string]bool{"Foo": true, "Bar": true}))
	})

	o.Spec("AliasedImportedDependencies", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "bar",
			Syntax: []*ast.File{
				parse(expect, `

    import baz "some/path/to/foo"

    type Bar interface{
        Bar(baz.Foo) baz.Bar
    }`),
			},
		})

		pkgName := "foo"
		pkg := &packages.Package{
			Name: pkgName,
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface {
        Foo()
    }

	type Bar interface {
		Bar()
	}`),
			},
		}
		done, err := pers.ConsistentlyReturn(mockGoDir.ImportOutput, pkg, nil)
		expect(err).To(not(haveOccurred()))
		defer done()

		found := types.Load(mockGoDir)
		expect(mockGoDir.ImportCalled).To(haveLen(2))
		expect(<-mockGoDir.ImportInput.Path).To(equal("some/path/to/foo"))

		expect(found).To(haveLen(1))
		mockables := found[0].ExportedTypes()
		expect(mockables).To(haveLen(1))

		dependencies := found[0].Dependencies(mockables[0].Type.(*ast.InterfaceType))
		expect(dependencies).To(haveLen(2))

		names := make(map[string]bool)
		for _, dependent := range dependencies {
			expect(dependent.PkgName).To(equal("baz"))
			expect(dependent.PkgPath).To(equal("some/path/to/foo"))
			names[dependent.Type.Name.String()] = true
		}
		expect(names).To(equal(map[string]bool{"Foo": true, "Bar": true}))
	})

	// TestAnonymousError is testing the only case (as of go 1.7) where
	// a builtin is an interface type.
	o.Spec("AnonymousError", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "foo",
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface{
        error
    }`),
			},
		})
		found := types.Load(mockGoDir)
		expect(found).To(haveLen(1))

		typs := found[0].ExportedTypes()
		expect(typs).To(haveLen(1))

		spec := typs[0]
		expect(spec).To(not(beNil()))

		inter := spec.Type.(*ast.InterfaceType)
		expect(inter.Methods.List).To(haveLen(1))
		err := inter.Methods.List[0]
		expect(err.Names[0].String()).To(equal("Error"))
		_, isFunc := err.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())
	})

	o.Spec("AnonymousLocalTypes", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "foo",
			Syntax: []*ast.File{
				parse(expect, `
    type Bar interface{
        Foo
        Bar()
    }`),
				parse(expect, `
    type Foo interface{
        Foo()
    }`),
			},
		})
		found := types.Load(mockGoDir)
		expect(found).To(haveLen(1))

		typs := found[0].ExportedTypes()
		expect(typs).To(haveLen(2))

		spec := find(expect, typs, "Bar")
		expect(spec).To(not(beNil()))
		inter := spec.Type.(*ast.InterfaceType)
		expect(inter.Methods.List).To(haveLen(2))
		foo := inter.Methods.List[0]
		expect(foo.Names[0].String()).To(equal("Foo"))
		_, isFunc := foo.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())
	})

	o.Spec("AnonymousImportedTypes", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "bar",
			Syntax: []*ast.File{
				parse(expect, `

    import "some/path/to/foo"

    type Bar interface{
        foo.Foo
        Bar()
    }`),
			},
		})

		pkgName := "foo"
		pkg := &packages.Package{
			Name: pkgName,
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface {
        Foo(x X) Y
    }

	type X int
	type Y int`),
			},
		}
		done, err := pers.ConsistentlyReturn(mockGoDir.ImportOutput, pkg, nil)
		expect(err).To(not(haveOccurred()))
		defer done()

		found := types.Load(mockGoDir)

		// 3 calls: 1 for the initial import, then deps imports for X and Y
		expect(mockGoDir.ImportCalled).To(haveLen(3))
		expect(<-mockGoDir.ImportInput.Path).To(equal("some/path/to/foo"))

		expect(found).To(haveLen(1))
		typs := found[0].ExportedTypes()
		expect(typs).To(haveLen(1))

		spec := typs[0]
		expect(spec).To(not(beNil()))
		inter := spec.Type.(*ast.InterfaceType)
		expect(inter.Methods.List).To(haveLen(2))

		foo := inter.Methods.List[0]
		expect(foo.Names[0].String()).To(equal("Foo"))
		f, isFunc := foo.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())
		expect(f.Params.List).To(haveLen(1))
		expect(f.Results.List).To(haveLen(1))
		expr, isSelector := f.Params.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("foo"))
		expect(expr.Sel.String()).To(equal("X"))
		expr, isSelector = f.Results.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("foo"))
		expect(expr.Sel.String()).To(equal("Y"))
	})

	o.Spec("AnonymousAliasedImportedTypes", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "bar",
			Syntax: []*ast.File{
				parse(expect, `

    import baz "some/path/to/foo"

    type Bar interface{
        baz.Foo
        Bar()
    }`),
			},
		})

		pkgName := "foo"
		pkg := &packages.Package{
			Name: pkgName,
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface {
        Foo(x X) Y
    }

	type X int
	type Y int`),
			},
		}
		done, err := pers.ConsistentlyReturn(mockGoDir.ImportOutput, pkg, nil)
		expect(err).To(not(haveOccurred()))
		defer done()

		found := types.Load(mockGoDir)

		// 3 calls: 1 for the initial import, then deps imports for X and Y
		expect(mockGoDir.ImportCalled).To(haveLen(3))
		expect(<-mockGoDir.ImportInput.Path).To(equal("some/path/to/foo"))

		expect(found).To(haveLen(1))
		typs := found[0].ExportedTypes()
		expect(typs).To(haveLen(1))

		spec := typs[0]
		expect(spec).To(not(beNil()))
		inter := spec.Type.(*ast.InterfaceType)
		expect(inter.Methods.List).To(haveLen(2))

		foo := inter.Methods.List[0]
		expect(foo.Names[0].String()).To(equal("Foo"))
		f, isFunc := foo.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())
		expect(f.Params.List).To(haveLen(1))
		expect(f.Results.List).To(haveLen(1))
		expr, isSelector := f.Params.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("baz"))
		expect(expr.Sel.String()).To(equal("X"))
		expr, isSelector = f.Results.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("baz"))
		expect(expr.Sel.String()).To(equal("Y"))
	})

	o.Spec("AnonymousImportedTypes_Recursion", func(expect expectation, mockGoDir *mockGoDir) {
		pers.ConsistentlyReturn(mockGoDir.PathOutput, "/some/path")
		pers.ConsistentlyReturn(mockGoDir.PackageOutput, &packages.Package{
			Name: "bar",
			Syntax: []*ast.File{
				parse(expect, `

    import "some/path/to/foo"

    type Bar interface{
        foo.Foo
        Bar()
    }`),
			},
		})

		pkgName := "foo"
		pkg := &packages.Package{
			Name: pkgName,
			Syntax: []*ast.File{
				parse(expect, `
    type Foo interface {
        Foo(func(X) Y) func(Y) X
    }`),
			},
		}
		done, err := pers.ConsistentlyReturn(mockGoDir.ImportOutput, pkg, nil)
		expect(err).To(not(haveOccurred()))
		defer done()

		found := types.Load(mockGoDir)

		// One call for the initial import, four more for dependency checking
		expect(mockGoDir.ImportCalled).To(haveLen(5))
		expect(<-mockGoDir.ImportInput.Path).To(equal("some/path/to/foo"))

		expect(found).To(haveLen(1))
		typs := found[0].ExportedTypes()
		expect(typs).To(haveLen(1))

		spec := typs[0]
		expect(spec).To(not(beNil()))
		inter := spec.Type.(*ast.InterfaceType)
		expect(inter.Methods.List).To(haveLen(2))

		foo := inter.Methods.List[0]
		expect(foo.Names[0].String()).To(equal("Foo"))
		f, isFunc := foo.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())
		expect(f.Params.List).To(haveLen(1))
		expect(f.Results.List).To(haveLen(1))

		input := f.Params.List[0]
		in, isFunc := input.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())

		expr, isSelector := in.Params.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("foo"))
		expect(expr.Sel.String()).To(equal("X"))
		expr, isSelector = in.Results.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("foo"))
		expect(expr.Sel.String()).To(equal("Y"))

		output := f.Params.List[0]
		out, isFunc := output.Type.(*ast.FuncType)
		expect(isFunc).To(beTrue())

		expr, isSelector = out.Params.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("foo"))
		expect(expr.Sel.String()).To(equal("X"))
		expr, isSelector = out.Results.List[0].Type.(*ast.SelectorExpr)
		expect(isSelector).To(beTrue())
		expect(expr.X.(*ast.Ident).String()).To(equal("foo"))
		expect(expr.Sel.String()).To(equal("Y"))
	})
}

func expectNamesToMatch(expect expectation, list []*ast.TypeSpec, names ...string) {
	listNames := make(map[string]struct{}, len(list))
	for _, spec := range list {
		listNames[spec.Name.String()] = struct{}{}
	}
	expectedNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		expectedNames[name] = struct{}{}
	}
	expect(listNames).To(equal(expectedNames))
}

func find(expect expectation, typs []*ast.TypeSpec, name string) *ast.TypeSpec {
	for _, typ := range typs {
		if typ.Name.String() == name {
			return typ
		}
	}
	return nil
}
