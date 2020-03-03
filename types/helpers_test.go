// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
)

const packagePrefix = "package foo\n\n"

func parse(expect expectation, code string) *ast.File {
	f, err := parser.ParseFile(token.NewFileSet(), "", packagePrefix+code, 0)
	expect(err).To(not(haveOccurred()))
	return f
}
