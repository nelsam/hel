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
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo() {
   m.FooCalled <- true
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))

	fields := method.Fields()
	expect(fields).To.Have.Len(1).Else.FailNow()

	expect(fields[0].Names[0].Name).To.Equal("FooCalled")
	ch, ok := fields[0].Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok := ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("bool")
}

func TestMockMethodParams(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
         Foo(foo, bar string, baz int)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(foo, bar string, baz int) {
   m.FooCalled <- true
   m.FooInput.Foo <- foo
   m.FooInput.Bar <- bar
   m.FooInput.Baz <- baz
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))

	fields := method.Fields()
	expect(fields).To.Have.Len(2)

	expect(fields[0].Names[0].Name).To.Equal("FooCalled")
	ch, ok := fields[0].Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok := ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("bool")

	expect(fields[1].Names[0].Name).To.Equal("FooInput")
	input, ok := fields[1].Type.(*ast.StructType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(input.Fields.List).To.Have.Len(2).Else.FailNow()

	fooBar := input.Fields.List[0]
	expect(fooBar.Names).To.Have.Len(2).Else.FailNow()
	expect(fooBar.Names[0].Name).To.Equal("Foo")
	expect(fooBar.Names[1].Name).To.Equal("Bar")
	ch, ok = fooBar.Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok = ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("string")

	baz := input.Fields.List[1]
	expect(baz.Names[0].Name).To.Equal("Baz")
	ch, ok = baz.Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok = ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("int")
}

func TestMockMethodReturns(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo() (foo, bar string, baz int)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo() (foo, bar string, baz int) {
   m.FooCalled <- true
   return <-m.FooOutput.Foo, <-m.FooOutput.Bar, <-m.FooOutput.Baz
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))

	fields := method.Fields()
	expect(fields).To.Have.Len(2)

	expect(fields[0].Names[0].Name).To.Equal("FooCalled")
	ch, ok := fields[0].Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok := ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("bool")

	expect(fields[1].Names[0].Name).To.Equal("FooOutput")
	input, ok := fields[1].Type.(*ast.StructType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(input.Fields.List).To.Have.Len(2).Else.FailNow()

	fooBar := input.Fields.List[0]
	expect(fooBar.Names).To.Have.Len(2).Else.FailNow()
	expect(fooBar.Names[0].Name).To.Equal("Foo")
	expect(fooBar.Names[1].Name).To.Equal("Bar")
	ch, ok = fooBar.Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok = ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("string")

	baz := input.Fields.List[1]
	expect(baz.Names[0].Name).To.Equal("Baz")
	ch, ok = baz.Type.(*ast.ChanType)
	expect(ok).To.Be.Ok().Else.FailNow()
	expect(ch.Dir).To.Equal(ast.SEND | ast.RECV)
	ident, ok = ch.Value.(*ast.Ident)
	expect(ident.Name).To.Equal("int")
}

func TestMockMethodWithBlockingReturn(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo()
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil().Else.FailNow()
	mock.SetBlockingReturn(true)
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo() () {
   m.FooCalled <- true
   <-m.FooOutput.BlockReturn
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

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
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(arg0 int, arg1 string) (string, error) {
   m.FooCalled <- true
   m.FooInput.Arg0 <- arg0
   m.FooInput.Arg1 <- arg1
   return <-m.FooOutput.Ret0, <-m.FooOutput.Ret1
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodLocalTypes(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo(bar bar.Bar, baz func(f Foo) error) (*Foo, func() Foo, error)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(bar bar.Bar, baz func(f Foo) error) (*Foo, func() Foo, error) {
   m.FooCalled <- true
   m.FooInput.Bar <- bar
   m.FooInput.Baz <- baz
   return <-m.FooOutput.Ret0, <-m.FooOutput.Ret1, <-m.FooOutput.Ret2
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))

	method.PrependLocalPackage("foo")

	expected, err = format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(bar bar.Bar, baz func(f foo.Foo) error) (*foo.Foo, func() foo.Foo, error) {
   m.FooCalled <- true
   m.FooInput.Bar <- bar
   m.FooInput.Baz <- baz
   return <-m.FooOutput.Ret0, <-m.FooOutput.Ret1, <-m.FooOutput.Ret2
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src = source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodLocalTypeNesting(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
   Foo(bar []Bar, bacon map[Foo]Bar) (baz []Baz, eggs map[Foo]Bar)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))
	method.PrependLocalPackage("foo")

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(bar []foo.Bar, bacon map[foo.Foo]foo.Bar) (baz []foo.Baz, eggs map[foo.Foo]foo.Bar) {
   m.FooCalled <- true
   m.FooInput.Bar <- bar
   m.FooInput.Bacon <- bacon
   return <-m.FooOutput.Baz, <-m.FooOutput.Eggs
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}

func TestMockMethodReceiverNameConflicts(t *testing.T) {
	expect := expect.New(t)

	spec := typeSpec(expect, `
 type Foo interface {
         Foo(m string)
 }`)
	mock, err := mocks.For(spec)
	expect(err).To.Be.Nil().Else.FailNow()
	method := mocks.MethodFor(mock, "Foo", method(expect, spec))

	expected, err := format.Source([]byte(`
 package foo

 func (m *mockFoo) Foo(m_ string) {
   m.FooCalled <- true
   m.FooInput.M <- m_
 }`))
	expect(err).To.Be.Nil().Else.FailNow()

	src := source(expect, "foo", []ast.Decl{method.Ast()}, nil)
	expect(src).To.Equal(string(expected))
}
