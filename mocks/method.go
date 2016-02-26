package mocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

const (
	inputFmt  = "arg%d"
	outputFmt = "ret%d"
)

type Method struct {
	receiver   Mock
	name       string
	implements *ast.FuncType
}

func MethodFor(receiver Mock, name string, typ *ast.FuncType) Method {
	return Method{
		receiver:   receiver,
		name:       name,
		implements: typ,
	}
}

func (m Method) Ast() *ast.FuncDecl {
	f := &ast.FuncDecl{}
	f.Name = &ast.Ident{Name: m.name}
	f.Type = m.implements
	f.Recv = m.recv()
	f.Body = m.body()
	return f
}

func (m Method) recv() *ast.FieldList {
	return &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "m"}},
				Type: &ast.StarExpr{
					X: &ast.Ident{Name: m.receiver.Name()},
				},
			},
		},
	}
}

func (m Method) sendOn(receiver string, fields ...string) *ast.SendStmt {
	return &ast.SendStmt{Chan: selectors(receiver, fields...)}
}

func (m Method) called() ast.Stmt {
	stmt := m.sendOn("m", m.name+"Called")
	stmt.Value = &ast.Ident{Name: "true"}
	return stmt
}

func (m Method) params() []*ast.Field {
	for idx, f := range m.implements.Params.List {
		if f.Names == nil {
			// altering the field directly is okay here, since it's needed anyway
			f.Names = []*ast.Ident{{Name: fmt.Sprintf(inputFmt, idx)}}
		}
	}
	return m.implements.Params.List
}

func (m Method) results() []*ast.Field {
	if m.implements.Results == nil {
		if !*m.receiver.blockingReturn {
			return nil
		}
		return []*ast.Field{
			{
				Names: []*ast.Ident{
					{Name: "blockReturn"},
				},
				Type: &ast.Ident{Name: "bool"},
			},
			{
				Names: []*ast.Ident{
					{Name: "unblockReturn"},
				},
				Type: &ast.Ident{Name: "bool"},
			},
		}
	}
	fields := make([]*ast.Field, 0, len(m.implements.Results.List))
	for idx, f := range m.implements.Results.List {
		if f.Names == nil {
			// to avoid changing the method definition, make a copy
			copy := *f
			f = &copy
			f.Names = []*ast.Ident{{Name: fmt.Sprintf(outputFmt, idx)}}
		}
		fields = append(fields, f)
	}
	return fields
}

func (m Method) inputs() (stmts []ast.Stmt) {
	for _, input := range m.params() {
		for _, name := range input.Names {
			stmt := m.sendOn("m", m.name+"Input", strings.Title(name.String()))
			stmt.Value = &ast.Ident{Name: name.String()}
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}

func (m Method) PrependLocalPackage(name string) {
	m.prependPackage(name, m.implements.Results)
	m.prependPackage(name, m.implements.Params)
}

func (m Method) prependPackage(name string, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		ident, ok := field.Type.(*ast.Ident)
		if !ok {
			continue
		}
		if !unicode.IsUpper(rune(ident.String()[0])) {
			// Assume a built-in type, at least for now
			continue
		}
		field.Type = selectors(name, ident.String())
	}
}

func (m Method) recvFrom(receiver string, fields ...string) *ast.UnaryExpr {
	return &ast.UnaryExpr{Op: token.ARROW, X: selectors(receiver, fields...)}
}

func (m Method) returnsExprs() (exprs []ast.Expr) {
	for _, output := range m.results() {
		for _, name := range output.Names {
			exprs = append(exprs, m.recvFrom("m", m.name+"Output", strings.Title(name.String())))
		}
	}
	return exprs
}

func (m Method) returns() ast.Stmt {
	if m.implements.Results == nil {
		if !*m.receiver.blockingReturn {
			return nil
		}
		return m.blockingReturn()
	}
	return &ast.ReturnStmt{Results: m.returnsExprs()}
}

func (m Method) blockingReturn() *ast.SelectStmt {
	blockingCase := &ast.CommClause{
		Comm: &ast.ExprStmt{X: m.returnsExprs()[0]},
		Body: []ast.Stmt{ &ast.ExprStmt{ X: m.returnsExprs()[1] } },
	}
	return &ast.SelectStmt {
		Body: &ast.BlockStmt{
			List: []ast.Stmt{ blockingCase, &ast.CommClause{} },
		},
	}
}

func (m Method) body() *ast.BlockStmt {
	stmts := []ast.Stmt{m.called()}
	stmts = append(stmts, m.inputs()...)
	if returnStmt := m.returns(); returnStmt != nil {
		stmts = append(stmts, m.returns())
	}
	return &ast.BlockStmt{
		List: stmts,
	}
}
