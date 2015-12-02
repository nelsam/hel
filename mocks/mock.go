package mocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"
)

type TypeFinder interface {
	ExportedTypes() []*ast.TypeSpec
}

type Mock struct {
	typeName   string
	implements *ast.InterfaceType
}

func New(typ *ast.TypeSpec) (*Mock, error) {
	inter, ok := typ.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("TypeSpec.Type expected to be *ast.InterfaceType, was %T", typ.Type)
	}
	m := new(Mock)
	m.typeName = typ.Name.String()
	m.implements = inter
	return m, nil
}

func (m *Mock) Name() string {
	return "mock" + strings.ToUpper(m.typeName[0:1]) + m.typeName[1:]
}

func (m *Mock) Methods() (methods []Method) {
	for _, method := range m.implements.Methods.List {
		switch methodType := method.Type.(type) {
		case *ast.FuncType:
			newMethod := Method{
				receiver:   m,
				name:       method.Names[0].String(),
				implements: methodType,
			}
			methods = append(methods, newMethod)
		}
	}
	return
}

func (m *Mock) Constructor(chanSize int) *ast.FuncDecl {
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

// TODO: this should return a *ast.GenDecl
func (m *Mock) Ast() *ast.TypeSpec {
	spec := &ast.TypeSpec{}
	spec.Name = &ast.Ident{Name: m.Name()}
	spec.Type = m.structType()
	return spec
}

func (m *Mock) makeChan(typ ast.Expr, size int) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "make"},
		Args: []ast.Expr{
			&ast.ChanType{Dir: ast.SEND | ast.RECV, Value: typ},
			&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(size)},
		},
	}
}

func (m *Mock) paramChanInit(method Method, chanSize int) []ast.Stmt {
	return m.typeChanInit(method.name+"Input", method.params(), chanSize)
}

func (m *Mock) returnChanInit(method Method, chanSize int) []ast.Stmt {
	return m.typeChanInit(method.name+"Output", method.results(), chanSize)
}

func (m *Mock) typeChanInit(fieldName string, fields []*ast.Field, chanSize int) (inputInits []ast.Stmt) {
	for _, field := range fields {
		for _, name := range field.Names {
			inputInits = append(inputInits, &ast.AssignStmt{
				Lhs: []ast.Expr{selectors("m", fieldName, name.String())},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{m.makeChan(field.Type, chanSize)},
			})
		}
	}
	return inputInits
}

func (m *Mock) chanInit(method Method, chanSize int) []ast.Stmt {
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

func (m *Mock) constructorBody(chanSize int) []ast.Stmt {
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

func (m *Mock) chanStruct(list []*ast.Field) *ast.StructType {
	typ := &ast.StructType{Fields: &ast.FieldList{}}
	for _, f := range list {
		typ.Fields.List = append(typ.Fields.List, &ast.Field{
			Names: f.Names,
			Type: &ast.ChanType{
				Dir:   ast.SEND | ast.RECV,
				Value: f.Type,
			},
		})
	}
	return typ
}

func (m *Mock) methodTypes(method Method) []*ast.Field {
	return []*ast.Field{
		{
			Names: []*ast.Ident{{Name: method.name + "Called"}},
			Type: &ast.ChanType{
				Dir:   ast.SEND | ast.RECV,
				Value: &ast.Ident{Name: "bool"},
			},
		},
		{
			Names: []*ast.Ident{{Name: method.name + "Input"}},
			Type:  m.chanStruct(method.params()),
		},
		{
			Names: []*ast.Ident{{Name: method.name + "Output"}},
			Type:  m.chanStruct(method.results()),
		},
	}
}

func (m *Mock) structType() *ast.StructType {
	structType := &ast.StructType{Fields: &ast.FieldList{}}
	for _, method := range m.Methods() {
		structType.Fields.List = append(structType.Fields.List, m.methodTypes(method)...)
	}
	return structType
}
