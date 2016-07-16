// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
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

func (m Mock) makeChan(typ ast.Expr, size int) *ast.CallExpr {
	switch src := typ.(type) {
	case *ast.ChanType:
		switch src.Dir {
		case ast.SEND, ast.RECV:
			typ = &ast.ParenExpr{X: src}
		}
	case *ast.Ellipsis:
		// The actual value of variadic types is a slice
		typ = &ast.ArrayType{Elt: src.Elt}
	}
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "make"},
		Args: []ast.Expr{
			&ast.ChanType{Dir: ast.SEND | ast.RECV, Value: typ},
			&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(size)},
		},
	}
}

func (m Mock) paramChanInit(method Method, chanSize int) []ast.Stmt {
	return m.typeChanInit(method.name+"Input", method.params(), chanSize)
}

func (m Mock) returnChanInit(method Method, chanSize int) []ast.Stmt {
	return m.typeChanInit(method.name+"Output", method.results(), chanSize)
}

func (m Mock) typeChanInit(fieldName string, fields []*ast.Field, chanSize int) (inputInits []ast.Stmt) {
	for _, field := range fields {
		for _, name := range field.Names {
			inputInits = append(inputInits, &ast.AssignStmt{
				Lhs: []ast.Expr{selectors("m", fieldName, strings.Title(name.String()))},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{m.makeChan(field.Type, chanSize)},
			})
		}
	}
	return inputInits
}

func (m Mock) chanInit(method Method, chanSize int) []ast.Stmt {
	stmts := []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{selectors("m", method.name+"Called")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{m.makeChan(&ast.Ident{Name: "bool"}, chanSize)},
		},
	}
	stmts = append(stmts, m.paramChanInit(method, chanSize)...)
	stmts = append(stmts, m.returnChanInit(method, chanSize)...)
	return stmts
}

func (m Mock) constructorBody(chanSize int) []ast.Stmt {
	structAlloc := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "m"}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: &ast.Ident{Name: m.Name()}}}},
	}
	stmts := []ast.Stmt{structAlloc}
	for _, method := range m.Methods() {
		stmts = append(stmts, m.chanInit(method, chanSize)...)
	}
	stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "m"}}})
	return stmts
}

func (m Mock) chanStruct(list []*ast.Field) *ast.StructType {
	typ := &ast.StructType{Fields: &ast.FieldList{}}
	for _, f := range list {
		chanValType := f.Type
		switch src := chanValType.(type) {
		case *ast.ChanType:
			// Receive-only channels require parens, and it seems unfair to leave
			// out send-only channels.
			switch src.Dir {
			case ast.SEND, ast.RECV:
				chanValType = &ast.ParenExpr{X: src}
			}
		case *ast.Ellipsis:
			// The actual value of variadic types is a slice
			chanValType = &ast.ArrayType{Elt: src.Elt}
		}
		names := make([]*ast.Ident, 0, len(f.Names))
		for _, name := range f.Names {
			newName := &ast.Ident{}
			*newName = *name
			names = append(names, newName)
			newName.Name = strings.Title(newName.Name)
		}
		typ.Fields.List = append(typ.Fields.List, &ast.Field{
			Names: names,
			Type: &ast.ChanType{
				Dir:   ast.SEND | ast.RECV,
				Value: chanValType,
			},
		})
	}
	return typ
}

func (m Mock) methodTypes(method Method) []*ast.Field {
	fields := []*ast.Field{
		{
			Names: []*ast.Ident{{Name: method.name + "Called"}},
			Type: &ast.ChanType{
				Dir:   ast.SEND | ast.RECV,
				Value: &ast.Ident{Name: "bool"},
			},
		},
	}
	if len(method.params()) > 0 {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{{Name: method.name + "Input"}},
			Type:  m.chanStruct(method.params()),
		})
	}
	if len(method.results()) > 0 {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{{Name: method.name + "Output"}},
			Type:  m.chanStruct(method.results()),
		})
	}
	return fields
}

func (m Mock) structType() *ast.StructType {
	structType := &ast.StructType{Fields: &ast.FieldList{}}
	for _, method := range m.Methods() {
		structType.Fields.List = append(structType.Fields.List, m.methodTypes(method)...)
	}
	return structType
}
