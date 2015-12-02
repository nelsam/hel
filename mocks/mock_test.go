package mocks_test

import (
	"go/ast"
	"go/format"
	"go/token"
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/hel/mocks"
)

const packagePrefix = "package foo\n\n"

func TestNewErrorsForNonInterfaceTypes(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, "type Foo func()")
	_, err := mocks.New(spec)
	expect(err).Not.To.Be.Nil()
	expect(err.Error()).To.Equal("TypeSpec.Type expected to be *ast.InterfaceType, was *ast.FuncType")
}

func TestMockName(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, "type Foo interface{}")
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Name()).To.Equal("mockFoo")
}

func TestMockAst(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Foo(foo string) int
  Bar(bar int) string
 }
 `)
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo
 
 type mockFoo struct {
  FooCalled chan bool
  FooInput struct {
   foo chan string
  }
  FooOutput struct {
   ret0 chan int
  }  
  BarCalled chan bool
  BarInput struct {
   bar chan int
  }
  BarOutput struct {
   ret0 chan string
  }
 }
 `))
	expect(err).To.Be.Nil()

	decls := []ast.Decl{
		&ast.GenDecl{
			Tok:   token.TYPE,
			Specs: []ast.Spec{m.Ast()},
		},
	}
	src := source(expect, "foo", decls, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockConstructor(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Foo(foo string) int
  Bar(bar int) string
 }
 `)
	m, err := mocks.New(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo
 
 func newMockFoo() *mockFoo {
  m := &mockFoo{}
  m.FooCalled = make(chan bool, 300)
  m.FooInput.foo = make(chan string, 300)
  m.FooOutput.ret0 = make(chan int, 300)
  m.BarCalled = make(chan bool, 300)
  m.BarInput.bar = make(chan int, 300)
  m.BarOutput.ret0 = make(chan string, 300)
  return m
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Constructor(300)}, nil)
	expect(src).To.Equal(string(expected))
}
