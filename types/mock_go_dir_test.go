package types_test

import "go/ast"

type mockGoDir struct {
	PathCalled chan bool
	PathOutput struct {
		ret0 chan string
	}
	PackagesCalled chan bool
	PackagesOutput struct {
		ret0 chan map[string]*ast.Package
	}
}

func newMockGoDir() *mockGoDir {
	m := &mockGoDir{}
	m.PathCalled = make(chan bool, 100)
	m.PathOutput.ret0 = make(chan string, 100)
	m.PackagesCalled = make(chan bool, 100)
	m.PackagesOutput.ret0 = make(chan map[string]*ast.Package, 100)
	return m
}
func (m *mockGoDir) Path() string {
	m.PathCalled <- true
	return <-m.PathOutput.ret0
}
func (m *mockGoDir) Packages() map[string]*ast.Package {
	m.PackagesCalled <- true
	return <-m.PackagesOutput.ret0
}
