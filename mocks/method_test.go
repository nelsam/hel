// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks_test

import (
	"go/ast"
	"go/format"
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/hel/mocks"
)

func TestMockSimpleMethod(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
         Foo()
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo() {
   m.FooCalled <- true
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodParams(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
         Foo(foo, bar string, baz int)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(foo, bar string, baz int) {
   m.FooCalled <- true
   m.FooInput.Foo <- foo
   m.FooInput.Bar <- bar
   m.FooInput.Baz <- baz
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodReturns(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo() (foo, bar string, baz int)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo() (foo, bar string, baz int) {
   m.FooCalled <- true
   return <-m.FooOutput.Foo, <-m.FooOutput.Bar, <-m.FooOutput.Baz
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodWithBlockingReturn(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo()
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	mock.SetBlockingReturn(true)
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo() () {
   m.FooCalled <- true
   <-m.FooOutput.BlockReturn
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodUnnamedValues(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo(int, string) (string, error)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(arg0 int, arg1 string) (string, error) {
   m.FooCalled <- true
   m.FooInput.Arg0 <- arg0
   m.FooInput.Arg1 <- arg1
   return <-m.FooOutput.Ret0, <-m.FooOutput.Ret1
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodLocalTypes(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo(bar bar.Bar, baz func(f Foo) error) (Foo, func() Foo, error)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(bar bar.Bar, baz func(f Foo) error) (Foo, func() Foo, error) {
   m.FooCalled <- true
   m.FooInput.Bar <- bar
   m.FooInput.Baz <- baz
   return <-m.FooOutput.Ret0, <-m.FooOutput.Ret1, <-m.FooOutput.Ret2
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))

	method.PrependLocalPackage("foo")

	expected, err = format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(bar bar.Bar, baz func(f foo.Foo) error) (foo.Foo, func() foo.Foo, error) {
   m.FooCalled <- true
   m.FooInput.Bar <- bar
   m.FooInput.Baz <- baz
   return <-m.FooOutput.Ret0, <-m.FooOutput.Ret1, <-m.FooOutput.Ret2
 }`))
	expect(err).To.Be.Nil()

	src = source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}
