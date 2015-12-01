package mocks

import (
	"fmt"
	"go/ast"
	"strings"
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

func (m *Mock) methodStruct(method Method) *ast.StructType {
	return &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: "called"}},
					Type: &ast.ChanType{
						Dir:   ast.SEND | ast.RECV,
						Value: &ast.Ident{Name: "bool"},
					},
				},
				{
					Names: []*ast.Ident{{Name: "input"}},
					Type:  m.chanStruct(method.params()),
				},
				{
					Names: []*ast.Ident{{Name: "output"}},
					Type:  m.chanStruct(method.results()),
				},
			},
		},
	}
}

func (m *Mock) structType() *ast.StructType {
	methodsType := &ast.StructType{
		Fields: &ast.FieldList{},
	}
	for _, method := range m.Methods() {
		field := &ast.Field{
			Names: []*ast.Ident{{Name: method.name}},
			Type:  m.methodStruct(method),
		}
		methodsType.Fields.List = append(methodsType.Fields.List, field)
	}
	structType := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: "methods"}},
					Type:  methodsType,
				},
			},
		},
	}
	return structType
}

func (m *Mock) Ast() *ast.TypeSpec {
	spec := &ast.TypeSpec{}
	spec.Name = &ast.Ident{Name: m.Name()}
	spec.Type = m.structType()
	return spec
}
