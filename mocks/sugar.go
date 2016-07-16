// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks

import "go/ast"

func selectors(receiver string, fields ...string) *ast.SelectorExpr {
	if len(fields) == 0 {
		return nil
	}
	selector := &ast.SelectorExpr{
		X:   &ast.Ident{Name: receiver},
		Sel: &ast.Ident{Name: fields[0]},
	}
	for _, field := range fields[1:] {
		selector = &ast.SelectorExpr{
			X:   selector,
			Sel: &ast.Ident{Name: field},
		}
	}
	return selector
}
