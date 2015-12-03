package mocks_test

import "go/ast"

type mockTypeFinder struct {
	ExportedTypesCalled chan bool
	ExportedTypesOutput struct {
		ret0 chan []*ast.TypeSpec
	}
}

func newMockTypeFinder() *mockTypeFinder {
	m := &mockTypeFinder{}
	m.ExportedTypesCalled = make(chan bool, 300)
	m.ExportedTypesOutput.ret0 = make(chan []*ast.TypeSpec, 300)
	return m
}

func (m *mockTypeFinder) ExportedTypes() []*ast.TypeSpec {
	m.ExportedTypesCalled <- true
	return <-m.ExportedTypesOutput.ret0
}
