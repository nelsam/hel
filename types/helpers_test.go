package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/a8m/expect"
)

const packagePrefix = "package foo\n\n"

func parse(expect func(interface{}) *expect.Expect, code string) *ast.File {
	f, err := parser.ParseFile(token.NewFileSet(), "", packagePrefix+code, 0)
	expect(err).To.Be.Nil()
	expect(f).Not.To.Be.Nil()
	return f
}
