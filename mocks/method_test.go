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
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
package foo
func (m *mockFoo) Foo() {
m.Foo.called <- true
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
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
 package foo
 
 func (m *mockFoo) Foo(foo, bar string, baz int) {
   m.Foo.called <- true
   m.Foo.input.foo <- foo
   m.Foo.input.bar <- bar
   m.Foo.input.baz <- baz
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
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
 package foo
 
 func (m *mockFoo) Foo() (foo, bar string, baz int) {
   m.Foo.called <- true
   return <-m.Foo.output.foo, <-m.Foo.output.bar, <-m.Foo.output.baz
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
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Methods()).To.Have.Len(1)

	method := m.Methods()[0]

	expected, err := format.Source([]byte(`
 package foo
 
 func (m *mockFoo) Foo(arg0 int, arg1 string) (string, error) {
   m.Foo.called <- true
   m.Foo.input.arg0 <- arg0
   m.Foo.input.arg1 <- arg1
   return <-m.Foo.output.ret0, <-m.Foo.output.ret1
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}
