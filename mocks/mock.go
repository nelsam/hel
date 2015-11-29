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

func (m *Mock) Ast() *ast.TypeSpec {
	return nil
}
