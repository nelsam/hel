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
	expect(fooPrefixes[0].ExportedTypes()[0].Name.String()).To.Equal("Foo")
	expect(fooPrefixes[0].ExportedTypes()[1].Name.String()).To.Equal("FooBar")

	fooPostfixes := found.Filter(".*Foo")
	expect(fooPostfixes).To.Have.Len(1)
	expect(fooPostfixes[0].Len()).To.Equal(2)
	expect(fooPostfixes[0].ExportedTypes()[0].Name.String()).To.Equal("Foo")
	expect(fooPostfixes[0].ExportedTypes()[1].Name.String()).To.Equal("BarFoo")

	fooContainers := found.Filter("Foo.*", ".*Foo")
	expect(fooContainers).To.Have.Len(1)
	expect(fooContainers[0].Len()).To.Equal(3)
	expect(fooContainers[0].ExportedTypes()[0].Name.String()).To.Equal("Foo")
	expect(fooContainers[0].ExportedTypes()[1].Name.String()).To.Equal("FooBar")
	expect(fooContainers[0].ExportedTypes()[2].Name.String()).To.Equal("BarFoo")
}
