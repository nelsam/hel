// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

type Mock struct {
	typeName       string
	implements     *ast.InterfaceType
	blockingReturn *bool
}

func For(typ *ast.TypeSpec) (Mock, error) {
	inter, ok := typ.Type.(*ast.InterfaceType)
	if !ok {
		return Mock{}, fmt.Errorf("TypeSpec.Type expected to be *ast.InterfaceType, was %T", typ.Type)
	}
	var blockingReturn bool
	m := Mock{
		typeName:       typ.Name.String(),
		implements:     inter,
		blockingReturn: &blockingReturn,
	}
	return m, nil
}

func (m Mock) Name() string {
	return "mock" + strings.ToUpper(m.typeName[0:1]) + m.typeName[1:]
}

func (m Mock) Methods() (methods []Method) {
	for _, method := range m.implements.Methods.List {
		switch methodType := method.Type.(type) {
		case *ast.FuncType:
			methods = append(methods, MethodFor(m, method.Names[0].String(), methodType))
		}
	}
	return
}

func (m Mock) PrependLocalPackage(name string) {
	for _, m := range m.Methods() {
		m.PrependLocalPackage(name)
	}
}

func (m Mock) SetBlockingReturn(blockingReturn bool) {
	*m.blockingReturn = blockingReturn
}

func (m Mock) Constructor(chanSize int) *ast.FuncDecl {
	decl := &ast.FuncDecl{}
	typeRunes := []rune(m.Name())
	typeRunes[0] = unicode.ToUpper(typeRunes[0])
	decl.Name = &ast.Ident{Name: "new" + string(typeRunes)}
	decl.Type = &ast.FuncType{
		Results: &ast.FieldList{List: []*ast.Field{{
			Type: &ast.StarExpr{
				X: &ast.Ident{Name: m.Name()},
			},
		}}},
	}
	decl.Body = &ast.BlockStmt{List: m.constructorBody(chanSize)}
	return decl
}

func (m Mock) Decl() *ast.GenDecl {
	spec := &ast.TypeSpec{}
	spec.Name = &ast.Ident{Name: m.Name()}
	spec.Type = m.structType()
	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{spec},
	}
}

func (m Mock) Ast(chanSize int) []ast.Decl {
	decls := []ast.Decl{
		m.Decl(),
		m.Constructor(chanSize),
	}
	for _, method := range m.Methods() {
		decls = append(decls, method.Ast())
	}
	return decls
}

func (m Mock) constructorBody(chanSize int) []ast.Stmt {
	structAlloc := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "m"}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: &ast.Ident{Name: m.Name()}}}},
	}
	stmts := []ast.Stmt{structAlloc}
	for _, method := range m.Methods() {
		stmts = append(stmts, method.chanInit(chanSize)...)
	}
	stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "m"}}})
	return stmts
}

func (m Mock) structType() *ast.StructType {
	structType := &ast.StructType{Fields: &ast.FieldList{}}
	for _, method := range m.Methods() {
		structType.Fields.List = append(structType.Fields.List, method.Fields()...)
	}
	return structType
}
