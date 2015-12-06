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
	expect(found[0].TestPackage()).To.Equal("foo")
}
