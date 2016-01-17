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
	ImportCalled chan bool
	ImportInput  struct {
		path, pkg chan string
	}
	ImportOutput struct {
		ret0 chan *ast.Package
		ret1 chan error
	}
}

func newMockGoDir() *mockGoDir {
	m := &mockGoDir{}
	m.PathCalled = make(chan bool, 100)
	m.PathOutput.ret0 = make(chan string, 100)
	m.PackagesCalled = make(chan bool, 100)
	m.PackagesOutput.ret0 = make(chan map[string]*ast.Package, 100)
	m.ImportCalled = make(chan bool, 100)
	m.ImportInput.path = make(chan string, 100)
	m.ImportInput.pkg = make(chan string, 100)
	m.ImportOutput.ret0 = make(chan *ast.Package, 100)
	m.ImportOutput.ret1 = make(chan error, 100)
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
func (m *mockGoDir) Import(path, pkg string) (*ast.Package, error) {
	m.ImportCalled <- true
	m.ImportInput.path <- path
	m.ImportInput.pkg <- pkg
	return <-m.ImportOutput.ret0, <-m.ImportOutput.ret1
}
