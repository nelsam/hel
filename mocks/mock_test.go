package mocks_test

import (
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
