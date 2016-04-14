package mocks_test

import (
	"go/ast"
	"go/format"
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/hel/mocks"
)

func TestNewErrorsForNonInterfaceTypes(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, "type Foo func()")
	_, err := mocks.For(spec)
	expect(err).Not.To.Be.Nil()
	expect(err.Error()).To.Equal("TypeSpec.Type expected to be *ast.InterfaceType, was *ast.FuncType")
}

func TestMockName(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, "type Foo interface{}")
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()
	expect(m.Name()).To.Equal("mockFoo")
}

func TestMockTypeDecl(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Foo(foo string) int
  Bar(bar int) Foo
  Baz()
  Bacon(func(Eggs) Eggs) func(Eggs) Eggs
 }
 `)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo

 type mockFoo struct {
  FooCalled chan bool
  FooInput struct {
   Foo chan string
  }
  FooOutput struct {
   Ret0 chan int
  }
  BarCalled chan bool
  BarInput struct {
   Bar chan int
  }
  BarOutput struct {
   Ret0 chan Foo
  }
  BazCalled chan bool
  BaconCalled chan bool
  BaconInput struct {
    Arg0 chan func(Eggs) Eggs
  }
  BaconOutput struct {
    Ret0 chan func(Eggs) Eggs
  }
 }
 `))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Decl()}, nil)
	expect(src).To.Equal(string(expected))

	m.PrependLocalPackage("foo")

	expected, err = format.Source([]byte(`
 package foo

 type mockFoo struct {
  FooCalled chan bool
  FooInput struct {
   Foo chan string
  }
  FooOutput struct {
   Ret0 chan int
  }
  BarCalled chan bool
  BarInput struct {
   Bar chan int
  }
  BarOutput struct {
   Ret0 chan foo.Foo
  }
  BazCalled chan bool
  BaconCalled chan bool
  BaconInput struct {
    Arg0 chan func(foo.Eggs) foo.Eggs
  }
  BaconOutput struct {
    Ret0 chan func(foo.Eggs) foo.Eggs
  }
 }
 `))
	expect(err).To.Be.Nil()

	src = source(expect, "foo", []ast.Decl{m.Decl()}, nil)
	expect(src).To.Equal(string(expected))

	m.SetBlockingReturn(true)

	expected, err = format.Source([]byte(`
 package foo

 type mockFoo struct {
  FooCalled chan bool
  FooInput struct {
   Foo chan string
  }
  FooOutput struct {
   Ret0 chan int
  }
  BarCalled chan bool
  BarInput struct {
   Bar chan int
  }
  BarOutput struct {
   Ret0 chan foo.Foo
  }
  BazCalled chan bool
  BazOutput struct {
   BlockReturn chan bool
  }
 }
 `))
	expect(err).To.Be.Nil()

	src = source(expect, "foo", []ast.Decl{m.Decl()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockTypeDecl_DirectionalChansGetParens(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Foo(foo chan<- int) <-chan int
 }
 `)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo

 type mockFoo struct {
  FooCalled chan bool
  FooInput struct {
   Foo chan (chan<- int)
  }
  FooOutput struct {
   Ret0 chan (<-chan int)
  }
 }
 `))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Decl()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockTypeDecl_VariadicMethods(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
type Foo interface {
 Foo(foo ...int)
}
`)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo

 type mockFoo struct {
  FooCalled chan bool
  FooInput struct {
   Foo chan []int
  }
 }
 `))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Decl()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockTypeDecl_ParamsWithoutTypes(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
type Foo interface {
 Foo(foo, bar string)
}
`)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
package foo

type mockFoo struct {
 FooCalled chan bool
 FooInput struct {
  Foo, Bar chan string
 }
}
`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Decl()}, nil)
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
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo

 func newMockFoo() *mockFoo {
  m := &mockFoo{}
  m.FooCalled = make(chan bool, 300)
  m.FooInput.Foo = make(chan string, 300)
  m.FooOutput.Ret0 = make(chan int, 300)
  m.BarCalled = make(chan bool, 300)
  m.BarInput.Bar = make(chan int, 300)
  m.BarOutput.Ret0 = make(chan string, 300)
  return m
 }`))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Constructor(300)}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockConstructor_DirectionalChansGetParens(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Foo(foo chan<- int) <-chan int
 }
 `)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo

 func newMockFoo() *mockFoo {
  m := &mockFoo{}
  m.FooCalled = make(chan bool, 200)
  m.FooInput.Foo = make(chan (chan<- int), 200)
  m.FooOutput.Ret0 = make(chan (<-chan int), 200)
  return m
 }
 `))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Constructor(200)}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockConstructor_VariadicParams(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Foo(foo ...int)
 }
 `)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expected, err := format.Source([]byte(`
 package foo

 func newMockFoo() *mockFoo {
  m := &mockFoo{}
  m.FooCalled = make(chan bool, 200)
  m.FooInput.Foo = make(chan []int, 200)
  return m
 }
 `))
	expect(err).To.Be.Nil()

	src := source(expect, "foo", []ast.Decl{m.Constructor(200)}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockAst(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
  Bar(bar string)
  Baz() (baz int)
 }`)
	m, err := mocks.For(spec)
	expect(err).To.Be.Nil()
	expect(m).Not.To.Be.Nil()

	expect(m.Methods()).To.Have.Len(2)

	decls := m.Ast(300)
	expect(decls).To.Have.Len(4)
	expect(decls[0]).To.Equal(m.Decl())
	expect(decls[1]).To.Equal(m.Constructor(300))
	expect(decls[2]).To.Equal(m.Methods()[0].Ast())
	expect(decls[3]).To.Equal(m.Methods()[1].Ast())
}
