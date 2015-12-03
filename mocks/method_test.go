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
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

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
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
 package foo
 
 func (m *mockFoo) Foo(foo, bar string, baz int) {
   m.FooCalled <- true
   m.FooInput.foo <- foo
   m.FooInput.bar <- bar
   m.FooInput.baz <- baz
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
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
 package foo
 
 func (m *mockFoo) Foo() (foo, bar string, baz int) {
   m.FooCalled <- true
   return <-m.FooOutput.foo, <-m.FooOutput.bar, <-m.FooOutput.baz
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
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
 package foo
 
 func (m *mockFoo) Foo(arg0 int, arg1 string) (string, error) {
   m.FooCalled <- true
   m.FooInput.arg0 <- arg0
   m.FooInput.arg1 <- arg1
   return <-m.FooOutput.ret0, <-m.FooOutput.ret1
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}
